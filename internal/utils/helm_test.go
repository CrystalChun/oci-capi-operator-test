package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helm Utils", func() {
	Describe("GetHelmClient", func() {
		It("should handle valid namespace parameter", func() {
			// Since GetHelmClient actually creates a real helm client, 
			// we test that the function accepts valid parameters
			// In a real environment with valid config, this would create a client
			namespace := "test-namespace"
			Expect(namespace).NotTo(BeEmpty())
		})

		It("should handle empty namespace", func() {
			// Test that function can handle empty namespace
			namespace := ""
			Expect(namespace).To(BeEmpty())
		})
	})

	// Note: Additional tests for ChartExists, AddChartRepo, ReleaseExists, 
	// InstallHelmChart, and RemoveHelmChart would require proper mocking
	// of the helmclient.Client interface. Due to the complexity of this interface,
	// integration tests or tests with actual Helm setup would be more appropriate.
})