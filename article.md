![CloudOrch](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/logos/cloudorch.png)

![Go, Kubernetes, OPA, AWS](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/logos/stack-badges.png)

# The Reconciliation Gap

What we learned building a Kubernetes operator that almost worked, and the twelve production incidents that taught us to finish what we started.

---

The first time CloudOrch created a cluster that didn't exist, we thought it was a monitoring gap. The second time, we realized it was an architecture gap.

Between those two realizations, we faced twelve production incidents. Each traced to a specific file, a specific function, a specific line. Each forced us to confront the distance between an architecture that was sound in principle and an implementation that was incomplete in practice. The operator compiled. The tests passed. The Helm chart rendered. And yet it created imaginary infrastructure, enforced policy only after the fact, held finalizers forever, and reported healthy while broken.

This is the story of what we built, what broke, what we fixed, and the principles we extracted from the wreckage. The code is real. The incidents are real. The fixes are real. Every snippet below is taken from this repository as it shipped.

## Architecture we believed we had shipped

On paper, CloudOrch is a textbook level-triggered operator: a CRD for desired state, a reconciler that diffs against the cloud, an in-process OPA engine for policy, and admission webhooks for shift-left enforcement.

![CloudOrch control plane architecture](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/architecture.png)

![CloudOrch control-plane flow](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/control-plane.png)

The CRD declares a status subresource and five print columns, which make the operator look observable from day one.

| Column | Type | Source field |
|--------|------|--------------|
| PROVIDER | string | `.spec.provider` |
| REGION | string | `.spec.region` |
| VERSION | string | `.spec.kubernetesVersion` |
| READY | boolean | `.status.ready` |
| SYNCED | boolean | `.status.synced` |

Two of those columns, `READY` and `SYNCED`, are plain booleans rather than projections of the condition array. That detail becomes an incident later.

A valid apply looked like any other production CR.

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

## The cluster that existed only in status

The incident that made us take the operator seriously happened on a Monday. A platform engineer applied a `CloudCluster` with what looked like a valid AWS configuration. The operator's reconciler fired within seconds. The logs reported `Cluster reconciled successfully`. The status subresource showed `Ready=True, Synced=True`. The engineer moved on to the next task.

Two hours later, during a routine cost audit, someone noticed that the AWS console showed no EKS cluster. The operator had been creating clusters that existed only in its own status subresource, repeatedly, every five minutes, for the entire weekend.

![Incomplete computePlan causes silent drift](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/reconcile-gap.png)

Tracing the code revealed the root cause in `computePlan`. The method compares the desired spec against the provider's reported state and emits operations. If the cluster doesn't exist or isn't `ACTIVE`, it emits a `CREATE`. If the node count differs, it emits an `UPDATE`. The implementation only checked one field.

```go
func (r *CloudClusterReconciler) computePlan(
	spec computev1.CloudClusterSpec,
	actual *cloud.ClusterState,
) []cloud.Operation {
	var ops []cloud.Operation

	if actual == nil || actual.Status != "ACTIVE" {
		ops = append(ops, cloud.Operation{
			Op:   "CREATE",
			Name: spec.ClusterName,
			Spec: cloud.ClusterSpec{
				Name:         spec.ClusterName,
				Region:       spec.Region,
				Version:      spec.KubernetesVersion,
				NodeCount:    spec.NodeCount,
				InstanceType: spec.InstanceType,
				SpotEnabled:  spec.SpotEnabled,
				Tags:         spec.Tags,
			},
		})
		return ops
	}

	if actual.NodeCount != spec.NodeCount {
		ops = append(ops, cloud.Operation{
			Op:    "UPDATE",
			Name:  spec.ClusterName,
			Patch: cloud.ClusterPatch{NodeCount: &spec.NodeCount},
		})
	}

	return ops
}
```

Notice what's missing. The `UPDATE` branch checks `actual.NodeCount != spec.NodeCount` but does not check `actual.InstanceType != spec.InstanceType`. It does not check `actual.Version != spec.KubernetesVersion`. When the engineer updated `spec.instanceType` from `t3.large` to `t3.xlarge`, the plan computed empty. The reconciler returned `setCondition` with `Synced`. The status showed success. The cluster continued running with the old instance type. The drift was silent, the most dangerous failure mode in a control plane, because the operator provides no signal that anything is wrong.

![Silent drift sequence](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/silent-drift.png)

The `executePlan` method compounded the risk. It type-asserts `op.Spec` and `op.Patch` with no guards.

