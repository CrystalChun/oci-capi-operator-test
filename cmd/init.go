package main

import (
	"context"
	"os"

	"github.com/openshift/oci-capi-operator/internal/components/crds"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewInitCommand() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initializes prerequesites for the OCI CAPI operator.",
		Long:  "Applies the required CRDs for the cluster-api and capi-provider-oci providers to the cluster.",
	}
	initCmd.Run = func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		setupLog := ctrl.Log.WithName("setup")
		if err := runInit(ctx, &setupLog); err != nil {
			setupLog.Error(err, "Failed to initialize operator")
			os.Exit(1)
		}
		os.Exit(0)
	}
	return initCmd
}

// Apply the CRDs to the cluster for this operator
func runInit(ctx context.Context, setupLog *logr.Logger) error {
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	cfg, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "Failed to get config")
		return err
	}
	client, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "Failed to create client")
		return err
	}

	// get the CRDs for the cluster-api provider
	capiCRDs, err := crds.GetClusterctlComponents(ctx, "cluster-api", v1alpha3.CoreProviderType)
	if err != nil {
		setupLog.Error(err, "Failed to get CRDs for CAPI")
		return err
	}
	// get the CRDs for the capi-provider-oci provider
	capociCRDs, err := crds.GetClusterctlComponents(ctx, "oci", v1alpha3.InfrastructureProviderType)
	if err != nil {
		setupLog.Error(err, "Failed to get CRDs for CAPOCI")
		return err
	}

	// apply all the CRDs
	for _, crd := range append(capiCRDs, capociCRDs...) {
		setupLog.Info("Applying CRD", "name", crd.GetName())
		err = client.Create(ctx, &crd)
		if err != nil && !errors.IsAlreadyExists(err) {
			setupLog.Error(err, "Failed to apply CRD", "name", crd.GetName())
			return err
		}
	}
	setupLog.Info("All CRDs applied successfully")
	return nil
}
