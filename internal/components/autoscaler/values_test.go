package autoscaler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Autoscaler Values", func() {
	var (
		defaultValues AutoscalerDeploymentValues
	)

	BeforeEach(func() {
		defaultValues = AutoscalerDeploymentValues{
			CloudProvider:        "oci",
			Name:                 "cluster-autoscaler",
			Namespace:            "default",
			ServiceAccountName:   "cluster-autoscaler",
			CreateRBAC:           false,
			CreateServiceAccount: false,
			RepositoryURL:        "https://kubernetes.github.io/autoscaler",
			Chart:                "cluster-autoscaler",
			Version:              "9.29.0",
		}
	})

	Context("GetValuesString", func() {
		It("should generate correct values string", func() {
			valuesStr := GetValuesString(&defaultValues)
			Expect(valuesStr).To(ContainSubstring("cloudProvider: oci"))
			Expect(valuesStr).To(ContainSubstring("fullnameOverride: cluster-autoscaler"))
			Expect(valuesStr).To(ContainSubstring("namespace: default"))
			Expect(valuesStr).To(ContainSubstring("create: false"))
			Expect(valuesStr).To(ContainSubstring("name: cluster-autoscaler"))
		})

		It("should handle different values", func() {
			values := AutoscalerDeploymentValues{
				CloudProvider:        "aws",
				Name:                 "test-autoscaler",
				Namespace:            "test-namespace",
				ServiceAccountName:   "test-sa",
				CreateRBAC:           true,
				CreateServiceAccount: true,
			}
			valuesStr := GetValuesString(&values)
			Expect(valuesStr).To(ContainSubstring("cloudProvider: aws"))
			Expect(valuesStr).To(ContainSubstring("fullnameOverride: test-autoscaler"))
			Expect(valuesStr).To(ContainSubstring("namespace: test-namespace"))
			Expect(valuesStr).To(ContainSubstring("create: true"))
			Expect(valuesStr).To(ContainSubstring("name: test-sa"))
		})
	})

	Context("GetAutoscalerDeploymentValues", func() {
		It("should use default values when instance values are empty", func() {
			instance := &capiv1alpha1.OCIClusterAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-autoscaler",
				},
				Spec: capiv1alpha1.OCIClusterAutoscalerSpec{
					ClusterAutoscaler: capiv1alpha1.ClusterAutoscalerConfig{},
				},
			}

			values := GetAutoscalerDeploymentValues(defaultValues, instance)
			Expect(values).To(Equal(defaultValues))
		})

		It("should override default values with instance values", func() {
			instance := &capiv1alpha1.OCIClusterAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-autoscaler",
				},
				Spec: capiv1alpha1.OCIClusterAutoscalerSpec{
					ClusterAutoscaler: capiv1alpha1.ClusterAutoscalerConfig{
						CloudProvider:        "aws",
						Name:                 "test-autoscaler",
						Namespace:            "test-namespace",
						ServiceAccountName:   "test-sa",
						CreateRBAC:           true,
						CreateServiceAccount: true,
						RepositoryURL:        "https://test-repo.com",
						Version:              "1.0.0",
					},
				},
			}

			values := GetAutoscalerDeploymentValues(defaultValues, instance)
			Expect(values.CloudProvider).To(Equal("aws"))
			Expect(values.Name).To(Equal("test-autoscaler"))
			Expect(values.Namespace).To(Equal("test-namespace"))
			Expect(values.ServiceAccountName).To(Equal("test-sa"))
			Expect(values.CreateRBAC).To(BeTrue())
			Expect(values.CreateServiceAccount).To(BeTrue())
			Expect(values.RepositoryURL).To(Equal("https://test-repo.com"))
			Expect(values.Version).To(Equal("1.0.0"))
		})

		It("should only override specified values", func() {
			instance := &capiv1alpha1.OCIClusterAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-autoscaler",
				},
				Spec: capiv1alpha1.OCIClusterAutoscalerSpec{
					ClusterAutoscaler: capiv1alpha1.ClusterAutoscalerConfig{
						CloudProvider: "aws",
						Name:          "test-autoscaler",
					},
				},
			}

			values := GetAutoscalerDeploymentValues(defaultValues, instance)
			Expect(values.CloudProvider).To(Equal("aws"))
			Expect(values.Name).To(Equal("test-autoscaler"))
			Expect(values.Namespace).To(Equal(defaultValues.Namespace))
			Expect(values.ServiceAccountName).To(Equal(defaultValues.ServiceAccountName))
			Expect(values.CreateRBAC).To(Equal(defaultValues.CreateRBAC))
			Expect(values.CreateServiceAccount).To(Equal(defaultValues.CreateServiceAccount))
			Expect(values.RepositoryURL).To(Equal(defaultValues.RepositoryURL))
			Expect(values.Version).To(Equal(defaultValues.Version))
		})
	})
})
