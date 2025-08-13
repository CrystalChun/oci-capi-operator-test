package enableautoscaler

import (
	"context"
	"fmt"

	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BootstrapConfigSecret is a secret that contains the bootstrap config for additional nodes that are added to the cluster.
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
		bootstrapConfigSecret.Data = map[string][]byte{ // TODO: confirm what this secret looks likeand the key is
			"bootstrap.ign": []byte(ignitionConfig),
		}
		return nil
	}

	return bootstrapConfigSecret, mutateFn
}

func KubeConfigSecret(capiSystemNamespace string, clusterName string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	kubeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kubeconfig", clusterName),
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		utils.SetDefaultLabels(kubeConfigSecret, clusterName)
		// This needs to contain the kubeconfig for the cluster.
		return nil
	}

	return kubeConfigSecret, mutateFn
}
