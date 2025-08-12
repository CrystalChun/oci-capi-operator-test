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
func GenerateCAPIComponents(ctx context.Context, provider string, providerType v1alpha3.ProviderType) ([]unstructured.Unstructured, error) {
	clusterctlClient, err := client.New(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("error creating clusterctl client: %w", err)
	}
	components, err := clusterctlClient.GetProviderComponents(ctx, provider, providerType, client.ComponentsOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting provider components: %w", err)
	}
	return components.Objs(), nil
	/*
		 	// Create new clusterctl config client
			configClient, err := configclient.New(ctx, "")
			if err != nil {
				return nil, fmt.Errorf("error creating clusterctl config client: %w", err)
			}
			// Create new clusterctl provider client
			providerConfig, err := configClient.Providers().Get(provider, providerType)
			if err != nil {
				return nil, fmt.Errorf("error creating clusterctl provider client: %w", err)
			}

			// Initialize new clusterctl repository components
			components, err := repository.NewComponents(repository.ComponentsInput{
				Provider: providerConfig,
				ConfigClient: configClient,
				Options: repository.ComponentsOptions{
					SkipTemplateProcess: true,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("error creating clusterctl repository components: %w", err)
			}
			return components.Objs(), nil
	*/
}
