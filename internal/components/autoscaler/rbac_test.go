package autoscaler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Autoscaler RBAC", func() {
	var (
		instance *capiv1alpha1.OCIClusterAutoscaler
		values   *AutoscalerDeploymentValues
	)

	BeforeEach(func() {
		instance = &capiv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}

		values = &AutoscalerDeploymentValues{
			Name:               "test-autoscaler",
			Namespace:          "test-namespace",
			ServiceAccountName: "test-sa",
		}
	})

	Context("ClusterRole", func() {
		It("should create cluster role with correct configuration", func() {
			clusterRole, mutateFn := ClusterRole(values.Name, instance)
			Expect(clusterRole).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify cluster role configuration
			Expect(clusterRole.GetName()).To(Equal("test-autoscaler-extra"))
			Expect(clusterRole.GetLabels()).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(clusterRole.GetLabels()).To(HaveKeyWithValue("capi.openshift.io/managed-by", "test-autoscaler"))

			// Verify rules
			rules := clusterRole.(*rbacv1.ClusterRole).Rules
			Expect(rules).To(HaveLen(1))
			Expect(rules[0].APIGroups).To(Equal([]string{"infrastructure.cluster.x-k8s.io"}))
			Expect(rules[0].Resources).To(Equal([]string{"*"}))
			Expect(rules[0].Verbs).To(Equal([]string{"get", "list", "watch", "update"}))
		})
	})

	Context("ClusterRoleBinding", func() {
		It("should create cluster role binding with correct configuration", func() {
			clusterRoleBinding, mutateFn := ClusterRoleBinding(values, instance)
			Expect(clusterRoleBinding).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify cluster role binding configuration
			Expect(clusterRoleBinding.GetName()).To(Equal("test-autoscaler-extra"))
			Expect(clusterRoleBinding.GetLabels()).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(clusterRoleBinding.GetLabels()).To(HaveKeyWithValue("capi.openshift.io/managed-by", "test-autoscaler"))

			// Verify role ref
			roleRef := clusterRoleBinding.(*rbacv1.ClusterRoleBinding).RoleRef
			Expect(roleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
			Expect(roleRef.Kind).To(Equal("ClusterRole"))
			Expect(roleRef.Name).To(Equal("test-autoscaler-extra"))

			// Verify subjects
			subjects := clusterRoleBinding.(*rbacv1.ClusterRoleBinding).Subjects
			Expect(subjects).To(HaveLen(1))
			Expect(subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(subjects[0].Name).To(Equal("test-sa"))
			Expect(subjects[0].Namespace).To(Equal("test-namespace"))
		})
	})
})
