package capoci

import (
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	ocicapiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"

	"github.com/go-openapi/swag"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewComponent returns a Component for the CAPOCI controller manager
func NewComponent(capociNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) *components.Component {
	namespace, namespaceMutateFn := Namespace(capociNamespace, autoscaler, scheme)
	serviceAccount, serviceAccountMutateFn := ServiceAccount(capociNamespace, scheme, autoscaler)

	deploy, deployMutateFn := CAPOCIDeployment(capociNamespace, scheme, autoscaler)
	service, serviceMutateFn := WebhookService(capociNamespace, scheme, autoscaler)

	mutatingWebhookConfiguration, mutatingWebhookConfigurationMutateFn := MutatingWebhookConfiguration(capociNamespace, scheme, autoscaler)
	validatingWebhookConfiguration, validatingWebhookConfigurationMutateFn := ValidatingWebhookConfiguration(capociNamespace, scheme, autoscaler)

	role, roleMutateFn := Role(capociNamespace, scheme, autoscaler)
	roleBinding, roleBindingMutateFn := RoleBinding(capociNamespace, scheme, autoscaler)
	return &components.Component{
		Name: "CAPOCI",
		Subcomponents: components.SubcomponentList{
			{Name: "namespace", Object: namespace, MutateFn: namespaceMutateFn},
			{Name: "serviceAccount", Object: serviceAccount, MutateFn: serviceAccountMutateFn},
			{Name: "deployment", Object: deploy, MutateFn: deployMutateFn},
			{Name: "service", Object: service, MutateFn: serviceMutateFn},
			{Name: "mutatingWebhookConfiguration", Object: mutatingWebhookConfiguration, MutateFn: mutatingWebhookConfigurationMutateFn},
			{Name: "validatingWebhookConfiguration", Object: validatingWebhookConfiguration, MutateFn: validatingWebhookConfigurationMutateFn},
			{Name: "role", Object: role, MutateFn: roleMutateFn},
			{Name: "roleBinding", Object: roleBinding, MutateFn: roleBindingMutateFn},
		},
	}
}

func CAPOCIDeployment(capociNamespace string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capoci-controller-manager",
			Namespace: capociNamespace,
			Labels: map[string]string{
				"cluster-x-k8s.io/provider":   "infrastructure-oci",
				"clusterctl.cluster.x-k8s.io": "",
				"control-plane":               "controller-manager",
			},
		},
	}

	mutateFn := func() error {
		deploy.Spec = appsv1.DeploymentSpec{
			ProgressDeadlineSeconds: swag.Int32(600), //default is 600
			Replicas:                swag.Int32(1),   //default is 1
			RevisionHistoryLimit:    swag.Int32(10),  //default is 10
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster-x-k8s.io/provider": "infrastructure-oci",
					"control-plane":             "controller-manager",
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType, //default is RollingUpdate
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{ //default is 25%
						Type:   intstr.String,
						StrVal: "25%",
					},
					MaxSurge: &intstr.IntOrString{ //default is 25%
						Type:   intstr.String,
						StrVal: "25%",
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"cluster-x-k8s.io/provider": "infrastructure-oci",
						"control-plane":             "controller-manager",
					},
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
								{
									Weight: 10,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/control-plane",
												Operator: corev1.NodeSelectorOpExists,
											},
										},
									},
								},
								{
									Weight: 10,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/master",
												Operator: corev1.NodeSelectorOpExists,
											},
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "manager",
							Image:           "ghcr.io/oracle/cluster-api-oci-controller:v0.20.2",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/manager",
							},
							Args: []string{
								"--leader-elect",
								"--feature-gates=MachinePool=true",
								"--metrics-bind-address=127.0.0.1:8080",
								"--logging-format=text",
								"--init-oci-clients-on-startup=true",
								"--enable-instance-metadata-service-lookup=false",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "AUTH_CONFIG_DIR",
									Value: "/etc/oci",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "webhook-server",
									ContainerPort: 9443,
									Protocol:      corev1.ProtocolTCP, //default is TCP
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(8081),
										Scheme: corev1.URISchemeHTTP, //default is HTTP
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       20,
								TimeoutSeconds:      1, //default is 1
								SuccessThreshold:    1, //default is 1
								FailureThreshold:    3, //default is 3
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(8081),
										Scheme: corev1.URISchemeHTTP, //default is HTTP
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10, //default is 10
								TimeoutSeconds:      1,  //default is 1
								SuccessThreshold:    1,  //default is 1
								FailureThreshold:    3,  //default is 3
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: swag.Bool(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								Privileged: swag.Bool(false), //default is false
								RunAsGroup: swag.Int64(65532),
								RunAsUser:  swag.Int64(65532),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "auth-config-dir",
									MountPath: "/etc/oci",
									ReadOnly:  true, //default is false
								},
								{
									Name:      "cert",
									MountPath: "/tmp/k8s-webhook-server/serving-certs", //TODO: update this with ocp cert
									ReadOnly:  true,                                    //default is false
								},
							},
						},
					},
					DNSPolicy:     corev1.DNSClusterFirst,
					RestartPolicy: corev1.RestartPolicyAlways, //default is Always
					SchedulerName: "default-scheduler",        //default is default-scheduler
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: swag.Bool(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					ServiceAccountName:            "capoci-controller-manager",
					TerminationGracePeriodSeconds: swag.Int64(30), //default is 30
					Tolerations: []corev1.Toleration{
						{
							Effect: corev1.TaintEffectNoSchedule,
							Key:    "node-role.kubernetes.io/control-plane",
						},
						{
							Effect: corev1.TaintEffectNoSchedule,
							Key:    "node-role.kubernetes.io/master",
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "auth-config-dir",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  "capoci-auth-config",
									DefaultMode: swag.Int32(420),
								},
							},
						},
						{
							Name: "cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "capoci-webhook-service-cert",
								},
							},
						},
					},
				},
			},
		}

		return controllerutil.SetControllerReference(instance, deploy, scheme)
	}

	return deploy, mutateFn
}

