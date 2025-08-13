package utils

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CertManagerCAInjectAnnotation  = "cert-manager.io/inject-ca-from"
	OpenshiftCABundleAnnotation    = "service.beta.openshift.io/inject-cabundle"
	OpenshiftServiceCertAnnotation = "service.beta.openshift.io/serving-cert-secret-name"
)

// GetSecretData gets the data of the specified key from the referenced secret
func GetSecretData(ctx context.Context, client client.Client, secretName string, namespace string, keyName string) ([]byte, error) {
	secret := &corev1.Secret{}

	if err := client.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: namespace,
	}, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	data, exists := secret.Data[keyName]
	if !exists {
		return nil, fmt.Errorf("key %s not found in secret %s", keyName, secretName)
	}
	return data, nil
}

// GetDeploymentCondition gets the condition from the status of the Deployment if it exists
func GetDeploymentCondition(conditions []appsv1.DeploymentCondition, conditionType appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

func EditDeploymentCerts(scheme *runtime.Scheme, obj *unstructured.Unstructured, secretName string) error {
	if obj == nil {
		return fmt.Errorf("unstructured object is nil")
	}

	deployment := &appsv1.Deployment{}
	if err := scheme.Convert(obj, deployment, nil); err != nil {
		return fmt.Errorf("failed to convert unstructured to Deployment: %w", err)
	}

	// Find and update the serving-cert volume
	for i, vol := range deployment.Spec.Template.Spec.Volumes {
		if vol.Name == "cert" {
			deployment.Spec.Template.Spec.Volumes[i] = corev1.Volume{
				Name: "cert",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secretName,
					},
				},
			}
			break
		}
	}

	// Convert back to unstructured
	if err := scheme.Convert(deployment, obj, nil); err != nil {
		return fmt.Errorf("failed to convert Deployment back to unstructured: %w", err)
	}

	return nil
}

func SetDefaultLabels(obj client.Object, instanceName string) error {
	if obj == nil {
		return fmt.Errorf("object is nil")
	}
	labels := map[string]string{
		"cluster.x-k8s.io/provider":    "cluster-api",
		"capi.openshift.io/managed-by": instanceName,
	}
	if objLabels := obj.GetLabels(); objLabels != nil {
		for key, value := range objLabels {
			labels[key] = value
		}
	}
	obj.SetLabels(labels)
	return nil
}

func SetControllerLabels(obj *unstructured.Unstructured, instanceName string) error {
	if obj == nil {
		return fmt.Errorf("unstructured object is nil")
	}
	labels := map[string]string{
		"cluster.x-k8s.io/provider":    "cluster-api",
		"capi.openshift.io/managed-by": instanceName,
	}
	if objLabels := obj.GetLabels(); objLabels != nil {
		for key, value := range objLabels {
			labels[key] = value
		}
	}
	obj.SetLabels(labels)
	return nil
}

func SetOpenshiftServiceCertAnnotation(obj *unstructured.Unstructured, name string) error {
	if obj == nil {
		return fmt.Errorf("unstructured object is nil")
	}
	annotations := map[string]string{
		OpenshiftServiceCertAnnotation: name,
	}
	if objAnnotations := obj.GetAnnotations(); objAnnotations != nil {
		for key, value := range objAnnotations {
			annotations[key] = value
		}
	}
	delete(annotations, CertManagerCAInjectAnnotation)
	obj.SetAnnotations(annotations)
	return nil
}

func SetOpenshiftCABundleAnnotation(obj *unstructured.Unstructured) error {
	if obj == nil {
		return fmt.Errorf("unstructured object is nil")
	}
	annotations := map[string]string{
		OpenshiftCABundleAnnotation: "true",
	}
	if objAnnotations := obj.GetAnnotations(); objAnnotations != nil {
		for key, value := range objAnnotations {
			annotations[key] = value
		}
	}
	delete(annotations, CertManagerCAInjectAnnotation)
	obj.SetAnnotations(annotations)
	return nil
}

func GetInfrastructureCluster(ctx context.Context, client client.Client) (*configv1.Infrastructure, error) {
	clusters := &configv1.InfrastructureList{}
	if err := client.List(ctx, clusters); err != nil {
		return nil, fmt.Errorf("failed to get infrastructure cluster: %w", err)
	}
	if len(clusters.Items) != 1 {
		return nil, fmt.Errorf("expected 1 infrastructure cluster, got %d", len(clusters.Items))
	}
	return &clusters.Items[0], nil
}

// GetClusterName gets the name of the cluster from the infrastructure cluster.
func GetClusterName(ctx context.Context, client client.Client) (string, error) {
	infrastructureCluster, err := GetInfrastructureCluster(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get infrastructure cluster: %w", err)
	}
	return infrastructureCluster.Status.InfrastructureName, nil
}

// GetClusterAPIServerInternalURL gets the API server internal URL of the cluster from the infrastructure cluster.
func GetClusterAPIServerInternalURL(ctx context.Context, client client.Client) (string, error) {
	infrastructureCluster, err := GetInfrastructureCluster(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get infrastructure cluster: %w", err)
	}
	return infrastructureCluster.Status.APIServerInternalURL, nil
}

// GetMachineConfigCA gets the CA certificate for the machine config server
func GetMachineConfigCA(ctx context.Context, client client.Client) (string, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: "machine-config-server-tls", Namespace: "openshift-machine-config-operator"}, secret); err != nil {
		return "", fmt.Errorf("failed to get machine config server CA: %w", err)
	}
	return string(secret.Data["tls.crt"]), nil
}
