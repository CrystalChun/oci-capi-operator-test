package enableautoscaler

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Enable Autoscaler Config", func() {
	var (
		ctx           context.Context
		fakeClient    client.Client
		scheme        *runtime.Scheme
		testInstance  *capiv1alpha1.OCIClusterAutoscaler
		baseConfig    Config
		testNetwork   *configv1.Network
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = configv1.AddToScheme(scheme)
		_ = capiv1alpha1.AddToScheme(scheme)

		testNetwork = &configv1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: configv1.NetworkSpec{
				ClusterNetwork: []configv1.ClusterNetworkEntry{
					{CIDR: "10.128.0.0/14"},
				},
				ServiceNetwork: []string{"172.30.0.0/16"},
			},
		}

		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(testNetwork).
			Build()

		testInstance = &capiv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-autoscaler",
				Namespace: "test-namespace",
			},
			Spec: capiv1alpha1.OCIClusterAutoscalerSpec{},
		}

		baseConfig = Config{
			AutoScalingConfig: AutoScalingConfig{
				CPUs:     2,
				Memory:   4,
				MinNodes: 1,
				MaxNodes: 3,
				Shape:    "oc3",
				ImageID:  "test-image",
			},
			ClusterConfig: ClusterConfig{
				CompartmentID: "test-compartment",
			},
			NetworkConfig: NetworkConfig{
				VCNID:                   "test-vcn",
				SubnetID:                "test-subnet",
				NetworkSecurityGroupID:  "test-nsg",
				ControlPlaneEndpoint:    "https://test.endpoint.com",
				APIServerLoadBalancerID: "test-lb",
			},
		}
	})

	Describe("Config structs", func() {
		It("should create AutoScalingConfig with all fields", func() {
			config := AutoScalingConfig{
				CPUs:     4,
				Memory:   8,
				MinNodes: 2,
				MaxNodes: 5,
				Shape:    "custom",
				ImageID:  "custom-image",
			}

			Expect(config.CPUs).To(Equal(int32(4)))
			Expect(config.Memory).To(Equal(int32(8)))
			Expect(config.MinNodes).To(Equal(int32(2)))
			Expect(config.MaxNodes).To(Equal(int32(5)))
			Expect(config.Shape).To(Equal("custom"))
			Expect(config.ImageID).To(Equal("custom-image"))
		})

		It("should create ClusterConfig with compartment ID", func() {
			config := ClusterConfig{
				CompartmentID: "test-compartment-id",
			}

			Expect(config.CompartmentID).To(Equal("test-compartment-id"))
		})

		It("should create NetworkConfig with all fields", func() {
			config := NetworkConfig{
				VCNID:                   "test-vcn-id",
				SubnetID:                "test-subnet-id",
				NetworkSecurityGroupID:  "test-nsg-id",
				ControlPlaneEndpoint:    "https://api.test.com",
				APIServerLoadBalancerID: "test-lb-id",
				ClusterNetworkCIDRBlock: "10.0.0.0/16",
				ServiceNetworkCIDRBlock: "172.30.0.0/16",
			}

			Expect(config.VCNID).To(Equal("test-vcn-id"))
			Expect(config.SubnetID).To(Equal("test-subnet-id"))
			Expect(config.NetworkSecurityGroupID).To(Equal("test-nsg-id"))
			Expect(config.ControlPlaneEndpoint).To(Equal("https://api.test.com"))
			Expect(config.APIServerLoadBalancerID).To(Equal("test-lb-id"))
			Expect(config.ClusterNetworkCIDRBlock).To(Equal("10.0.0.0/16"))
			Expect(config.ServiceNetworkCIDRBlock).To(Equal("172.30.0.0/16"))
		})
	})

	Describe("SetAutoScalingConfig", func() {
		It("should override CPUs and Memory when ShapeConfig is specified", func() {
			testInstance.Spec.Autoscaling.ShapeConfig = &capiv1alpha1.ShapeConfig{
				CPUs:   8,
				Memory: 16,
			}

			result, err := SetAutoScalingConfig(ctx, fakeClient, testInstance, baseConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.AutoScalingConfig.CPUs).To(Equal(int32(8)))
			Expect(result.AutoScalingConfig.Memory).To(Equal(int32(16)))
			Expect(result.AutoScalingConfig.MinNodes).To(Equal(int32(1))) // unchanged
			Expect(result.AutoScalingConfig.MaxNodes).To(Equal(int32(3))) // unchanged
		})

		It("should override MinNodes and MaxNodes when specified", func() {
			testInstance.Spec.Autoscaling.MinNodes = 5
			testInstance.Spec.Autoscaling.MaxNodes = 10

			result, err := SetAutoScalingConfig(ctx, fakeClient, testInstance, baseConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.AutoScalingConfig.MinNodes).To(Equal(int32(5)))
			Expect(result.AutoScalingConfig.MaxNodes).To(Equal(int32(10)))
			Expect(result.AutoScalingConfig.CPUs).To(Equal(int32(2))) // unchanged
			Expect(result.AutoScalingConfig.Memory).To(Equal(int32(4))) // unchanged
		})

		It("should override Shape when specified", func() {
			testInstance.Spec.Autoscaling.Shape = "custom-shape"

			result, err := SetAutoScalingConfig(ctx, fakeClient, testInstance, baseConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.AutoScalingConfig.Shape).To(Equal("custom-shape"))
		})

		It("should override ImageID when specified", func() {
			testInstance.Spec.Autoscaling.ImageID = "custom-image-id"

			result, err := SetAutoScalingConfig(ctx, fakeClient, testInstance, baseConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.AutoScalingConfig.ImageID).To(Equal("custom-image-id"))
		})

		It("should not override values when they are zero/empty", func() {
			testInstance.Spec.Autoscaling.ShapeConfig = &capiv1alpha1.ShapeConfig{
				CPUs:   0, // zero value, should not override
				Memory: 8,
			}
			testInstance.Spec.Autoscaling.MinNodes = 0 // zero value, should not override
			testInstance.Spec.Autoscaling.MaxNodes = 5
			testInstance.Spec.Autoscaling.Shape = "" // empty, should not override
			testInstance.Spec.Autoscaling.ImageID = "new-image"

			result, err := SetAutoScalingConfig(ctx, fakeClient, testInstance, baseConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.AutoScalingConfig.CPUs).To(Equal(int32(2))) // unchanged
			Expect(result.AutoScalingConfig.Memory).To(Equal(int32(8))) // changed
			Expect(result.AutoScalingConfig.MinNodes).To(Equal(int32(1))) // unchanged
			Expect(result.AutoScalingConfig.MaxNodes).To(Equal(int32(5))) // changed
			Expect(result.AutoScalingConfig.Shape).To(Equal("oc3")) // unchanged
			Expect(result.AutoScalingConfig.ImageID).To(Equal("new-image")) // changed
		})

		It("should set network config from cluster when not specified", func() {
			// Need to copy the config properly to avoid modifying the original
			testConfig := Config{
				AutoScalingConfig: baseConfig.AutoScalingConfig,
				ClusterConfig:     baseConfig.ClusterConfig,
				NetworkConfig: NetworkConfig{
					VCNID:                   baseConfig.NetworkConfig.VCNID,
					SubnetID:                baseConfig.NetworkConfig.SubnetID,
					NetworkSecurityGroupID:  baseConfig.NetworkConfig.NetworkSecurityGroupID,
					ControlPlaneEndpoint:    baseConfig.NetworkConfig.ControlPlaneEndpoint,
					APIServerLoadBalancerID: baseConfig.NetworkConfig.APIServerLoadBalancerID,
					ClusterNetworkCIDRBlock: "", // Clear this to test retrieval
					ServiceNetworkCIDRBlock: "", // Clear this to test retrieval
				},
			}

			result, err := SetAutoScalingConfig(ctx, fakeClient, testInstance, testConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.NetworkConfig.ClusterNetworkCIDRBlock).To(Equal("10.128.0.0/14"))
			Expect(result.NetworkConfig.ServiceNetworkCIDRBlock).To(Equal("172.30.0.0/16"))
		})

		It("should return error when network config fails", func() {
			emptyClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			testConfig := Config{
				AutoScalingConfig: baseConfig.AutoScalingConfig,
				ClusterConfig:     baseConfig.ClusterConfig,
				NetworkConfig: NetworkConfig{
					VCNID:                   baseConfig.NetworkConfig.VCNID,
					SubnetID:                baseConfig.NetworkConfig.SubnetID,
					NetworkSecurityGroupID:  baseConfig.NetworkConfig.NetworkSecurityGroupID,
					ControlPlaneEndpoint:    baseConfig.NetworkConfig.ControlPlaneEndpoint,
					APIServerLoadBalancerID: baseConfig.NetworkConfig.APIServerLoadBalancerID,
					ClusterNetworkCIDRBlock: "", // This will cause the error
					ServiceNetworkCIDRBlock: baseConfig.NetworkConfig.ServiceNetworkCIDRBlock,
				},
			}

			result, err := SetAutoScalingConfig(ctx, emptyClient, testInstance, testConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to set network config"))
			Expect(result).To(Equal(testConfig)) // should return original config on error
		})
	})

	Describe("SetNetworkConfig", func() {
		It("should retrieve cluster network CIDR when not set", func() {
			config := baseConfig
			config.NetworkConfig.ClusterNetworkCIDRBlock = ""
			config.NetworkConfig.ServiceNetworkCIDRBlock = "existing-service-cidr"

			result, err := SetNetworkConfig(ctx, fakeClient, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.NetworkConfig.ClusterNetworkCIDRBlock).To(Equal("10.128.0.0/14"))
			Expect(result.NetworkConfig.ServiceNetworkCIDRBlock).To(Equal("existing-service-cidr")) // unchanged
		})

		It("should retrieve service network CIDR when not set", func() {
			config := baseConfig
			config.NetworkConfig.ClusterNetworkCIDRBlock = "existing-cluster-cidr"
			config.NetworkConfig.ServiceNetworkCIDRBlock = ""

			result, err := SetNetworkConfig(ctx, fakeClient, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.NetworkConfig.ClusterNetworkCIDRBlock).To(Equal("existing-cluster-cidr")) // unchanged
			Expect(result.NetworkConfig.ServiceNetworkCIDRBlock).To(Equal("172.30.0.0/16"))
		})

		It("should retrieve both network CIDRs when not set", func() {
			config := baseConfig
			config.NetworkConfig.ClusterNetworkCIDRBlock = ""
			config.NetworkConfig.ServiceNetworkCIDRBlock = ""

			result, err := SetNetworkConfig(ctx, fakeClient, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.NetworkConfig.ClusterNetworkCIDRBlock).To(Equal("10.128.0.0/14"))
			Expect(result.NetworkConfig.ServiceNetworkCIDRBlock).To(Equal("172.30.0.0/16"))
		})

		It("should not change network CIDRs when already set", func() {
			config := baseConfig
			config.NetworkConfig.ClusterNetworkCIDRBlock = "custom-cluster-cidr"
			config.NetworkConfig.ServiceNetworkCIDRBlock = "custom-service-cidr"

			result, err := SetNetworkConfig(ctx, fakeClient, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.NetworkConfig.ClusterNetworkCIDRBlock).To(Equal("custom-cluster-cidr"))
			Expect(result.NetworkConfig.ServiceNetworkCIDRBlock).To(Equal("custom-service-cidr"))
		})

		It("should return error when cluster network CIDR retrieval fails", func() {
			emptyClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			config := baseConfig
			config.NetworkConfig.ClusterNetworkCIDRBlock = ""

			result, err := SetNetworkConfig(ctx, emptyClient, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get cluster network CIDR block"))
			Expect(result).To(Equal(config)) // should return original config on error
		})

		It("should return error when service network CIDR retrieval fails", func() {
			emptyClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			config := baseConfig
			config.NetworkConfig.ClusterNetworkCIDRBlock = "existing-cluster-cidr" // Set this so it won't fail on cluster CIDR
			config.NetworkConfig.ServiceNetworkCIDRBlock = ""

			result, err := SetNetworkConfig(ctx, emptyClient, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get service network CIDR block"))
			Expect(result).To(Equal(config)) // should return original config on error
		})
	})
})