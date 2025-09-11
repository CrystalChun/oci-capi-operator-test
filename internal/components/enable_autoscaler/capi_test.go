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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/utils"
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("CAPI Components", func() {
	var (
		instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler
		config   Config
	)

	BeforeEach(func() {
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
				ClusterNetworkCIDRBlock: "10.0.0.0/16",
				ServiceNetworkCIDRBlock: "10.1.0.0/16",
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

	Context("OCICluster", func() {
		It("should create OCICluster with correct configuration", func() {
			obj, mutateFn := OCICluster("capi-system", "test-cluster", instance, config)
			ociCluster, ok := obj.(*infrastructurev1beta2.OCICluster)
			Expect(ok).To(BeTrue(), "Object should be an OCICluster")

			// Verify initial state
			Expect(ociCluster.Name).To(Equal("test-cluster"))
			Expect(ociCluster.Namespace).To(Equal("capi-system"))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels and annotations
			defaultLabels := utils.GetDefaultLabels(instance.Name)
			Expect(ociCluster.Labels).To(Equal(defaultLabels))
			Expect(ociCluster.Annotations).To(HaveKeyWithValue("cluster.x-k8s.io/skip-apiserver-lb-management", "true"))

			// Verify spec
			Expect(ociCluster.Spec.CompartmentId).To(Equal(config.ClusterConfig.CompartmentID))
			Expect(ociCluster.Spec.ControlPlaneEndpoint.Host).To(Equal(config.NetworkConfig.ControlPlaneEndpoint))
			Expect(ociCluster.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(6443)))

			// Verify network spec
			Expect(*ociCluster.Spec.NetworkSpec.APIServerLB.LoadBalancerId).To(Equal(config.NetworkConfig.APIServerLoadBalancerID))
			Expect(ociCluster.Spec.NetworkSpec.SkipNetworkManagement).To(BeTrue())
			Expect(*ociCluster.Spec.NetworkSpec.Vcn.ID).To(Equal(config.NetworkConfig.VCNID))
			Expect(ociCluster.Spec.NetworkSpec.Vcn.Subnets).To(HaveLen(1))
			Expect(*ociCluster.Spec.NetworkSpec.Vcn.Subnets[0].ID).To(Equal(config.NetworkConfig.SubnetID))
			Expect(ociCluster.Spec.NetworkSpec.Vcn.NetworkSecurityGroup.List).To(HaveLen(1))
			Expect(*ociCluster.Spec.NetworkSpec.Vcn.NetworkSecurityGroup.List[0].ID).To(Equal(config.NetworkConfig.NetworkSecurityGroupID))
		})

		It("should preserve existing annotations", func() {
			obj, mutateFn := OCICluster("capi-system", "test-cluster", instance, config)
			ociCluster := obj.(*infrastructurev1beta2.OCICluster)

			// Add existing annotation
			ociCluster.Annotations = map[string]string{
				"existing-key": "existing-value",
			}

			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			Expect(ociCluster.Annotations).To(HaveKeyWithValue("existing-key", "existing-value"))
			Expect(ociCluster.Annotations).To(HaveKeyWithValue("cluster.x-k8s.io/skip-apiserver-lb-management", "true"))
		})
	})

	Context("CAPICluster", func() {
		It("should create Cluster with correct configuration", func() {
			obj, mutateFn := CAPICluster("capi-system", "test-cluster", instance, config)
			cluster, ok := obj.(*capiv1beta1.Cluster)
			Expect(ok).To(BeTrue(), "Object should be a Cluster")

			// Verify initial state
			Expect(cluster.Name).To(Equal("test-cluster"))
			Expect(cluster.Namespace).To(Equal("capi-system"))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels
			defaultLabels := utils.GetDefaultLabels(instance.Name)
			Expect(cluster.Labels).To(Equal(defaultLabels))

			// Verify spec
			Expect(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks).To(ConsistOf(config.NetworkConfig.ClusterNetworkCIDRBlock))
			Expect(cluster.Spec.ClusterNetwork.Services.CIDRBlocks).To(ConsistOf(config.NetworkConfig.ServiceNetworkCIDRBlock))
			Expect(cluster.Spec.ClusterNetwork.ServiceDomain).To(Equal("cluster.local"))

			// Verify infrastructure ref
			Expect(cluster.Spec.InfrastructureRef.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(cluster.Spec.InfrastructureRef.Kind).To(Equal("OCICluster"))
			Expect(cluster.Spec.InfrastructureRef.Name).To(Equal("test-cluster"))
			Expect(cluster.Spec.InfrastructureRef.Namespace).To(Equal("capi-system"))
		})
	})

	Context("OCIMachineTemplate", func() {
		It("should create OCIMachineTemplate with correct configuration", func() {
			obj, mutateFn := OCIMachineTemplate("capi-system", "test-cluster", instance, config)
			template, ok := obj.(*infrastructurev1beta2.OCIMachineTemplate)
			Expect(ok).To(BeTrue(), "Object should be an OCIMachineTemplate")

			// Verify initial state
			Expect(template.Name).To(Equal("test-cluster-autoscaling"))
			Expect(template.Namespace).To(Equal("capi-system"))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify spec
			Expect(template.Spec.Template.Spec.ImageId).To(Equal(config.AutoScalingConfig.ImageID))
			Expect(template.Spec.Template.Spec.Shape).To(Equal(config.AutoScalingConfig.Shape))
			Expect(template.Spec.Template.Spec.ShapeConfig.Ocpus).To(Equal("2"))
			Expect(template.Spec.Template.Spec.ShapeConfig.MemoryInGBs).To(Equal("4"))
			Expect(template.Spec.Template.Spec.IsPvEncryptionInTransitEnabled).To(BeFalse())
		})
	})

	Context("MachineDeployment", func() {
		It("should create MachineDeployment with correct configuration", func() {
			obj, mutateFn := MachineDeployment("capi-system", "test-cluster", instance, config)
			deployment, ok := obj.(*capiv1beta1.MachineDeployment)
			Expect(ok).To(BeTrue(), "Object should be a MachineDeployment")

			// Verify initial state
			Expect(deployment.Name).To(Equal("test-cluster"))
			Expect(deployment.Namespace).To(Equal("capi-system"))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels
			defaultLabels := utils.GetDefaultLabels(instance.Name)
			Expect(deployment.Labels).To(Equal(defaultLabels))

			// Verify annotations
			Expect(deployment.Annotations).To(HaveKeyWithValue("capacity.cluster-autoscaler.kubernetes.io/cpu", "2"))
			Expect(deployment.Annotations).To(HaveKeyWithValue("capacity.cluster-autoscaler.kubernetes.io/memory", "4G"))
			Expect(deployment.Annotations).To(HaveKeyWithValue("cluster.x-k8s.io/cluster-api-autoscaler-node-group-min-size", "1"))
			Expect(deployment.Annotations).To(HaveKeyWithValue("cluster.x-k8s.io/cluster-api-autoscaler-node-group-max-size", "3"))

			// Verify spec
			Expect(deployment.Spec.ClusterName).To(Equal("test-cluster"))
			Expect(deployment.Spec.Template.Spec.ClusterName).To(Equal("test-cluster"))
			Expect(*deployment.Spec.Template.Spec.Bootstrap.DataSecretName).To(Equal("test-cluster-bootstrap"))

			// Verify infrastructure ref
			infraRef := deployment.Spec.Template.Spec.InfrastructureRef
			Expect(infraRef.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(infraRef.Kind).To(Equal("OCIMachineTemplate"))
			Expect(infraRef.Name).To(Equal("test-cluster-autoscaling"))
			Expect(infraRef.Namespace).To(Equal("capi-system"))
		})

		It("should preserve existing annotations", func() {
			obj, mutateFn := MachineDeployment("capi-system", "test-cluster", instance, config)
			deployment := obj.(*capiv1beta1.MachineDeployment)

			// Add existing annotation
			deployment.Annotations = map[string]string{
				"existing-key": "existing-value",
			}

			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			Expect(deployment.Annotations).To(HaveKeyWithValue("existing-key", "existing-value"))
			Expect(deployment.Annotations).To(HaveKeyWithValue("capacity.cluster-autoscaler.kubernetes.io/cpu", "2"))
		})
	})

	Context("ValidateMinMaxNodes", func() {
		It("should pass with valid min/max values", func() {
			err := ValidateMinMaxNodes(instance, config)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail when min > max", func() {
			config.AutoScalingConfig.MinNodes = 5
			config.AutoScalingConfig.MaxNodes = 3

			err := ValidateMinMaxNodes(instance, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("min nodes must be less than max nodes"))
		})

		It("should fail with negative min nodes", func() {
			config.AutoScalingConfig.MinNodes = -1

			err := ValidateMinMaxNodes(instance, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("min nodes must be equal to or greater than 0"))
		})

		It("should fail with negative max nodes", func() {
			config.AutoScalingConfig.MaxNodes = -1

			err := ValidateMinMaxNodes(instance, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("max nodes must be equal to or greater than 0"))
		})

		It("should use instance values over config values", func() {
			instance.Spec.Autoscaling.MinNodes = 2
			instance.Spec.Autoscaling.MaxNodes = 4

			err := ValidateMinMaxNodes(instance, config)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.Autoscaling.MinNodes = 5
			instance.Spec.Autoscaling.MaxNodes = 3

			err = ValidateMinMaxNodes(instance, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("min nodes must be less than max nodes"))
		})
	})
})
