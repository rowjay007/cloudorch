/*
CloudCluster is the Schema for the cloudclusters API.
It defines a managed Kubernetes cluster (EKS/GKE/AKS).
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CloudClusterSpec defines the desired state of CloudCluster.
type CloudClusterSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=40
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=20
	Region string `json:"region"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=1.28;1.29;1.30;1.31
	KubernetesVersion string `json:"kubernetesVersion"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	NodeCount int32 `json:"nodeCount"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=t3.medium;t3.large;t3.xlarge;m5.large;m5.xlarge;m5.2xlarge
	InstanceType string `json:"instanceType"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	SpotEnabled bool `json:"spotEnabled,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=5
	// +kubebuilder:validation:MaxLength=23
	// +kubebuilder:validation:Pattern=^[a-z][a-z0-9-]*[a-z0-9]$
	ClusterName string `json:"clusterName"`
}

// CloudClusterStatus defines the observed state of CloudCluster.
type CloudClusterStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	ClusterID string `json:"clusterId,omitempty"`

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
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="VERSION",type=string,JSONPath=`.spec.kubernetesVersion`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="SYNCED",type=boolean,JSONPath=`.status.synced`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// CloudCluster is the Schema for the cloudclusters API.
type CloudCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudClusterSpec   `json:"spec,omitempty"`
	Status CloudClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudClusterList contains a list of CloudCluster.
type CloudClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudCluster `json:"items"`
}

func (in *CloudCluster) DeepCopyObject() runtime.Object {
	out := new(CloudCluster)
	in.DeepCopyInto(out)
	return out
}

func (in *CloudCluster) DeepCopyInto(out *CloudCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
}

func (in *CloudClusterList) DeepCopyObject() runtime.Object {
	out := new(CloudClusterList)
	in.DeepCopyInto(out)
	return out
}

func (in *CloudClusterList) DeepCopyInto(out *CloudClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]CloudCluster, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}