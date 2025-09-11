/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package autoscaler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/utils"
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
			Name:               "cluster-autoscaler",
			Namespace:          "kube-system",
			ServiceAccountName: "cluster-autoscaler-sa",
		}
	})

	Context("ClusterRole", func() {
		It("should create a ClusterRole with correct configuration", func() {
			obj, mutateFn := ClusterRole(values.Name, instance)
			clusterRole, ok := obj.(*rbacv1.ClusterRole)
			Expect(ok).To(BeTrue(), "Object should be a ClusterRole")

			// Verify initial state
			Expect(clusterRole.Name).To(Equal("cluster-autoscaler-extra"))
			Expect(clusterRole.Rules).To(BeEmpty())

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify final state
			Expect(clusterRole.Rules).To(HaveLen(1))
			rule := clusterRole.Rules[0]
			Expect(rule.APIGroups).To(ConsistOf("infrastructure.cluster.x-k8s.io"))
			Expect(rule.Resources).To(ConsistOf("*"))
			Expect(rule.Verbs).To(ConsistOf("get", "list", "watch", "update"))

			// Verify labels
			expectedLabels := utils.GetDefaultLabels(instance.Name)
			Expect(clusterRole.Labels).To(Equal(expectedLabels))
		})
	})

	Context("ClusterRoleBinding", func() {
		It("should create a ClusterRoleBinding with correct configuration", func() {
			obj, mutateFn := ClusterRoleBinding(values, instance)
			binding, ok := obj.(*rbacv1.ClusterRoleBinding)
			Expect(ok).To(BeTrue(), "Object should be a ClusterRoleBinding")

			// Verify initial state
			Expect(binding.Name).To(Equal("cluster-autoscaler-extra"))
			Expect(binding.RoleRef).To(Equal(rbacv1.RoleRef{}))
			Expect(binding.Subjects).To(BeEmpty())

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify final state
			Expect(binding.RoleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
			Expect(binding.RoleRef.Kind).To(Equal("ClusterRole"))
			Expect(binding.RoleRef.Name).To(Equal("cluster-autoscaler-extra"))

			Expect(binding.Subjects).To(HaveLen(1))
			subject := binding.Subjects[0]
			Expect(subject.Kind).To(Equal("ServiceAccount"))
			Expect(subject.Name).To(Equal(values.ServiceAccountName))
			Expect(subject.Namespace).To(Equal(values.Namespace))

			// Verify labels
			expectedLabels := utils.GetDefaultLabels(instance.Name)
			Expect(binding.Labels).To(Equal(expectedLabels))
		})

		It("should handle different names and namespaces", func() {
			values.Name = "custom-autoscaler"
			values.Namespace = "custom-ns"
			values.ServiceAccountName = "custom-sa"

			obj, mutateFn := ClusterRoleBinding(values, instance)
			binding, ok := obj.(*rbacv1.ClusterRoleBinding)
			Expect(ok).To(BeTrue())

			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			Expect(binding.Name).To(Equal("custom-autoscaler-extra"))
			Expect(binding.RoleRef.Name).To(Equal("custom-autoscaler-extra"))
			Expect(binding.Subjects[0].Name).To(Equal("custom-sa"))
			Expect(binding.Subjects[0].Namespace).To(Equal("custom-ns"))
		})
	})
})
