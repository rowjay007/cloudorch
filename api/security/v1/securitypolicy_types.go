/*
SecurityPolicy is the Schema for the securitypolicies API.
It defines security group / firewall rule / NSG constructs.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityPolicySpec defines the desired state of SecurityPolicy.
type SecurityPolicySpec struct {
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
	PolicyName string `json:"policyName"`

	// +kubebuilder:validation:Required
	IngressRules []IngressRule `json:"ingressRules"`

	// +kubebuilder:validation:Required
	EgressRules []EgressRule `json:"egressRules"`

	// +kubebuilder:validation:Optional
	AttachedResources []string `json:"attachedResources,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AuditEnabled bool `json:"auditEnabled,omitempty"`

	// +kubebuilder:validation:Optional
	ComplianceLabels map[string]string `json:"complianceLabels,omitempty"`
}

// IngressRule defines an inbound firewall rule.
type IngressRule struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=TCP;UDP;ICMP;HTTP;HTTPS;ALL
	Protocol string `json:"protocol"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	FromPort int32 `json:"fromPort"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	ToPort int32 `json:"toPort"`

	// +kubebuilder:validation:Required
	CIDRBlocks []string `json:"cidrBlocks"`

	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
}

// EgressRule defines an outbound firewall rule.
type EgressRule struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=TCP;UDP;ICMP;HTTP;HTTPS;ALL
	Protocol string `json:"protocol"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	FromPort int32 `json:"fromPort"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Maximum=65535
	ToPort int32 `json:"toPort"`

	// +kubebuilder:validation:Required
	CIDRBlocks []string `json:"cidrBlocks"`

	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
}

// SecurityPolicyStatus defines the observed state of SecurityPolicy.
type SecurityPolicyStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	SecurityGroupID string `json:"securityGroupId,omitempty"`

	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Synced bool `json:"synced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="POLICY",type=string,JSONPath=`.spec.policyName`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// SecurityPolicy is the Schema for the securitypolicies API.
type SecurityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityPolicySpec   `json:"spec,omitempty"`
	Status SecurityPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityPolicyList contains a list of SecurityPolicy.
type SecurityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecurityPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityPolicy{}, &SecurityPolicyList{})
}
