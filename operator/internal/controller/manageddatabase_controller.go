package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/MaksOther/go_test/operator/api/v1alpha1"
	"github.com/MaksOther/go_test/operator/internal/provisioner"
)

const (
	refreshInterval = 10 * time.Second
	retryInterval   = 5 * time.Second
)

type ManagedDatabaseReconciler struct {
	client.Client
	Provisioner *provisioner.Client
}

func (r *ManagedDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var database v1alpha1.ManagedDatabase
	if err := r.Get(ctx, req.NamespacedName, &database); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var result ctrl.Result
	var err error

	switch nextAction(&database) {
	case actionCreate:
		result, err = r.createDatabase(ctx, &database)
	case actionMarkUnknown:
		err = r.markUnknown(ctx, &database)
	case actionRefresh:
		result, err = r.refreshDatabase(ctx, &database)
	}

	if apierrors.IsConflict(err) {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	return result, err
}

func (r *ManagedDatabaseReconciler) createDatabase(ctx context.Context, database *v1alpha1.ManagedDatabase) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	now := metav1.Now()
	database.Status.State = v1alpha1.StateCreating
	database.Status.Message = "sending create request to the provisioning API"
	database.Status.CreateRequestedAt = &now
	if err := r.Status().Update(ctx, database); err != nil {
		return ctrl.Result{}, err
	}
	beforeCreate := database.DeepCopy()

	created, err := r.Provisioner.CreateDatabase(ctx, provisioner.CreateRequest{
		Name:   externalName(database),
		Engine: database.Spec.Engine,
		SizeGB: database.Spec.SizeGB,
	})

	result := ctrl.Result{}
	switch {
	case err == nil:
		log.Info("database created", "databaseID", created.ID)
		database.Status.DatabaseID = created.ID
		database.Status.State = v1alpha1.StateProvisioning
		database.Status.Message = "database is being provisioned"
		result.RequeueAfter = refreshInterval
	case errors.Is(err, provisioner.ErrUnavailable):
		database.Status.State = v1alpha1.StatePending
		database.Status.Message = "provisioning API is unavailable, will retry"
		database.Status.CreateRequestedAt = nil
		result.RequeueAfter = retryInterval
	case errors.Is(err, provisioner.ErrBadRequest):
		database.Status.State = v1alpha1.StateFailed
		database.Status.Message = err.Error()
		database.Status.CreateRequestedAt = nil
	default:
		log.Error(err, "create request failed, the database may exist")
		database.Status.State = v1alpha1.StateUnknown
		database.Status.Message = fmt.Sprintf("create request failed (%v), the database may or may not exist. "+
			"Not retrying to avoid a duplicate", err)
	}

	if err := r.Status().Patch(ctx, database, client.MergeFrom(beforeCreate)); err != nil {
		if database.Status.DatabaseID != "" {
			log.Error(err, "failed to save database ID", "databaseID", database.Status.DatabaseID)
		}
		return ctrl.Result{}, err
	}
	return result, nil
}

func (r *ManagedDatabaseReconciler) markUnknown(ctx context.Context, database *v1alpha1.ManagedDatabase) error {
	database.Status.State = v1alpha1.StateUnknown
	database.Status.Message = "a create request was sent but its result was never saved, the database may or may not exist. " +
		"Not retrying to avoid a duplicate"
	return r.Status().Update(ctx, database)
}

func (r *ManagedDatabaseReconciler) refreshDatabase(ctx context.Context, database *v1alpha1.ManagedDatabase) (ctrl.Result, error) {
	beforeRefresh := database.DeepCopy()
	result := ctrl.Result{}

	external, err := r.Provisioner.GetDatabase(ctx, database.Status.DatabaseID)
	switch {
	case errors.Is(err, provisioner.ErrNotFound):
		database.Status.State = v1alpha1.StateFailed
		database.Status.Message = fmt.Sprintf("database %s does not exist in the provisioning API", database.Status.DatabaseID)
	case err != nil:
		return ctrl.Result{}, err
	case external.State == provisioner.StateReady:
		database.Status.State = v1alpha1.StateReady
		database.Status.Endpoint = external.Endpoint
		database.Status.Message = "database is ready"
	case external.State == provisioner.StateFailed:
		database.Status.State = v1alpha1.StateFailed
		database.Status.Message = "provisioning failed in the provisioning API, delete this resource to clean up"
	default:
		database.Status.State = v1alpha1.StateProvisioning
		database.Status.Message = "database is being provisioned"
		result.RequeueAfter = refreshInterval
	}

	if err := r.Status().Patch(ctx, database, client.MergeFrom(beforeRefresh)); err != nil {
		return ctrl.Result{}, err
	}
	return result, nil
}

func externalName(database *v1alpha1.ManagedDatabase) string {
	return fmt.Sprintf("%s-%s", database.Name, string(database.UID)[:8])
}

func (r *ManagedDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ManagedDatabase{}).
		Named("manageddatabase").
		Complete(r)
}
