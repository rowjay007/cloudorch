# CloudOrch

**Kubernetes-Native Multi-Cloud Infrastructure Orchestration Engine**

A production-grade Kubernetes Operator built with Go 1.23+ and controller-runtime v0.18. CloudOrch extends the Kubernetes API with Custom Resource Definitions (CRDs) to manage multi-cloud infrastructure declaratively.

## Architecture

CloudOrch follows the **Operator Pattern** with a level-triggered reconciliation loop:

```
┌─────────────────┐     ┌──────────────────────┐     ┌──────────────┐
│  Kubernetes API │────▶│  CloudOrch Operator  │────▶│ Cloud APIs   │
│  (etcd)         │◀────│  (Reconciler Loop)   │◀────│ (AWS/GCP/Azure) │
└─────────────────┘     └──────────────────────┘     └──────────────┘
                               │
                               ▼
                        ┌──────────────┐
                        │ OPA Policies │
                        │ (Rego)       │
                        └──────────────┘
```

## Components

| Component | Purpose |
|-----------|---------|
| `operator-core` | controller-runtime Manager, leader election, health probes |
| `crd-registry` | 12 CRDs with OpenAPI v3 schemas |
| `cloud-adapters` | AWS/GCP/Azure provider implementations (Strategy Pattern) |
| `policy-engine` | Embedded OPA Rego policy evaluation |
| `webhook-server` | Validating + Mutating admission webhooks |
| `controllers` | Reconcilers for each CRD |

## 8 Design Patterns Implemented

1. **Kubernetes Operator Pattern** — controller-runtime Reconciler with informers and work queues
2. **Level-Triggered Reconciliation Loop** — continuous diff of desired vs actual state
3. **Finalizer Pattern** — safe resource deletion with cleanup guarantee
4. **Owner References** — cascading garbage collection
5. **Strategy Pattern** — pluggable cloud provider adapters
6. **Admission Webhooks** — shift-left policy enforcement
7. **Status Conditions** — observable operator contract (Ready, Synced, Degraded, PolicyViolation)
8. **Leader Election** — zero-downtime HA operator

## Cloud Providers

- **AWS** — EKS, RDS, S3, ElastiCache, VPC, ALB/NLB, Route53
- **GCP** — GKE, Cloud SQL, GCS, Memorystore, VPC
- **Azure** — AKS, Azure DB, Blob Storage, Azure Cache, VNET

## Quick Start

### Prerequisites

- Go 1.23+
- Kubernetes 1.28+
- kubectl configured to a cluster
- cert-manager (for webhooks)

### Build

```bash
make build
```

### Run locally (with envtest)

```bash
make test
```

### Deploy to cluster

```bash
make deploy
```

## CRDs

| Category | CRDs |
|----------|------|
| Compute | CloudCluster, NodePool, ServerlessFunction |
| Data | ManagedDatabase, ObjectStore, CacheCluster |
| Network | VirtualNetwork, LoadBalancer, DNSZone |
| Security | SecurityPolicy, ServiceAccount, SecretStore |

## Non-Functional Requirements

| Metric | Target |
|--------|--------|
| Reconcile Loop P99 | < 2s per CR |
| Drift Detection | < 5 minutes |
| Webhook Latency P99 | < 100ms |
| OPA Decision | < 10ms |
| Concurrency | 50 parallel reconcilers |
| Test Coverage | ≥ 80% |

## Example Usage

```yaml
apiVersion: compute.cloudorch.io/v1
kind: CloudCluster
metadata:
  name: my-eks-cluster
spec:
  provider: aws
  region: us-east-1
  kubernetesVersion: "1.30"
  nodeCount: 3
  instanceType: t3.large
  clusterName: my-cluster
  tags:
    environment: production
    team: platform
```

Apply with:
```bash
kubectl apply -f cluster.yaml
```

Check status:
```bash
kubectl describe cloudcluster my-eks-cluster
kubectl get cloudclusters
```

## License

MIT