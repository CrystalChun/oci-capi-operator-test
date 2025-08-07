package autoscaler

import (
	ocicapiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ServiceAccount(namespace string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-cluster-autoscaler",
			Namespace: namespace,
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(instance, serviceAccount, scheme)
	}

	return serviceAccount, mutateFn
}

func ClusterRole(instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}

	mutateFn := func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"infrastructure.cluster.x-k8s.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
		}
		return nil
	}

	return clusterRole, mutateFn
}

func ClusterRoleBinding(namespace string, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}

	mutateFn := func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "oci-cluster-autoscaler-extra",
		}
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "oci-cluster-autoscaler",
				Namespace: namespace,
			},
		}
		return nil
	}

	return clusterRoleBinding, mutateFn
}
