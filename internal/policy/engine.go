// Package policy implements the OPA Rego policy engine for CloudOrch.
// Policies are loaded from ConfigMap or OCI registry and hot-reloaded
// on ConfigMap update without operator restart.
package policy

import (
	"context"
	"fmt"
	"sync"

	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
)

// Violation represents a single policy violation.
type Violation struct {
	Rule     string
	Message  string
	Severity string // "error" | "warning"
}

// Engine evaluates OPA Rego policies against CloudOrch CRDs.
type Engine struct {
	mu       sync.RWMutex
	compiler *rego.Compiler
	policies []string
	logger   *zap.Logger
}

// NewEngine creates a new policy engine with the given Rego policies.
func NewEngine(policies []string, logger *zap.Logger) *Engine {
	e := &Engine{
		policies: policies,
		logger:   logger,
	}
	e.compile()
	return e
}

// compile rebuilds the OPA compiler from the current policy set.
func (e *Engine) compile() {
	var modules []rego.Module
	for i, p := range e.policies {
		modules = append(modules, rego.Module{
			FileName: fmt.Sprintf("policy_%d.rego", i),
			Raw:      []byte(p),
		})
	}

	compiler := rego.NewCompiler()
	compiler.Compile(modules)
	if compiler.Failed() {
		e.logger.Error(fmt.Errorf("policy compilation failed"), "errors", compiler.Errors)
		return
	}

	e.mu.Lock()
	e.compiler = compiler
	e.mu.Unlock()
	e.logger.Info("policy engine compiled successfully", "policies", len(policies))
}

// Evaluate evaluates all policies against a CloudCluster CR.
// Returns a list of violations (empty if all policies pass).
func (e *Engine) Evaluate(ctx context.Context, cluster interface{}) ([]Violation, error) {
	e.mu.RLock()
	compiler := e.compiler
	e.mu.RUnlock()

	if compiler == nil {
		return nil, fmt.Errorf("policy engine not compiled")
	}

	// Prepare the input document for OPA evaluation.
	input := map[string]interface{}{
		"cluster": cluster,
	}

	// Evaluate the "cloudorch" policy set.
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

// HotReload replaces the current policy set and recompiles.
func (e *Engine) HotReload(policies []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies = policies
	e.compile()
}

// DefaultPolicies returns the built-in Rego policies as strings.
func DefaultPolicies() []string {
	return []string{
		// allowed_regions.rego
		`package cloudorch

default allow = true

# Allowed regions per provider
allowed_regions["aws"] = ["us-east-1", "us-east-2", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-southeast-1"]
allowed_regions["gcp"] = ["us-east1", "us-west1", "europe-west1", "asia-northeast1", "asia-southeast1"]
allowed_regions["azure"] = ["eastus", "westus", "northeurope", "westeurope", "southeastasia"]

violation[msg] {
    input.cluster.spec.provider == provider
    not allowed_regions[provider][_] == input.cluster.spec.region
    msg := sprintf("Region %q is not allowed for provider %q", [input.cluster.spec.region, provider])
}
`,
		// instance_types.rego
		`package cloudorch

# Allowed instance types per provider
allowed_instance_types["aws"] = ["t3.medium", "t3.large", "t3.xlarge", "m5.large", "m5.xlarge", "m5.2xlarge"]
allowed_instance_types["gcp"] = ["e2-medium", "e2-standard-2", "e2-standard-4", "n1-standard-1", "n1-standard-2", "n2-standard-2"]
allowed_instance_types["azure"] = ["Standard_B2s", "Standard_D2s_v3", "Standard_D4s_v3", "Standard_D8s_v3"]

violation[msg] {
    input.cluster.spec.provider == provider
    not allowed_instance_types[provider][_] == input.cluster.spec.instanceType
    msg := sprintf("Instance type %q is not allowed for provider %q", [input.cluster.spec.instanceType, provider])
}
`,
		// required_tags.rego
		`package cloudorch

required_tags = ["environment", "team", "cost-center"]

violation[msg] {
    tag := required_tags[_]
    not input.cluster.spec.tags[tag]
    msg := sprintf("Required tag %q is missing", [tag])
}
`,
		// cost_threshold.rego
		`package cloudorch

# Default cost threshold: $5000/month
cost_threshold = 5000

violation[msg] {
    input.cluster.spec.estimatedMonthlyCost > cost_threshold
    msg := sprintf("Estimated monthly cost $%.2f exceeds threshold $%.2f", [input.cluster.spec.estimatedMonthlyCost, cost_threshold])
}
`,
	}
}
