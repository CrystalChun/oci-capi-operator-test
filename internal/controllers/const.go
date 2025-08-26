package controllers

const (
	FinalizerName  = "ociclusterautoscaler.capi.openshift.io/finalizer"
	ManagedByLabel = "capi.openshift.io/managed-by"

	OCICAPIClusterName    = "oci-capi-cluster"
	CAPISystemNamespace   = "capi-system"
	CAPOCISystemNamespace = "cluster-api-provider-oci-system"

	CAPIDeploymentName   = "capi-manager"
	CAPOCIDeploymentName = "capoci-controller-manager"

	CAPIServiceAccountName   = "capi-manager"
	CAPOCIServiceAccountName = "capoci-controller-manager"

	CAPIWebhookServiceName   = "capi-webhook-service"
	CAPOCIWebhookServiceName = "capoci-webhook-service"

	AutoscalerRepoURL        = "https://kubernetes.github.io/autoscaler"
	AutoscalerChartName      = "oci-cluster-autoscaler/cluster-autoscaler"
	AutoscalerDeploymentName = "oci-cluster-autoscaler"
	AutoScalerCloudProvider  = "clusterapi"
)
