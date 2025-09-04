package autoscaler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Autoscaler Values", func() {
	Describe("GetValuesString", func() {
		It("should generate correct values string with all fields", func() {
			values := &AutoscalerDeploymentValues{
				CloudProvider:        "oci",
				Name:                 "cluster-autoscaler",
				Namespace:            "cluster-autoscaler-system",
				ServiceAccountName:   "cluster-autoscaler-sa",
				CreateRBAC:           true,
				CreateServiceAccount: true,
			}

			valuesStr := GetValuesString(values)
			Expect(valuesStr).To(ContainSubstring("cloudProvider: oci"))
			Expect(valuesStr).To(ContainSubstring("fullnameOverride: cluster-autoscaler"))
			Expect(valuesStr).To(ContainSubstring("namespace: cluster-autoscaler-system"))
			Expect(valuesStr).To(ContainSubstring("create: true"))
			Expect(valuesStr).To(ContainSubstring("name: cluster-autoscaler-sa"))
		})

		It("should generate correct values string with RBAC disabled", func() {
			values := &AutoscalerDeploymentValues{
				CloudProvider:        "aws",
				Name:                 "test-autoscaler",
				Namespace:            "test-namespace",
				ServiceAccountName:   "test-sa",
				CreateRBAC:           false,
				CreateServiceAccount: false,
			}

			valuesStr := GetValuesString(values)
			Expect(valuesStr).To(ContainSubstring("cloudProvider: aws"))
			Expect(valuesStr).To(ContainSubstring("fullnameOverride: test-autoscaler"))
			Expect(valuesStr).To(ContainSubstring("namespace: test-namespace"))
			Expect(valuesStr).To(ContainSubstring("create: false"))
			Expect(valuesStr).To(ContainSubstring("name: test-sa"))
		})

		It("should handle empty values", func() {
			values := &AutoscalerDeploymentValues{}
			valuesStr := GetValuesString(values)
			Expect(valuesStr).To(ContainSubstring("cloudProvider: "))
			Expect(valuesStr).To(ContainSubstring("fullnameOverride: "))
			Expect(valuesStr).To(ContainSubstring("namespace: "))
			Expect(valuesStr).To(ContainSubstring("create: false"))
			Expect(valuesStr).To(ContainSubstring("name: "))
		})
	})

	Describe("GetAutoscalerDeploymentValues", func() {
		var (
			originalValues AutoscalerDeploymentValues
			instance       *capiv1alpha1.OCIClusterAutoscaler
		)

		BeforeEach(func() {
			originalValues = AutoscalerDeploymentValues{
				CloudProvider:        "default-provider",
				Name:                 "default-name",
				Namespace:            "default-namespace",
				ServiceAccountName:   "default-sa",
				CreateRBAC:           false,
				CreateServiceAccount: false,
				RepositoryURL:        "https://default.repo.com",
				Chart:                "default-chart",
				Version:              "1.0.0",
			}

			instance = &capiv1alpha1.OCIClusterAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-instance",
					Namespace: "test-namespace",
				},
				Spec: capiv1alpha1.OCIClusterAutoscalerSpec{},
			}
		})

		It("should override CloudProvider when specified in instance", func() {
			instance.Spec.ClusterAutoscaler.CloudProvider = "oci"
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.CloudProvider).To(Equal("oci"))
			Expect(result.Name).To(Equal("default-name")) // unchanged
		})

		It("should override Name when specified in instance", func() {
			instance.Spec.ClusterAutoscaler.Name = "custom-autoscaler"
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.Name).To(Equal("custom-autoscaler"))
			Expect(result.CloudProvider).To(Equal("default-provider")) // unchanged
		})

		It("should override Namespace when specified in instance", func() {
			instance.Spec.ClusterAutoscaler.Namespace = "custom-namespace"
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.Namespace).To(Equal("custom-namespace"))
		})

		It("should override ServiceAccountName when specified in instance", func() {
			instance.Spec.ClusterAutoscaler.ServiceAccountName = "custom-sa"
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.ServiceAccountName).To(Equal("custom-sa"))
		})

		It("should override CreateRBAC when set to true in instance", func() {
			instance.Spec.ClusterAutoscaler.CreateRBAC = true
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.CreateRBAC).To(BeTrue())
		})

		It("should not override CreateRBAC when false in instance (keeps original)", func() {
			originalValues.CreateRBAC = true
			instance.Spec.ClusterAutoscaler.CreateRBAC = false
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.CreateRBAC).To(BeTrue()) // original value preserved
		})

		It("should override CreateServiceAccount when set to true in instance", func() {
			instance.Spec.ClusterAutoscaler.CreateServiceAccount = true
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.CreateServiceAccount).To(BeTrue())
		})

		It("should override RepositoryURL when specified in instance", func() {
			instance.Spec.ClusterAutoscaler.RepositoryURL = "https://custom.repo.com"
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.RepositoryURL).To(Equal("https://custom.repo.com"))
		})

		It("should override Version when specified in instance", func() {
			instance.Spec.ClusterAutoscaler.Version = "2.0.0"
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.Version).To(Equal("2.0.0"))
		})

		It("should override multiple fields when specified in instance", func() {
			instance.Spec.ClusterAutoscaler.CloudProvider = "oci"
			instance.Spec.ClusterAutoscaler.Name = "custom-autoscaler"
			instance.Spec.ClusterAutoscaler.Namespace = "custom-namespace"
			instance.Spec.ClusterAutoscaler.ServiceAccountName = "custom-sa"
			instance.Spec.ClusterAutoscaler.CreateRBAC = true
			instance.Spec.ClusterAutoscaler.CreateServiceAccount = true
			instance.Spec.ClusterAutoscaler.RepositoryURL = "https://custom.repo.com"
			instance.Spec.ClusterAutoscaler.Version = "2.0.0"

			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.CloudProvider).To(Equal("oci"))
			Expect(result.Name).To(Equal("custom-autoscaler"))
			Expect(result.Namespace).To(Equal("custom-namespace"))
			Expect(result.ServiceAccountName).To(Equal("custom-sa"))
			Expect(result.CreateRBAC).To(BeTrue())
			Expect(result.CreateServiceAccount).To(BeTrue())
			Expect(result.RepositoryURL).To(Equal("https://custom.repo.com"))
			Expect(result.Version).To(Equal("2.0.0"))
			Expect(result.Chart).To(Equal("default-chart")) // unchanged
		})

		It("should preserve original values when instance has empty spec", func() {
			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result).To(Equal(originalValues))
		})

		It("should only override non-empty fields from instance", func() {
			instance.Spec.ClusterAutoscaler.CloudProvider = "oci"
			instance.Spec.ClusterAutoscaler.Name = "" // empty, should not override
			instance.Spec.ClusterAutoscaler.Version = "2.0.0"

			result := GetAutoscalerDeploymentValues(originalValues, instance)
			Expect(result.CloudProvider).To(Equal("oci"))
			Expect(result.Name).To(Equal("default-name")) // preserved
			Expect(result.Version).To(Equal("2.0.0"))
		})
	})

	Describe("AutoscalerDeploymentValues struct", func() {
		It("should create struct with all fields", func() {
			values := AutoscalerDeploymentValues{
				CloudProvider:        "oci",
				Name:                 "test-autoscaler",
				Namespace:            "test-namespace",
				ServiceAccountName:   "test-sa",
				CreateRBAC:           true,
				CreateServiceAccount: true,
				RepositoryURL:        "https://test.repo.com",
				Chart:                "autoscaler",
				Version:              "1.0.0",
			}

			Expect(values.CloudProvider).To(Equal("oci"))
			Expect(values.Name).To(Equal("test-autoscaler"))
			Expect(values.Namespace).To(Equal("test-namespace"))
			Expect(values.ServiceAccountName).To(Equal("test-sa"))
			Expect(values.CreateRBAC).To(BeTrue())
			Expect(values.CreateServiceAccount).To(BeTrue())
			Expect(values.RepositoryURL).To(Equal("https://test.repo.com"))
			Expect(values.Chart).To(Equal("autoscaler"))
			Expect(values.Version).To(Equal("1.0.0"))
		})
	})
})