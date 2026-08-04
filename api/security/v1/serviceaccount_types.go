/*
ServiceAccount is the Schema for the serviceaccounts API.
It defines an IAM Role / Service Account / Managed Identity with workload identity binding.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccountSpec defines the desired state of ServiceAccount.
type ServiceAccountSpec struct {
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
	AccountName string `json:"accountName"`

	// +kubebuilder:validation:Required
	Permissions []Permission `json:"permissions"`

	// +kubebuilder:validation:Required
	TrustPolicy TrustPolicy `json:"trustPolicy"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	ManagedIdentity bool `json:"managedIdentity,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	PodAnnotationKeys []string `json:"podAnnotationKeys,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// Permission defines an IAM permission scope.
type Permission struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Action string `json:"action"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Resource string `json:"resource"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Allow;Deny
	Effect string `json:"effect,omitempty"`
}

// TrustPolicy defines who can assume this identity.
type TrustPolicy struct {
	// +kubebuilder:validation:Required
	PrincipalType string `json:"principalType"`

	// +kubebuilder:validation:Required
	PrincipalIDs []string `json:"principalIds"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=sts:AssumeRole;sts:AssumeRoleWithWebIdentity;sts:AssumeRoleWithSAML
	Action string `json:"action,omitempty"`
}

// ServiceAccountStatus defines the observed state of ServiceAccount.
type ServiceAccountStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	RoleARN string `json:"roleArn,omitempty"`

	// +optional
	AccountID string `json:"accountId,omitempty"`

	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Synced bool `json:"synced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="ACCOUNT",type=string,JSONPath=`.spec.accountName`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// ServiceAccount is the Schema for the serviceaccounts API.
type ServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceAccountSpec   `json:"spec,omitempty"`
	Status ServiceAccountStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceAccountList contains a list of ServiceAccount.
type ServiceAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceAccount{}, &ServiceAccountList{})
}