func Namespace(capociNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: capociNamespace,
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, namespace, scheme)
	}

	return namespace, mutateFn
}

func ServiceAccount(capociNamespace string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capoci-controller-manager",
			Namespace: capociNamespace,
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(instance, sa, scheme)
	}

	return sa, mutateFn
}

func OCICredentialsSecret(namespace string, privateKey []byte, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	// Create or update secret with OCI credentials for CAPOCI
	secretName := "oci-credentials"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}

	mutateFn := func() error {
		secret.Type = corev1.SecretTypeOpaque
		secret.Data = map[string][]byte{
			"tenancy":     []byte(autoscaler.Spec.OCI.TenancyID),
			"user":        []byte(autoscaler.Spec.OCI.UserID),
			"region":      []byte(autoscaler.Spec.OCI.Region),
			"fingerprint": []byte(autoscaler.Spec.OCI.Fingerprint),
			"key":         privateKey,
		}
		return controllerutil.SetControllerReference(autoscaler, secret, scheme)
	}

	return secret, mutateFn
}

func WebhookService(capociNamespace string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capoci-webhook-service",
			Namespace: capociNamespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/provider": "infrastructure-oci",
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": "capoci-webhook-service-cert",
			},
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(instance, service, scheme)
	}

	return service, mutateFn
}

func MutatingWebhookConfiguration(capociNamespace string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	mutatingWebhookConfiguration := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capoci-mutating-webhook-configuration",
			Namespace: capociNamespace,
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(instance, mutatingWebhookConfiguration, scheme)
	}

	return mutatingWebhookConfiguration, mutateFn
}

func ValidatingWebhookConfiguration(capociNamespace string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	validatingWebhookConfiguration := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capoci-validating-webhook-configuration",
			Namespace: capociNamespace,
		},
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(instance, validatingWebhookConfiguration, scheme)
	}

	return validatingWebhookConfiguration, mutateFn
}

// TODO: capoci-validating-webhook-configuration and mutating, service for webhook
// TODO: clusterrole, clusterrolebinding
// TODO: configmap Creating ConfigMap="capoci-manager-config" Namespace="cluster-api-provider-oci-system"
