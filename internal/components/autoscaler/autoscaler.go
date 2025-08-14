package autoscaler

import (
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"
	"github.com/openshift/oci-capi-operator/internal/utils"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// InstallAutoscaler installs the cluster-autoscaler Helm chart
func InstallAutoscaler(values *AutoscalerDeploymentValues, restConfig *rest.Config) error {
	valuesStr := GetValuesString(values)
	err := utils.InstallHelmChart(values.Name, values.Namespace, values.RepositoryURL, values.Chart, valuesStr, restConfig)
	if err != nil {
		return err
	}
	return nil
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
