package components

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Components", func() {
	var (
		testComponent *Component
		testObject1   *corev1.ConfigMap
		testObject2   *corev1.Secret
		mutateFn1     controllerutil.MutateFn
		mutateFn2     controllerutil.MutateFn
	)

	BeforeEach(func() {
		testObject1 = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cm",
				Namespace: "test-namespace",
			},
		}

		testObject2 = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-namespace",
			},
		}

		mutateFn1 = func() error {
			testObject1.Data = map[string]string{"key1": "value1"}
			return nil
		}

		mutateFn2 = func() error {
			testObject2.StringData = map[string]string{"key2": "value2"}
			return nil
		}

		testComponent = &Component{
			Name: "TestComponent",
			Subcomponents: SubcomponentList{
				{
					Name:     "configmap",
					Object:   testObject1,
					MutateFn: mutateFn1,
				},
				{
					Name:     "secret",
					Object:   testObject2,
					MutateFn: mutateFn2,
				},
			},
		}
	})

	Describe("Component struct", func() {
		It("should create component with name and subcomponents", func() {
			component := &Component{
				Name:          "MyComponent",
				Subcomponents: SubcomponentList{},
			}

			Expect(component.Name).To(Equal("MyComponent"))
			Expect(component.Subcomponents).To(HaveLen(0))
		})

		It("should handle component with multiple subcomponents", func() {
			Expect(testComponent.Name).To(Equal("TestComponent"))
			Expect(testComponent.Subcomponents).To(HaveLen(2))
		})
	})

	Describe("GetName", func() {
		It("should return component name", func() {
			name := testComponent.GetName()
			Expect(name).To(Equal("TestComponent"))
		})

		It("should return empty string for empty name", func() {
			emptyComponent := &Component{Name: ""}
			name := emptyComponent.GetName()
			Expect(name).To(Equal(""))
		})
	})

	Describe("GetSubcomponents", func() {
		It("should return all subcomponents", func() {
			subcomponents := testComponent.GetSubcomponents()
			Expect(subcomponents).To(HaveLen(2))
			Expect(subcomponents[0].Name).To(Equal("configmap"))
			Expect(subcomponents[1].Name).To(Equal("secret"))
		})

		It("should return empty list for component with no subcomponents", func() {
			emptyComponent := &Component{
				Name:          "Empty",
				Subcomponents: SubcomponentList{},
			}
			subcomponents := emptyComponent.GetSubcomponents()
			Expect(subcomponents).To(HaveLen(0))
		})
	})

	Describe("Subcomponent struct", func() {
		It("should create subcomponent with all fields", func() {
			subcomp := Subcomponent{
				Name:     "test-subcomponent",
				Object:   testObject1,
				MutateFn: mutateFn1,
			}

			Expect(subcomp.Name).To(Equal("test-subcomponent"))
			Expect(subcomp.Object).To(Equal(testObject1))
			Expect(subcomp.MutateFn).NotTo(BeNil())
		})

		It("should allow nil mutate function", func() {
			subcomp := Subcomponent{
				Name:     "test-subcomponent",
				Object:   testObject1,
				MutateFn: nil,
			}

			Expect(subcomp.Name).To(Equal("test-subcomponent"))
			Expect(subcomp.Object).To(Equal(testObject1))
			Expect(subcomp.MutateFn).To(BeNil())
		})
	})

	Describe("SubcomponentList", func() {
		It("should behave as a slice", func() {
			list := SubcomponentList{
				{Name: "first", Object: testObject1, MutateFn: mutateFn1},
				{Name: "second", Object: testObject2, MutateFn: mutateFn2},
			}

			Expect(list).To(HaveLen(2))
			Expect(list[0].Name).To(Equal("first"))
			Expect(list[1].Name).To(Equal("second"))
		})

		It("should support append operations", func() {
			list := SubcomponentList{}
			newSubcomp := Subcomponent{
				Name:     "appended",
				Object:   testObject1,
				MutateFn: mutateFn1,
			}

			list = append(list, newSubcomp)
			Expect(list).To(HaveLen(1))
			Expect(list[0].Name).To(Equal("appended"))
		})
	})

	Describe("Object interface compliance", func() {
		It("should ensure objects implement client.Object", func() {
			for _, subcomp := range testComponent.Subcomponents {
				var _ client.Object = subcomp.Object
				Expect(subcomp.Object.GetName()).NotTo(BeEmpty())
				Expect(subcomp.Object.GetNamespace()).NotTo(BeEmpty())
			}
		})
	})

	Describe("MutateFn functionality", func() {
		It("should execute mutate functions correctly", func() {
			// Verify initial state
			Expect(testObject1.Data).To(BeNil())
			Expect(testObject2.StringData).To(BeNil())

			// Execute mutate functions
			for _, subcomp := range testComponent.Subcomponents {
				if subcomp.MutateFn != nil {
					err := subcomp.MutateFn()
					Expect(err).NotTo(HaveOccurred())
				}
			}

			// Verify mutations were applied
			Expect(testObject1.Data).To(HaveKeyWithValue("key1", "value1"))
			Expect(testObject2.StringData).To(HaveKeyWithValue("key2", "value2"))
		})

		It("should handle nil mutate functions gracefully", func() {
			subcomp := Subcomponent{
				Name:     "no-mutate",
				Object:   testObject1,
				MutateFn: nil,
			}

			Expect(subcomp.MutateFn).To(BeNil())
			// This should not panic when MutateFn is nil
		})

		It("should handle mutate function errors", func() {
			errorMutateFn := func() error {
				return errors.New("test error")
			}

			subcomp := Subcomponent{
				Name:     "error-mutate",
				Object:   testObject1,
				MutateFn: errorMutateFn,
			}

			err := subcomp.MutateFn()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("test error"))
		})
	})

	Describe("Component composition", func() {
		It("should allow creating components from other components", func() {
			subComponent1 := &Component{
				Name: "SubComponent1",
				Subcomponents: SubcomponentList{
					{Name: "sub1", Object: testObject1, MutateFn: mutateFn1},
				},
			}

			subComponent2 := &Component{
				Name: "SubComponent2",
				Subcomponents: SubcomponentList{
					{Name: "sub2", Object: testObject2, MutateFn: mutateFn2},
				},
			}

			// Components can be used to organize subcomponents
			Expect(subComponent1.GetSubcomponents()).To(HaveLen(1))
			Expect(subComponent2.GetSubcomponents()).To(HaveLen(1))

			// We can combine them conceptually
			allSubcomponents := append(subComponent1.GetSubcomponents(), subComponent2.GetSubcomponents()...)
			Expect(allSubcomponents).To(HaveLen(2))
		})
	})
})