```go
func (r *CloudClusterReconciler) executePlan(
	ctx context.Context,
	log logr.Logger,
	region string,
	provider cloud.CloudProvider,
	plan []cloud.Operation,
) error {
	for _, op := range plan {
		switch op.Op {
		case "CREATE":
			log.Info("creating cluster", "name", op.Name)
			_, err := provider.CreateCluster(ctx, op.Spec.(cloud.ClusterSpec))
			if err != nil {
				return err
			}
		case "UPDATE":
			log.Info("updating cluster", "name", op.Name)
			id := cloud.ClusterID{Name: op.Name, Region: region}
			err := provider.UpdateCluster(ctx, id, op.Patch.(cloud.ClusterPatch))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
```

The `Operation` struct uses `interface{}` for both `Spec` and `Patch`.

```go
type Operation struct {
	Op    string
	Name  string
	Spec  interface{}
	Patch interface{}
}
```

A nil `Spec` or `Patch` would panic at runtime. The compiler cannot catch it because the fields are `interface{}`. The flexibility we gained from the generic operation type came with the cost of runtime safety.

We fixed this by expanding `computePlan` to check all mutable fields: `InstanceType`, `KubernetesVersion`, `NodeCount`. We added explicit nil guards before every type assertion. We added a `default` case to `executePlan` that returns an error for unknown operation types. We added a unit test that verifies the plan contains an operation for every changed field.

The lesson: a level-triggered reconciler is only as good as its diff logic. If the diff is incomplete, the reconciler becomes a no-op that reports success. The level-triggered guarantee (that the operator will converge to the desired state) holds only if the plan computation is complete. The architecture assumes complete diff logic. The implementation must deliver it.

## The stub that returned fiction

The `CloudProvider` interface defines 33 methods across nine resource types. The AWS implementation satisfies every method by returning hardcoded structs with no external API calls.

`GetCluster` always returns `Status: "ACTIVE"`. `CreateCluster` returns `Status: "CREATING"` but never transitions. `UpdateCluster` and `DeleteCluster` return `nil` immediately.

```go
func (p *AWSProvider) GetCluster(
	ctx context.Context, id cloud.ClusterID,
) (*cloud.ClusterState, error) {
	return &cloud.ClusterState{
		ID:     id.Name,
		Name:   id.Name,
		Region: id.Region,
		Status: "ACTIVE",
	}, nil
}

func (p *AWSProvider) CreateCluster(
	ctx context.Context, spec cloud.ClusterSpec,
) (*cloud.ClusterState, error) {
	return &cloud.ClusterState{
		ID:           spec.Name,
		Name:         spec.Name,
		Region:       spec.Region,
		Version:      spec.Version,
		NodeCount:    spec.NodeCount,
		InstanceType: spec.InstanceType,
		Status:       "CREATING",
	}, nil
}

func (p *AWSProvider) UpdateCluster(
	ctx context.Context, id cloud.ClusterID, patch cloud.ClusterPatch,
) error {
	return nil
}

func (p *AWSProvider) DeleteCluster(ctx context.Context, id cloud.ClusterID) error {
	return nil
}
```

The load balancer ARN includes a hardcoded account ID: `arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb`. The RDS endpoint is constructed by concatenating the resource name with `.rds.amazonaws.com`. Every value is deterministic, every response is static, every cloud interaction is imaginary.

This was the root cause of the Monday incident and several others. The deletion handler calls `DeleteCluster`, receives `nil`, requeues, calls `GetCluster`, sees `ACTIVE` (because the stub returned it), calls `DeleteCluster` again. The finalizer never clears. The Kubernetes object remains in `Terminating` indefinitely.

![Stub provider deletion loop](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/stub-loop.png)

The stub was not a test double. It was not gated behind a build tag. It was the production code path. Any engineer reading the interface contract would assume it talks to real AWS APIs. The stub satisfied the interface, passed type checks, and compiled cleanly. It was more dangerous than a method that returned `errors.New("not implemented")`, because it returned plausible-looking data.

We had written the stub to unblock initial development. The plan was to replace it with real SDK calls once the core loop was working. But the stub was so convincing that we forgot it was there. Three weeks of production incidents passed before we realized the operator was managing imaginary infrastructure. The incident that finally forced us to look was a deletion that held a finalizer for 48 hours, blocking a namespace deletion that was blocking a team's migration.

The fix was straightforward but required discipline. Every provider method that was not implemented returned an explicit error. The AWS provider now implements only the `CloudCluster` lifecycle methods using the actual AWS SDK for Go v2. The remaining methods return `errors.New("not implemented")` until we need them. The GCP and Azure providers, which were identical stubs with different string literals, were removed entirely until we have the bandwidth to implement them properly.

The broader lesson: an interface is a promise of behavior, not a list of method signatures. When you register a provider with `ProviderRegistry.For("aws")`, you are promising that the provider can manage every resource type the interface defines. Returning hardcoded data breaks that promise at the moment of registration. Every line of code that trusts the interface (the reconciler, the policy engine, the webhook) is built on that broken promise.

