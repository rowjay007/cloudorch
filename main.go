package main

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	computev1 "github.com/rowjay/cloudorch/api/compute/v1"
	"github.com/rowjay/cloudorch/internal/cloud"
	"github.com/rowjay/cloudorch/internal/cloud/aws"
	"github.com/rowjay/cloudorch/internal/controllers"
	"github.com/rowjay/cloudorch/internal/policy"
)

const (
	operatorName = "cloudorch"
)

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	log := ctrl.Log.WithName(operatorName)
	ctx := ctrl.SetupSignalHandler()

	var scheme = runtime.NewScheme()
	_ = computev1.AddToScheme(scheme)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		LeaderElection:          true,
		LeaderElectionID:        "cloudorch-leader-election",
		LeaderElectionNamespace: "cloudorch-system",
		HealthProbeBindAddress:  ":8081",
	})
	if err != nil {
		log.Error(err, "failed to create manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "failed to add healthz check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "failed to add readyz check")
		os.Exit(1)
	}

	awsProvider := aws.NewAWSProvider(os.Getenv("AWS_REGION"))
	providers := cloud.NewProviderRegistry(map[string]cloud.CloudProvider{
		"aws": awsProvider,
	})

	policyEngine := policy.NewEngine(policy.DefaultPolicies(), nil)

	if err := (&controllers.CloudClusterReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Providers: providers,
		Policy:    policyEngine,
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "failed to setup CloudCluster reconciler")
		os.Exit(1)
	}

	log.Info("starting CloudOrch operator",
		"version", "v0.1.0",
		"providers", []string{"aws"},
	)

	if err := mgr.Start(ctx); err != nil {
		log.Error(err, "manager exited non-zero")
		os.Exit(1)
	}
}