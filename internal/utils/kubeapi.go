package utils

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		"service.beta.openshift.io/serving-cert-secret-name": name,
	}
	if objAnnotations := obj.GetAnnotations(); objAnnotations != nil {
		for key, value := range objAnnotations {
			annotations[key] = value
		}
	}
	obj.SetAnnotations(annotations)
	return nil
}

func SetOpenshiftCABundleAnnotation(obj *unstructured.Unstructured) error {
	if obj == nil {
		return fmt.Errorf("unstructured object is nil")
	}
	annotations := map[string]string{
		"service.beta.openshift.io/inject-cabundle": "true",
	}
	if objAnnotations := obj.GetAnnotations(); objAnnotations != nil {
		for key, value := range objAnnotations {
			annotations[key] = value
		}
	}
	obj.SetAnnotations(annotations)
	return nil
}
