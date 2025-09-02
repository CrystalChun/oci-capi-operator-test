package crds

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1alpha3 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
)

var _ = Describe("CRDs", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("GetClusterctlComponents", func() {
		It("should get CRDs for core provider", func() {
			components, err := GetClusterctlComponents(ctx, "cluster-api", v1alpha3.CoreProviderType)
			Expect(err).NotTo(HaveOccurred())
			Expect(components).To(BeAssignableToTypeOf([]unstructured.Unstructured{}))

			// Verify each component is a CRD and has the required annotation
			for _, component := range components {
				Expect(component.GetKind()).To(Equal("CustomResourceDefinition"))
				annotations := component.GetAnnotations()
				Expect(annotations).To(HaveKeyWithValue("service.beta.openshift.io/inject-cabundle", "true"))
			}
		})

		It("should get CRDs for infrastructure provider", func() {
			components, err := GetClusterctlComponents(ctx, "oci", v1alpha3.InfrastructureProviderType)
			Expect(err).NotTo(HaveOccurred())
			Expect(components).To(BeAssignableToTypeOf([]unstructured.Unstructured{}))

			// Verify each component is a CRD and has the required annotation
			for _, component := range components {
				Expect(component.GetKind()).To(Equal("CustomResourceDefinition"))
				annotations := component.GetAnnotations()
				Expect(annotations).To(HaveKeyWithValue("service.beta.openshift.io/inject-cabundle", "true"))
			}
		})

		It("should fail with invalid provider", func() {
			_, err := GetClusterctlComponents(ctx, "invalid-provider", v1alpha3.CoreProviderType)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error generating CAPI components"))
		})

		It("should return empty list when no CRDs are found", func() {
			// This is a hypothetical case since we can't easily create a provider with no CRDs
			components, err := GetClusterctlComponents(ctx, "cluster-api", v1alpha3.BootstrapProviderType)
			Expect(err).NotTo(HaveOccurred())
			Expect(components).To(BeEmpty())
		})
	})
})
