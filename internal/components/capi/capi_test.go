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

package capi

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	securityv1 "github.com/openshift/api/security/v1"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CAPI Components", func() {
	var (
		instance *capiv1alpha1.OCIClusterAutoscaler
	)

	BeforeEach(func() {
		instance = &capiv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}
	})

	Context("CAPINamespace", func() {
		It("should create a namespace with correct configuration", func() {
			obj, mutateFn := CAPINamespace("capi-system", instance)
			namespace, ok := obj.(*corev1.Namespace)
			Expect(ok).To(BeTrue(), "Object should be a Namespace")

			// Verify initial state
			Expect(namespace.Name).To(Equal("capi-system"))
			Expect(namespace.Labels).To(BeEmpty())

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels
			defaultLabels := utils.GetDefaultLabels(instance.Name)
			Expect(namespace.Labels).To(Equal(defaultLabels))
		})
	})

	Context("SecurityContextConstraints", func() {
		It("should create SCC with correct configuration", func() {
			obj, mutateFn := SecurityContextConstraints(
				"capi-system",
				"capoci-system",
				"capoci-sa",
				"capi-sa",
				instance,
			)
			scc, ok := obj.(*securityv1.SecurityContextConstraints)
			Expect(ok).To(BeTrue(), "Object should be a SecurityContextConstraints")

			// Verify initial state
			Expect(scc.Name).To(Equal("oci-capi"))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration
			Expect(scc.RunAsUser.Type).To(Equal(securityv1.RunAsUserStrategyRunAsAny))
			Expect(scc.SELinuxContext.Type).To(Equal(securityv1.SELinuxStrategyRunAsAny))
			Expect(scc.SeccompProfiles).To(ConsistOf("runtime/default"))

			// Verify service account users
			expectedUsers := []string{
				"system:serviceaccount:capoci-system:capoci-sa",
				"system:serviceaccount:capi-system:capi-sa",
			}
			Expect(scc.Users).To(ConsistOf(expectedUsers))

			// Verify labels
			defaultLabels := utils.GetDefaultLabels(instance.Name)
			Expect(scc.Labels).To(Equal(defaultLabels))
		})
	})

	Context("ClusterRoleBinding", func() {
		It("should create ClusterRoleBinding with correct configuration", func() {
			obj, mutateFn := ClusterRoleBinding("test-binding", "test-sa", "test-ns")
			binding, ok := obj.(*rbacv1.ClusterRoleBinding)
			Expect(ok).To(BeTrue(), "Object should be a ClusterRoleBinding")

			// Verify initial state
			Expect(binding.Name).To(Equal("test-binding"))
			Expect(binding.RoleRef).To(Equal(rbacv1.RoleRef{}))
			Expect(binding.Subjects).To(BeEmpty())

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration
			Expect(binding.RoleRef.Kind).To(Equal("ClusterRole"))
			Expect(binding.RoleRef.Name).To(Equal("cluster-admin"))

			Expect(binding.Subjects).To(HaveLen(1))
			subject := binding.Subjects[0]
			Expect(subject.Kind).To(Equal("ServiceAccount"))
			Expect(subject.Name).To(Equal("test-sa"))
			Expect(subject.Namespace).To(Equal("test-ns"))
		})
	})

	Context("ServiceAccountSecret", func() {
		It("should create Secret with correct configuration", func() {
			obj, mutateFn := ServiceAccountSecret("test-sa", "test-ns")
			secret, ok := obj.(*corev1.Secret)
			Expect(ok).To(BeTrue(), "Object should be a Secret")

			// Verify initial state
			Expect(secret.Name).To(Equal("test-sa-token"))
			Expect(secret.Namespace).To(Equal("test-ns"))
			Expect(secret.Type).To(BeEmpty())
			Expect(secret.Annotations).To(BeEmpty())

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration
			Expect(secret.Type).To(Equal(corev1.SecretTypeServiceAccountToken))
			Expect(secret.Annotations).To(HaveKeyWithValue("kubernetes.io/service-account.name", "test-sa"))
		})
	})

	Context("GetComponents", func() {
		It("should return component with all subcomponents", func() {
			component := GetComponents(
				"capi-system",
				"capoci-system",
				"capi-sa",
				"capoci-sa",
				"test-binding",
				instance,
			)

			Expect(component.Name).To(Equal("CAPI"))
			Expect(component.Subcomponents).To(HaveLen(4))

			// Verify SCC subcomponent
			scc := component.Subcomponents[0]
			Expect(scc.Name).To(Equal("scc"))
			_, ok := scc.Object.(*securityv1.SecurityContextConstraints)
			Expect(ok).To(BeTrue())
			Expect(scc.MutateFn).NotTo(BeNil())

			// Verify Namespace subcomponent
			ns := component.Subcomponents[1]
			Expect(ns.Name).To(Equal("namespace"))
			_, ok = ns.Object.(*corev1.Namespace)
			Expect(ok).To(BeTrue())
			Expect(ns.MutateFn).NotTo(BeNil())

			// Verify ClusterRoleBinding subcomponent
			crb := component.Subcomponents[2]
			Expect(crb.Name).To(Equal("clusterRoleBinding"))
			_, ok = crb.Object.(*rbacv1.ClusterRoleBinding)
			Expect(ok).To(BeTrue())
			Expect(crb.MutateFn).NotTo(BeNil())

			// Verify ServiceAccountSecret subcomponent
			secret := component.Subcomponents[3]
			Expect(secret.Name).To(Equal("serviceAccountSecret"))
			_, ok = secret.Object.(*corev1.Secret)
			Expect(ok).To(BeTrue())
			Expect(secret.MutateFn).NotTo(BeNil())

			// Test that all mutation functions work
			for _, sub := range component.Subcomponents {
				err := sub.MutateFn()
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})
