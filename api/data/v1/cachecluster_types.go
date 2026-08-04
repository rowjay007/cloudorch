/*
CacheCluster is the Schema for the cacheclusters API.
It defines a managed caching cluster (ElastiCache / Memorystore / Azure Cache).
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CacheClusterSpec defines the desired state of CacheCluster.
type CacheClusterSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=aws;gcp;azure
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=20
	Region string `json:"region"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=redis;memcached
	Engine string `json:"engine"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	NodeType string `json:"nodeType"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	NumNodes int32 `json:"numNodes"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=365
	SnapshotRetention int32 `json:"snapshotRetention"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=true
	TransitEncryption bool `json:"transitEncryption"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=true
	AtRestEncryption bool `json:"atRestEncryption"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	MultiAZ bool `json:"multiAZ,omitempty"`

	// +kubebuilder:validation:Optional
	ParameterGroupName string `json:"parameterGroupName,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// CacheClusterStatus defines the observed state of CacheCluster.
type CacheClusterStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// +optional
	Port int32 `json:"port,omitempty"`

	// +optional
	ConfigurationEndpoint string `json:"configurationEndpoint,omitempty"`

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

// CacheCluster is the Schema for the cacheclusters API.
type CacheCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CacheClusterSpec   `json:"spec,omitempty"`
	Status CacheClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CacheClusterList contains a list of CacheCluster.
type CacheClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CacheCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CacheCluster{}, &CacheClusterList{})
}
