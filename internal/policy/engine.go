package policy

import (
	"context"
	"fmt"
	"sync"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"go.uber.org/zap"
)

type Violation struct {
	Rule     string
	Message  string
	Severity string
}

type Engine struct {
	mu       sync.RWMutex
	compiler *ast.Compiler
	policies []string
	logger   *zap.Logger
}

func NewEngine(policies []string, logger *zap.Logger) *Engine {
	e := &Engine{
		policies: policies,
		logger:   logger,
	}
	e.compile()
	return e
}

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
	e.logger.Info("policy engine compiled successfully", zap.Int("policies", len(e.policies)))
}

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

func (e *Engine) HotReload(policies []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies = policies
	e.compile()
}

func DefaultPolicies() []string {
	return []string{
		`package cloudorch

default allow = true

allowed_regions["aws"] = ["us-east-1", "us-east-2", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-southeast-1"]
allowed_regions["gcp"] = ["us-east1", "us-west1", "europe-west1", "asia-northeast1", "asia-southeast1"]
allowed_regions["azure"] = ["eastus", "westus", "northeurope", "westeurope", "southeastasia"]

violation[msg] {
    input.cluster.spec.provider == provider
    not allowed_regions[provider][_] == input.cluster.spec.region
    msg := sprintf("Region %q is not allowed for provider %q", [input.cluster.spec.region, provider])
}
`,
		`package cloudorch

allowed_instance_types["aws"] = ["t3.medium", "t3.large", "t3.xlarge", "m5.large", "m5.xlarge", "m5.2xlarge"]
allowed_instance_types["gcp"] = ["e2-medium", "e2-standard-2", "e2-standard-4", "n1-standard-1", "n1-standard-2", "n2-standard-2"]
allowed_instance_types["azure"] = ["Standard_B2s", "Standard_D2s_v3", "Standard_D4s_v3", "Standard_D8s_v3"]

violation[msg] {
    input.cluster.spec.provider == provider
    not allowed_instance_types[provider][_] == input.cluster.spec.instanceType
    msg := sprintf("Instance type %q is not allowed for provider %q", [input.cluster.spec.instanceType, provider])
}
`,
		`package cloudorch

required_tags = ["environment", "team", "cost-center"]

violation[msg] {
    tag := required_tags[_]
    not input.cluster.spec.tags[tag]
    msg := sprintf("Required tag %q is missing", [tag])
}
`,
		`package cloudorch

cost_threshold = 5000

violation[msg] {
    input.cluster.spec.estimatedMonthlyCost > cost_threshold
    msg := sprintf("Estimated monthly cost $%.2f exceeds threshold $%.2f", [input.cluster.spec.estimatedMonthlyCost, cost_threshold])
}
`,
	}
}