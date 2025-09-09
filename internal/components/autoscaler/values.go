package autoscaler

import (
	"fmt"

	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
)

type AutoscalerDeploymentValues struct {
	CloudProvider string
	// Name is both the name of the autoscaler Helm chart and the name of the Helm release
	Name                 string
	Namespace            string
	ServiceAccountName   string
	CreateRBAC           bool
	CreateServiceAccount bool
	RepositoryURL        string
	Chart                string
	Version              string
}

var valuesFmt = `
cloudProvider: %s
fullnameOverride: %s
autoDiscovery:
  namespace: %s
rbac:
  create: %t
  serviceAccount:
    create: %t
    name: %s`

// GetValuesString gets the value overrides as a string for the autoscaler helm chart deployment
func GetValuesString(values *AutoscalerDeploymentValues) string {
	return fmt.Sprintf(valuesFmt,
		values.CloudProvider,
		values.Name,
		values.Namespace,
		values.CreateRBAC,
		values.CreateServiceAccount,
		values.ServiceAccountName,
	)
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
	if instance.Spec.ClusterAutoscaler.Version != "" {
		originalValues.Version = instance.Spec.ClusterAutoscaler.Version
	}
	return originalValues
}