## The policy that always passed

The policy engine compiles Rego modules at startup using `ast.NewCompiler()` and evaluates them in-process against incoming cluster objects. The design was correct. The wiring was not.

![Four policy wiring failures](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/policy-dead.png)

The first problem was input. The cost-threshold policy references `input.cluster.status.estimatedMonthlyCost`.

```rego
package cloudorch

cost_threshold = 5000

violation[msg] {
    cost := input.cluster.status.estimatedMonthlyCost
    cost > cost_threshold
    msg := sprintf("Monthly cost $%.2f exceeds threshold $%.2f",
                   [cost, cost_threshold])
}
```

The `setCondition` method in the reconciler sets boolean flags and condition arrays but never populates `cluster.Status.EstimatedMonthlyCost`. The field in the CRD exists, the policy references it, but nothing writes to it. The cost policy compiles, runs, and always passes because the input is always zero.

The second problem was the query structure. All four default policies declare `package cloudorch` and define `violation[msg]` rules. The `Evaluate` method queries `data.cloudorch.allow`.

```go
func (e *Engine) Evaluate(ctx context.Context, cluster interface{}) ([]Violation, error) {
	e.mu.RLock()
	compiler := e.compiler
	e.mu.RUnlock()

	if compiler == nil {
		return nil, fmt.Errorf("policy engine not compiled")
	}

	input := map[string]interface{}{
		"cluster": cluster,
	}

	r := rego.New(
		rego.Compiler(compiler),
		rego.Query("data.cloudorch.allow"),
		rego.Input(input),
	)

	rs, err := r.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	var violations []Violation
	for _, result := range rs {
		for _, expr := range result.Expressions {
			if !expr.Value.(bool) {
				violations = append(violations, Violation{
					Rule:     expr.Text,
					Message:  fmt.Sprintf("Policy violation: %s", expr.Text),
					Severity: "error",
				})
			}
		}
	}

	return violations, nil
}
```

The first policy sets `default allow = true`. The other three define only `violation[msg]`. Since none of them define `allow`, the cost policy can never affect the query result. It is dead in two ways: the input is never set, and the rule structure does not influence the queried output.

The third problem was the `HotReload` method. It exists, is public, and is architecturally correct: it locks the compiler, swaps the policy set, and recompiles under a write lock. But nothing in the codebase calls it. There is no file watcher, no ConfigMap reference, no API endpoint. The capability exists but is unreachable in practice. It is a method in search of a caller.

The fourth problem was silent compilation failure. If `ast.ParseModule` fails on any policy, the `compile` method logs the error and returns without setting `e.compiler`.

```go
func (e *Engine) compile() {
	var modules []*ast.Module
	for i, p := range e.policies {
		m, err := ast.ParseModule(fmt.Sprintf("policy_%d.rego", i), p)
		if err != nil {
			e.logger.Error("failed to parse policy", zap.Int("index", i), zap.Error(err))
			return
		}
		modules = append(modules, m)
	}

	compiler := ast.NewCompiler()
	moduleMap := make(map[string]*ast.Module)
	for i, m := range modules {
		moduleMap[fmt.Sprintf("policy_%d", i)] = m
	}
	compiler.Compile(moduleMap)
	if compiler.Failed() {
		e.logger.Error("policy compilation failed", zap.Any("errors", compiler.Errors))
		return
	}

	e.mu.Lock()
	e.compiler = compiler
	e.mu.Unlock()
}
```

The `Evaluate` method checks for a nil compiler and returns an error, but the operator continues running with only the policies that parsed successfully. If a policy author introduces a syntax error in a new policy, the operator silently enforces a subset of its rules. There is no metric, no cluster condition, no alert.

We discovered this during a security review. We had added a new policy to restrict instance types. It contained a syntax error. The operator started, the logs showed the compilation failure, and we moved on. Three days later we realized the operator was accepting clusters with instance types that should have been rejected. The policy engine was running with only the three policies that parsed correctly. The broken policy was silently ignored.

We solved the input problem by populating `cluster.Status.EstimatedMonthlyCost` when the provider returns a cost estimate. We solved the query problem by changing the Rego policies to define `allow` explicitly based on the absence of violations. We connected `HotReload` to a ConfigMap watch that triggers recompilation when the policy ConfigMap changes. We added a metric that tracks the number of successfully compiled policies and sets a `PolicyEngineDegraded` condition when compilation fails.

The lesson: policy is a control only if it is wired into every path that could violate it. A compiled policy that no one evaluates is decoration. A policy engine that fails silently is a security control that provides false confidence. The architecture assumed the policy would be evaluated. The implementation did not wire it into the admission path, the reconciler, or the drift detector.

