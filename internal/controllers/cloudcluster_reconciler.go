package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	computev1 "github.com/rowjay/cloudorch/api/compute/v1"
	"github.com/rowjay/cloudorch/internal/cloud"
	"github.com/rowjay/cloudorch/internal/policy"
)

const cloudOrchFinalizer = "cloudorch.io/finalizer"

type CloudClusterReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Providers *cloud.ProviderRegistry
	Policy    *policy.Engine
}

func (r *CloudClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.Log.FromContext(ctx)

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
		return r.setCondition(ctx, &cluster, "Ready", "PolicyViolation", formatViolations(violations))
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
	if err := r.executePlan(ctx, log, provider, plan); err != nil {
		return r.setCondition(ctx, &cluster, "Ready", "ReconcileFailed", err.Error())
	}

	return r.setCondition(ctx, &cluster, "Ready", "Synced", "Cluster reconciled successfully")
}

func (r *CloudClusterReconciler) handleDeletion(ctx context.Context, log logr.Logger, cluster *computev1.CloudCluster, provider cloud.CloudProvider) (ctrl.Result, error) {
	clusterID := cloud.ClusterID{Name: cluster.Name, Region: cluster.Spec.Region}
	_, err := provider.GetCluster(ctx, clusterID)
	if err == nil {
		log.Info("destroying cloud cluster", "name", cluster.Name, "region", cluster.Spec.Region)
		if err := provider.DeleteCluster(ctx, clusterID); err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Cloud resource is gone. Remove finalizer.
	if controllerutil.ContainsFinalizer(cluster, cloudOrchFinalizer) {
		controllerutil.RemoveFinalizer(cluster, cloudOrchFinalizer)
		return ctrl.Result{}, r.Update(ctx, cluster)
	}

	return ctrl.Result{}, nil
}

func (r *CloudClusterReconciler) computePlan(spec computev1.CloudClusterSpec, actual *cloud.ClusterState) []cloud.Operation {
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

func (r *CloudClusterReconciler) executePlan(ctx context.Context, log logr.Logger, provider cloud.CloudProvider, plan []cloud.Operation) error {
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
			err := provider.UpdateCluster(ctx, cloud.ClusterID{Name: op.Name, Region: ""}, op.Patch.(cloud.ClusterPatch))
			if err != nil {
				return err
			}
		case "DELETE":
			log.Info("deleting cluster", "name", op.Name)
			err := provider.DeleteCluster(ctx, cloud.ClusterID{Name: op.Name, Region: ""})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *CloudClusterReconciler) setCondition(ctx context.Context, cluster *computev1.CloudCluster, conditionType, reason, message string) (ctrl.Result, error) {
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

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, r.Status().Update(ctx, cluster)
}

func (r *CloudClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.CloudCluster{}).
		Owns(&computev1.NodePool{}).
		Complete(r)
}

func formatViolations(violations []policy.Violation) string {
	msg := "Policy violations: "
	for i, v := range violations {
		if i > 0 {
			msg += "; "
		}
		msg += v.Message
	}
	return msg
}
