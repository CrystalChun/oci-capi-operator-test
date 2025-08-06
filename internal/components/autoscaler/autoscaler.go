package autoscaler

import (
	"fmt"

	"github.com/go-openapi/swag"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	ocicapiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewComponent(capiSystemNamespace string, image string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) *components.Component {
	deploy, deployMutateFn := AutoscalerDeployment(capiSystemNamespace, image, scheme, autoscaler)
	serviceAccount, serviceAccountMutateFn := ServiceAccount(capiSystemNamespace, scheme, autoscaler)
	clusterRole, clusterRoleMutateFn := ClusterRole(capiSystemNamespace, autoscaler)
	clusterRoleBinding, clusterRoleBindingMutateFn := ClusterRoleBinding(capiSystemNamespace, autoscaler)

	ociCluster, ociClusterMutateFn := CAPIOCICluster(capiSystemNamespace, autoscaler)
	cluster, clusterMutateFn := CAPICluster(capiSystemNamespace, autoscaler)
	machineTemplate, machineTemplateMutateFn := OCIMachineTemplate(capiSystemNamespace, autoscaler)
	machineDeployment, machineDeploymentMutateFn := MachineDeployment(capiSystemNamespace, autoscaler)

	return &components.Component{
		Name: "autoscaler",
		Subcomponents: components.SubcomponentList{
			{Name: "deployment", Object: deploy, MutateFn: deployMutateFn},
			{Name: "ociCluster", Object: ociCluster, MutateFn: ociClusterMutateFn},
			{Name: "cluster", Object: cluster, MutateFn: clusterMutateFn},
			{Name: "machineTemplate", Object: machineTemplate, MutateFn: machineTemplateMutateFn},
			{Name: "machineDeployment", Object: machineDeployment, MutateFn: machineDeploymentMutateFn},
			{Name: "serviceAccount", Object: serviceAccount, MutateFn: serviceAccountMutateFn},
			{Name: "clusterRole", Object: clusterRole, MutateFn: clusterRoleMutateFn},
			{Name: "clusterRoleBinding", Object: clusterRoleBinding, MutateFn: clusterRoleBindingMutateFn},
		},
	}
}

func AutoscalerDeployment(capiSystemNamespace string, image string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-cluster-autoscaler",
			Namespace: "oci-cluster-autoscaler",
		},
	}

	mutateFn := func() error {
		// TODO: fill this in
		return controllerutil.SetControllerReference(instance, deploy, scheme)
	}

	return deploy, mutateFn
}

func CAPIOCICluster(capiSystemNamespace string, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) { // Create OCICluster
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

func CAPICluster(capiSystemNamespace string, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
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

func OCIMachineTemplate(capiSystemNamespace string, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	// Create OCIMachineTemplate
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

func MachineDeployment(capiSystemNamespace string, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	// Create MachineDeployment
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

func ServiceAccount(capiSystemNamespace string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-cluster-autoscaler",
			Namespace: capiSystemNamespace,
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(instance, serviceAccount, scheme)
	}

	return serviceAccount, mutateFn
}

func ClusterRole(capiSystemNamespace string, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}

	mutateFn := func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"infrastructure.cluster.x-k8s.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
		}
		return nil
	}

	return clusterRole, mutateFn
}

func ClusterRoleBinding(capiSystemNamespace string, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}

	mutateFn := func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "oci-cluster-autoscaler-extra",
		}
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "oci-cluster-autoscaler",
				Namespace: capiSystemNamespace,
			},
		}
		return nil
	}

	return clusterRoleBinding, mutateFn
}
