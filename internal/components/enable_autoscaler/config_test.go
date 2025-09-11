/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package enableautoscaler

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Config", func() {
	var (
		ctx      context.Context
		instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler
		config   Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		instance = &ocicapioperatorv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
			Spec: ocicapioperatorv1alpha1.OCIClusterAutoscalerSpec{
				Autoscaling: ocicapioperatorv1alpha1.AutoscalingConfig{},
			},
		}

		config = Config{
			AutoScalingConfig: AutoScalingConfig{
				CPUs:     2,
				Memory:   4,
				MinNodes: 1,
				MaxNodes: 3,
				Shape:    "oc3",
				ImageID:  "",
			},
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
				ServiceNetworkCIDRBlock: "10.1.0.0/16",
			},
		}
	})

	Context("SetAutoScalingConfig", func() {
		It("should use default values when instance values are empty", func() {
			mockClient := &MockClient{}
			result, err := SetAutoScalingConfig(ctx, mockClient, instance, config)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.AutoScalingConfig).To(Equal(config.AutoScalingConfig))
		})

		It("should override values from instance", func() {
			instance.Spec.Autoscaling = ocicapioperatorv1alpha1.AutoscalingConfig{
				MinNodes: 2,
				MaxNodes: 5,
				Shape:    "custom-shape",
				ImageID:  "custom-image",
				ShapeConfig: &ocicapioperatorv1alpha1.ShapeConfig{
					CPUs:   4,
					Memory: 8,
				},
			}

			mockClient := &MockClient{}
			result, err := SetAutoScalingConfig(ctx, mockClient, instance, config)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.AutoScalingConfig.CPUs).To(Equal(int32(4)))
			Expect(result.AutoScalingConfig.Memory).To(Equal(int32(8)))
			Expect(result.AutoScalingConfig.MinNodes).To(Equal(int32(2)))
			Expect(result.AutoScalingConfig.MaxNodes).To(Equal(int32(5)))
			Expect(result.AutoScalingConfig.Shape).To(Equal("custom-shape"))
			Expect(result.AutoScalingConfig.ImageID).To(Equal("custom-image"))
			// Networking config should be the same as the default config
			Expect(result.NetworkConfig.ClusterNetworkCIDRBlock).To(Equal("10.0.0.0/16"))
			Expect(result.NetworkConfig.ServiceNetworkCIDRBlock).To(Equal("10.1.0.0/16"))
			Expect(result.NetworkConfig.NetworkSecurityGroupID).To(Equal("test-nsg"))
			Expect(result.NetworkConfig.ControlPlaneEndpoint).To(Equal("test-endpoint"))
			Expect(result.NetworkConfig.APIServerLoadBalancerID).To(Equal("test-lb"))
			Expect(result.NetworkConfig.VCNID).To(Equal("test-vcn"))
			Expect(result.NetworkConfig.SubnetID).To(Equal("test-subnet"))
		})

		It("should handle partial overrides", func() {
			instance.Spec.Autoscaling = ocicapioperatorv1alpha1.AutoscalingConfig{
				MinNodes: 2,
				Shape:    "custom-shape",
			}

			mockClient := &MockClient{}
			result, err := SetAutoScalingConfig(ctx, mockClient, instance, config)
			Expect(err).NotTo(HaveOccurred())

			// Overridden values
			Expect(result.AutoScalingConfig.MinNodes).To(Equal(int32(2)))
			Expect(result.AutoScalingConfig.Shape).To(Equal("custom-shape"))

			// Default values
			Expect(result.AutoScalingConfig.CPUs).To(Equal(int32(2)))
			Expect(result.AutoScalingConfig.Memory).To(Equal(int32(4)))
			Expect(result.AutoScalingConfig.MaxNodes).To(Equal(int32(3)))
			Expect(result.AutoScalingConfig.ImageID).To(Equal(""))
		})
	})

	Context("SetNetworkConfig", func() {
		It("should use existing values when CIDRs are set", func() {
			mockClient := &MockClient{}
			result, err := SetNetworkConfig(ctx, mockClient, config)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.NetworkConfig.ClusterNetworkCIDRBlock).To(Equal("10.0.0.0/16"))
			Expect(result.NetworkConfig.ServiceNetworkCIDRBlock).To(Equal("10.1.0.0/16"))
		})
	})
})