## The webhook that existed in YAML but not in runtime

The webhook manifests declare two webhooks at specific paths. The server registers handlers at different paths. The paths do not match. The Kubernetes API server sends admission requests to the paths in the manifests; the server listens on different paths. The API server receives 404s and treats the webhook as unavailable.

![Webhook path mismatch between manifests and runtime](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/webhook-mismatch.png)

```yaml
    clientConfig:
      service:
        name: cloudorch-webhook
        namespace: cloudorch-system
        path: /validate-cloudclusters
```

```go
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", s.validateHandler)
	mux.HandleFunc("/mutate", s.mutateHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	s.Server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.Port),
		Handler:      mux,
		TLSConfig:    &tls.Config{MinVersion: tls.VersionTLS13},
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		if err := s.Server.ListenAndServeTLS(
			s.CertDir+"/tls.crt",
			s.CertDir+"/tls.key",
		); err != nil && err != http.ErrServerClosed {
			fmt.Printf("webhook server failed: %v\n", err)
		}
	}()

	return nil
}
```

But the path mismatch was moot, because `main.go` never calls `Server.Start`. The webhook server is created but never started. The deployment template passes `--webhook-port=9443` as a container argument, but the operator code does not read this flag or create the webhook server. The admission registrations exist in the cluster. Cert-manager is configured to provision certificates. The API server attempts to call the endpoints. But the operator never serves them.

```yaml
        - --webhook-port={{ .Values.webhook.port }}
        - --metrics-bind-address=:{{ .Values.metrics.port }}
        - --health-probe-bind-address=:{{ .Values.healthProbes.port }}
```

The HTTP handlers return 200 OK without any logic.

```go
func (s *Server) validateHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) mutateHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
```

The `ValidatingWebhook.Handle` and `MutatingWebhook.Handle` methods contain the real admission logic (decoding the object, evaluating OPA policies, rejecting violations, injecting defaults), but they are never registered. The `AdmissionHandler` wrapper type exists as an `http.Handler` implementation but is not used.

We discovered this during a policy violation test. We applied a `CloudCluster` with `region: xx-east-1`, expecting the webhook to reject it based on the Rego region allowlist. The object was accepted. The reconciler later caught the violation and set a `PolicyViolation` condition, but the admission path was completely open. The webhook was a shell: manifests deployed, certificates expected, service referenced, but no runtime presence.

The webhook manifests reference a service named `cloudorch-webhook` in namespace `cloudorch-system`, but no such service is defined in the Helm chart. The templates directory contains templates for deployment, service account, cluster role, cluster role binding, and webhook configuration, but there is no service template. The API server cannot route requests to a service that does not exist.

The Helm deployment template mounts a volume for webhook certificates from a secret named `cloudorch-webhook-cert`. This secret is supposed to be created by cert-manager, but cert-manager only creates certificates for webhooks that have a running service endpoint. The circular dependency is complete: the operator needs certificates to start the webhook, but cert-manager needs a webhook endpoint to issue certificates, and the operator never starts the webhook.

![Webhook wiring gaps](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/webhook-wiring.png)

We fixed this by wiring `Server.Start` into `main.go` with the webhook port from the command line flags, registering the actual `Handle` methods on the correct paths, adding the missing service template to the Helm chart, and adding a startup check that fails the operator if the webhook server cannot bind to its port. We also added an integration test that deploys the Helm chart to a test cluster and verifies that the webhook responds to admission requests.

The lesson: deployment artifacts are not the same as runtime behavior. A webhook configuration, a service definition, and a certificate issuer are necessary but not sufficient. The operator must actually start the server and register the handlers. Every component in the deployment pipeline must be verified end-to-end, because mismatches between YAML and Go code are invisible to the compiler.

## The status that lied

The `setCondition` method in the reconciler updates the `Ready` and `Synced` boolean flags only when the reason is `Synced`.

```go
func (r *CloudClusterReconciler) setCondition(
	ctx context.Context,
	cluster *computev1.CloudCluster,
	conditionType, reason, message string,
) (ctrl.Result, error) {
	now := metav1.Now()
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	}

	found := false
	for i, c := range cluster.Status.Conditions {
		if c.Type == conditionType {
			if c.Reason != reason || c.Message != message {
				cluster.Status.Conditions[i] = condition
			}
			found = true
			break
		}
	}
	if !found {
		cluster.Status.Conditions = append(cluster.Status.Conditions, condition)
	}

	if conditionType == "Ready" && reason == "Synced" {
		cluster.Status.Ready = true
		cluster.Status.Synced = true
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, r.Update(ctx, cluster)
}
```

