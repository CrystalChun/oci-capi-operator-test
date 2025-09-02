package utils

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1alpha3 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
)

var _ = Describe("Clusterctl Utils", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("GenerateCAPIComponents", func() {
		It("should generate CAPI components successfully", func() {
			components, err := GenerateCAPIComponents(ctx, "cluster-api", v1alpha3.CoreProviderType, "test-namespace")
			Expect(err).NotTo(HaveOccurred())
			Expect(components).To(BeAssignableToTypeOf([]unstructured.Unstructured{}))
		})

		It("should fail with invalid provider", func() {
			_, err := GenerateCAPIComponents(ctx, "invalid-provider", v1alpha3.CoreProviderType, "test-namespace")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting provider components"))
		})

		It("should fail with empty provider", func() {
			_, err := GenerateCAPIComponents(ctx, "", v1alpha3.CoreProviderType, "test-namespace")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting provider components"))
		})

		It("should generate components with different provider types", func() {
			providerTypes := []v1alpha3.ProviderType{
				v1alpha3.CoreProviderType,
				v1alpha3.BootstrapProviderType,
				v1alpha3.ControlPlaneProviderType,
				v1alpha3.InfrastructureProviderType,
			}

			for _, providerType := range providerTypes {
				components, err := GenerateCAPIComponents(ctx, "cluster-api", providerType, "test-namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(components).To(BeAssignableToTypeOf([]unstructured.Unstructured{}))
			}
		})

		It("should generate components with different namespaces", func() {
			namespaces := []string{
				"default",
				"kube-system",
				"custom-namespace",
			}

			for _, namespace := range namespaces {
				components, err := GenerateCAPIComponents(ctx, "cluster-api", v1alpha3.CoreProviderType, namespace)
				Expect(err).NotTo(HaveOccurred())
				Expect(components).To(BeAssignableToTypeOf([]unstructured.Unstructured{}))
			}
		})
	})
})
