package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Component", func() {
	var (
		component *Component
	)

	BeforeEach(func() {
		// Create a test component with subcomponents
		component = &Component{
			Name: "TestComponent",
			Subcomponents: SubcomponentList{
				{
					Name: "ConfigMap1",
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap-1",
							Namespace: "test-namespace",
						},
					},
					MutateFn: func() error {
						return nil
					},
				},
				{
					Name: "ConfigMap2",
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap-2",
							Namespace: "test-namespace",
						},
					},
					MutateFn: func() error {
						return nil
					},
				},
			},
		}
	})

	Context("GetName", func() {
		It("should return the component name", func() {
			Expect(component.GetName()).To(Equal("TestComponent"))
		})
	})

	Context("GetSubcomponents", func() {
		It("should return all subcomponents", func() {
			subcomponents := component.GetSubcomponents()
			Expect(subcomponents).To(HaveLen(2))
			Expect(subcomponents[0].Name).To(Equal("ConfigMap1"))
			Expect(subcomponents[1].Name).To(Equal("ConfigMap2"))
		})

		It("should return subcomponents with valid objects", func() {
			subcomponents := component.GetSubcomponents()
			for _, subcomponent := range subcomponents {
				Expect(subcomponent.Object).NotTo(BeNil())
				configMap := subcomponent.Object.(*corev1.ConfigMap)
				Expect(configMap.Namespace).To(Equal("test-namespace"))
			}
		})

		It("should return subcomponents with valid mutate functions", func() {
			subcomponents := component.GetSubcomponents()
			for _, subcomponent := range subcomponents {
				Expect(subcomponent.MutateFn).NotTo(BeNil())
				err := subcomponent.MutateFn()
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Context("SubcomponentList", func() {
		It("should be assignable to a slice of Subcomponent", func() {
			var list SubcomponentList
			list = []Subcomponent{
				{
					Name: "Test",
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test",
						},
					},
					MutateFn: func() error {
						return nil
					},
				},
			}
			Expect(list).To(HaveLen(1))
			Expect(list[0].Name).To(Equal("Test"))
		})
	})
})
