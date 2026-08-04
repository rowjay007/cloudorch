/*
SecretStore is the Schema for the secretstores API.
It defines a cloud secrets manager (AWS Secrets Manager / GCP Secret Manager / Azure Key Vault).
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretStoreSpec defines the desired state of SecretStore.
type SecretStoreSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=aws;gcp;azure
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=20
	Region string `json:"region"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=^[a-z][a-z0-9-]*[a-z0-9]$
	StoreName string `json:"storeName"`

	// +kubebuilder:validation:Required
	SecretRefs []SecretReference `json:"secretRefs"`

	// +kubebuilder:validation:Optional
	RotationSchedule string `json:"rotationSchedule,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=none;automatic
	ReplicationPolicy string `json:"replicationPolicy,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	EncryptionEnabled bool `json:"encryptionEnabled,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// SecretReference references a secret within the store.
type SecretReference struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	FromK8sSecret string `json:"fromK8sSecret,omitempty"`
}

// SecretStoreStatus defines the observed state of SecretStore.
type SecretStoreStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	StoreARN string `json:"storeArn,omitempty"`

	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Synced bool `json:"synced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="STORE",type=string,JSONPath=`.spec.storeName`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// SecretStore is the Schema for the secretstores API.
type SecretStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretStoreSpec   `json:"spec,omitempty"`
	Status SecretStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretStoreList contains a list of SecretStore.
type SecretStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretStore{}, &SecretStoreList{})
}
