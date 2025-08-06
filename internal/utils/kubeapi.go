package utils

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
