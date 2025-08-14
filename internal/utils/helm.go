package utils

import (
	"context"
	"fmt"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/rest"
)

func InstallHelmChart(name, namespace, url, chartName, values string, cfg *rest.Config) error {
	chartRepo := repo.Entry{
		Name: name,
		URL:  url,
	}

	helmClient, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace: namespace,
		},
		RestConfig: cfg,
	})
	if err != nil {
		return fmt.Errorf("error creating helm client: %w", err)
	}

	err = helmClient.AddOrUpdateChartRepo(chartRepo)
	if err != nil {
		return fmt.Errorf("error adding chart repo: %w", err)
	}

	release, err := helmClient.InstallChart(context.Background(), &helmclient.ChartSpec{
		ReleaseName: name,
		ChartName:   chartName,
		Namespace:   namespace,
		ValuesYaml:  values,
		Wait:        true,
		Timeout:     300 * time.Second,
	}, &helmclient.GenericHelmOptions{})
	if err != nil {
		return fmt.Errorf("error installing chart: %w", err)
	}
	fmt.Println(release)
	return nil
}
