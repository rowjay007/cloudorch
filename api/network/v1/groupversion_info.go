package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is group version used to register these objects.
var GroupVersion = schema.GroupVersion{Group: "network.cloudorch.io", Version: "v1"}

// SchemeBuilder registers addFuncs to the scheme.
var SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

// AddToScheme adds the types in this group-version to the given scheme.
func AddToScheme(scheme *runtime.Scheme) error {
	return SchemeBuilder.AddToScheme(scheme)
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&VirtualNetwork{},
		&VirtualNetworkList{},
		&LoadBalancer{},
		&LoadBalancerList{},
		&DNSZone{},
		&DNSZoneList{},
	)
	return nil
}
