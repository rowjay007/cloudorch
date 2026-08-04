// Package drift implements the drift detector that periodically
// compares desired state (etcd) vs actual state (cloud APIs).
package drift

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/rowjay/cloudorch/internal/cloud"
)

// Detector runs scheduled full-reconcile loops to detect drift.
type Detector struct {
	client.Client
	Scheme    *runtime.Scheme
	Providers *cloud.ProviderRegistry
	Interval  time.Duration
	Log       logr.Logger
}

// NewDetector creates a new drift detector.
func NewDetector(mgr manager.Manager, providers *cloud.ProviderRegistry, interval time.Duration) *Detector {
	return &Detector{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Providers: providers,
		Interval:  interval,
		Log:       ctrl.Log.WithName("drift-detector"),
	}
}

// Start runs the drift detection loop.
func (d *Detector) Start(ctx context.Context) error {
	ticker := time.NewTicker(d.Interval)
	defer ticker.Stop()

	// Run immediately on start.
	d.detect(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			d.detect(ctx)
		}
	}
}

// detect performs a full drift check across all CloudCluster CRs.
func (d *Detector) detect(ctx context.Context) {
	d.Log.Info("starting drift detection")

	var clusters computev1.CloudClusterList
	if err := d.List(ctx, &clusters); err != nil {
		d.Log.Error(err, "failed to list cloud clusters")
		return
	}

	for i := range clusters.Items {
		cluster := &clusters.Items[i]
		provider, err := d.Providers.For(cluster.Spec.Provider)
		if err != nil {
			d.Log.Error(err, "unknown provider", "cluster", cluster.Name)
			continue
		}

		actual, err := provider.GetCluster(ctx, cloud.ClusterID{
			Name:   cluster.Name,
			Region: cluster.Spec.Region,
		})
		if err != nil {
			d.Log.Error(err, "failed to get cluster state", "cluster", cluster.Name)
			continue
		}

		// Compare desired vs actual.
		if actual == nil || actual.Status != "ACTIVE" {
			d.Log.Info("drift detected: cluster not active", "cluster", cluster.Name, "region", cluster.Spec.Region)
			continue
		}

		if actual.NodeCount != cluster.Spec.NodeCount {
			d.Log.Info("drift detected: node count mismatch",
				"cluster", cluster.Name,
				"desired", cluster.Spec.NodeCount,
				"actual", actual.NodeCount,
			)
		}
	}

	d.Log.Info("drift detection complete")
}
