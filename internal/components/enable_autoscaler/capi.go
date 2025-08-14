package enableautoscaler

import (
	"context"
	"fmt"

	"github.com/go-openapi/swag"
	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/utils"
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func OCICluster(capiSystemNamespace, clusterName string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler, config Config) (client.Object, func() error) { // Create OCICluster
	ociCluster := &infrastructurev1beta2.OCICluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		utils.SetDefaultLabels(ociCluster, instance.Name)
		ociCluster.Spec = infrastructurev1beta2.OCIClusterSpec{
			CompartmentId: config.ClusterConfig.CompartmentID,
			NetworkSpec: infrastructurev1beta2.NetworkSpec{
				SkipNetworkManagement: true,
				Vcn: infrastructurev1beta2.VCN{
					ID: swag.String(config.NetworkConfig.VCNID),
					Subnets: []*infrastructurev1beta2.Subnet{
						{
							ID:   swag.String(config.NetworkConfig.SubnetID),
							Name: "private",
							Role: "worker",
						},
					},
					NetworkSecurityGroup: infrastructurev1beta2.NetworkSecurityGroup{
						List: []*infrastructurev1beta2.NSG{
							{
								ID:   swag.String(config.NetworkConfig.NetworkSecurityGroupID),
								Name: "cluster-compute-nsg",
								Role: "worker",
							},
						},
					},
				},
			},
		}
		return nil
	}

	return ociCluster, mutateFn
}

func CAPICluster(capiSystemNamespace, clusterName string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler, config Config) (client.Object, func() error) {
	cluster := &capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		utils.SetDefaultLabels(cluster, instance.Name)
		cluster.Spec = capiv1beta1.ClusterSpec{
			ClusterNetwork: &capiv1beta1.ClusterNetwork{
				Pods: &capiv1beta1.NetworkRanges{
					CIDRBlocks: []string{config.NetworkConfig.ClusterNetworkCIDRBlock},
				},
				ServiceDomain: "cluster.local",
				Services: &capiv1beta1.NetworkRanges{
					CIDRBlocks: []string{config.NetworkConfig.ServiceNetworkCIDRBlock},
				},
			},
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
				Kind:       "OCICluster",
				Name:       clusterName,
				Namespace:  capiSystemNamespace,
			},
		}
		return nil
	}

	return cluster, mutateFn
}

func OCIMachineTemplate(capiSystemNamespace, clusterName string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler, config Config) (client.Object, func() error) {
	cpu := config.AutoScalingConfig.CPUs
	memory := config.AutoScalingConfig.Memory
	shape := config.AutoScalingConfig.Shape
	imageID := config.AutoScalingConfig.ImageID
	if instance.Spec.Autoscaling.ShapeConfig.CPUs != 0 {
		cpu = instance.Spec.Autoscaling.ShapeConfig.CPUs
	}
	if instance.Spec.Autoscaling.ShapeConfig.Memory != 0 {
		memory = instance.Spec.Autoscaling.ShapeConfig.Memory
	}
	if instance.Spec.Autoscaling.Shape != "" {
		shape = instance.Spec.Autoscaling.Shape
	}
	if instance.Spec.Autoscaling.ImageID != "" {
		imageID = instance.Spec.Autoscaling.ImageID
	}
	machineTemplate := &infrastructurev1beta2.OCIMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-autoscaling", clusterName),
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		machineTemplate.Spec = infrastructurev1beta2.OCIMachineTemplateSpec{
			Template: infrastructurev1beta2.OCIMachineTemplateResource{
				Spec: infrastructurev1beta2.OCIMachineSpec{
					ImageId: imageID,
					Shape:   shape,
					ShapeConfig: infrastructurev1beta2.ShapeConfig{
						Ocpus:       fmt.Sprintf("%d", cpu),
						MemoryInGBs: fmt.Sprintf("%d", memory), // TODO: check if this is correct
					},
					IsPvEncryptionInTransitEnabled: false,
				},
			},
		}
		return nil
	}

	return machineTemplate, mutateFn
}

