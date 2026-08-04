/*
LoadBalancer is the Schema for the loadbalancers API.
It defines a cloud load balancer (ALB/NLB/GCP LB/Azure LB).
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LoadBalancerSpec defines the desired state of LoadBalancer.
type LoadBalancerSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=aws;gcp;azure
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=20
	Region string `json:"region"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=application;network
	Scheme string `json:"scheme"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HTTP;HTTPS;TCP;TLS;UDP
	Protocol string `json:"protocol"`

	// +kubebuilder:validation:Required
	Targets []LoadBalancerTarget `json:"targets"`

	// +kubebuilder:validation:Optional
	HealthCheck LoadBalancerHealthCheck `json:"healthCheck,omitempty"`

	// +kubebuilder:validation:Optional
	SSLCertificateRef SecretReference `json:"sslCertificateRef,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AccessLogs bool `json:"accessLogs,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// LoadBalancerTarget defines a target group for the load balancer.
type LoadBalancerTarget struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// +kubebuilder:validation:Required
	Protocol string `json:"protocol"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	TargetGroupARN string `json:"targetGroupArn,omitempty"`

	// +kubebuilder:validation:Optional
	VPCID string `json:"vpcId,omitempty"`
}

// LoadBalancerHealthCheck defines health check settings.
type LoadBalancerHealthCheck struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HTTP;HTTPS;TCP
	Protocol string `json:"protocol"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Path string `json:"path,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=120
	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=10
	HealthyThreshold int32 `json:"healthyThreshold,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=10
	UnhealthyThreshold int32 `json:"unhealthyThreshold,omitempty"`
}

// SecretReference references a Kubernetes Secret.
type SecretReference struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

// LoadBalancerStatus defines the observed state of LoadBalancer.
type LoadBalancerStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	DNSName string `json:"dnsName,omitempty"`

	// +optional
	LoadBalancerARN string `json:"loadBalancerArn,omitempty"`

	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Synced bool `json:"synced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="SCHEME",type=string,JSONPath=`.spec.scheme`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// LoadBalancer is the Schema for the loadbalancers API.
type LoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoadBalancerSpec   `json:"spec,omitempty"`
	Status LoadBalancerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LoadBalancerList contains a list of LoadBalancer.
type LoadBalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoadBalancer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoadBalancer{}, &LoadBalancerList{})
}
