package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatePending      = "Pending"
	StateCreating     = "Creating"
	StateProvisioning = "Provisioning"
	StateReady        = "Ready"
	StateFailed       = "Failed"
	StateUnknown      = "Unknown"
	StateDeleting     = "Deleting"
)

type ManagedDatabaseSpec struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="engine cannot be changed after creation"
	Engine string `json:"engine"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="sizeGB cannot be changed, the provisioning API has no resize operation"
	SizeGB int `json:"sizeGB"`
}

type ManagedDatabaseStatus struct {
	State             string       `json:"state,omitempty"`
	Message           string       `json:"message,omitempty"`
	DatabaseID        string       `json:"databaseID,omitempty"`
	Endpoint          string       `json:"endpoint,omitempty"`
	CreateRequestedAt *metav1.Time `json:"createRequestedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mdb
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.engine`
// +kubebuilder:printcolumn:name="Size",type=integer,JSONPath=`.spec.sizeGB`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ManagedDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedDatabaseSpec   `json:"spec,omitempty"`
	Status ManagedDatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ManagedDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedDatabase{}, &ManagedDatabaseList{})
}
