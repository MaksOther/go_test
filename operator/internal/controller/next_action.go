package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/MaksOther/go_test/operator/api/v1alpha1"
)

const (
	finalizerName        = "demo.example.com/external-database"
	databaseIDAnnotation = "demo.example.com/database-id"
)

type action string

const (
	actionNothing         action = "Nothing"
	actionAddFinalizer    action = "AddFinalizer"
	actionCreate          action = "Create"
	actionMarkUnknown     action = "MarkUnknown"
	actionRefresh         action = "Refresh"
	actionAdopt           action = "Adopt"
	actionDelete          action = "Delete"
	actionRemoveFinalizer action = "RemoveFinalizer"
)

func nextAction(database *v1alpha1.ManagedDatabase) action {
	status := database.Status

	if !database.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(database, finalizerName) {
			return actionNothing
		}
		if status.DatabaseID != "" {
			return actionDelete
		}
		return actionRemoveFinalizer
	}

	if !controllerutil.ContainsFinalizer(database, finalizerName) {
		return actionAddFinalizer
	}

	if status.DatabaseID != "" {
		if status.State == v1alpha1.StateReady || status.State == v1alpha1.StateFailed {
			return actionNothing
		}
		return actionRefresh
	}

	if database.Annotations[databaseIDAnnotation] != "" {
		return actionAdopt
	}

	if status.State == v1alpha1.StateFailed || status.State == v1alpha1.StateUnknown {
		return actionNothing
	}

	if status.CreateRequestedAt != nil {
		return actionMarkUnknown
	}

	return actionCreate
}
