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
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Autoscaler", func() {
	var (
		instance *capiv1alpha1.OCIClusterAutoscaler
		values   *AutoscalerDeploymentValues
		scheme   *runtime.Scheme
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		err := clientgoscheme.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
		err = rbacv1.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())

		instance = &capiv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}

		values = &AutoscalerDeploymentValues{
			CloudProvider:        "oci",
			Name:                 "cluster-autoscaler",
			Namespace:            "kube-system",
			ServiceAccountName:   "cluster-autoscaler-sa",
			CreateRBAC:           true,
			CreateServiceAccount: true,
			RepositoryURL:        "https://kubernetes.github.io/autoscaler",
			Chart:                "cluster-autoscaler",
			Version:              "1.0.0",
		}
	})

	Context("GetComponents", func() {
		It("should return component with correct subcomponents", func() {
			component := GetComponents(values, instance, scheme)

			Expect(component.Name).To(Equal("Autoscaler"))
			Expect(component.Subcomponents).To(HaveLen(2))

			// Verify ClusterRole subcomponent
			clusterRole := component.Subcomponents[0]
			Expect(clusterRole.Name).To(Equal("clusterRole"))
			_, ok := clusterRole.Object.(*rbacv1.ClusterRole)
			Expect(ok).To(BeTrue())
			Expect(clusterRole.MutateFn).NotTo(BeNil())

			// Verify ClusterRoleBinding subcomponent
			clusterRoleBinding := component.Subcomponents[1]
			Expect(clusterRoleBinding.Name).To(Equal("clusterRoleBinding"))
			_, ok = clusterRoleBinding.Object.(*rbacv1.ClusterRoleBinding)
			Expect(ok).To(BeTrue())
			Expect(clusterRoleBinding.MutateFn).NotTo(BeNil())
		})

		It("should create subcomponents that can be mutated", func() {
			component := GetComponents(values, instance, scheme)

			// Test ClusterRole mutation
			clusterRole := component.Subcomponents[0]
			err := clusterRole.MutateFn()
			Expect(err).NotTo(HaveOccurred())
			role := clusterRole.Object.(*rbacv1.ClusterRole)
			Expect(role.Rules).To(HaveLen(1))
			defaultLabels := utils.GetDefaultLabels(instance.Name)
			Expect(role.Labels).To(Equal(defaultLabels))

			// Test ClusterRoleBinding mutation
			clusterRoleBinding := component.Subcomponents[1]
			err = clusterRoleBinding.MutateFn()
			Expect(err).NotTo(HaveOccurred())
			binding := clusterRoleBinding.Object.(*rbacv1.ClusterRoleBinding)
			Expect(binding.Subjects).To(HaveLen(1))
			Expect(binding.Labels).To(Equal(defaultLabels))
		})
	})
})
