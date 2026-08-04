/*
NodePool is the Schema for the nodepools API.
It defines an auto-scaling node group within a CloudCluster.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodePoolSpec defines the desired state of NodePool.
type NodePoolSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=40
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=20
	Region string `json:"region"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=5
	// +kubebuilder:validation:MaxLength=23
	// +kubebuilder:validation:Pattern=^[a-z][a-z0-9-]*[a-z0-9]$
	ClusterName string `json:"clusterName"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=23
	// +kubebuilder:validation:Pattern=^[a-z][a-z0-9-]*[a-z0-9]$
	PoolName string `json:"poolName"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=t3.medium;t3.large;t3.xlarge;m5.large;m5.xlarge;m5.2xlarge;m5.4xlarge;c5.large;c5.xlarge;r5.large;r5.xlarge
	InstanceType string `json:"instanceType"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000
	MinSize int32 `json:"minSize"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	MaxSize int32 `json:"maxSize"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000
	DesiredSize int32 `json:"desiredSize"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	SpotEnabled bool `json:"spotEnabled,omitempty"`

	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Optional
	Taints []NodeTaint `json:"taints,omitempty"`
}

// NodeTaint represents a Kubernetes taint applied to nodes.
type NodeTaint struct {
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// +kubebuilder:validation:Required
	Value string `json:"value,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=NoSchedule;PreferNoSchedule;NoExecute
	Effect string `json:"effect"`
}

// NodePoolStatus defines the observed state of NodePool.
type NodePoolStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	CurrentSize int32 `json:"currentSize,omitempty"`

	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Synced bool `json:"synced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="CLUSTER",type=string,JSONPath=`.spec.clusterName`
// +kubebuilder:printcolumn:name="POOL",type=string,JSONPath=`.spec.poolName`
// +kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// NodePool is the Schema for the nodepools API.
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolSpec   `json:"spec,omitempty"`
	Status NodePoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NodePoolList contains a list of NodePool.
type NodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodePool{}, &NodePoolList{})
}
