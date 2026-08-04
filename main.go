package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	computev1 "github.com/rowjay/cloudorch/api/compute/v1"
	data1 "github.com/rowjay/cloudorch/api/data/v1"
	network1 "github.com/rowjay/cloudorch/api/network/v1"
	security1 "github.com/rowjay/cloudorch/api/security/v1"
	"github.com/rowjay/cloudorch/internal/cloud"
	"github.com/rowjay/cloudorch/internal/cloud/aws"
	"github.com/rowjay/cloudorch/internal/cloud/azure"
	"github.com/rowjay/cloudorch/internal/cloud/gcp"
	"github.com/rowjay/cloudorch/internal/controllers"
	"github.com/rowjay/cloudorch/internal/cost"
	"github.com/rowjay/cloudorch/internal/drift"
	"github.com/rowjay/cloudorch/internal/policy"
	"github.com/rowjay/cloudorch/internal/reconciler"
	"github.com/rowjay/cloudorch/internal/webhook"
)

const (
	operatorName = "cloudorch"
	webhookPort  = 9443
)

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	log := ctrl.Log.WithName(operatorName)
	ctx := ctrl.SetupSignalHandler()

	var scheme = runtime.NewScheme()
	_ = computev1.AddToScheme(scheme)
	_ = data1.AddToScheme(scheme)
	_ = network1.AddToScheme(scheme)
	_ = security1.AddToScheme(scheme)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		LeaderElection:          true,
		LeaderElectionID:        "cloudorch-leader-election",
		LeaderElectionNamespace: "cloudorch-system",
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookPort,
			CertDir: "/tmp/k8s-webhook-server/serving-certs",
		}),
		Metrics: filters.WithAuthenticationAndAuthorization,
		HealthProbeBindAddress: ":8081",
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
	gcpProvider := gcp.NewGCPProvider(os.Getenv("GCP_REGION"))
	azureProvider := azure.NewAzureProvider(os.Getenv("AZURE_REGION"))

	providers := cloud.NewProviderRegistry(map[string]cloud.CloudProvider{
		"aws":   awsProvider,
		"gcp":   gcpProvider,
		"azure": azureProvider,
	})

	policyEngine, err := policy.NewEngine(ctx, policy.Config{
		RegoPath:     "/etc/cloudorch/policies",
		HotReload:    true,
		PollInterval: 30 * time.Second,
	})
	if err != nil {
		log.Error(err, "failed to create policy engine")
		os.Exit(1)
	}

	costOptimizer := cost.NewOptimizer()

	driftDetector := drift.NewDetector(drift.Config{
		Interval: 5 * time.Minute,
		Providers: providers,
		Policy:    policyEngine,
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Log:       log.WithName("drift-detector"),
	})

	if err := (&controllers.CloudClusterReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Providers: providers,
		Policy:    policyEngine,
		Cost:      costOptimizer,
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "failed to setup CloudCluster reconciler")
		os.Exit(1)
	}

	if err := setupWebhooks(mgr, policyEngine); err != nil {
		log.Error(err, "failed to setup webhooks")
		os.Exit(1)
	}

	diffEngine := reconciler.NewDiffEngine(reconciler.Config{
		MaxConcurrency: 50,
		RetryBackoff:   100 * time.Millisecond,
	})
	go diffEngine.Start(ctx)

	go driftDetector.Start(ctx)

	certWatcher, err := certwatcher.New("/tmp/k8s-webhook-server/serving-certs", nil)
	if err != nil {
		log.Error(err, "failed to create cert watcher")
		os.Exit(1)
	}
	go func() {
		if err := certWatcher.Start(ctx); err != nil {
			log.Error(err, "cert watcher failed")
		}
	}()

	log.Info("starting CloudOrch operator",
		"version", "v0.1.0",
		"webhookPort", webhookPort,
		"providers", []string{"aws", "gcp", "azure"},
	)

	if err := mgr.Start(ctx); err != nil {
		log.Error(err, "manager exited non-zero")
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan
	log.Info("shutting down CloudOrch operator")
}

func setupWebhooks(mgr ctrl.Manager, policyEngine *policy.Engine) error {
	validating := &webhook.ValidatingWebhook{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Policy: policyEngine,
	}

	mutating := &webhook.MutatingWebhook{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}

	mgr.GetWebhookServer().Register("/validate-cloudclusters", &webhook.AdmissionHandler{
		Webhook: validating,
	})
	mgr.GetWebhookServer().Register("/mutate-cloudclusters", &webhook.AdmissionHandler{
		Webhook: mutating,
	})

	return nil
}

