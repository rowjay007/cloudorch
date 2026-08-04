/*
DNSZone is the Schema for the dnszones API.
It defines a DNS zone (Route53 / Cloud DNS / Azure DNS).
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DNSZoneSpec defines the desired state of DNSZone.
type DNSZoneSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=aws;gcp;azure
	Provider string `json:"provider"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=20
	Region string `json:"region"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ZoneName string `json:"zoneName"`

	// +kubebuilder:validation:Required
	Records []DNSRecord `json:"records"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=300
	TTL int32 `json:"ttl,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	DNSSEC bool `json:"dnssec,omitempty"`

	// +kubebuilder:validation:Optional
	HealthCheckRef string `json:"healthCheckRef,omitempty"`

	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// DNSRecord defines a DNS record in the zone.
type DNSRecord struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=A;AAAA;CNAME;MX;TXT;NS;SOA;PTR;SRV
	Type string `json:"type"`

	// +kubebuilder:validation:Required
	Value string `json:"value"`

	// +kubebuilder:validation:Optional
	TTL int32 `json:"ttl,omitempty"`

	// +kubebuilder:validation:Optional
	Weight int32 `json:"weight,omitempty"`

	// +kubebuilder:validation:Optional
	SetIdentifier string `json:"setIdentifier,omitempty"`
}

// DNSZoneStatus defines the observed state of DNSZone.
type DNSZoneStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	ZoneID string `json:"zoneId,omitempty"`

	// +optional
	NameServers []string `json:"nameServers,omitempty"`

	// +optional
	Ready bool `json:"ready,omitempty"`

	// +optional
	Synced bool `json:"synced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="ZONE",type=string,JSONPath=`.spec.zoneName`
// +kubebuilder:printcolumn:name="REGION",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="READY",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// DNSZone is the Schema for the dnszones API.
type DNSZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSZoneSpec   `json:"spec,omitempty"`
	Status DNSZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DNSZoneList contains a list of DNSZone.
type DNSZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSZone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DNSZone{}, &DNSZoneList{})
}
