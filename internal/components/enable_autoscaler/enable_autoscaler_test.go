package enableautoscaler

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Enable Autoscaler", func() {
	var (
		ctx              context.Context
		k8sClient        client.Client
		scheme           *runtime.Scheme
		instance         *ocicapioperatorv1alpha1.OCIClusterAutoscaler
		capiSystemNS     string
		clusterName      string
		serviceAccountSA string
		config           Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		instance = &ocicapioperatorv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}

		capiSystemNS = "capi-system"
		clusterName = "test-cluster"
		serviceAccountSA = "capi-sa"

		config = Config{
			ClusterConfig: ClusterConfig{
				CompartmentID: "test-compartment",
			},
			NetworkConfig: NetworkConfig{
				VCNID:                   "test-vcn",
				SubnetID:                "test-subnet",
				NetworkSecurityGroupID:  "test-nsg",
				ControlPlaneEndpoint:    "test-endpoint",
				APIServerLoadBalancerID: "test-lb",
				ClusterNetworkCIDRBlock: "10.0.0.0/16",
				ServiceNetworkCIDRBlock: "172.16.0.0/16",
			},
			AutoScalingConfig: AutoScalingConfig{
				CPUs:     2,
				Memory:   4,
				MinNodes: 1,
				MaxNodes: 3,
				Shape:    "oc3",
				ImageID:  "test-image",
			},
		}
	})

	Context("GetComponents", func() {
		It("should return component with correct subcomponents", func() {
			component := GetComponents(ctx, k8sClient, capiSystemNS, clusterName, serviceAccountSA, instance, config)
			Expect(component).NotTo(BeNil())
			Expect(component.Name).To(Equal("EnableAutoscaler"))

			// Verify subcomponents
			Expect(component.Subcomponents).To(HaveLen(6))
			Expect(component.Subcomponents[0].Name).To(Equal("bootstrapConfigSecret"))
			Expect(component.Subcomponents[1].Name).To(Equal("kubeConfigSecret"))
			Expect(component.Subcomponents[2].Name).To(Equal("machineTemplate"))
			Expect(component.Subcomponents[3].Name).To(Equal("machineDeployment"))
			Expect(component.Subcomponents[4].Name).To(Equal("ociCluster"))
			Expect(component.Subcomponents[5].Name).To(Equal("cluster"))

			// Verify each subcomponent has an object and mutate function
			for _, subcomponent := range component.Subcomponents {
				Expect(subcomponent.Object).NotTo(BeNil())
				Expect(subcomponent.MutateFn).NotTo(BeNil())
			}
		})

		It("should create subcomponents that can be mutated", func() {
			component := GetComponents(ctx, k8sClient, capiSystemNS, clusterName, serviceAccountSA, instance, config)

			// Test each subcomponent's mutate function
			for _, subcomponent := range component.Subcomponents {
				err := subcomponent.MutateFn()
				if subcomponent.Name == "bootstrapConfigSecret" || subcomponent.Name == "kubeConfigSecret" {
					// These will fail because we haven't set up all the required resources
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})
	})
})
