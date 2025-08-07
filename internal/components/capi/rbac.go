package capi

import (
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"

	securityv1 "github.com/openshift/api/security/v1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// SecurityContextConstraints defines the SCC for the CAPI manager and CAPOCI controller manager
func SecurityContextConstraints(capiSystemNamespace string, scheme *runtime.Scheme, autoscaler *capiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-capi",
		},
	}

	mutateFn := func() error {
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
		return controllerutil.SetControllerReference(autoscaler, scc, scheme)
	}

	return scc, mutateFn
}

func ServiceAccount(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capi-manager",
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, serviceAccount, scheme)
	}

	return serviceAccount, mutateFn
}

func Role(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capi-leader-election-role",
			Namespace: capiSystemNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"create", "get", "update", "watch", "update", "patch", "delete", "list"},
			},
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, role, scheme)
	}

	return role, mutateFn
}

func RoleBinding(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capi-leader-election-rolebinding",
			Namespace: capiSystemNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "capi-leader-election-role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "capi-manager",
				Namespace: capiSystemNamespace,
			},
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, roleBinding, scheme)
	}

	return roleBinding, mutateFn
}

func ClusterRole(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capi-aggregated-manager-role",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"addons.cluster.x-k8s.io"},
				Resources: []string{"clusterresourcesets/finalizers", "clusterresourcesets/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{
				APIGroups: []string{
					"addons.cluster.x-k8s.io",
					"bootstrap.cluster.x-k8s.io",
					"controlplane.cluster.x-k8s.io",
					"infrastructure.cluster.x-k8s.io",
				},
				Resources: []string{"*"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions", "customresourcedefinitions/status"},
				ResourceNames: []string{
					"clusterclasses.cluster.x-k8s.io",
					"clusterresourcesetbindings.addons.cluster.x-k8s.io",
					"clusterresourcesets.addons.cluster.x-k8s.io",
					"clusters.cluster.x-k8s.io",
					"extensionconfigs.runtime.cluster.x-k8s.io",
					"ipaddressclaims.ipam.cluster.x-k8s.io",
					"ipaddresses.ipam.cluster.x-k8s.io",
					"machinedeployments.cluster.x-k8s.io",
					"machinedrainrules.cluster.x-k8s.io",
					"machinehealthchecks.cluster.x-k8s.io",
					"machinepools.cluster.x-k8s.io",
					"machines.cluster.x-k8s.io",
					"machinesets.cluster.x-k8s.io",
				},
				Verbs: []string{"patch", "update"},
			},
			{
				APIGroups: []string{"authentication.k8s.io"},
				Resources: []string{"tokenreviews"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"subjectaccessreviews"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{"cluster.x-k8s.io"},
				Resources: []string{
					"clusterclasses",
					"clusterclasses/status",
					"clusters",
					"clusters/finalizers",
					"clusters/status",
					"machinedrainrules",
					"machinehealthchecks/finalizers",
					"machinehealthchecks/status",
				},
				Verbs: []string{"get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"cluster.x-k8s.io"},
				Resources: []string{
					"machinedeployments",
					"machinedeployments/finalizers",
					"machinedeployments/status",
					"machinehealthchecks",
					"machinepools",
					"machinepools/finalizers",
					"machinepools/status",
					"machines",
					"machines/finalizers",
					"machines/status",
					"machinesets",
					"machinesets/finalizers",
					"machinesets/status",
				},
				Verbs: []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"ipam.cluster.x-k8s.io"},
				Resources: []string{"ipaddressclaims", "ipaddresses"},
				Verbs:     []string{"get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"ipam.cluster.x-k8s.io"},
				Resources: []string{"ipaddressclaims/status"},
				Verbs:     []string{"patch", "update"},
			},
			{
				APIGroups: []string{"runtime.cluster.x-k8s.io"},
				Resources: []string{"extensionconfigs", "extensionconfigs/status"},
				Verbs:     []string{"get", "list", "patch", "update", "watch"},
			},
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, clusterRole, scheme)
	}

	return clusterRole, mutateFn
}

func ClusterRoleBinding(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capi-aggregated-manager-rolebinding",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "capi-aggregated-manager-role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "capi-manager",
				Namespace: capiSystemNamespace,
			},
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, clusterRoleBinding, scheme)
	}

	return clusterRoleBinding, mutateFn
}
