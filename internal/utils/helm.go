package utils

import (
	"context"
	"fmt"
	"strings"

	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/rest"
)

// GetHelmClient creates a new Helm client from the provided namespace and configuration
func GetHelmClient(namespace string, cfg *rest.Config) (helmclient.Client, error) {
	helmClient, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace: namespace,
		},
		RestConfig: cfg,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating helm client: %w", err)
	}
	return helmClient, nil
}

// ChartExists checks if a chart exists in the Helm client
func ChartExists(helmClient helmclient.Client, name string) (bool, error) {
	chart, _, err := helmClient.GetChart(name, &action.ChartPathOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return false, fmt.Errorf("error getting chart: %w", err)
	}
	return chart != nil, nil
}

// AddChartRepo adds a chart repository to the Helm client
func AddChartRepo(helmClient helmclient.Client, name, url string) error {
	chartRepo := repo.Entry{
		Name: name,
		URL:  url,
	}
	err := helmClient.AddOrUpdateChartRepo(chartRepo)
	if err != nil {
		return fmt.Errorf("error adding chart repo: %w", err)
	}
	return nil
}

// ReleaseExists checks if a release exists in the Helm client
func ReleaseExists(helmClient helmclient.Client, name string) (bool, error) {
	release, err := helmClient.GetRelease(name)
	if err != nil {
		return false, fmt.Errorf("error getting release: %w", err)
	}
	return release != nil, nil
}

// InstallHelmChart installs a Helm chart
func InstallHelmChart(helmClient helmclient.Client, chartSpec *helmclient.ChartSpec) error {
	_, err := helmClient.InstallChart(context.Background(), chartSpec, &helmclient.GenericHelmOptions{})
	if err != nil {
		return fmt.Errorf("error installing chart: %w", err)
	}
	return nil
}

// RemoveHelmChart removes a Helm chart
func RemoveHelmChart(helmClient helmclient.Client, chartSpec *helmclient.ChartSpec) error {
	err := helmClient.UninstallRelease(chartSpec)
	if err != nil {
		return fmt.Errorf("error removing chart: %w", err)
	}
	return nil
}
