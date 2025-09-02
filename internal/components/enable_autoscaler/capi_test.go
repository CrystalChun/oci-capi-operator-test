package enableautoscaler

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("CAPI", func() {
	var (
		instance     *ocicapioperatorv1alpha1.OCIClusterAutoscaler
		capiSystemNS string
		clusterName  string
		config       Config
		scheme       *runtime.Scheme
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(infrastructurev1beta2.AddToScheme(scheme)).To(Succeed())
		Expect(capiv1beta1.AddToScheme(scheme)).To(Succeed())

		instance = &ocicapioperatorv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}

		capiSystemNS = "capi-system"
		clusterName = "test-cluster"

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

	Context("OCICluster", func() {
		It("should create OCI cluster with correct configuration", func() {
			cluster, mutateFn := OCICluster(capiSystemNS, clusterName, instance, config)
			Expect(cluster).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify cluster configuration
			c := cluster.(*infrastructurev1beta2.OCICluster)
			Expect(c.Name).To(Equal(clusterName))
			Expect(c.Namespace).To(Equal(capiSystemNS))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration after mutation
			Expect(c.Spec.CompartmentId).To(Equal(config.ClusterConfig.CompartmentID))
			Expect(c.Spec.ControlPlaneEndpoint.Host).To(Equal(config.NetworkConfig.ControlPlaneEndpoint))
			Expect(c.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(6443)))
			Expect(*c.Spec.NetworkSpec.APIServerLB.LoadBalancerId).To(Equal(config.NetworkConfig.APIServerLoadBalancerID))
			Expect(*c.Spec.NetworkSpec.Vcn.ID).To(Equal(config.NetworkConfig.VCNID))
			Expect(*c.Spec.NetworkSpec.Vcn.Subnets[0].ID).To(Equal(config.NetworkConfig.SubnetID))
			Expect(*c.Spec.NetworkSpec.Vcn.NetworkSecurityGroup.List[0].ID).To(Equal(config.NetworkConfig.NetworkSecurityGroupID))
		})
	})

	Context("CAPICluster", func() {
		It("should create CAPI cluster with correct configuration", func() {
			cluster, mutateFn := CAPICluster(capiSystemNS, clusterName, instance, config)
			Expect(cluster).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify cluster configuration
			c := cluster.(*capiv1beta1.Cluster)
			Expect(c.Name).To(Equal(clusterName))
			Expect(c.Namespace).To(Equal(capiSystemNS))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration after mutation
			Expect(c.Spec.ClusterNetwork.Pods.CIDRBlocks).To(Equal([]string{config.NetworkConfig.ClusterNetworkCIDRBlock}))
			Expect(c.Spec.ClusterNetwork.Services.CIDRBlocks).To(Equal([]string{config.NetworkConfig.ServiceNetworkCIDRBlock}))
			Expect(c.Spec.InfrastructureRef.Kind).To(Equal("OCICluster"))
			Expect(c.Spec.InfrastructureRef.Name).To(Equal(clusterName))
			Expect(c.Spec.InfrastructureRef.Namespace).To(Equal(capiSystemNS))
		})
	})

	Context("OCIMachineTemplate", func() {
		It("should create OCI machine template with correct configuration", func() {
			template, mutateFn := OCIMachineTemplate(capiSystemNS, clusterName, instance, config)
			Expect(template).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify template configuration
			t := template.(*infrastructurev1beta2.OCIMachineTemplate)
			Expect(t.Name).To(Equal(fmt.Sprintf("%s-autoscaling", clusterName)))
			Expect(t.Namespace).To(Equal(capiSystemNS))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration after mutation
			Expect(t.Spec.Template.Spec.ImageId).To(Equal(config.AutoScalingConfig.ImageID))
			Expect(t.Spec.Template.Spec.Shape).To(Equal(config.AutoScalingConfig.Shape))
			Expect(t.Spec.Template.Spec.ShapeConfig.Ocpus).To(Equal(fmt.Sprintf("%d", config.AutoScalingConfig.CPUs)))
			Expect(t.Spec.Template.Spec.ShapeConfig.MemoryInGBs).To(Equal(fmt.Sprintf("%d", config.AutoScalingConfig.Memory)))
		})
	})

	Context("MachineDeployment", func() {
		It("should create machine deployment with correct configuration", func() {
			deployment, mutateFn := MachineDeployment(capiSystemNS, clusterName, instance, config)
			Expect(deployment).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify deployment configuration
			d := deployment.(*capiv1beta1.MachineDeployment)
			Expect(d.Name).To(Equal(clusterName))
			Expect(d.Namespace).To(Equal(capiSystemNS))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration after mutation
			Expect(d.Annotations).To(HaveKeyWithValue("capacity.cluster-autoscaler.kubernetes.io/cpu", fmt.Sprintf("%d", config.AutoScalingConfig.CPUs)))
			Expect(d.Annotations).To(HaveKeyWithValue("capacity.cluster-autoscaler.kubernetes.io/memory", fmt.Sprintf("%dG", config.AutoScalingConfig.Memory)))
			Expect(d.Annotations).To(HaveKeyWithValue("cluster.x-k8s.io/cluster-api-autoscaler-node-group-min-size", fmt.Sprintf("%d", config.AutoScalingConfig.MinNodes)))
			Expect(d.Annotations).To(HaveKeyWithValue("cluster.x-k8s.io/cluster-api-autoscaler-node-group-max-size", fmt.Sprintf("%d", config.AutoScalingConfig.MaxNodes)))

			Expect(d.Spec.ClusterName).To(Equal(clusterName))
			Expect(d.Spec.Template.Spec.ClusterName).To(Equal(clusterName))
			Expect(*d.Spec.Template.Spec.Bootstrap.DataSecretName).To(Equal(fmt.Sprintf("%s-bootstrap", clusterName)))
			Expect(d.Spec.Template.Spec.InfrastructureRef.Kind).To(Equal("OCIMachineTemplate"))
			Expect(d.Spec.Template.Spec.InfrastructureRef.Name).To(Equal(fmt.Sprintf("%s-autoscaling", clusterName)))
			Expect(d.Spec.Template.Spec.InfrastructureRef.Namespace).To(Equal(capiSystemNS))
		})
	})

	Context("ValidateMinMaxNodes", func() {
		It("should validate min/max nodes successfully", func() {
			err := ValidateMinMaxNodes(instance, config)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail when min nodes is greater than max nodes", func() {
			config.AutoScalingConfig.MinNodes = 5
			config.AutoScalingConfig.MaxNodes = 3
			err := ValidateMinMaxNodes(instance, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("min nodes must be less than max nodes"))
		})

		It("should fail when min nodes is negative", func() {
			config.AutoScalingConfig.MinNodes = -1
			err := ValidateMinMaxNodes(instance, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("min nodes must be equal to or greater than 0"))
		})

		It("should fail when max nodes is negative", func() {
			config.AutoScalingConfig.MaxNodes = -1
			err := ValidateMinMaxNodes(instance, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("max nodes must be equal to or greater than 0"))
		})
	})
})
