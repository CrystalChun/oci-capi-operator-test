package autoscaler

import (
	"fmt"

	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
)

type AutoscalerDeploymentValues struct {
	CloudProvider        string `default:"clusterapi"`
	Name                 string `default:"oci-cluster-autoscaler"`
	Namespace            string `default:"capi-system"`
	ServiceAccountName   string `default:"oci-cluster-autoscaler"`
	CreateRBAC           bool   `default:"true"`
	CreateServiceAccount bool   `default:"true"`
	RepositoryURL        string `default:"https://kubernetes-sigs.github.io/cluster-api-autoscaler"`
	Chart                string `default:"cluster-api-autoscaler"`
	Version              string `default:"9.4.0"`
}

var valuesFmt = `cloudProvider: %s
fullnameOverride: %s
autoDiscovery:
  namespace: %s
rbac:
  create: %t
  serviceAccount:
    create: %t
    name: %s`

func GetValuesString(values *AutoscalerDeploymentValues) string {
	return fmt.Sprintf(valuesFmt, values.CloudProvider, values.Name, values.Namespace, values.CreateRBAC, values.CreateServiceAccount, values.ServiceAccountName)
}

// GetAutoscalerDeploymentValues gets the autoscaler deployment values from the instance if it's set
func GetAutoscalerDeploymentValues(originalValues AutoscalerDeploymentValues, instance *capiv1alpha1.OCIClusterAutoscaler) AutoscalerDeploymentValues {
	if instance.Spec.ClusterAutoscaler.CloudProvider != "" {
		originalValues.CloudProvider = instance.Spec.ClusterAutoscaler.CloudProvider
	}
	if instance.Spec.ClusterAutoscaler.Name != "" {
		originalValues.Name = instance.Spec.ClusterAutoscaler.Name
	}
	if instance.Spec.ClusterAutoscaler.Namespace != "" {
		originalValues.Namespace = instance.Spec.ClusterAutoscaler.Namespace
	}
	if instance.Spec.ClusterAutoscaler.ServiceAccountName != "" {
		originalValues.ServiceAccountName = instance.Spec.ClusterAutoscaler.ServiceAccountName
	}
	if instance.Spec.ClusterAutoscaler.CreateRBAC {
		originalValues.CreateRBAC = true
	}
	if instance.Spec.ClusterAutoscaler.CreateServiceAccount {
		originalValues.CreateServiceAccount = true
	}
	if instance.Spec.ClusterAutoscaler.RepositoryURL != "" {
		originalValues.RepositoryURL = instance.Spec.ClusterAutoscaler.RepositoryURL
	}
	return originalValues
}