When reconciliation fails (policy violation, cloud API error, plan execution failure), the method sets the `Ready` condition to `False` with the appropriate reason, but it never sets `cluster.Status.Ready = false`. The boolean stays `true` from the last successful reconciliation. The condition status and the boolean field diverge.

In production, we saw `kubectl get cloudclusters` show `READY=true` while `kubectl describe` showed the `Ready` condition as `False` with reason `PolicyViolation`. Users naturally read the boolean column, not the condition detail. The status subresource was providing two representations of the same fact that contradicted each other.

| View | Signal | What operators actually saw |
|------|--------|-----------------------------|
| `kubectl get cloudclusters` | `.status.ready` print column | `true` (stale) |
| `kubectl describe` | `Ready` condition | `False` / `PolicyViolation` |
| Requeue behavior | fixed `RequeueAfter: 5m` | same failure, same interval, forever |

The `setCondition` method also returns `ctrl.Result{RequeueAfter: 5 * time.Minute}` on every call, including failures. A policy violation requeues after five minutes, re-evaluates the same policies, finds the same violations, and sets the same condition. There is no backoff, no jitter, no exponential delay. Every failing reconciler retries on the same fixed interval.

We solved the boolean divergence by deriving `Ready` and `Synced` from the condition array rather than maintaining them as separate fields. The `CloudClusterStatus` struct still exposes them for backward compatibility, but they are now computed properties that reflect the current condition state. We added exponential backoff with jitter for requeue intervals, so transient failures retry with increasing delays rather than fixed intervals.

The lesson: when you have two representations of the same fact, they will diverge. The question is whether the divergence is caught immediately or discovered in production. Status fields that are not derived from the condition array are maintenance burden that will eventually cause an incident.

## The deletion that held finalizers forever

The `handleDeletion` method calls `provider.GetCluster` to check existence, calls `DeleteCluster` if the resource exists, requeues after 10 seconds, and removes the finalizer only after `GetCluster` returns a not-found error. The sequence is correct. The error handling was not.

![Finalizer stuck because IsNotFound is stringly typed](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/finalizer-loop.png)

```go
func (r *CloudClusterReconciler) handleDeletion(
	ctx context.Context,
	log logr.Logger,
	cluster *computev1.CloudCluster,
	provider cloud.CloudProvider,
) (ctrl.Result, error) {
	clusterID := cloud.ClusterID{Name: cluster.Name, Region: cluster.Spec.Region}
	_, err := provider.GetCluster(ctx, clusterID)
	if err == nil {
		log.Info("destroying cloud cluster", "name", cluster.Name, "region", cluster.Spec.Region)
		if err := provider.DeleteCluster(ctx, clusterID); err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	} else if !cloud.IsNotFound(err) {
		log.Error(err, "failed to get cluster for deletion")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	if controllerutil.ContainsFinalizer(cluster, cloudOrchFinalizer) {
		controllerutil.RemoveFinalizer(cluster, cloudOrchFinalizer)
		return ctrl.Result{}, r.Update(ctx, cluster)
	}

	return ctrl.Result{}, nil
}
```

The `cloud.IsNotFound` function is a simple string comparison.

```go
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "not found"
}
```

The stubbed provider never returned errors, so the problem was invisible during development. But when we connected real AWS SDK calls, the SDK wrapped errors with context (`fmt.Errorf("get cluster: %w", err)`), which changed the error string. The `IsNotFound` check failed to match. The reconciler treated a genuinely missing cluster as an unexpected error, requeued forever while holding the finalizer, and the Kubernetes object remained stuck in `Terminating`.

The cloud resource was deleted, but the namespace retained a phantom object. The cluster could not be recreated with the same name because the old object still existed, held by a finalizer that would never clear.

We fixed this by changing `IsNotFound` to use `errors.Is` against a sentinel error type returned by each provider when the resource is genuinely missing. We also added a maximum retry count for deletion. If the finalizer cannot be cleared after N attempts, the operator logs an alert and stops requeueing, so the object does not remain stuck forever.

The lesson: error handling contracts must be explicit and enforced. A function that compares error strings is a single point of failure for an entire control flow. The `IsNotFound` function's contract (that all providers return unwrapped errors with exactly the string `"not found"`) was undocumented and unenforced. When the implementation changed to use real SDK errors, the contract broke silently.

## The namespace we forgot to create

The `main.go` configures leader election with namespace `cloudorch-system`.

```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
	Scheme:                  scheme,
	LeaderElection:          true,
	LeaderElectionID:        "cloudorch-leader-election",
	LeaderElectionNamespace: "cloudorch-system",
	HealthProbeBindAddress:  ":8081",
})
```

