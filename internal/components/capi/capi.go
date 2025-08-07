package capi

import (
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"

	"github.com/go-openapi/swag"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewComponent returns a Component for the CAPI controller manager
func NewComponent(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) *components.Component {
	namespace, namespaceMutateFn := CAPINamespace(capiSystemNamespace, autoscaler, scheme)
	//caBundleConfigMap, caBundleConfigMapMutateFn := CABundleConfigMap(capiSystemNamespace, autoscaler, scheme)

	scc, sccMutateFn := SecurityContextConstraints(capiSystemNamespace, scheme, autoscaler)
	serviceAccount, serviceAccountMutateFn := ServiceAccount(capiSystemNamespace, autoscaler, scheme)

	deploy, deployMutateFn := CAPIDeployment(capiSystemNamespace, autoscaler, scheme)

	validatingWebhook, validatingWebhookMutateFn := ValidatingWebhookConfiguration(capiSystemNamespace, autoscaler, scheme)
	mutatingWebhook, mutatingWebhookMutateFn := MutatingWebhookConfiguration(capiSystemNamespace, autoscaler, scheme)
	admissionWebhookService, admissionWebhookServiceMutateFn := AdmissionWebhookService(capiSystemNamespace, autoscaler, scheme)

	role, roleMutateFn := Role(capiSystemNamespace, autoscaler, scheme)
	roleBinding, roleBindingMutateFn := RoleBinding(capiSystemNamespace, autoscaler, scheme)
	clusterRole, clusterRoleMutateFn := ClusterRole(capiSystemNamespace, autoscaler, scheme)
	clusterRoleBinding, clusterRoleBindingMutateFn := ClusterRoleBinding(capiSystemNamespace, autoscaler, scheme)

	return &components.Component{
		Name: "capi",
		Subcomponents: components.SubcomponentList{
			{Name: "namespace", Object: namespace, MutateFn: namespaceMutateFn},
			{Name: "scc", Object: scc, MutateFn: sccMutateFn},
			{Name: "serviceAccount", Object: serviceAccount, MutateFn: serviceAccountMutateFn},
			{Name: "deployment", Object: deploy, MutateFn: deployMutateFn},
			{Name: "validatingWebhook", Object: validatingWebhook, MutateFn: validatingWebhookMutateFn},
			{Name: "mutatingWebhook", Object: mutatingWebhook, MutateFn: mutatingWebhookMutateFn},
			{Name: "admissionWebhookService", Object: admissionWebhookService, MutateFn: admissionWebhookServiceMutateFn},
			{Name: "role", Object: role, MutateFn: roleMutateFn},
			{Name: "roleBinding", Object: roleBinding, MutateFn: roleBindingMutateFn},
			{Name: "clusterRole", Object: clusterRole, MutateFn: clusterRoleMutateFn},
			{Name: "clusterRoleBinding", Object: clusterRoleBinding, MutateFn: clusterRoleBindingMutateFn},
		},
	}
}

func CAPINamespace(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: capiSystemNamespace,
		},
	}
	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, namespace, scheme)
	}
	return namespace, mutateFn
}

func CAPIDeployment(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
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
	}
	mutateFn := func() error {
		deploy.Spec = appsv1.DeploymentSpec{
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
					DNSPolicy:          corev1.DNSClusterFirst,
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

					Volumes: []corev1.Volume{ //TODO: check if this is correct within openshift
						{
							Name: "cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "capi-webhook-service-cert",
								},
							},
						},
					},
				},
			},
		}
		return controllerutil.SetControllerReference(autoscaler, deploy, scheme)
	}
	return deploy, mutateFn
}

func AdmissionWebhookService(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capi-webhook-service",
			Namespace: capiSystemNamespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/provider": "cluster-api",
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": "capi-webhook-service-cert",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"cluster.x-k8s.io/provider": "cluster-api",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 443,
					TargetPort: intstr.IntOrString{
						StrVal: "webhook-server",
					},
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type:            corev1.ServiceTypeClusterIP,
			SessionAffinity: corev1.ServiceAffinityNone,
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, service, scheme)
	}

	return service, mutateFn
}
