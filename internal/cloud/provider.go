package cloud

import (
	"context"
)

type CloudProvider interface {
	Name() string
	Regions() []string

	GetCluster(ctx context.Context, id ClusterID) (*ClusterState, error)
	CreateCluster(ctx context.Context, spec ClusterSpec) (*ClusterState, error)
	UpdateCluster(ctx context.Context, id ClusterID, patch ClusterPatch) error
	DeleteCluster(ctx context.Context, id ClusterID) error

	GetDatabase(ctx context.Context, id DatabaseID) (*DatabaseState, error)
	CreateDatabase(ctx context.Context, spec DatabaseSpec) (*DatabaseState, error)
	UpdateDatabase(ctx context.Context, id DatabaseID, patch DatabasePatch) error
	DeleteDatabase(ctx context.Context, id DatabaseID) error

	GetObjectStore(ctx context.Context, id ObjectStoreID) (*ObjectStoreState, error)
	CreateObjectStore(ctx context.Context, spec ObjectStoreSpec) (*ObjectStoreState, error)
	UpdateObjectStore(ctx context.Context, id ObjectStoreID, patch ObjectStorePatch) error
	DeleteObjectStore(ctx context.Context, id ObjectStoreID) error

	GetCacheCluster(ctx context.Context, id CacheClusterID) (*CacheClusterState, error)
	CreateCacheCluster(ctx context.Context, spec CacheClusterSpec) (*CacheClusterState, error)
	UpdateCacheCluster(ctx context.Context, id CacheClusterID, patch CacheClusterPatch) error
	DeleteCacheCluster(ctx context.Context, id CacheClusterID) error

	GetVirtualNetwork(ctx context.Context, id VirtualNetworkID) (*VirtualNetworkState, error)
	CreateVirtualNetwork(ctx context.Context, spec VirtualNetworkSpec) (*VirtualNetworkState, error)
	UpdateVirtualNetwork(ctx context.Context, id VirtualNetworkID, patch VirtualNetworkPatch) error
	DeleteVirtualNetwork(ctx context.Context, id VirtualNetworkID) error

	GetLoadBalancer(ctx context.Context, id LoadBalancerID) (*LoadBalancerState, error)
	CreateLoadBalancer(ctx context.Context, spec LoadBalancerSpec) (*LoadBalancerState, error)
	UpdateLoadBalancer(ctx context.Context, id LoadBalancerID, patch LoadBalancerPatch) error
	DeleteLoadBalancer(ctx context.Context, id LoadBalancerID) error

	GetDNSZone(ctx context.Context, id DNSZoneID) (*DNSZoneState, error)
	CreateDNSZone(ctx context.Context, spec DNSZoneSpec) (*DNSZoneState, error)
	UpdateDNSZone(ctx context.Context, id DNSZoneID, patch DNSZonePatch) error
	DeleteDNSZone(ctx context.Context, id DNSZoneID) error

	GetSecurityPolicy(ctx context.Context, id SecurityPolicyID) (*SecurityPolicyState, error)
	CreateSecurityPolicy(ctx context.Context, spec SecurityPolicySpec) (*SecurityPolicyState, error)
	UpdateSecurityPolicy(ctx context.Context, id SecurityPolicyID, patch SecurityPolicyPatch) error
	DeleteSecurityPolicy(ctx context.Context, id SecurityPolicyID) error

	EstimateMonthlyCost(ctx context.Context, resources []ResourceSpec) (*CostEstimate, error)
}


type ClusterID struct {
	Name   string
	Region string
}

type DatabaseID struct {
	Name   string
	Region string
}

type ObjectStoreID struct {
	Name   string
	Region string
}

type CacheClusterID struct {
	Name   string
	Region string
}

type VirtualNetworkID struct {
	Name   string
	Region string
}

type LoadBalancerID struct {
	Name   string
	Region string
}

type DNSZoneID struct {
	Name   string
	Region string
}

type SecurityPolicyID struct {
	Name   string
	Region string
}


type ClusterState struct {
	ID           string
	Name         string
	Region       string
	Version      string
	Endpoint     string
	Status       string // "CREATING" | "ACTIVE" | "DELETING" | "FAILED"
	NodeCount    int32
	InstanceType string
}

type DatabaseState struct {
	ID       string
	Name     string
	Region   string
	Endpoint string
	Port     int32
	Status   string
}

type ObjectStoreState struct {
	ID       string
	Name     string
	Region   string
	ARN      string
	Endpoint string
	Status   string
}

