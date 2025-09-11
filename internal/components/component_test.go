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

package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Component", func() {
	var component *Component

	BeforeEach(func() {
		// Create a test component with some subcomponents
		component = &Component{
			Name: "test-component",
			Subcomponents: SubcomponentList{
				{
					Name: "sub1",
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-cm-1",
						},
					},
				},
				{
					Name: "sub2",
					Object: &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-secret-1",
						},
					},
				},
			},
		}
	})

	It("should return the correct component name", func() {
		Expect(component.GetName()).To(Equal("test-component"))
	})

	It("should return all subcomponents", func() {
		subcomponents := component.GetSubcomponents()
		Expect(subcomponents).To(HaveLen(2))
		Expect(subcomponents[0].Name).To(Equal("sub1"))
		Expect(subcomponents[1].Name).To(Equal("sub2"))
	})

	It("should have properly typed subcomponent objects", func() {
		subcomponents := component.GetSubcomponents()

		// First subcomponent should be a ConfigMap
		configMap, ok := subcomponents[0].Object.(*corev1.ConfigMap)
		Expect(ok).To(BeTrue())
		Expect(configMap.Name).To(Equal("test-cm-1"))

		// Second subcomponent should be a Secret
		secret, ok := subcomponents[1].Object.(*corev1.Secret)
		Expect(ok).To(BeTrue())
		Expect(secret.Name).To(Equal("test-secret-1"))
	})
})
