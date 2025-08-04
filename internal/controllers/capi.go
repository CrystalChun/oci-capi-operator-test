package controllers

import (
	"context"
	"fmt"

	ocicapiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-openapi/swag"
)

// createOrUpdateCAPIResources creates or updates all CAPI-related resources
func (r *OCIClusterAutoscalerReconciler) activateAutoscalerResources(ctx context.Context, instance *ocicapiv1alpha1.OCIClusterAutoscaler) error {
	err := r.createCAPIOCICluster(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create CAPI cluster: %w", err)
	}

	err = r.createCAPICluster(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create CAPI cluster: %w", err)
	}

	// Create OCIMachineTemplate
	err = r.createOCIMachineTemplate(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create OCIMachineTemplate: %w", err)
	}

	err = r.createMachineDeployment(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create MachineDeployment: %w", err)
	}

	return nil
}

func (r *OCIClusterAutoscalerReconciler) createCAPIOCICluster(ctx context.Context, instance *ocicapiv1alpha1.OCIClusterAutoscaler) error { // Create OCICluster
	ociCluster := &infrastructurev1beta2.OCICluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: capiSystemNamespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": instance.Name,
			},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ociCluster, func() error {
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
	})
	if err != nil {
		return fmt.Errorf("failed to create/update OCICluster: %w", err)
	}
	return nil
}

func (r *OCIClusterAutoscalerReconciler) createCAPICluster(ctx context.Context, instance *ocicapiv1alpha1.OCIClusterAutoscaler) error {
	// Create Cluster
	cluster := &capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: capiSystemNamespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": instance.Name,
			},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cluster, func() error {
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
	})
	if err != nil {
		return fmt.Errorf("failed to create/update Cluster: %w", err)
	}
	return nil
}

func (r *OCIClusterAutoscalerReconciler) createOCIMachineTemplate(ctx context.Context, instance *ocicapiv1alpha1.OCIClusterAutoscaler) error {
	// Create OCIMachineTemplate
	machineTemplate := &infrastructurev1beta2.OCIMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-autoscaling", instance.Name),
			Namespace: capiSystemNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, machineTemplate, func() error {
		machineTemplate.Spec = infrastructurev1beta2.OCIMachineTemplateSpec{
			Template: infrastructurev1beta2.OCIMachineTemplateResource{
				Spec: infrastructurev1beta2.OCIMachineSpec{
					ImageId: instance.Spec.OCI.ImageID,
					Shape:   instance.Spec.Autoscaling.Shape,
					ShapeConfig: infrastructurev1beta2.ShapeConfig{
						Ocpus:       fmt.Sprintf("%d", instance.Spec.Autoscaling.ShapeConfig.CPUs),
						MemoryInGBs: fmt.Sprintf("%d", instance.Spec.Autoscaling.ShapeConfig.MemoryInGBs), // TODO: check if this is correct
					},
					IsPvEncryptionInTransitEnabled: false,
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create/update OCIMachineTemplate: %w", err)
	}
	return nil
}

func (r *OCIClusterAutoscalerReconciler) createMachineDeployment(ctx context.Context, instance *ocicapiv1alpha1.OCIClusterAutoscaler) error {
	// Create MachineDeployment
	machineDeployment := &capiv1beta1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: capiSystemNamespace,
			Annotations: map[string]string{
				"capacity.cluster-autoscaler.kubernetes.io/cpu":               fmt.Sprintf("%d", instance.Spec.Autoscaling.ShapeConfig.CPUs),
				"capacity.cluster-autoscaler.kubernetes.io/memory":            fmt.Sprintf("%dG", instance.Spec.Autoscaling.ShapeConfig.MemoryInGBs),
				"cluster.x-k8s.io/cluster-api-autoscaler-node-group-min-size": fmt.Sprintf("%d", instance.Spec.Autoscaling.MinSize),
				"cluster.x-k8s.io/cluster-api-autoscaler-node-group-max-size": fmt.Sprintf("%d", instance.Spec.Autoscaling.MaxSize),
			},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, machineDeployment, func() error {
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
	})
	if err != nil {
		return fmt.Errorf("failed to create/update MachineDeployment: %w", err)
	}
	return nil
}
