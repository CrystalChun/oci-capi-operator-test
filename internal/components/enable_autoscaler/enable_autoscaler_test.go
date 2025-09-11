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
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("Enable Autoscaler", func() {
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

	Context("GetComponents", func() {
		It("should return component with all subcomponents", func() {
			mockClient := &MockClient{}
			component := GetComponents(ctx, mockClient, "capi-system", "test-cluster", "capi-sa", instance, config)

			Expect(component.Name).To(Equal("EnableAutoscaler"))
			Expect(component.Subcomponents).To(HaveLen(6))

			// Verify BootstrapConfigSecret subcomponent
			bootstrapSecret := component.Subcomponents[0]
			Expect(bootstrapSecret.Name).To(Equal("bootstrapConfigSecret"))
			_, ok := bootstrapSecret.Object.(*corev1.Secret)
			Expect(ok).To(BeTrue())
			Expect(bootstrapSecret.MutateFn).NotTo(BeNil())

			// Verify KubeConfigSecret subcomponent
			kubeConfigSecret := component.Subcomponents[1]
			Expect(kubeConfigSecret.Name).To(Equal("kubeConfigSecret"))
			_, ok = kubeConfigSecret.Object.(*corev1.Secret)
			Expect(ok).To(BeTrue())
			Expect(kubeConfigSecret.MutateFn).NotTo(BeNil())

			// Verify MachineTemplate subcomponent
			machineTemplate := component.Subcomponents[2]
			Expect(machineTemplate.Name).To(Equal("machineTemplate"))
			_, ok = machineTemplate.Object.(*infrastructurev1beta2.OCIMachineTemplate)
			Expect(ok).To(BeTrue())
			Expect(machineTemplate.MutateFn).NotTo(BeNil())

			// Verify MachineDeployment subcomponent
			machineDeployment := component.Subcomponents[3]
			Expect(machineDeployment.Name).To(Equal("machineDeployment"))
			_, ok = machineDeployment.Object.(*capiv1beta1.MachineDeployment)
			Expect(ok).To(BeTrue())
			Expect(machineDeployment.MutateFn).NotTo(BeNil())

			// Verify OCICluster subcomponent
			ociCluster := component.Subcomponents[4]
			Expect(ociCluster.Name).To(Equal("ociCluster"))
			_, ok = ociCluster.Object.(*infrastructurev1beta2.OCICluster)
			Expect(ok).To(BeTrue())
			Expect(ociCluster.MutateFn).NotTo(BeNil())

			// Verify CAPICluster subcomponent
			capiCluster := component.Subcomponents[5]
			Expect(capiCluster.Name).To(Equal("cluster"))
			_, ok = capiCluster.Object.(*capiv1beta1.Cluster)
			Expect(ok).To(BeTrue())
			Expect(capiCluster.MutateFn).NotTo(BeNil())
		})

		It("should create components with correct names and namespaces", func() {
			mockClient := &MockClient{}
			component := GetComponents(ctx, mockClient, "custom-ns", "custom-cluster", "custom-sa", instance, config)

			// Check OCICluster
			ociCluster := component.Subcomponents[4].Object.(*infrastructurev1beta2.OCICluster)
			Expect(ociCluster.Name).To(Equal("custom-cluster"))
			Expect(ociCluster.Namespace).To(Equal("custom-ns"))

			// Check CAPICluster
			capiCluster := component.Subcomponents[5].Object.(*capiv1beta1.Cluster)
			Expect(capiCluster.Name).To(Equal("custom-cluster"))
			Expect(capiCluster.Namespace).To(Equal("custom-ns"))

			// Check MachineTemplate
			machineTemplate := component.Subcomponents[2].Object.(*infrastructurev1beta2.OCIMachineTemplate)
			Expect(machineTemplate.Name).To(Equal("custom-cluster-autoscaling"))
			Expect(machineTemplate.Namespace).To(Equal("custom-ns"))

			// Check MachineDeployment
			machineDeployment := component.Subcomponents[3].Object.(*capiv1beta1.MachineDeployment)
			Expect(machineDeployment.Name).To(Equal("custom-cluster"))
			Expect(machineDeployment.Namespace).To(Equal("custom-ns"))
		})
	})
})
