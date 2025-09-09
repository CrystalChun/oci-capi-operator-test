package enableautoscaler

import (
	"context"

	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetComponents(ctx context.Context, client client.Client, capiSystemNamespace string, clusterName string, capiServiceAccountName string, autoscaler *ocicapioperatorv1alpha1.OCIClusterAutoscaler, config Config) *components.Component {
	bootstrapConfigSecret, bootstrapConfigSecretMutateFn := BootstrapConfigSecret(ctx, client, capiSystemNamespace, clusterName, autoscaler)
	kubeConfigSecret, kubeConfigSecretMutateFn := KubeConfigSecret(ctx, client, capiSystemNamespace, clusterName, capiServiceAccountName, autoscaler)

	ociCluster, ociClusterMutateFn := OCICluster(capiSystemNamespace, clusterName, autoscaler, config)
	cluster, clusterMutateFn := CAPICluster(capiSystemNamespace, clusterName, autoscaler, config)
	machineTemplate, machineTemplateMutateFn := OCIMachineTemplate(capiSystemNamespace, clusterName, autoscaler, config)
	machineDeployment, machineDeploymentMutateFn := MachineDeployment(capiSystemNamespace, clusterName, autoscaler, config)

	return &components.Component{
		Name: "EnableAutoscaler",
		Subcomponents: components.SubcomponentList{
			{Name: "bootstrapConfigSecret", Object: bootstrapConfigSecret, MutateFn: bootstrapConfigSecretMutateFn},
			{Name: "kubeConfigSecret", Object: kubeConfigSecret, MutateFn: kubeConfigSecretMutateFn},
			{Name: "machineTemplate", Object: machineTemplate, MutateFn: machineTemplateMutateFn},
			{Name: "machineDeployment", Object: machineDeployment, MutateFn: machineDeploymentMutateFn},
			{Name: "ociCluster", Object: ociCluster, MutateFn: ociClusterMutateFn},
			{Name: "cluster", Object: cluster, MutateFn: clusterMutateFn},
		},
	}
}