func MachineDeployment(capiSystemNamespace, clusterName string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler, config Config) (client.Object, func() error) {
	cpu := config.AutoScalingConfig.CPUs
	memory := config.AutoScalingConfig.Memory
	minNodes := config.AutoScalingConfig.MinNodes
	maxNodes := config.AutoScalingConfig.MaxNodes
	if instance.Spec.Autoscaling.ShapeConfig.CPUs != 0 {
		cpu = instance.Spec.Autoscaling.ShapeConfig.CPUs
	}
	if instance.Spec.Autoscaling.ShapeConfig.Memory != 0 {
		memory = instance.Spec.Autoscaling.ShapeConfig.Memory
	}
	if instance.Spec.Autoscaling.MinNodes != 0 {
		minNodes = instance.Spec.Autoscaling.MinNodes
	}
	if instance.Spec.Autoscaling.MaxNodes != 0 {
		maxNodes = instance.Spec.Autoscaling.MaxNodes
	}

	machineDeployment := &capiv1beta1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		utils.SetDefaultLabels(machineDeployment, instance.Name)
		annotations := map[string]string{
			"capacity.cluster-autoscaler.kubernetes.io/cpu":               fmt.Sprintf("%d", cpu),
			"capacity.cluster-autoscaler.kubernetes.io/memory":            fmt.Sprintf("%dG", memory),
			"cluster.x-k8s.io/cluster-api-autoscaler-node-group-min-size": fmt.Sprintf("%d", minNodes),
			"cluster.x-k8s.io/cluster-api-autoscaler-node-group-max-size": fmt.Sprintf("%d", maxNodes),
		}
		if machineDeployment.Annotations != nil {
			for key, value := range machineDeployment.Annotations {
				annotations[key] = value
			}
		}
		machineDeployment.Annotations = annotations
		machineDeployment.Spec = capiv1beta1.MachineDeploymentSpec{
			ClusterName: clusterName,
			Template: capiv1beta1.MachineTemplateSpec{
				Spec: capiv1beta1.MachineSpec{
					ClusterName: clusterName,
					Bootstrap: capiv1beta1.Bootstrap{
						DataSecretName: swag.String(fmt.Sprintf("%s-bootstrap", clusterName)),
					},
					InfrastructureRef: corev1.ObjectReference{
						APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
						Kind:       "OCIMachineTemplate",
						Name:       fmt.Sprintf("%s-autoscaling", clusterName),
						Namespace:  capiSystemNamespace,
					},
				},
			},
		}
		return nil
	}

	return machineDeployment, mutateFn
}

// ValidateMinMaxNodes validates the min and max nodes values for the autoscaler
func ValidateMinMaxNodes(autoscaler *ocicapioperatorv1alpha1.OCIClusterAutoscaler, config Config) error {
	minNodes := config.AutoScalingConfig.MinNodes
	maxNodes := config.AutoScalingConfig.MaxNodes
	if autoscaler.Spec.Autoscaling.MinNodes != 0 {
		minNodes = autoscaler.Spec.Autoscaling.MinNodes
	}
	if autoscaler.Spec.Autoscaling.MaxNodes != 0 {
		maxNodes = autoscaler.Spec.Autoscaling.MaxNodes
	}

	// ensure that min nodes is less than max nodes
	if minNodes > maxNodes {
		return fmt.Errorf("min nodes must be less than max nodes")
	} else if minNodes < 0 {
		return fmt.Errorf("min nodes must be equal to or greater than 0")
	} else if maxNodes < 0 {
		return fmt.Errorf("max nodes must be equal to or greater than 0")
	}
	return nil
}

// SetNetworkConfig finds the network CIDRs in the cluster if it is not set in the config
// then sets them in the config
func SetNetworkConfig(ctx context.Context, client client.Client, config Config) (Config, error) {
	if config.NetworkConfig.ClusterNetworkCIDRBlock == "" {
		clusterNetworkCIDRBlock, err := utils.GetClusterNetworkCIDRBlock(ctx, client)
		if err != nil {
			return config, fmt.Errorf("failed to get cluster network CIDR block: %w", err)
		}
		config.NetworkConfig.ClusterNetworkCIDRBlock = clusterNetworkCIDRBlock
	}
	if config.NetworkConfig.ServiceNetworkCIDRBlock == "" {
		serviceNetworkCIDRBlock, err := utils.GetServiceNetworkCIDRBlock(ctx, client)
		if err != nil {
			return config, fmt.Errorf("failed to get service network CIDR block: %w", err)
		}
		config.NetworkConfig.ServiceNetworkCIDRBlock = serviceNetworkCIDRBlock
	}
	return config, nil
}
