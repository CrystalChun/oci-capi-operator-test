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

package capoci

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CAPOCI Components", func() {
	var (
		instance *capiv1alpha1.OCIClusterAutoscaler
		auth     *CAPOCICredentials
	)

	BeforeEach(func() {
		instance = &capiv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}

		auth = &CAPOCICredentials{
			TenancyID:            "test-tenancy",
			UserID:               "test-user",
			Region:               "test-region",
			Fingerprint:          "test-fingerprint",
			PrivateKey:           "test-key",
			UseInstancePrincipal: "false",
			Passphrase:           "test-passphrase",
		}
	})

	Context("Namespace", func() {
		It("should create a namespace with correct configuration", func() {
			obj, mutateFn := Namespace("capoci-system", instance)
			namespace, ok := obj.(*corev1.Namespace)
			Expect(ok).To(BeTrue(), "Object should be a Namespace")

			// Verify initial state
			Expect(namespace.Name).To(Equal("capoci-system"))
			Expect(namespace.Labels).To(BeEmpty())

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels
			defaultLabels := utils.GetDefaultLabels(instance.Name)
			Expect(namespace.Labels).To(Equal(defaultLabels))
		})
	})

	Context("AuthConfigSecret", func() {
		It("should create a secret with correct configuration", func() {
			obj, mutateFn := AuthConfigSecret(instance, "capoci-system", auth)
			secret, ok := obj.(*corev1.Secret)
			Expect(ok).To(BeTrue(), "Object should be a Secret")

			// Verify initial state
			Expect(secret.Name).To(Equal("capoci-auth-config"))
			Expect(secret.Namespace).To(Equal("capoci-system"))
			Expect(secret.Labels).To(BeEmpty())
			Expect(secret.Data).To(BeEmpty())

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels
			defaultLabels := utils.GetDefaultLabels(instance.Name)
			Expect(secret.Labels).To(Equal(defaultLabels))

			// Verify secret data
			Expect(secret.Data).To(HaveLen(7))
			Expect(string(secret.Data["tenancy"])).To(Equal(auth.TenancyID))
			Expect(string(secret.Data["user"])).To(Equal(auth.UserID))
			Expect(string(secret.Data["region"])).To(Equal(auth.Region))
			Expect(string(secret.Data["fingerprint"])).To(Equal(auth.Fingerprint))
			Expect(string(secret.Data["key"])).To(Equal(auth.PrivateKey))
			Expect(string(secret.Data["useInstancePrincipal"])).To(Equal(auth.UseInstancePrincipal))
			Expect(string(secret.Data["passphrase"])).To(Equal(auth.Passphrase))
		})

		It("should handle empty credentials", func() {
			emptyAuth := &CAPOCICredentials{}
			obj, mutateFn := AuthConfigSecret(instance, "capoci-system", emptyAuth)
			secret, ok := obj.(*corev1.Secret)
			Expect(ok).To(BeTrue())

			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// All values should be empty but present
			Expect(secret.Data).To(HaveLen(7))
			for _, value := range secret.Data {
				Expect(string(value)).To(BeEmpty())
			}
		})
	})

	Context("GetComponents", func() {
		It("should return component with all subcomponents", func() {
			component := GetComponents("capoci-system", instance, auth)

			Expect(component.Name).To(Equal("CAPOCI"))
			Expect(component.Subcomponents).To(HaveLen(2))

			// Verify Namespace subcomponent
			ns := component.Subcomponents[0]
			Expect(ns.Name).To(Equal("namespace"))
			_, ok := ns.Object.(*corev1.Namespace)
			Expect(ok).To(BeTrue())
			Expect(ns.MutateFn).NotTo(BeNil())

			// Verify AuthConfigSecret subcomponent
			secret := component.Subcomponents[1]
			Expect(secret.Name).To(Equal("authConfigSecret"))
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