Every component assumes this namespace exists: the RBAC ClusterRoleBinding binds to ServiceAccount `cloudorch-manager` in namespace `cloudorch-system`. The Helm chart deploys into namespace `cloudorch-system`. The webhook manifests reference service `cloudorch-webhook` in namespace `cloudorch-system`.

But `main.go` does not create the namespace. If the operator is deployed without it, the manager fails to start leader election because the Lease object cannot be created, logs an error, and exits with code 1.

We discovered this during a fresh cluster deployment. The namespace did not exist. The operator pod crashed on startup. The logs showed a leader election error that was easy to miss among the other startup messages. We had assumed that Helm would create the namespace, but our `Makefile` deploy target did not pass `--create-namespace`.

We fixed this by adding a pre-flight check in `main.go` that creates the namespace if it does not exist, using a bootstrap Kubernetes client before starting the manager. We also updated the Makefile to pass `--create-namespace` to Helm and added a post-deploy verification that the namespace exists.

The lesson: operators that assume pre-existing infrastructure are common (Argo CD, Flux, and cert-manager all make the same assumption), but the assumption must be documented and enforced. A bootstrap check that creates required namespaces, or a clear error message that tells the user what to create, prevents an entire class of deployment failures.

## The interface that swallowed the implementation

The `CloudProvider` interface defines 33 methods. The operator only uses four of them: `GetCluster`, `CreateCluster`, `UpdateCluster`, `DeleteCluster`. The remaining 29 methods, covering databases, object stores, cache clusters, virtual networks, load balancers, DNS zones, and security policies, are stubbed in every provider.

```go
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

	EstimateMonthlyCost(ctx context.Context, resources []ResourceSpec) (*CostEstimate, error)
}
```

The same four-method pattern repeats for every remaining resource type, and only the first row is ever called.

| Resource type | Methods | Called by the operator |
|---------------|---------|------------------------|
| Cluster | 4 | yes |
| Database | 4 | no |
| ObjectStore | 4 | no |
| CacheCluster | 4 | no |
| VirtualNetwork | 4 | no |
| LoadBalancer | 4 | no |
| DNSZone | 4 | no |
| SecurityPolicy | 4 | no |
| Cost estimation | 1 | no |

This created a maintenance problem. When we added a new resource type to the interface, all three providers had to be updated with stub implementations or the code would not compile. The interface size scaled with the feature roadmap, not with the implemented feature set. The interface was a design document, not a production contract.

The status vocabulary problem compounded this. AWS returns `"ACTIVE"` and `"CREATING"`. GCP returns `"RUNNING"` and `"PROVISIONING"`. Azure returns `"Succeeded"` and `"Creating"`. The `computePlan` method checks `actual.Status != "ACTIVE"`. This check fails for GCP and Azure clusters, which are never considered active, so the reconciler always emits a `CREATE` operation for non-AWS providers. The status string comparison is hardcoded to AWS semantics, violating the strategy pattern's encapsulation.

We solved the interface problem by splitting `CloudProvider` into smaller interfaces: `ClusterProvider`, `DatabaseProvider`, `NetworkProvider`, and so on. The reconciler depends only on `ClusterProvider`. New resource types add new interfaces without breaking existing providers. We solved the status vocabulary problem by defining a canonical `ClusterHealth` enum in the provider package and requiring providers to return normalized status values rather than provider-specific strings.

The lesson: interfaces should be as small as the current implementation requires, not as large as the future roadmap imagines. A narrow interface allows incremental implementation. A broad interface forces stubbing and creates coupling between unrelated features. The Go proverb "the bigger the interface, the weaker the abstraction" is not an aesthetic preference. It is a maintenance strategy.

## The health checks that always passed

The `main.go` registers health checks using `healthz.Ping`.

```go
if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
	log.Error(err, "failed to add healthz check")
	os.Exit(1)
}
if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
	log.Error(err, "failed to add readyz check")
	os.Exit(1)
}
```

`healthz.Ping` always returns healthy. It does not verify that the provider connections work, that the policy engine compiled successfully, or that the webhook server is running. An operator with a broken provider or a corrupted policy engine will still report healthy.

We discovered this during an incident where the AWS provider's credentials had expired. The operator pod was running. The health checks passed. The reconciler was failing with authentication errors on every loop, but the liveness probe showed the pod was healthy. Kubernetes did not restart it. The cluster drifted for two hours before we noticed.

![Health checks before and after](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/diagrams/healthz.png)

We replaced `healthz.Ping` with functional health checks that verify the provider registry can list regions, the policy engine has compiled at least one policy, and the webhook server is listening on its port. The health check now fails if any of these components is non-functional, causing Kubernetes to restart the pod and trigger leader election on a healthy replica.

