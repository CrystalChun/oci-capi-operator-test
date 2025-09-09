package enableautoscaler

import (
	"context"
	"encoding/base64"
	"fmt"

	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BootstrapConfigSecret creates a secret that contains the bootstrap config for additional workernodes that are added to the cluster.
func BootstrapConfigSecret(ctx context.Context, client client.Client, capiSystemNamespace string, clusterName string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	bootstrapConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-bootstrap", clusterName),
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		utils.SetDefaultLabels(bootstrapConfigSecret, clusterName)
		ignitionConfig, err := utils.GenerateIgnitionConfig(ctx, client)
		if err != nil {
			return fmt.Errorf("failed to generate ignition config: %w", err)
		}

		bootstrapConfigSecret.Data = map[string][]byte{
			"value":  []byte(ignitionConfig),
			"format": []byte("ignition"),
		}
		return nil
	}

	return bootstrapConfigSecret, mutateFn
}

var kubeconfigFmt = `apiVersion: v1
kind: Config
clusters:
- name: %s
  cluster:
    server: https://kubernetes.default.svc
    certificate-authority-data: %s
contexts:
- name: %s
  context:
    cluster: %s
    user: %s
    namespace: %s
current-context: %s
users:
- name: %s
  user:
    token: %s
`

// KubeConfigSecret creates a secret for CAPI so it can access this cluster.
func KubeConfigSecret(ctx context.Context, client client.Client, capiSystemNamespace string, clusterName string, capiServiceAccountName string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	kubeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kubeconfig", clusterName),
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		secret, err := utils.GetSecret(ctx, client, fmt.Sprintf("%s-token", capiServiceAccountName), capiSystemNamespace)
		if err != nil {
			return fmt.Errorf("failed to get service account secret: %w", err)
		}

		caCrt := base64.StdEncoding.EncodeToString(secret.Data["ca.crt"])
		token := string(secret.Data["token"])
		kubeconfig := fmt.Sprintf(kubeconfigFmt, clusterName, caCrt, clusterName, clusterName, capiServiceAccountName, capiSystemNamespace, clusterName, capiServiceAccountName, token)

		utils.SetDefaultLabels(kubeConfigSecret, clusterName)
		kubeConfigSecret.Data = map[string][]byte{
			"value": []byte(kubeconfig),
		}
		labels := kubeConfigSecret.GetLabels()
		labels["cluster.x-k8s.io/cluster-name"] = clusterName
		labels["clusterctl.cluster.x-k8s.io/move"] = ""
		kubeConfigSecret.SetLabels(labels)
		return nil
	}

	return kubeConfigSecret, mutateFn
}
