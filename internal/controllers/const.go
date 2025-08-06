package controllers

const (
	FinalizerName = "ociclusterautoscaler.capi.openshift.io/finalizer"

	OCICAPIClusterName    = "oci-capi-cluster"
	CAPISystemNamespace   = "capi-system"
	CAPOCISystemNamespace = "cluster-api-provider-oci-system"

	// Default Images
	ClusterAutoscalerImage = "registry.k8s.io/autoscaling/cluster-autoscaler:v1.29.0"
)
