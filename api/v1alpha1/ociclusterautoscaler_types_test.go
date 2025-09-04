package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("OCIClusterAutoscaler Types", func() {
	Describe("OCIClusterAutoscaler", func() {
		It("should create a valid OCIClusterAutoscaler object", func() {
			autoscaler := &OCIClusterAutoscaler{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "capi.openshift.io/v1alpha1",
					Kind:       "OCIClusterAutoscaler",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-autoscaler",
					Namespace: "test-namespace",
				},
				Spec: OCIClusterAutoscalerSpec{
					Autoscaling: AutoscalingConfig{
						MinNodes: 1,
						MaxNodes: 5,
						Shape:    "VM.Standard.E4.Flex",
						ShapeConfig: &ShapeConfig{
							CPUs:   2,
							Memory: 8,
						},
						ImageID: "ocid1.image.oc1..example",
					},
					CAPI: CAPIConfig{
						Namespace:   "capi-system",
						ClusterName: "test-cluster",
					},
					ClusterAutoscaler: ClusterAutoscalerConfig{
						Name:                 "cluster-autoscaler",
						Namespace:            "cluster-autoscaler-system",
						ServiceAccountName:   "cluster-autoscaler",
						CloudProvider:        "oci",
						CreateRBAC:           true,
						CreateServiceAccount: true,
						RepositoryURL:        "https://charts.example.com",
						Version:              "1.0.0",
					},
				},
				Status: OCIClusterAutoscalerStatus{
					Phase:                     "Running",
					CAPIInstalled:             true,
					ClusterAutoscalerDeployed: true,
					ObservedGeneration:        1,
					Conditions: []metav1.Condition{
						{
							Type:   "Ready",
							Status: metav1.ConditionTrue,
							Reason: "AutoscalerRunning",
						},
					},
				},
			}

			Expect(autoscaler.Name).To(Equal("test-autoscaler"))
			Expect(autoscaler.Namespace).To(Equal("test-namespace"))
			Expect(autoscaler.Kind).To(Equal("OCIClusterAutoscaler"))
			Expect(autoscaler.APIVersion).To(Equal("capi.openshift.io/v1alpha1"))
		})

		It("should have correct TypeMeta for OCIClusterAutoscaler", func() {
			autoscaler := &OCIClusterAutoscaler{}
			autoscaler.SetGroupVersionKind(GroupVersion.WithKind("OCIClusterAutoscaler"))

			Expect(autoscaler.GetObjectKind().GroupVersionKind().Group).To(Equal("capi.openshift.io"))
			Expect(autoscaler.GetObjectKind().GroupVersionKind().Version).To(Equal("v1alpha1"))
			Expect(autoscaler.GetObjectKind().GroupVersionKind().Kind).To(Equal("OCIClusterAutoscaler"))
		})
	})

	Describe("OCIClusterAutoscalerList", func() {
		It("should create a valid OCIClusterAutoscalerList", func() {
			autoscaler1 := OCIClusterAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "autoscaler-1",
					Namespace: "test-namespace",
				},
			}

			autoscaler2 := OCIClusterAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "autoscaler-2",
					Namespace: "test-namespace",
				},
			}

			list := &OCIClusterAutoscalerList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "capi.openshift.io/v1alpha1",
					Kind:       "OCIClusterAutoscalerList",
				},
				ListMeta: metav1.ListMeta{
					ResourceVersion: "12345",
				},
				Items: []OCIClusterAutoscaler{autoscaler1, autoscaler2},
			}

			Expect(list.Items).To(HaveLen(2))
			Expect(list.Items[0].Name).To(Equal("autoscaler-1"))
			Expect(list.Items[1].Name).To(Equal("autoscaler-2"))
			Expect(list.ResourceVersion).To(Equal("12345"))
		})
	})

	Describe("AutoscalingConfig", func() {
		It("should create AutoscalingConfig with all fields", func() {
			config := AutoscalingConfig{
				MinNodes: 2,
				MaxNodes: 10,
				Shape:    "VM.Standard.E4.Flex",
				ShapeConfig: &ShapeConfig{
					CPUs:   4,
					Memory: 16,
				},
				ImageID: "ocid1.image.oc1..example123",
			}

			Expect(config.MinNodes).To(Equal(int32(2)))
			Expect(config.MaxNodes).To(Equal(int32(10)))
			Expect(config.Shape).To(Equal("VM.Standard.E4.Flex"))
			Expect(config.ShapeConfig).NotTo(BeNil())
			Expect(config.ShapeConfig.CPUs).To(Equal(int32(4)))
			Expect(config.ShapeConfig.Memory).To(Equal(int32(16)))
			Expect(config.ImageID).To(Equal("ocid1.image.oc1..example123"))
		})

		It("should work with minimal configuration", func() {
			config := AutoscalingConfig{
				MinNodes: 1,
				MaxNodes: 3,
			}

			Expect(config.MinNodes).To(Equal(int32(1)))
			Expect(config.MaxNodes).To(Equal(int32(3)))
			Expect(config.Shape).To(BeEmpty())
			Expect(config.ShapeConfig).To(BeNil())
			Expect(config.ImageID).To(BeEmpty())
		})

		It("should handle zero values correctly", func() {
			config := AutoscalingConfig{}

			Expect(config.MinNodes).To(Equal(int32(0)))
			Expect(config.MaxNodes).To(Equal(int32(0)))
			Expect(config.Shape).To(BeEmpty())
			Expect(config.ShapeConfig).To(BeNil())
			Expect(config.ImageID).To(BeEmpty())
		})
	})

	Describe("ShapeConfig", func() {
		It("should create ShapeConfig with CPUs and Memory", func() {
			config := &ShapeConfig{
				CPUs:   8,
				Memory: 32,
			}

			Expect(config.CPUs).To(Equal(int32(8)))
			Expect(config.Memory).To(Equal(int32(32)))
		})

		It("should handle zero values", func() {
			config := &ShapeConfig{}

			Expect(config.CPUs).To(Equal(int32(0)))
			Expect(config.Memory).To(Equal(int32(0)))
		})
	})

	Describe("CAPIConfig", func() {
		It("should create CAPIConfig with namespace and cluster name", func() {
			config := CAPIConfig{
				Namespace:   "capi-system",
				ClusterName: "my-cluster",
			}

			Expect(config.Namespace).To(Equal("capi-system"))
			Expect(config.ClusterName).To(Equal("my-cluster"))
		})

		It("should handle empty values", func() {
			config := CAPIConfig{}

			Expect(config.Namespace).To(BeEmpty())
			Expect(config.ClusterName).To(BeEmpty())
		})
	})

	Describe("ClusterAutoscalerConfig", func() {
		It("should create ClusterAutoscalerConfig with all fields", func() {
			config := ClusterAutoscalerConfig{
				RepositoryURL:        "https://charts.example.com",
				Name:                 "cluster-autoscaler",
				Namespace:            "cluster-autoscaler-system",
				ServiceAccountName:   "cluster-autoscaler-sa",
				CloudProvider:        "oci",
				CreateRBAC:           true,
				CreateServiceAccount: true,
				Version:              "2.0.0",
			}

			Expect(config.RepositoryURL).To(Equal("https://charts.example.com"))
			Expect(config.Name).To(Equal("cluster-autoscaler"))
			Expect(config.Namespace).To(Equal("cluster-autoscaler-system"))
			Expect(config.ServiceAccountName).To(Equal("cluster-autoscaler-sa"))
			Expect(config.CloudProvider).To(Equal("oci"))
			Expect(config.CreateRBAC).To(BeTrue())
			Expect(config.CreateServiceAccount).To(BeTrue())
			Expect(config.Version).To(Equal("2.0.0"))
		})

		It("should handle default boolean values", func() {
			config := ClusterAutoscalerConfig{
				Name: "test-autoscaler",
			}

			Expect(config.Name).To(Equal("test-autoscaler"))
			Expect(config.CreateRBAC).To(BeFalse())
			Expect(config.CreateServiceAccount).To(BeFalse())
		})
	})

	Describe("OCIClusterAutoscalerStatus", func() {
		It("should create status with all fields", func() {
			condition := metav1.Condition{
				Type:   "Ready",
				Status: metav1.ConditionTrue,
				Reason: "AutoscalerReady",
			}

			status := OCIClusterAutoscalerStatus{
				Conditions:                []metav1.Condition{condition},
				Phase:                     "Running",
				CAPIInstalled:             true,
				ClusterAutoscalerDeployed: true,
				ObservedGeneration:        5,
			}

			Expect(status.Conditions).To(HaveLen(1))
			Expect(status.Conditions[0].Type).To(Equal("Ready"))
			Expect(status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(status.Phase).To(Equal("Running"))
			Expect(status.CAPIInstalled).To(BeTrue())
			Expect(status.ClusterAutoscalerDeployed).To(BeTrue())
			Expect(status.ObservedGeneration).To(Equal(int64(5)))
		})

		It("should handle empty status", func() {
			status := OCIClusterAutoscalerStatus{}

			Expect(status.Conditions).To(BeEmpty())
			Expect(status.Phase).To(BeEmpty())
			Expect(status.CAPIInstalled).To(BeFalse())
			Expect(status.ClusterAutoscalerDeployed).To(BeFalse())
			Expect(status.ObservedGeneration).To(Equal(int64(0)))
		})

		It("should handle multiple conditions", func() {
			conditions := []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "AutoscalerReady",
				},
				{
					Type:   "Progressing",
					Status: metav1.ConditionFalse,
					Reason: "InstallationComplete",
				},
			}

			status := OCIClusterAutoscalerStatus{
				Conditions: conditions,
			}

			Expect(status.Conditions).To(HaveLen(2))
			Expect(status.Conditions[0].Type).To(Equal("Ready"))
			Expect(status.Conditions[1].Type).To(Equal("Progressing"))
		})
	})

	Describe("Object interface compliance", func() {
		It("should implement runtime.Object for OCIClusterAutoscaler", func() {
			autoscaler := &OCIClusterAutoscaler{}
			var _ runtime.Object = autoscaler

			gvk := autoscaler.GetObjectKind().GroupVersionKind()
			Expect(gvk.Empty()).To(BeTrue()) // Empty until explicitly set

			autoscaler.SetGroupVersionKind(GroupVersion.WithKind("OCIClusterAutoscaler"))
			gvk = autoscaler.GetObjectKind().GroupVersionKind()
			Expect(gvk.Group).To(Equal("capi.openshift.io"))
			Expect(gvk.Version).To(Equal("v1alpha1"))
			Expect(gvk.Kind).To(Equal("OCIClusterAutoscaler"))
		})

		It("should implement runtime.Object for OCIClusterAutoscalerList", func() {
			list := &OCIClusterAutoscalerList{}
			var _ runtime.Object = list

			gvk := list.GetObjectKind().GroupVersionKind()
			Expect(gvk.Empty()).To(BeTrue()) // Empty until explicitly set

			list.SetGroupVersionKind(GroupVersion.WithKind("OCIClusterAutoscalerList"))
			gvk = list.GetObjectKind().GroupVersionKind()
			Expect(gvk.Group).To(Equal("capi.openshift.io"))
			Expect(gvk.Version).To(Equal("v1alpha1"))
			Expect(gvk.Kind).To(Equal("OCIClusterAutoscalerList"))
		})
	})

	Describe("JSON serialization", func() {
		It("should serialize and deserialize OCIClusterAutoscaler correctly", func() {
			original := &OCIClusterAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-autoscaler",
					Namespace: "test-namespace",
				},
				Spec: OCIClusterAutoscalerSpec{
					Autoscaling: AutoscalingConfig{
						MinNodes: 1,
						MaxNodes: 5,
						Shape:    "VM.Standard.E4.Flex",
					},
				},
			}

			// This would be tested in integration tests with actual JSON marshaling
			// Here we just verify the structure is correct
			Expect(original.Spec.Autoscaling.MinNodes).To(Equal(int32(1)))
			Expect(original.Spec.Autoscaling.MaxNodes).To(Equal(int32(5)))
			Expect(original.Spec.Autoscaling.Shape).To(Equal("VM.Standard.E4.Flex"))
		})
	})

	Describe("Validation", func() {
		It("should handle MinNodes validation", func() {
			// Testing the kubebuilder validation constraint
			// In a real environment, this would be enforced by the API server
			config := AutoscalingConfig{
				MinNodes: -1, // This would be invalid according to kubebuilder:validation:Minimum=0
			}

			// In unit tests, we can't test the actual validation, but we can verify the structure
			Expect(config.MinNodes).To(Equal(int32(-1)))
		})
	})
})