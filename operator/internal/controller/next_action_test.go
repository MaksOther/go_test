package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/MaksOther/go_test/operator/api/v1alpha1"
)

func TestNextAction(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name        string
		finalizers  []string
		annotations map[string]string
		deleting    bool
		status      v1alpha1.ManagedDatabaseStatus
		want        action
	}{
		{
			name: "new resource gets a finalizer first",
			want: actionAddFinalizer,
		},
		{
			name:       "resource with finalizer and empty status is created",
			finalizers: []string{finalizerName},
			want:       actionCreate,
		},
		{
			name:       "create was requested but result was not saved",
			finalizers: []string{finalizerName},
			status:     v1alpha1.ManagedDatabaseStatus{State: v1alpha1.StateCreating, CreateRequestedAt: &now},
			want:       actionMarkUnknown,
		},
		{
			name:       "unknown state is never created again",
			finalizers: []string{finalizerName},
			status:     v1alpha1.ManagedDatabaseStatus{State: v1alpha1.StateUnknown, CreateRequestedAt: &now},
			want:       actionNothing,
		},
		{
			name:        "unknown state with database id annotation is adopted",
			finalizers:  []string{finalizerName},
			annotations: map[string]string{databaseIDAnnotation: "db-12345678"},
			status:      v1alpha1.ManagedDatabaseStatus{State: v1alpha1.StateUnknown, CreateRequestedAt: &now},
			want:        actionAdopt,
		},
		{
			name:       "rejected create is not retried",
			finalizers: []string{finalizerName},
			status:     v1alpha1.ManagedDatabaseStatus{State: v1alpha1.StateFailed},
			want:       actionNothing,
		},
		{
			name:       "provisioning database is refreshed",
			finalizers: []string{finalizerName},
			status:     v1alpha1.ManagedDatabaseStatus{State: v1alpha1.StateProvisioning, DatabaseID: "db-12345678", CreateRequestedAt: &now},
			want:       actionRefresh,
		},
		{
			name:       "ready database needs nothing",
			finalizers: []string{finalizerName},
			status:     v1alpha1.ManagedDatabaseStatus{State: v1alpha1.StateReady, DatabaseID: "db-12345678"},
			want:       actionNothing,
		},
		{
			name:        "annotation is ignored when database id is already known",
			finalizers:  []string{finalizerName},
			annotations: map[string]string{databaseIDAnnotation: "db-87654321"},
			status:      v1alpha1.ManagedDatabaseStatus{State: v1alpha1.StateReady, DatabaseID: "db-12345678"},
			want:        actionNothing,
		},
		{
			name:       "deleted resource with database id deletes the database",
			finalizers: []string{finalizerName},
			deleting:   true,
			status:     v1alpha1.ManagedDatabaseStatus{State: v1alpha1.StateReady, DatabaseID: "db-12345678"},
			want:       actionDelete,
		},
		{
			name:       "deleted resource without database id is released",
			finalizers: []string{finalizerName},
			deleting:   true,
			status:     v1alpha1.ManagedDatabaseStatus{State: v1alpha1.StateUnknown, CreateRequestedAt: &now},
			want:       actionRemoveFinalizer,
		},
		{
			name:     "deleted resource without finalizer is never created",
			deleting: true,
			want:     actionNothing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database := &v1alpha1.ManagedDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "orders",
					Finalizers:  tt.finalizers,
					Annotations: tt.annotations,
				},
				Status: tt.status,
			}
			if tt.deleting {
				database.DeletionTimestamp = &now
			}

			if got := nextAction(database); got != tt.want {
				t.Errorf("nextAction() = %s, want %s", got, tt.want)
			}
		})
	}
}