type CacheClusterState struct {
	ID       string
	Name     string
	Region   string
	Endpoint string
	Port     int32
	Status   string
}

type VirtualNetworkState struct {
	ID     string
	Name   string
	CIDR   string
	VPCID  string
	Status string
}

type LoadBalancerState struct {
	ID      string
	Name    string
	DNSName string
	ARN     string
	Status  string
}

type DNSZoneState struct {
	ID          string
	Name        string
	ZoneID      string
	NameServers []string
	Status      string
}

type SecurityPolicyState struct {
	ID     string
	Name   string
	SGID   string
	Status string
}


type ClusterSpec struct {
	Name         string
	Region       string
	Version      string
	NodeCount    int32
	InstanceType string
	SpotEnabled  bool
	Tags         map[string]string
	VPCID        string
	SubnetIDs    []string
}

type ClusterPatch struct {
	NodeCount    *int32
	InstanceType *string
}

type DatabaseSpec struct {
	Name                string
	Region              string
	Engine              string
	Version             string
	InstanceClass       string
	MultiAZ             bool
	BackupRetentionDays int32
	StorageEncrypted    bool
	StorageGB           int32
	Tags                map[string]string
}

type DatabasePatch struct {
	BackupRetentionDays *int32
	StorageGB           *int32
}

type ObjectStoreSpec struct {
	Name               string
	Region             string
	AccessPolicy       string
	Versioning         bool
	LifecycleRules     []LifecycleRule
	ReplicationRegions []string
	EncryptionKeyRef   string
	Tags               map[string]string
}

type ObjectStorePatch struct {
	AccessPolicy *string
	Versioning   *bool
}

type CacheClusterSpec struct {
	Name              string
	Region            string
	Engine            string
	NodeType          string
	NumNodes          int32
	SnapshotRetention int32
	TransitEncryption bool
	AtRestEncryption  bool
	MultiAZ           bool
	Tags              map[string]string
}

type CacheClusterPatch struct {
	NumNodes          *int32
	NodeType          *string
	SnapshotRetention *int32
}

type VirtualNetworkSpec struct {
	Name            string
	Region          string
	CIDRBlock       string
	Subnets         []Subnet
	NATGateway      bool
	FlowLogsEnabled bool
	PeeringRefs     []string
	Tags            map[string]string
}

type VirtualNetworkPatch struct {
	NATGateway      *bool
	FlowLogsEnabled *bool
}

type Subnet struct {
	CIDRBlock        string
	Name             string
	Public           bool
	AvailabilityZone string
}

type LoadBalancerSpec struct {
	Name              string
	Region            string
	Scheme            string
	Protocol          string
	Targets           []LoadBalancerTarget
	HealthCheck       LoadBalancerHealthCheck
	SSLCertificateRef string
	AccessLogs        bool
	Tags              map[string]string
}

type LoadBalancerTarget struct {
	Port           int32
	Protocol       string
	TargetGroupARN string
	VPCID          string
}

type LoadBalancerHealthCheck struct {
	Protocol           string
	Path               string
	IntervalSeconds    int32
	HealthyThreshold   int32
	UnhealthyThreshold int32
}

type LoadBalancerPatch struct {
	Scheme *string
}

type DNSZoneSpec struct {
	Name           string
	Region         string
	ZoneName       string
	Records        []DNSRecord
	TTL            int32
	DNSSEC         bool
	HealthCheckRef string
	Tags           map[string]string
}

type DNSRecord struct {
	Name          string
	Type          string
	Value         string
	TTL           int32
	Weight        int32
	SetIdentifier string
}

type DNSZonePatch struct {
	TTL    *int32
	DNSSEC *bool
}

type SecurityPolicySpec struct {
	Name              string
	Region            string
	PolicyName        string
	IngressRules      []IngressRule
	EgressRules       []EgressRule
	AttachedResources []string
	AuditEnabled      bool
	ComplianceLabels  map[string]string
}

type IngressRule struct {
	Protocol    string
	FromPort    int32
	ToPort      int32
	CIDRBlocks  []string
	Description string
}

type EgressRule struct {
	Protocol    string
	FromPort    int32
	ToPort      int32
	CIDRBlocks  []string
	Description string
}

type SecurityPolicyPatch struct {
	IngressRules []IngressRule
	EgressRules  []EgressRule
}


type ResourceSpec struct {
	Type         string
	Provider     string
	Region       string
	InstanceType string
}

type CostEstimate struct {
	MonthlyCost float64
	Currency    string
	Breakdown   map[string]float64
}


func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "not found"
}
