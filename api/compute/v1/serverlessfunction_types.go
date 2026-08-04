/*
ServerlessFunction is the Schema for the serverlessfunctions API.
It defines a serverless function (Lambda / Cloud Functions / Azure Functions).
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServerlessFunctionSpec defines the desired state of ServerlessFunction.
type ServerlessFunctionSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=aws;gcp;azure
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=20
	Region string `json:"region"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Runtime string `json:"runtime"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Handler string `json:"handler"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=128
	// +kubebuilder:validation:Maximum=10240
	MemoryMB int32 `json:"memoryMB"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=900
	Timeout int32 `json:"timeout"`

	// +kubebuilder:validation:Optional
	CodeURI string `json:"codeURI,omitempty"`

	// +kubebuilder:validation:Optional
	EnvironmentSecrets []SecretReference `json:"environmentSecrets,omitempty"`

	// +kubebuilder:validation:Optional
	Triggers []FunctionTrigger `json:"triggers,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// SecretReference references a Kubernetes Secret containing environment variables.
type SecretReference struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

// FunctionTrigger defines an event source for the serverless function.
type FunctionTrigger struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HTTP;S3;SQS;SNS;PubSub;EventHub
	Type string `json:"type"`

	// +kubebuilder:validation:Required
	Resource string `json:"resource"`

	// +kubebuilder:validation:Optional
	Events []string `json:"events,omitempty"`
}

// ServerlessFunctionStatus defines the observed state of ServerlessFunction.
type ServerlessFunctionStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	FunctionARN string `json:"functionArn,omitempty"`

	// +optional
	InvokeURL string `json:"invokeUrl,omitempty"`

	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Synced bool `json:"synced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="RUNTIME",type=string,JSONPath=`.spec.runtime`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// ServerlessFunction is the Schema for the serverlessfunctions API.
type ServerlessFunction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServerlessFunctionSpec   `json:"spec,omitempty"`
	Status ServerlessFunctionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServerlessFunctionList contains a list of ServerlessFunction.
type ServerlessFunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServerlessFunction `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServerlessFunction{}, &ServerlessFunctionList{})
}
