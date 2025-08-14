package utils

import (
	"context"
	"fmt"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/rest"
)

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

func InstallHelmChart(helmClient helmclient.Client, name, namespace, chartName, values string) error {
	release, err := helmClient.InstallChart(context.Background(), &helmclient.ChartSpec{
		ReleaseName: name,
		ChartName:   chartName,
		Namespace:   namespace,
		ValuesYaml:  values,
		Wait:        true,
		Version:     "9.45.0", // this "should" work with Kubernetes 1.32.0, ocp 4.19
		Timeout:     300 * time.Second,
	}, &helmclient.GenericHelmOptions{})
	if err != nil {
		return fmt.Errorf("error installing chart: %w", err)
	}
	fmt.Println(release)
	return nil
}
