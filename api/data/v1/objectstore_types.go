/*
ObjectStore is the Schema for the objectstores API.
It defines an object storage bucket (S3 / GCS / Azure Blob).
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectStoreSpec defines the desired state of ObjectStore.
type ObjectStoreSpec struct {
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
	// +kubebuilder:validation:Pattern=^[a-z0-9][a-z0-9.-]*[a-z0-9]$
	BucketName string `json:"bucketName"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=private;public-read;authenticated-read
	AccessPolicy string `json:"accessPolicy"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=true
	Versioning bool `json:"versioning"`

	// +kubebuilder:validation:Optional
	LifecycleRules []LifecycleRule `json:"lifecycleRules,omitempty"`

	// +kubebuilder:validation:Optional
	ReplicationRegions []string `json:"replicationRegions,omitempty"`

	// +kubebuilder:validation:Optional
	EncryptionKeyRef SecretReference `json:"encryptionKeyRef,omitempty"`

	// +kubebuilder:validation:Optional
	CORSRules []CORSRule `json:"corsRules,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// LifecycleRule defines an object lifecycle expiration rule.
type LifecycleRule struct {
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	ExpirationDays int32 `json:"expirationDays"`

	// +kubebuilder:validation:Optional
	Prefix string `json:"prefix,omitempty"`

	// +kubebuilder:validation:Optional
	Status string `json:"status,omitempty"`
}

// CORSRule defines a Cross-Origin Resource Sharing rule.
type CORSRule struct {
	// +kubebuilder:validation:Required
	AllowedOrigins []string `json:"allowedOrigins"`

	// +kubebuilder:validation:Required
	AllowedMethods []string `json:"allowedMethods"`

	// +kubebuilder:validation:Optional
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	MaxAgeSeconds int32 `json:"maxAgeSeconds,omitempty"`
}

// ObjectStoreStatus defines the observed state of ObjectStore.
type ObjectStoreStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	BucketARN string `json:"bucketArn,omitempty"`

	// +optional
	Endpoint string `json:"endpoint,omitempty"`

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
// +kubebuilder:printcolumn:name="BUCKET",type=string,JSONPath=`.spec.bucketName`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// ObjectStore is the Schema for the objectstores API.
type ObjectStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObjectStoreSpec   `json:"spec,omitempty"`
	Status ObjectStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ObjectStoreList contains a list of ObjectStore.
type ObjectStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ObjectStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ObjectStore{}, &ObjectStoreList{})
}
