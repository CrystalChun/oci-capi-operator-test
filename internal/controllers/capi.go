package controllers

import (
	"context"
	"fmt"

	ocicapiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-openapi/swag"
)

func (r *OCIClusterAutoscalerReconciler) deployCAPI(ctx context.Context, instance *ocicapiv1alpha1.OCIClusterAutoscaler) error {
	err := r.createCAPIDeployment(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create CAPI deployment: %w", err)
	}
	return nil
}

func (r *OCIClusterAutoscalerReconciler) createCAPIDeployment(ctx context.Context, instance *ocicapiv1alpha1.OCIClusterAutoscaler) error {
	// Taken directly from the CAPI deployed after running `clusterctl init --bootstrap - --control-plane - --infrastructure oci``
	CAPIManagerImage := "registry.k8s.io/cluster-api/cluster-api-controller:v1.10.4"
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capi-controller-manager",
			Namespace: capiSystemNamespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/provider":   "cluster-api",
				"clusterctl.cluster.x-k8s.io": "",
				"control-plane":               "controller-manager",
			},
		},
		Spec: appsv1.DeploymentSpec{
			RevisionHistoryLimit:    swag.Int32(10),
			ProgressDeadlineSeconds: swag.Int32(600),
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{
						StrVal: "25%",
						Type:   intstr.String,
					},
					MaxSurge: &intstr.IntOrString{
						StrVal: "25%",
						Type:   intstr.String,
					},
				},
			},
			Replicas: swag.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster.x-k8s.io/provider": "cluster-api",
					"control-plane":             "controller-manager",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"cluster.x-k8s.io/provider": "cluster-api",
						"control-plane":             "controller-manager",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "capi-manager",
					//ServiceAccount:     "capi-manager",
					DNSPolicy: corev1.DNSClusterFirst,
					DNSConfig: &corev1.PodDNSConfig{
						Options: []corev1.PodDNSConfigOption{
							{Name: "ndots", Value: swag.String("1")},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
					SchedulerName: "default-scheduler",
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: swag.Bool(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "manager",
							Image: CAPIManagerImage,
							Args: []string{
								"--leader-elect",
								"--diagnostics-address=:8443",
								"--insecure-diagnostics=false",
								"--feature-gates=MachinePool=true,ClusterResourceSet=true,ClusterTopology=false,RuntimeSDK=false,MachineSetPreflightChecks=true,MachineWaitForVolumeDetachConsiderVolumeAttachments=true,PriorityQueue=false",
							},
							Command: []string{
								"/manager",
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9443,
									Name:          "webhook-server",
									Protocol:      corev1.ProtocolTCP,
								},
								{
									ContainerPort: 9440,
									Name:          "healthz",
									Protocol:      corev1.ProtocolTCP,
								},
								{
									ContainerPort: 8443,
									Name:          "metrics",
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "POD_UID",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.IntOrString{StrVal: "healthz"},
										Scheme: corev1.URISchemeHTTP,
									},
								},
								PeriodSeconds:    10,
								TimeoutSeconds:   1,
								SuccessThreshold: 1,
								FailureThreshold: 3,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.IntOrString{StrVal: "healthz"},
										Scheme: corev1.URISchemeHTTP,
									},
								},
								PeriodSeconds:    10,
								TimeoutSeconds:   1,
								SuccessThreshold: 1,
								FailureThreshold: 3,
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: swag.Bool(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								Privileged: swag.Bool(false),
								RunAsGroup: swag.Int64(65532),
								RunAsUser:  swag.Int64(65532),
							},
							VolumeMounts: []corev1.VolumeMount{ //TODO: check if this is correct within openshift
								{
									Name:      "cert",
									MountPath: "/tmp/k8s-webhook-server/serving-certs",
									ReadOnly:  true,
								},
							},
						},
					},
					TerminationGracePeriodSeconds: swag.Int64(30),
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
						{
							Key:    "node-role.kubernetes.io/control-plane",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},

					Volumes: []corev1.Volume{
						{
							Name: "cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  "capi-webhook-service-cert",
									DefaultMode: swag.Int32(420),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create/update CAPIDeployment: %w", err)
	}
	return nil
}

// activateAutoscalerResources creates or updates all CAPI-related custom resources for the autoscaler
// This includes:
// - CAPI OCICluster
// - CAPI Cluster
// - CAPI OCIMachineTemplate
// - CAPI MachineDeployment
func (r *OCIClusterAutoscalerReconciler) activateAutoscalerResources(ctx context.Context, instance *ocicapiv1alpha1.OCIClusterAutoscaler) error {
	// Create OCICluster
	err := r.createCAPIOCICluster(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create CAPI cluster: %w", err)
	}

	// Create Cluster
	err = r.createCAPICluster(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create CAPI cluster: %w", err)
	}

	// Create OCIMachineTemplate
	err = r.createOCIMachineTemplate(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create OCIMachineTemplate: %w", err)
	}

	// Create MachineDeployment
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
						MemoryInGBs: fmt.Sprintf("%d", instance.Spec.Autoscaling.ShapeConfig.Memory), // TODO: check if this is correct
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
				"capacity.cluster-autoscaler.kubernetes.io/memory":            fmt.Sprintf("%dG", instance.Spec.Autoscaling.ShapeConfig.Memory),
				"cluster.x-k8s.io/cluster-api-autoscaler-node-group-min-size": fmt.Sprintf("%d", instance.Spec.Autoscaling.MinNodes),
				"cluster.x-k8s.io/cluster-api-autoscaler-node-group-max-size": fmt.Sprintf("%d", instance.Spec.Autoscaling.MaxNodes),
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
