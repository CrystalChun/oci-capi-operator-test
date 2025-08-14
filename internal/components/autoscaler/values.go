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

func GetAutoscalerDeploymentValues(defaultValues AutoscalerDeploymentValues, autoscaler *capiv1alpha1.OCIClusterAutoscaler) AutoscalerDeploymentValues {
	cloudProvider := defaultValues.CloudProvider
	name := defaultValues.Name
	namespace := defaultValues.Namespace
	serviceAccountName := defaultValues.ServiceAccountName
	createRBAC := defaultValues.CreateRBAC
	createServiceAccount := defaultValues.CreateServiceAccount
	repoURL := defaultValues.RepositoryURL

	if autoscaler.Spec.ClusterAutoscaler.CloudProvider != "" {
		cloudProvider = autoscaler.Spec.ClusterAutoscaler.CloudProvider
	}
	if autoscaler.Spec.ClusterAutoscaler.Name != "" {
		name = autoscaler.Spec.ClusterAutoscaler.Name
	}
	if autoscaler.Spec.ClusterAutoscaler.Namespace != "" {
		namespace = autoscaler.Spec.ClusterAutoscaler.Namespace
	}
	if autoscaler.Spec.ClusterAutoscaler.ServiceAccountName != "" {
		serviceAccountName = autoscaler.Spec.ClusterAutoscaler.ServiceAccountName
	}
	if autoscaler.Spec.ClusterAutoscaler.CreateRBAC {
		createRBAC = true
	}
	if autoscaler.Spec.ClusterAutoscaler.CreateServiceAccount {
		createServiceAccount = true
	}
	if autoscaler.Spec.ClusterAutoscaler.RepositoryURL != "" {
		repoURL = autoscaler.Spec.ClusterAutoscaler.RepositoryURL
	}
	return AutoscalerDeploymentValues{
		Name:                 name,
		Namespace:            namespace,
		ServiceAccountName:   serviceAccountName,
		CloudProvider:        cloudProvider,
		CreateRBAC:           createRBAC,
		CreateServiceAccount: createServiceAccount,
		RepositoryURL:        repoURL,
	}
}
