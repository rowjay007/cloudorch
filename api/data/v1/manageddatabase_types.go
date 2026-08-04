/*
ManagedDatabase is the Schema for the manageddatabases API.
It defines a managed database instance (RDS / Cloud SQL / Azure DB).
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ManagedDatabaseSpec defines the desired state of ManagedDatabase.
type ManagedDatabaseSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=aws;gcp;azure
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=20
	Region string `json:"region"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=postgres;mysql;mariadb;sqlserver;oracle
	Engine string `json:"engine"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	InstanceClass string `json:"instanceClass"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=false
	MultiAZ bool `json:"multiAZ"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=365
	BackupRetentionDays int32 `json:"backupRetentionDays"`

	// +kubebuilder:validation:Required
	CredentialSecretRef SecretReference `json:"credentialSecretRef"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="03:00-04:00"
	MaintenanceWindow string `json:"maintenanceWindow,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	StorageEncrypted bool `json:"storageEncrypted,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=20
	// +kubebuilder:validation:Maximum=65536
	StorageGB int32 `json:"storageGB,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// SecretReference references a Kubernetes Secret containing database credentials.
type SecretReference struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

// ManagedDatabaseStatus defines the observed state of ManagedDatabase.
type ManagedDatabaseStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// +optional
	Port int32 `json:"port,omitempty"`

	// +optional
	DBInstanceIdentifier string `json:"dbInstanceIdentifier,omitempty"`

	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Synced bool `json:"synced,omitempty"`

	// +optional
	EstimatedMonthlyCost float64 `json:"estimatedMonthlyCost,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="ENGINE",type=string,JSONPath=`.spec.engine`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// ManagedDatabase is the Schema for the manageddatabases API.
type ManagedDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedDatabaseSpec   `json:"spec,omitempty"`
	Status ManagedDatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedDatabaseList contains a list of ManagedDatabase.
type ManagedDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedDatabase{}, &ManagedDatabaseList{})
}
