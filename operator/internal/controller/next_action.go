package controller

import (
	"github.com/MaksOther/go_test/operator/api/v1alpha1"
)

type action string

const (
	actionNothing     action = "Nothing"
	actionCreate      action = "Create"
	actionMarkUnknown action = "MarkUnknown"
	actionRefresh     action = "Refresh"
)

func nextAction(database *v1alpha1.ManagedDatabase) action {
	status := database.Status

	if status.DatabaseID != "" {
		if status.State == v1alpha1.StateReady || status.State == v1alpha1.StateFailed {
			return actionNothing
		}
		return actionRefresh
	}

	if status.State == v1alpha1.StateFailed || status.State == v1alpha1.StateUnknown {
		return actionNothing
	}

	if status.CreateRequestedAt != nil {
		return actionMarkUnknown
	}

	return actionCreate
}
