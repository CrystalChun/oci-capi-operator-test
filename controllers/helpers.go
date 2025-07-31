package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	securityv1 "github.com/openshift/api/security/v1"
	ocicapiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	capiSystemNamespace = "capi-system"
	capociNamespace     = "cluster-api-provider-oci-system"
)

// createOrUpdateSCC creates or updates the SecurityContextConstraints for CAPI
func (r *OCICAPIClusterReconciler) createOrUpdateSCC(ctx context.Context, instance *ocicapiv1alpha1.OCICAPICluster) error {
	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-capi",
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, scc, func() error {
		scc.RunAsUser = securityv1.RunAsUserStrategyOptions{
			Type: securityv1.RunAsUserStrategyRunAsAny,
		}
		scc.SELinuxContext = securityv1.SELinuxContextStrategyOptions{
			Type: securityv1.SELinuxStrategyRunAsAny,
		}
		scc.SeccompProfiles = []string{"runtime/default"}
		scc.Users = []string{
			"system:serviceaccount:cluster-api-provider-oci-system:capoci-controller-manager",
			"system:serviceaccount:capi-system:capi-manager",
		}
		return nil
	})

	return err
}

// createOrUpdateClusterAutoscalerRBAC creates or updates RBAC for cluster-autoscaler
func (r *OCICAPIClusterReconciler) createOrUpdateClusterAutoscalerRBAC(ctx context.Context, instance *ocicapiv1alpha1.OCICAPICluster) error {
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"infrastructure.cluster.x-k8s.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, binding, func() error {
		binding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "oci-cluster-autoscaler-extra",
		}
		binding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "oci-cluster-autoscaler",
				Namespace: capiSystemNamespace,
			},
		}
		return nil
	})

	return err
}

// createOrUpdateBootstrapIgnition creates or updates the bootstrap ignition secret
func (r *OCICAPIClusterReconciler) createOrUpdateBootstrapIgnition(ctx context.Context, instance *ocicapiv1alpha1.OCICAPICluster) error {
	// Get cluster information from infrastructure
	infraConfig := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: "cluster-infrastructure-02-config", Namespace: "kube-system"}, infraConfig)
	if err != nil {
		return fmt.Errorf("failed to get infrastructure config: %w", err)
	}

	// Get machine config server certificate
	mcsTLSSecret := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: "machine-config-server-tls", Namespace: "openshift-machine-config-operator"}, mcsTLSSecret)
	if err != nil {
		return fmt.Errorf("failed to get MCS TLS secret: %w", err)
	}

	// Create bootstrap ignition
	ignitionConfig := map[string]interface{}{
		"ignition": map[string]interface{}{
			"config": map[string]interface{}{
				"merge": []map[string]interface{}{
					{
						"source": fmt.Sprintf("https://%s:22623/config/worker", instance.Spec.Network.ControlPlaneEndpoint),
					},
				},
			},
			"security": map[string]interface{}{
				"tls": map[string]interface{}{
					"certificateAuthorities": []map[string]interface{}{
						{
							"source": fmt.Sprintf("data:text/plain;charset=utf-8;base64,%s",
								base64.StdEncoding.EncodeToString(mcsTLSSecret.Data["tls.crt"])),
						},
					},
				},
			},
			"version": "3.2.0",
		},
	}

	// Create the bootstrap secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-bootstrap", instance.Name),
			Namespace: capiSystemNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data["format"] = []byte("ignition")
		ignitionBytes, err := json.Marshal(ignitionConfig)
		if err != nil {
			return err
		}
		secret.Data["value"] = ignitionBytes
		return nil
	})

	return err
}

// createOrUpdateClusterAutoscaler creates or updates the cluster-autoscaler deployment
func (r *OCICAPIClusterReconciler) createOrUpdateClusterAutoscaler(ctx context.Context, instance *ocicapiv1alpha1.OCICAPICluster) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-cluster-autoscaler",
			Namespace: capiSystemNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		deploy.Spec = appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "oci-cluster-autoscaler",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "oci-cluster-autoscaler",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "oci-cluster-autoscaler",
					Containers: []corev1.Container{
						{
							Name:  "cluster-autoscaler",
							Image: "registry.k8s.io/autoscaling/cluster-autoscaler:v1.29.0",
							Args: []string{
								"--cloud-provider=clusterapi",
								"--namespace=capi-system",
								fmt.Sprintf("--min-nodes=%d", instance.Spec.Autoscaling.MinNodes),
								fmt.Sprintf("--max-nodes=%d", instance.Spec.Autoscaling.MaxNodes),
							},
						},
					},
				},
			},
		}
		return nil
	})

	return err
}

// createOrUpdateCertificateApprover creates or updates the certificate approver deployment
func (r *OCICAPIClusterReconciler) createOrUpdateCertificateApprover(ctx context.Context, instance *ocicapiv1alpha1.OCICAPICluster) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-certificate-approver",
			Namespace: capiSystemNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		deploy.Spec = appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "oci-certificate-approver",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "oci-certificate-approver",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "oci-certificate-approver",
					Containers: []corev1.Container{
						{
							Name:  "approver",
							Image: "quay.io/openshift/origin-cli:latest",
							Command: []string{
								"/bin/bash",
								"-c",
								certApproverScript,
							},
						},
					},
				},
			},
		}
		return nil
	})

	return err
}

const certApproverScript = `
#!/bin/bash
set -euo pipefail

get_csr_hostname() {
    local csr=$1
    signer_name=$(oc get csr $csr -ojsonpath={.spec.signerName})
    case $signer_name in
        kubernetes.io/kube-apiserver-client-kubelet)
            oc get csr $csr -ojsonpath={.spec.request} \
                | base64 -d \
                | openssl req -in - -subject -noout \
                | xargs -n1 \
                | grep CN=system:node \
                | cut -d : -f 3 \
                | cut -d . -f 1
            ;;
        kubernetes.io/kubelet-serving)
            oc get csr $csr -ojsonpath={.spec.username} \
                | cut -d : -f 3 \
                | cut -d . -f 1
            ;;
    esac
}

check_and_sign_csrs(){
    local csr
    for csr in $(oc get csr -ojson | jq '.items[] | select(.status=={}) | .metadata.name' -r); do
        node_name=$(get_csr_hostname $csr)
        if oc get ocimachine -oname -n capi-system | cut -d / -f 2 | grep -xq $node_name; then
            echo "Approving certificate $csr for node $node_name"
            oc adm certificate approve $csr
        fi
    done
}

while true; do
    check_and_sign_csrs
    sleep 10
done
`
