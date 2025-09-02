package enableautoscaler

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Config", func() {
	var (
		ctx       context.Context
		k8sClient client.Client
		scheme    *runtime.Scheme
		instance  *ocicapioperatorv1alpha1.OCIClusterAutoscaler
		config    Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(configv1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		instance = &ocicapioperatorv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}

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
				ClusterNetworkCIDRBlock: "",
				ServiceNetworkCIDRBlock: "",
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

		// Create test network config
		network := &configv1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: configv1.NetworkSpec{
				ClusterNetwork: []configv1.ClusterNetworkEntry{
					{
						CIDR: "10.0.0.0/16",
					},
				},
				ServiceNetwork: []string{"172.16.0.0/16"},
			},
		}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())
	})

	Context("SetAutoScalingConfig", func() {
		It("should use default values when instance values are not set", func() {
			updatedConfig, err := SetAutoScalingConfig(ctx, k8sClient, instance, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedConfig.AutoScalingConfig).To(Equal(config.AutoScalingConfig))
		})

		It("should override values with instance values when set", func() {
			instance.Spec.Autoscaling = ocicapioperatorv1alpha1.AutoscalingConfig{
				ShapeConfig: &ocicapioperatorv1alpha1.ShapeConfig{
					CPUs:   4,
					Memory: 8,
				},
				MinNodes: 2,
				MaxNodes: 5,
				Shape:    "oc4",
				ImageID:  "new-image",
			}

			updatedConfig, err := SetAutoScalingConfig(ctx, k8sClient, instance, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedConfig.AutoScalingConfig.CPUs).To(Equal(int32(4)))
			Expect(updatedConfig.AutoScalingConfig.Memory).To(Equal(int32(8)))
			Expect(updatedConfig.AutoScalingConfig.MinNodes).To(Equal(int32(2)))
			Expect(updatedConfig.AutoScalingConfig.MaxNodes).To(Equal(int32(5)))
			Expect(updatedConfig.AutoScalingConfig.Shape).To(Equal("oc4"))
			Expect(updatedConfig.AutoScalingConfig.ImageID).To(Equal("new-image"))
		})
	})

	Context("SetNetworkConfig", func() {
		It("should set network CIDRs from cluster when not set in config", func() {
			updatedConfig, err := SetNetworkConfig(ctx, k8sClient, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedConfig.NetworkConfig.ClusterNetworkCIDRBlock).To(Equal("10.0.0.0/16"))
			Expect(updatedConfig.NetworkConfig.ServiceNetworkCIDRBlock).To(Equal("172.16.0.0/16"))
		})

		It("should keep existing network CIDRs when set in config", func() {
			config.NetworkConfig.ClusterNetworkCIDRBlock = "192.168.0.0/16"
			config.NetworkConfig.ServiceNetworkCIDRBlock = "10.96.0.0/12"

			updatedConfig, err := SetNetworkConfig(ctx, k8sClient, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedConfig.NetworkConfig.ClusterNetworkCIDRBlock).To(Equal("192.168.0.0/16"))
			Expect(updatedConfig.NetworkConfig.ServiceNetworkCIDRBlock).To(Equal("10.96.0.0/12"))
		})

		It("should fail when network config does not exist", func() {
			// Delete the network config
			network := &configv1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
			}
			Expect(k8sClient.Delete(ctx, network)).To(Succeed())

			_, err := SetNetworkConfig(ctx, k8sClient, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get cluster network CIDR block"))
		})
	})
})
