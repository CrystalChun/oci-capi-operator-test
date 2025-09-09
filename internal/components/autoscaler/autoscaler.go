package autoscaler

import (
	"fmt"
	"strings"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"
	"github.com/openshift/oci-capi-operator/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// InstallAutoscaler installs the cluster-autoscaler Helm chart
func InstallAutoscaler(instance *capiv1alpha1.OCIClusterAutoscaler, values *AutoscalerDeploymentValues, restConfig *rest.Config) error {
	valuesStr := GetValuesString(values)
	helmClient, err := utils.GetHelmClient(values.Namespace, restConfig)
	if err != nil {
		return err
	}
	chartExists, err := utils.ChartExists(helmClient, values.Chart)
	if err != nil {
		return err
	}
	if !chartExists {
		fmt.Printf("Chart %s does not exist, adding repo %s\n", values.Chart, values.RepositoryURL)
		err = utils.AddChartRepo(helmClient, values.Name, values.RepositoryURL)
		if err != nil {
			return err
		}
	}
	releaseExists, err := utils.ReleaseExists(helmClient, values.Name)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	if releaseExists {
		fmt.Printf("Release %s already exists, skipping install\n", values.Name)
		return nil
	}
	chartSpec := &helmclient.ChartSpec{
		ReleaseName: values.Name,
		ChartName:   values.Chart,
		Namespace:   values.Namespace,
		ValuesYaml:  valuesStr,
		Version:     values.Version,
		Wait:        true,
		Timeout:     300 * time.Second,
		Labels:      utils.GetDefaultLabels(instance.Name),
	}

	err = utils.InstallHelmChart(helmClient, chartSpec)
	if err != nil {
		return err
	}
	return nil
}

func RemoveAutoscaler(values *AutoscalerDeploymentValues, restConfig *rest.Config) error {
	helmClient, err := utils.GetHelmClient(values.Namespace, restConfig)
	if err != nil {
		return err
	}
	releaseExists, err := utils.ReleaseExists(helmClient, values.Name)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	if !releaseExists {
		fmt.Printf("Release %s does not exist, skipping removal\n", values.Name)
		return nil
	}
	chartSpec := &helmclient.ChartSpec{
		ReleaseName: values.Name,
		ChartName:   values.Chart,
		Namespace:   values.Namespace,
	}
	return utils.RemoveHelmChart(helmClient, chartSpec)
}

// GetComponents returns a Component for the cluster-autoscaler which includes a list of subcomponents
func GetComponents(values *AutoscalerDeploymentValues, instance *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) *components.Component {
	clusterRole, clusterRoleMutateFn := ClusterRole(values.Name, instance)
	clusterRoleBinding, clusterRoleBindingMutateFn := ClusterRoleBinding(values, instance)

	return &components.Component{
		Name: "Autoscaler",
		Subcomponents: components.SubcomponentList{
			{Name: "clusterRole", Object: clusterRole, MutateFn: clusterRoleMutateFn},
			{Name: "clusterRoleBinding", Object: clusterRoleBinding, MutateFn: clusterRoleBindingMutateFn},
		},
	}
}
