package capoci

import (
	ocicapiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Role(capociNamespace string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capoci-leader-election-role",
			Namespace: capociNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(instance, role, scheme)
	}

	return role, mutateFn
}

func RoleBinding(capociNamespace string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capoci-leader-election-rolebinding",
			Namespace: capociNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "capoci-leader-election-role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "capoci-controller-manager",
				Namespace: capociNamespace,
			},
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(instance, roleBinding, scheme)
	}

	return roleBinding, mutateFn
}
