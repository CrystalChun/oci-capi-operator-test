/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	securityv1 "github.com/openshift/api/security/v1"

	"github.com/go-logr/logr"
	"github.com/kelseyhightower/envconfig"
	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/openshift/oci-capi-operator/internal/components/capoci"
	"github.com/openshift/oci-capi-operator/internal/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))

	utilruntime.Must(ocicapioperatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(infrastructurev1beta2.AddToScheme(scheme))
	utilruntime.Must(capiv1beta1.AddToScheme(scheme))
	utilruntime.Must(securityv1.Install(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	ctrl.SetLogger(zap.New(zap.JSONEncoder(func(o *zapcore.EncoderConfig) {
		o.EncodeTime = zapcore.RFC3339TimeEncoder
	})))
	cmd := &cobra.Command{
		Use: "oci-capi-operator",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
			os.Exit(1)
		},
	}
	cmd.AddCommand(NewInitCommand())
	cmd.AddCommand(NewRunCommand())

	if err := cmd.Execute(); err != nil {
		setupLog.Error(err, "problem running operator")
		os.Exit(1)
	}
}

type Options struct {
	CAPOCICredentials capoci.CAPOCICredentials
	RunOptions        RunOptions
}

type RunOptions struct {
	MetricsAddr          string
	EnableLeaderElection bool
	ProbeAddr            string
	SecureMetrics        bool
	EnableHTTP2          bool
}

func run(ctx context.Context, options Options, setupLog *logr.Logger) error {

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !options.RunOptions.EnableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsOpts := ctrl.Options{
		Scheme:                 scheme,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: options.RunOptions.ProbeAddr,
		LeaderElection:         options.RunOptions.EnableLeaderElection,
		LeaderElectionID:       "1af242a3.openshift.io",
	}
	if options.RunOptions.MetricsAddr != "0" {
		metricsOpts.HealthProbeBindAddress = options.RunOptions.MetricsAddr
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), metricsOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	if err = envconfig.Process("", &options); err != nil {
		setupLog.Error(err, "failed to process environment variables")
		os.Exit(1)
	}

	setupLog.Info("OCI credentials", "tenancyID", options.CAPOCICredentials.TenancyID, "userID", options.CAPOCICredentials.UserID, "region", options.CAPOCICredentials.Region, "fingerprint", options.CAPOCICredentials.Fingerprint)

	if err = (&controllers.OCIClusterAutoscalerReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		CAPOCICredentials: options.CAPOCICredentials,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OCIClusterAutoscaler")
		return err
	}

	if err = (&controllers.CertificateApprovalReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CertificateApproval")
		return err
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}

func NewRunCommand() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Runs the OCI CAPI operator",
	}

	options := Options{}

	runCmd.Flags().StringVar(&options.RunOptions.MetricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	runCmd.Flags().StringVar(&options.RunOptions.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	runCmd.Flags().BoolVar(&options.RunOptions.EnableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	runCmd.Flags().BoolVar(&options.RunOptions.SecureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	runCmd.Flags().BoolVar(&options.RunOptions.EnableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	runCmd.Run = func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		setupLog = ctrl.Log.WithName("setup")
		if err := run(ctx, options, &setupLog); err != nil {
			setupLog.Error(err, "problem running operator")
			os.Exit(1)
		}
	}
	return runCmd
}
