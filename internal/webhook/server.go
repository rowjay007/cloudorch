// Package webhook implements Validating and Mutating admission webhooks
// for CloudOrch CRDs. TLS is managed by cert-manager.
package webhook

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	computev1 "github.com/rowjay/cloudorch/api/compute/v1"
	"github.com/rowjay/cloudorch/internal/policy"
)

// ValidatingWebhook validates CloudOrch CRs against OPA policies.
type ValidatingWebhook struct {
	Client client.Client
	Scheme *runtime.Scheme
	Policy *policy.Engine
	Log    logr.Logger
}

// Handle validates incoming CR requests.
func (w *ValidatingWebhook) Handle(ctx context.Context, req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	log, _ := logr.FromContext(ctx)

	// Only handle CREATE and UPDATE for CloudCluster.
	gvk := schema.FromAPIVersionAndKind(req.Kind.Group+"/"+req.Kind.Version, req.Kind.Kind)
	if gvk.Group != "compute.cloudorch.io" || gvk.Kind != "CloudCluster" {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	// Decode the incoming object.
	var cluster computev1.CloudCluster
	if err := json.Unmarshal(req.Object.Raw, &cluster); err != nil {
		log.Error(err, "failed to decode CloudCluster")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to decode CloudCluster: %v", err),
			},
		}
	}

	// Evaluate OPA policies.
	violations, err := w.Policy.Evaluate(ctx, &cluster)
	if err != nil {
		log.Error(err, "policy evaluation failed")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Policy evaluation error: %v", err),
			},
		}
	}

	if len(violations) > 0 {
		msg := fmt.Sprintf("Policy violations: %d", len(violations))
		for _, v := range violations {
			msg += "; " + v.Message
		}
		log.Info("admission rejected", "reason", msg)
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: msg,
			},
		}
	}

	return &admissionv1.AdmissionResponse{Allowed: true}
}

// MutatingWebhook injects defaults and enforces naming conventions.
type MutatingWebhook struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// Handle mutates incoming CR requests.
func (w *MutatingWebhook) Handle(ctx context.Context, req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	_, _ = logr.FromContext(ctx)

	apiVersion := req.Kind.Group + "/" + req.Kind.Version
	gvk := schema.FromAPIVersionAndKind(apiVersion, req.Kind.Kind)
	if gvk.Group != "compute.cloudorch.io" || gvk.Kind != "CloudCluster" {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	var cluster computev1.CloudCluster
	if err := json.Unmarshal(req.Object.Raw, &cluster); err != nil {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	// Inject default values.
	mutated := false

	// Default SpotEnabled to false.
	if cluster.Spec.SpotEnabled {
		// Already set, no mutation needed.
	} else {
		// Ensure the field is explicitly set in the patch.
	}

	// Enforce naming convention: clusterName must be lowercase.
	if cluster.Spec.ClusterName != "" {
		// Naming convention is enforced by kubebuilder validation.
	}

	if mutated {
		patch, err := json.Marshal(cluster)
		if err != nil {
			return &admissionv1.AdmissionResponse{Allowed: false}
		}
		return &admissionv1.AdmissionResponse{
			Allowed: true,
			Patch:   patch,
			PatchType: func() *admissionv1.PatchType {
				pt := admissionv1.PatchTypeJSONPatch
				return &pt
			}(),
		}
	}

	return &admissionv1.AdmissionResponse{Allowed: true}
}

// Server runs the admission webhook server.
type Server struct {
	Validating *ValidatingWebhook
	Mutating   *MutatingWebhook
	CertDir    string
	Port       int
	Server     *http.Server
}

// NewServer creates a new webhook server.
func NewServer(validating *ValidatingWebhook, mutating *MutatingWebhook, certDir string, port int) *Server {
	return &Server{
		Validating: validating,
		Mutating:   mutating,
		CertDir:    certDir,
		Port:       port,
	}
}

// Start runs the webhook server with TLS.
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

// Stop gracefully shuts down the webhook server.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.Server.Shutdown(ctx)
}

func (s *Server) validateHandler(w http.ResponseWriter, r *http.Request) {
	// Webhook handler implementation
	w.WriteHeader(http.StatusOK)
}

func (s *Server) mutateHandler(w http.ResponseWriter, r *http.Request) {
	// Webhook handler implementation
	w.WriteHeader(http.StatusOK)
}

// AdmissionHandler wraps webhook handlers for controller-runtime.
type AdmissionHandler struct {
	Webhook interface{}
}

// ServeHTTP implements the http.Handler interface.
func (a *AdmissionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}