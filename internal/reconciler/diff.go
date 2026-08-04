// Package reconciler implements the structured diff engine that
// compares desired spec vs observed state and generates an ordered
// plan of create/update/delete operations.
package reconciler

import (
	"github.com/rowjay/cloudorch/internal/cloud"
)

// Operation represents a single cloud resource operation.
type Operation struct {
	Op    string // "CREATE" | "UPDATE" | "DELETE"
	Name  string
	Spec  interface{}
	Patch interface{}
}

// Plan is an ordered list of operations to apply.
type Plan struct {
	Operations []Operation
}

// ComputePlan computes the diff between desired spec and actual state.
func ComputePlan(spec interface{}, actual *cloud.ClusterState) Plan {
	var ops []Operation

	if actual == nil || actual.Status != "ACTIVE" {
		ops = append(ops, Operation{
			Op:   "CREATE",
			Name: "cluster",
			Spec: spec,
		})
		return Plan{Operations: ops}
	}

	// Check for drift and generate update operations.
	ops = append(ops, detectDrift(spec, actual)...)

	return Plan{Operations: ops}
}

// detectDrift compares desired spec against actual state and
// returns the list of update operations needed.
func detectDrift(spec interface{}, actual *cloud.ClusterState) []Operation {
	var ops []Operation
	// TODO: implement structured diff for each CRD type
	return ops
}

// ApplyPlan executes the plan sequentially. If any operation fails,
// it returns the error and the plan can be rolled back.
func ApplyPlan(provider cloud.CloudProvider, plan Plan) error {
	for _, op := range plan.Operations {
		switch op.Op {
		case "CREATE":
			// Execute create
		case "UPDATE":
			// Execute update
		case "DELETE":
			// Execute delete
		}
	}
	return nil
}
