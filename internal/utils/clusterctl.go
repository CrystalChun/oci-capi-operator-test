package utils

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1alpha3 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

// GenerateCAPIComponents uses the clusterctl library to generate all CAPI resources
// for a given provider and returns the resources as a list of unstructured objects
func GenerateCAPIComponents(ctx context.Context, provider string, providerType v1alpha3.ProviderType, namespace string) ([]unstructured.Unstructured, error) {
	clusterctlClient, err := client.New(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("error creating clusterctl client: %w", err)
	}
	components, err := clusterctlClient.GetProviderComponents(ctx, provider, providerType, client.ComponentsOptions{
		TargetNamespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting provider components: %w", err)
	}
	return components.Objs(), nil
}
