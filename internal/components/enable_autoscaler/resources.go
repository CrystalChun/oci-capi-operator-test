package enableautoscaler

import (
	"context"
	"fmt"

	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"
	"github.com/openshift/oci-capi-operator/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/go-openapi/swag"

	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AutoScalingConfig struct {
	CPUs     int32  `envconfig:"CPUs" default:"2"`
	Memory   int32  `envconfig:"MEMORY" default:"4"`
	MinNodes int32  `envconfig:"MIN_NODES" default:"1"`
	MaxNodes int32  `envconfig:"MAX_NODES" default:"3"`
	Shape    string `envconfig:"SHAPE" default:"oc3"`
}

func NewComponent(capiSystemNamespace string, image string, autoscaler *ocicapioperatorv1alpha1.OCIClusterAutoscaler, autoscalerConfig AutoScalingConfig, scheme *runtime.Scheme) *components.Component {
	ociCluster, ociClusterMutateFn := OCICluster(capiSystemNamespace, autoscaler)
	cluster, clusterMutateFn := CAPICluster(capiSystemNamespace, autoscaler)
	machineTemplate, machineTemplateMutateFn := OCIMachineTemplate(capiSystemNamespace, autoscaler)
	machineDeployment, machineDeploymentMutateFn := MachineDeployment(capiSystemNamespace, autoscaler, autoscalerConfig)

	return &components.Component{
		Name: "EnableAutoscaler",
		Subcomponents: components.SubcomponentList{
			{Name: "machineTemplate", Object: machineTemplate, MutateFn: machineTemplateMutateFn},
			{Name: "machineDeployment", Object: machineDeployment, MutateFn: machineDeploymentMutateFn},
			{Name: "ociCluster", Object: ociCluster, MutateFn: ociClusterMutateFn},
			{Name: "cluster", Object: cluster, MutateFn: clusterMutateFn},
		},
	}
}

func OCICluster(capiSystemNamespace string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) { // Create OCICluster
	ociCluster := &infrastructurev1beta2.OCICluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		utils.SetDefaultLabels(ociCluster, instance.Name)
		ociCluster.Spec = infrastructurev1beta2.OCIClusterSpec{
			CompartmentId: instance.Spec.OCI.CompartmentID,
			NetworkSpec: infrastructurev1beta2.NetworkSpec{
				SkipNetworkManagement: true,
				Vcn: infrastructurev1beta2.VCN{
					ID: swag.String(instance.Spec.OCI.Network.VCNID),
					Subnets: []*infrastructurev1beta2.Subnet{
						{
							ID:   swag.String(instance.Spec.OCI.Network.SubnetID),
							Name: "private",
							Role: "worker",
						},
					},
					NetworkSecurityGroup: infrastructurev1beta2.NetworkSecurityGroup{
						List: []*infrastructurev1beta2.NSG{
							{
								ID:   swag.String(instance.Spec.OCI.Network.NetworkSecurityGroupID),
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

func CAPICluster(capiSystemNamespace string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	cluster := &capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		utils.SetDefaultLabels(cluster, instance.Name)
		cluster.Spec = capiv1beta1.ClusterSpec{
			ClusterNetwork: &capiv1beta1.ClusterNetwork{
				Pods: &capiv1beta1.NetworkRanges{
					CIDRBlocks: []string{"10.128.0.0/14"},
				},
				ServiceDomain: "cluster.local",
				Services: &capiv1beta1.NetworkRanges{
					CIDRBlocks: []string{"172.30.0.0/16"},
				},
			},
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
				Kind:       "OCICluster",
				Name:       instance.Name,
				Namespace:  capiSystemNamespace,
			},
		}
		return nil
	}

	return cluster, mutateFn
}

func OCIMachineTemplate(capiSystemNamespace string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	machineTemplate := &infrastructurev1beta2.OCIMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-autoscaling", instance.Name),
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		machineTemplate.Spec = infrastructurev1beta2.OCIMachineTemplateSpec{
			Template: infrastructurev1beta2.OCIMachineTemplateResource{
				Spec: infrastructurev1beta2.OCIMachineSpec{
					ImageId: instance.Spec.OCI.ImageID,
					Shape:   instance.Spec.Autoscaling.Shape,
					ShapeConfig: infrastructurev1beta2.ShapeConfig{
						Ocpus:       fmt.Sprintf("%d", instance.Spec.Autoscaling.ShapeConfig.CPUs),
						MemoryInGBs: fmt.Sprintf("%d", instance.Spec.Autoscaling.ShapeConfig.Memory), // TODO: check if this is correct
					},
					IsPvEncryptionInTransitEnabled: false,
				},
			},
		}
		return nil
	}

	return machineTemplate, mutateFn
}

func MachineDeployment(capiSystemNamespace string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler, autoscalerConfig AutoScalingConfig) (client.Object, func() error) {
	cpu := autoscalerConfig.CPUs
	memory := autoscalerConfig.Memory
	minNodes := autoscalerConfig.MinNodes
	maxNodes := autoscalerConfig.MaxNodes
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

	// ensure that min nodes is less than max nodes
	if minNodes > maxNodes {
		return nil, func() error {
			return fmt.Errorf("min nodes must be less than max nodes")
		}
	}

	machineDeployment := &capiv1beta1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
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
			ClusterName: instance.Name,
			Template: capiv1beta1.MachineTemplateSpec{
				Spec: capiv1beta1.MachineSpec{
					ClusterName: instance.Name,
					Bootstrap: capiv1beta1.Bootstrap{
						DataSecretName: swag.String(fmt.Sprintf("%s-bootstrap", instance.Name)),
					},
					InfrastructureRef: corev1.ObjectReference{
						APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
						Kind:       "OCIMachineTemplate",
						Name:       fmt.Sprintf("%s-autoscaling", instance.Name),
						Namespace:  capiSystemNamespace,
					},
				},
			},
		}
		return nil
	}

	return machineDeployment, mutateFn
}

// BootstrapConfigSecret is a secret that contains the bootstrap config for additional nodes that are added to the cluster.

func BootstrapConfigSecret(ctx context.Context, client client.Client, capiSystemNamespace string, clusterName string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	bootstrapConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-bootstrap", clusterName),
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		utils.SetDefaultLabels(bootstrapConfigSecret, clusterName)
		ignitionConfig, err := utils.GenerateIgnitionConfig(ctx, client)
		if err != nil {
			return fmt.Errorf("failed to generate ignition config: %w", err)
		}
		bootstrapConfigSecret.Data = map[string][]byte{ // TODO: confirm what this secret looks likeand the key is
			"bootstrap.ign": []byte(ignitionConfig),
		}
		return nil
	}

	return bootstrapConfigSecret, mutateFn
}

func KubeConfigSecret(capiSystemNamespace string, clusterName string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	kubeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kubeconfig", clusterName),
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		utils.SetDefaultLabels(kubeConfigSecret, clusterName)
		// This needs to contain the kubeconfig for the cluster.
		return nil
	}

	return kubeConfigSecret, mutateFn
}
