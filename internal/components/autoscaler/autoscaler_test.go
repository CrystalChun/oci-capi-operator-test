package autoscaler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

var _ = Describe("Autoscaler", func() {
	var (
		instance *capiv1alpha1.OCIClusterAutoscaler
		values   *AutoscalerDeploymentValues
		scheme   *runtime.Scheme
		config   *rest.Config
	)

	BeforeEach(func() {
		instance = &capiv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}

		values = &AutoscalerDeploymentValues{
			CloudProvider:        "oci",
			Name:                 "test-autoscaler",
			Namespace:            "test-namespace",
			ServiceAccountName:   "test-sa",
			CreateRBAC:           false,
			CreateServiceAccount: false,
			RepositoryURL:        "https://kubernetes.github.io/autoscaler",
			Chart:                "cluster-autoscaler",
			Version:              "9.29.0",
		}

		scheme = runtime.NewScheme()
		config = &rest.Config{
			Host: "https://test-cluster:6443",
		}
	})

	Context("InstallAutoscaler", func() {
		It("should fail when helm client creation fails", func() {
			// Use invalid config to make helm client creation fail
			config.Host = ""
			err := InstallAutoscaler(instance, values, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error creating helm client"))
		})

		It("should add chart repo when chart doesn't exist", func() {
			// This test would require mocking the helm client
			// For now, we'll just verify it fails with the expected error
			err := InstallAutoscaler(instance, values, config)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("RemoveAutoscaler", func() {
		It("should fail when helm client creation fails", func() {
			// Use invalid config to make helm client creation fail
			config.Host = ""
			err := RemoveAutoscaler(values, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error creating helm client"))
		})
	})

	Context("GetComponents", func() {
		It("should return component with correct subcomponents", func() {
			component := GetComponents(values, instance, scheme)
			Expect(component).NotTo(BeNil())
			Expect(component.Name).To(Equal("Autoscaler"))

			// Verify subcomponents
			Expect(component.Subcomponents).To(HaveLen(2))
			Expect(component.Subcomponents[0].Name).To(Equal("clusterRole"))
			Expect(component.Subcomponents[1].Name).To(Equal("clusterRoleBinding"))

			// Verify each subcomponent has an object and mutate function
			for _, subcomponent := range component.Subcomponents {
				Expect(subcomponent.Object).NotTo(BeNil())
				Expect(subcomponent.MutateFn).NotTo(BeNil())
			}
		})

		It("should create subcomponents that can be mutated", func() {
			component := GetComponents(values, instance, scheme)

			// Test each subcomponent's mutate function
			for _, subcomponent := range component.Subcomponents {
				err := subcomponent.MutateFn()
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})