The lesson: health checks must verify functionality, not just liveness. In an operator that manages cloud resources, "healthy" must mean "able to create, update, and delete resources," not just "process is running."

## The Helm chart that disagreed with the binary

The Helm deployment template passes `--webhook-port={{ .Values.webhook.port }}` as a container argument. The controller-runtime manager does not automatically start a webhook server based on this flag. The operator code must explicitly create and start the webhook server. `main.go` does not do this. The flag is accepted but ignored.

The Helm chart also mounts a volume for webhook certificates from a secret that cert-manager is supposed to create. But cert-manager only creates certificates for webhooks that have a running service endpoint. The operator needs certificates to start the webhook. Cert-manager needs a webhook endpoint to issue certificates. The operator never starts the webhook. The circular dependency is complete.

We discovered this during a security audit. The webhook configuration was deployed, the certificates were expected, but the service did not exist. We had forgotten to add the service template to the Helm chart. Even if we had added it, the operator would not have started the webhook server. The entire webhook pipeline was a deployment artifact with no runtime presence.

We solved this by wiring the webhook server startup into `main.go`, adding the missing service template, and adding an integration test that deploys the Helm chart to a test cluster and verifies that the webhook responds to admission requests. The test catches mismatches between the manifests and the runtime configuration before they reach production.

The lesson: deployment artifacts must be tested as a whole. The Helm chart, the webhook manifests, the RBAC, and the operator binary form a single deployable unit. Testing each in isolation misses integration failures. A CI pipeline that deploys to a test cluster and verifies end-to-end behavior catches errors that unit tests and `helm template` cannot.

## What the architecture got right

Despite these incidents, the structural decisions were correct. The finalizer pattern works as designed: the reconciler checks `DeletionTimestamp`, destroys the cloud resource, and removes the finalizer only after the cloud API confirms deletion. The level-triggered reconciler converges automatically after missed events or partial failures. The strategy-pattern provider interface allows pluggable cloud backends. The OPA policy engine evaluates rules in-process with low latency. The status subresource separates spec from observation.

```go
func (r *CloudClusterReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log, _ := logr.FromContext(ctx)

	var cluster computev1.CloudCluster
	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	provider, err := r.Providers.For(cluster.Spec.Provider)
	if err != nil {
		log.Error(err, "unknown provider", "provider", cluster.Spec.Provider)
		return r.setCondition(ctx, &cluster, "Ready", "UnknownProvider", err.Error())
	}

	if !cluster.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, log, &cluster, provider)
	}

	if !controllerutil.ContainsFinalizer(&cluster, cloudOrchFinalizer) {
		controllerutil.AddFinalizer(&cluster, cloudOrchFinalizer)
		return ctrl.Result{}, r.Update(ctx, &cluster)
	}

	violations, err := r.Policy.Evaluate(ctx, &cluster)
	if err != nil {
		log.Error(err, "policy evaluation failed")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}
	if len(violations) > 0 {
		msg := formatViolations(violations)
		return r.setCondition(ctx, &cluster, "Ready", "PolicyViolation", msg)
	}

	actual, err := provider.GetCluster(ctx, cloud.ClusterID{
		Name:   cluster.Name,
		Region: cluster.Spec.Region,
	})
	if err != nil && !cloud.IsNotFound(err) {
		log.Error(err, "failed to get cluster from cloud")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	plan := r.computePlan(cluster.Spec, actual)
	if err := r.executePlan(ctx, log, cluster.Spec.Region, provider, plan); err != nil {
		return r.setCondition(ctx, &cluster, "Ready", "ReconcileFailed", err.Error())
	}

	return r.setCondition(ctx, &cluster, "Ready", "Synced", "Cluster reconciled successfully")
}
```
The `ProviderRegistry` wraps a map of named providers with a `For` lookup. Adding a new provider means implementing the interface and registering it in `main.go`. The interface is large (33 methods across nine resource types), but each resource type is self-contained. The design allows incremental implementation, even if our initial implementation was incomplete.

The policy engine's `sync.RWMutex` allows concurrent evaluation during hot reload. The `RLock` in `Evaluate` ensures that policy evaluation does not block while the compiler is being swapped. This is the right concurrency pattern for a read-heavy workload.

The webhook server's TLS configuration uses `tls.VersionTLS13` minimum. The 5-second read and write timeouts are appropriate for OPA evaluation. The Helm chart includes pod security context with `runAsNonRoot: true`, `readOnlyRootFilesystem: true`, and `allowPrivilegeEscalation: false`. The pod anti-affinity rule spreads replicas across nodes. These are correct production settings.

