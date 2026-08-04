/*
VirtualNetwork is the Schema for the virtualnetworks API.
It defines a VPC / VNet network construct.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualNetworkSpec defines the desired state of VirtualNetwork.
type VirtualNetworkSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=aws;gcp;azure
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=20
	Region string `json:"region"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$
	CIDRBlock string `json:"cidrBlock"`

	// +kubebuilder:validation:Required
	Subnets []Subnet `json:"subnets"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	NATGateway bool `json:"natGateway,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	FlowLogsEnabled bool `json:"flowLogsEnabled,omitempty"`

	// +kubebuilder:validation:Optional
	PeeringRefs []string `json:"peeringRefs,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// Subnet defines a subnet within a VirtualNetwork.
type Subnet struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$
	CIDRBlock string `json:"cidrBlock"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Public bool `json:"public,omitempty"`

	// +kubebuilder:validation:Optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`
}

// VirtualNetworkStatus defines the observed state of VirtualNetwork.
type VirtualNetworkStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	VPCID string `json:"vpcId,omitempty"`

	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Synced bool `json:"synced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="CIDR",type=string,JSONPath=`.spec.cidrBlock`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// VirtualNetwork is the Schema for the virtualnetworks API.
type VirtualNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualNetworkSpec   `json:"spec,omitempty"`
	Status VirtualNetworkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualNetworkList contains a list of VirtualNetwork.
type VirtualNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualNetwork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualNetwork{}, &VirtualNetworkList{})
}
