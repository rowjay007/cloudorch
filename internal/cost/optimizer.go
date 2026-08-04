// Package cost implements the cost optimizer that queries real-time
// pricing APIs and annotates CRs with estimated monthly costs.
package cost

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/rowjay/cloudorch/internal/cloud"
	computev1 "github.com/rowjay/cloudorch/api/compute/v1"
)

// Optimizer queries cloud pricing APIs and annotates CRs with cost estimates.
type Optimizer struct {
	client.Client
	Scheme    *runtime.Scheme
	Providers *cloud.ProviderRegistry
	Log       logr.Logger
}

// NewOptimizer creates a new cost optimizer.
func NewOptimizer(mgr manager.Manager, providers *cloud.ProviderRegistry) *Optimizer {
	return &Optimizer{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Providers: providers,
		Log:       ctrl.Log.WithName("cost-optimizer"),
	}
}

// AnnotateEstimate queries the cloud pricing API and annotates
// the CR with an estimated monthly cost.
func (o *Optimizer) AnnotateEstimate(ctx context.Context, cluster *computev1.CloudCluster, provider cloud.CloudProvider) error {
	resources := []cloud.ResourceSpec{
		{
			Type:         "cluster",
			Provider:     cluster.Spec.Provider,
			Region:       cluster.Spec.Region,
			InstanceType: cluster.Spec.InstanceType,
		},
	}

	estimate, err := provider.EstimateMonthlyCost(ctx, resources)
	if err != nil {
		o.Log.Error(err, "failed to estimate cost", "cluster", cluster.Name)
		return err
	}

	// Update the CR status with the cost estimate.
	cluster.Status.EstimatedMonthlyCost = estimate.MonthlyCost
	return o.Status().Update(ctx, cluster)
}

// SuggestRightsizing analyzes cluster usage and suggests rightsizing.
func (o *Optimizer) SuggestRightsizing(ctx context.Context, cluster *computev1.CloudCluster) []Right-sizingSuggestion {
	// TODO: integrate cloud-specific cost advisor APIs
	return nil
}

// Right-sizingSuggestion represents a cost optimization recommendation.
type Right-sizingSuggestion struct {
	ResourceID   string
	CurrentType  string
	SuggestedType string
	MonthlySavings float64
	Reason       string
}
