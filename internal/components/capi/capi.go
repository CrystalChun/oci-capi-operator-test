package capi

import (
	"context"

	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha3 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
)

// NewComponent returns a Component for the CAPI controller manager
func NewComponent(ctx context.Context, capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, webhookServiceName string, scheme *runtime.Scheme) ([]unstructured.Unstructured, error) {
	components, err := utils.GenerateCAPIComponents(ctx, "cluster-api", v1alpha3.CoreProviderType, capiSystemNamespace)
	if err != nil {
		return nil, err
	}
	reconcileComponents := []unstructured.Unstructured{}

	for _, component := range components {
		utils.SetControllerLabels(&component, autoscaler.Name)
		switch component.GetKind() {
		case "Service":
			utils.SetOpenshiftServiceCertAnnotation(&component, webhookServiceName)
			reconcileComponents = append(reconcileComponents, component)
		case "ValidatingWebhookConfiguration", "MutatingWebhookConfiguration":
			utils.SetOpenshiftCABundleAnnotation(&component)
			reconcileComponents = append(reconcileComponents, component)
		case "Deployment":
			utils.EditDeploymentCerts(scheme, &component, webhookServiceName)
			reconcileComponents = append(reconcileComponents, component)
		case "Certificate", "Issuer", "Namespace", "Secret", "CustomResourceDefinition": // skip these
		default:
			reconcileComponents = append(reconcileComponents, component)
		}
	}
	return reconcileComponents, nil
}

func CAPINamespace(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: capiSystemNamespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/provider": "cluster-api",
			},
		},
	}
	mutateFn := func() error {
		utils.SetDefaultLabels(namespace, autoscaler.Name)
		return nil
	}
	return namespace, mutateFn
}