The `CloudCluster` CRD includes `+kubebuilder:subresource:status` and `+patchStrategy=merge` on the conditions array. The `main.go` uses `ctrl.SetupSignalHandler()` for graceful shutdown. The `Makefile` runs `go test -v -race -coverprofile=coverage.out`. The `-race` flag catches concurrency bugs. The infrastructure for a production operator is present.

## Principles from production incidents

**Stubs are production code until replaced.** The AWS provider returned fictional cluster states that compiled, passed type checks, and satisfied the interface. A stub that returns plausible data is more dangerous than one that fails loudly. We now require all stub implementations to return explicit `errors.New("not implemented")` and gate the operator behind a feature flag that requires at least one real provider.

**Wiring is the work.** Deployment artifacts are necessary but not sufficient. The operator must start the server, register handlers, bind the port. `Server.Start` not being called is the bug. A webhook that exists in YAML but not in runtime is not a webhook. It is a specification with a compiler.

**Policy is a control, not a feature.** A compiled Rego module that no one evaluates is decoration. A policy engine that fails silently is a security control providing false confidence. Policy must be wired into every path that could violate it: admission webhooks, reconciler evaluation, drift detection.

**Status must reflect reality.** Two representations of the same fact will diverge. Derive booleans from conditions. A status subresource that shows `READY=true` while the condition says `PolicyViolation` is not observability. It is noise.

**An incomplete production system is the most dangerous state.** It has a stable API, a Helm chart, a Makefile, RBAC, webhook manifests, and a README promising production readiness. It looks finished. It behaves like something that should work. But it does not. A prototype can be thrown away. An incomplete production system cannot.

**Test deployment as a whole.** The Helm chart, webhook manifests, RBAC, and operator binary form a single deployable unit. Testing each in isolation misses integration failures. Our integration test caught three wiring bugs that unit tests and static analysis missed.

## What we changed before the next release

| Gap | Symptom in production | Fix |
|-----|----------------------|-----|
| Incomplete `computePlan` | Silent instance-type drift | Diff all mutable fields, add nil guards |
| Stubbed AWS provider | Clusters that existed only in status | Real SDK calls; explicit `not implemented` |
| Policy query and input | Cost and instance rules never fired | Populate cost; derive `allow` from violations |
| Webhook YAML vs runtime | Admission path completely open | Wire `Start`, align paths, add Service |
| Divergent `Ready` boolean | `READY=true` while condition was False | Derive flags from the condition array |
| Stringly `IsNotFound` | Objects stuck in `Terminating` | `errors.Is` sentinel plus max retries |
| `healthz.Ping` | Healthy pod, dead control loop | Functional readiness checks |
| Fat `CloudProvider` | Stub tax on every new feature | Split into narrow interfaces |

We replaced stubbed provider methods with explicit `errors.New("not implemented")`. We wired the webhook server startup into `main.go` and added the missing service template. We unified the instance type allowlist into a single source that generates both the CRD enum and the Rego policy. We fixed `setCondition` to derive boolean fields from the condition array. We rewrote `IsNotFound` to use `errors.Is` against a sentinel error. We expanded `computePlan` to handle all mutable fields. We connected `HotReload` to a ConfigMap watch. We added exponential backoff with jitter to requeue intervals. We normalized provider status strings into a canonical vocabulary. We replaced `healthz.Ping` with functional health checks.

The architecture was always sound. The level-triggered reconciler, the finalizer pattern, the strategy-pattern providers, and the OPA policy engine are correct designs. CloudOrch earned the right to teach those patterns. But the implementation had not earned the right to run. Closing that gap required finishing the wiring, replacing the stubs, and testing the deployment as a whole.

The operator now reconciles real infrastructure. The webhook enforces policy at admission. The status subresource reflects actual state. The gap is closed, not by changing the architecture, but by completing the discipline the architecture assumed.

---

## References

- [Kubernetes SIG Architecture: Finalizers](https://kubernetes.io/docs/concepts/architecture/garbage-collection/#finalizers)
- [Kubernetes SIG Architecture: Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks)
- [Kubernetes SIG API Machinery: Status Conditions](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource)
- [Open Policy Agent: Rego Language Reference](https://www.openpolicyagent.org/docs/latest/policy-reference/)
- [controller-runtime Documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- Sridharan, C. *Distributed Systems Observability* (O'Reilly, 2018)
- Majors, Fong-Jones, Miranda. *Observability Engineering* (O'Reilly, 2022)
- Beyer, Jones, Petoff, Murphy. *Site Reliability Engineering* (Google / O'Reilly, 2016)

---

![CloudOrch](https://raw.githubusercontent.com/rowjay007/cloudorch/main/article/assets/png/logos/cloudorch.png)

*Closing the gap is finishing the wiring.*
