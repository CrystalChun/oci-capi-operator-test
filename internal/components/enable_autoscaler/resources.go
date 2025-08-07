package enableautoscaler

import (
	"fmt"

	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/go-openapi/swag"

	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewComponent(capiSystemNamespace string, image string, autoscaler *ocicapioperatorv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) *components.Component {
	ociCluster, ociClusterMutateFn := OCICluster(capiSystemNamespace, autoscaler)
	cluster, clusterMutateFn := CAPICluster(capiSystemNamespace, autoscaler)
	machineTemplate, machineTemplateMutateFn := OCIMachineTemplate(capiSystemNamespace, autoscaler)
	machineDeployment, machineDeploymentMutateFn := MachineDeployment(capiSystemNamespace, autoscaler)

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
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": instance.Name,
			},
		},
	}

	mutateFn := func() error {
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
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": instance.Name,
			},
		},
	}

	mutateFn := func() error {
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

func MachineDeployment(capiSystemNamespace string, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	machineDeployment := &capiv1beta1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: capiSystemNamespace,
			Annotations: map[string]string{
				"capacity.cluster-autoscaler.kubernetes.io/cpu":               fmt.Sprintf("%d", instance.Spec.Autoscaling.ShapeConfig.CPUs),
				"capacity.cluster-autoscaler.kubernetes.io/memory":            fmt.Sprintf("%dG", instance.Spec.Autoscaling.ShapeConfig.Memory),
				"cluster.x-k8s.io/cluster-api-autoscaler-node-group-min-size": fmt.Sprintf("%d", instance.Spec.Autoscaling.MinNodes),
				"cluster.x-k8s.io/cluster-api-autoscaler-node-group-max-size": fmt.Sprintf("%d", instance.Spec.Autoscaling.MaxNodes),
			},
		},
	}

	mutateFn := func() error {
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
