package utils

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("KubeAPI Utils", func() {
	var (
		ctx           context.Context
		fakeClient    client.Client
		scheme        *runtime.Scheme
		testSecret    *corev1.Secret
		testConfigMap *corev1.ConfigMap
		testNetwork   *configv1.Network
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = configv1.AddToScheme(scheme)
		_ = appsv1.AddToScheme(scheme)

		testSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"test-key": []byte("test-value"),
				"tls.crt":  []byte("test-cert"),
			},
		}

		testConfigMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-root-ca.crt",
				Namespace: "kube-system",
			},
			Data: map[string]string{
				"ca.crt": "test-ca-cert",
			},
		}

		testNetwork = &configv1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: configv1.NetworkSpec{
				ClusterNetwork: []configv1.ClusterNetworkEntry{
					{CIDR: "10.128.0.0/14"},
				},
				ServiceNetwork: []string{"172.30.0.0/16"},
			},
		}

		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(testSecret, testConfigMap, testNetwork).
			Build()
	})

	Describe("GetSecret", func() {
		It("should successfully retrieve an existing secret", func() {
			secret, err := GetSecret(ctx, fakeClient, "test-secret", "test-namespace")
			Expect(err).NotTo(HaveOccurred())
			Expect(secret).NotTo(BeNil())
			Expect(secret.Name).To(Equal("test-secret"))
			Expect(secret.Namespace).To(Equal("test-namespace"))
		})

		It("should return error for non-existent secret", func() {
			secret, err := GetSecret(ctx, fakeClient, "non-existent", "test-namespace")
			Expect(err).To(HaveOccurred())
			Expect(secret).To(BeNil())
		})
	})

	Describe("GetSecretData", func() {
		It("should successfully retrieve secret data", func() {
			data, err := GetSecretData(ctx, fakeClient, "test-secret", "test-namespace", "test-key")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal("test-value"))
		})

		It("should return error for non-existent secret", func() {
			data, err := GetSecretData(ctx, fakeClient, "non-existent", "test-namespace", "test-key")
			Expect(err).To(HaveOccurred())
			Expect(data).To(BeNil())
		})

		It("should return error for non-existent key", func() {
			data, err := GetSecretData(ctx, fakeClient, "test-secret", "test-namespace", "non-existent-key")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key non-existent-key not found"))
			Expect(data).To(BeNil())
		})
	})

	Describe("GetDeploymentCondition", func() {
		var conditions []appsv1.DeploymentCondition

		BeforeEach(func() {
			conditions = []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   appsv1.DeploymentProgressing,
					Status: corev1.ConditionFalse,
				},
			}
		})

		It("should return existing condition", func() {
			condition := GetDeploymentCondition(conditions, appsv1.DeploymentAvailable)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Type).To(Equal(appsv1.DeploymentAvailable))
			Expect(condition.Status).To(Equal(corev1.ConditionTrue))
		})

		It("should return nil for non-existent condition", func() {
			condition := GetDeploymentCondition(conditions, appsv1.DeploymentReplicaFailure)
			Expect(condition).To(BeNil())
		})
	})

	Describe("EditDeploymentCerts", func() {
		var (
			deployment *appsv1.Deployment
			obj        *unstructured.Unstructured
		)

		BeforeEach(func() {
			deployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{
									Name: "cert",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "old-secret",
										},
									},
								},
								{
									Name: "other-volume",
									VolumeSource: corev1.VolumeSource{
										EmptyDir: &corev1.EmptyDirVolumeSource{},
									},
								},
							},
						},
					},
				},
			}

			var err error
			obj = &unstructured.Unstructured{}
			err = scheme.Convert(deployment, obj, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should successfully update cert volume", func() {
			err := EditDeploymentCerts(scheme, obj, "new-secret")
			Expect(err).NotTo(HaveOccurred())

			// Convert back to verify changes
			updatedDeployment := &appsv1.Deployment{}
			err = scheme.Convert(obj, updatedDeployment, nil)
			Expect(err).NotTo(HaveOccurred())

			// Check that cert volume was updated
			certVolume := updatedDeployment.Spec.Template.Spec.Volumes[0]
			Expect(certVolume.Name).To(Equal("cert"))
			Expect(certVolume.Secret.SecretName).To(Equal("new-secret"))

			// Check that other volume was not affected
			otherVolume := updatedDeployment.Spec.Template.Spec.Volumes[1]
			Expect(otherVolume.Name).To(Equal("other-volume"))
			Expect(otherVolume.EmptyDir).NotTo(BeNil())
		})

		It("should return error for nil object", func() {
			err := EditDeploymentCerts(scheme, nil, "new-secret")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unstructured object is nil"))
		})
	})

	Describe("GetDefaultLabels", func() {
		It("should return correct default labels", func() {
			labels := GetDefaultLabels("test-instance")
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue(ManagedByLabel, "test-instance"))
		})
	})

	Describe("SetDefaultLabels", func() {
		var testObj *corev1.ConfigMap

		BeforeEach(func() {
			testObj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"existing-label": "existing-value",
					},
				},
			}
		})

		It("should set default labels and preserve existing ones", func() {
			err := SetDefaultLabels(testObj, "test-instance")
			Expect(err).NotTo(HaveOccurred())

			labels := testObj.GetLabels()
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue(ManagedByLabel, "test-instance"))
			Expect(labels).To(HaveKeyWithValue("existing-label", "existing-value"))
		})

		It("should return error for nil object", func() {
			err := SetDefaultLabels(nil, "test-instance")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("object is nil"))
		})
	})

	Describe("SetOpenshiftServiceCertAnnotation", func() {
		var obj *unstructured.Unstructured

		BeforeEach(func() {
			obj = &unstructured.Unstructured{}
			obj.SetName("test-service")
		})

		It("should set service cert annotation for matching name", func() {
			err := SetOpenshiftServiceCertAnnotation(obj, "test-service")
			Expect(err).NotTo(HaveOccurred())

			annotations := obj.GetAnnotations()
			Expect(annotations).To(HaveKeyWithValue(OpenshiftServiceCertAnnotation, "test-service"))
			Expect(annotations).NotTo(HaveKey(CertManagerCAInjectAnnotation))
		})

		It("should not set annotation for different name", func() {
			err := SetOpenshiftServiceCertAnnotation(obj, "different-service")
			Expect(err).NotTo(HaveOccurred())

			annotations := obj.GetAnnotations()
			Expect(annotations).NotTo(HaveKey(OpenshiftServiceCertAnnotation))
		})

		It("should return error for nil object", func() {
			err := SetOpenshiftServiceCertAnnotation(nil, "test-service")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unstructured object is nil"))
		})
	})

	Describe("SetOpenshiftCABundleAnnotation", func() {
		var obj *unstructured.Unstructured

		BeforeEach(func() {
			obj = &unstructured.Unstructured{}
			obj.SetAnnotations(map[string]string{
				CertManagerCAInjectAnnotation: "should-be-removed",
				"existing-annotation":         "should-be-kept",
			})
		})

		It("should set CA bundle annotation and remove cert-manager annotation", func() {
			err := SetOpenshiftCABundleAnnotation(obj)
			Expect(err).NotTo(HaveOccurred())

			annotations := obj.GetAnnotations()
			Expect(annotations).To(HaveKeyWithValue(OpenshiftCABundleAnnotation, "true"))
			Expect(annotations).NotTo(HaveKey(CertManagerCAInjectAnnotation))
			Expect(annotations).To(HaveKeyWithValue("existing-annotation", "should-be-kept"))
		})

		It("should return error for nil object", func() {
			err := SetOpenshiftCABundleAnnotation(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unstructured object is nil"))
		})
	})

	Describe("GetKubeconfigCA", func() {
		It("should successfully retrieve kubeconfig CA", func() {
			ca, err := GetKubeconfigCA(ctx, fakeClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(ca).To(Equal("test-ca-cert"))
		})

		It("should return error when configmap doesn't exist", func() {
			emptyClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			ca, err := GetKubeconfigCA(ctx, emptyClient)
			Expect(err).To(HaveOccurred())
			Expect(ca).To(BeEmpty())
		})
	})

	Describe("GetClusterNetworkCIDRBlock", func() {
		It("should successfully retrieve cluster network CIDR", func() {
			cidr, err := GetClusterNetworkCIDRBlock(ctx, fakeClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(cidr).To(Equal("10.128.0.0/14"))
		})

		It("should return error when network config doesn't exist", func() {
			emptyClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			cidr, err := GetClusterNetworkCIDRBlock(ctx, emptyClient)
			Expect(err).To(HaveOccurred())
			Expect(cidr).To(BeEmpty())
		})
	})

	Describe("GetServiceNetworkCIDRBlock", func() {
		It("should successfully retrieve service network CIDR", func() {
			cidr, err := GetServiceNetworkCIDRBlock(ctx, fakeClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(cidr).To(Equal("172.30.0.0/16"))
		})

		It("should return error when network config doesn't exist", func() {
			emptyClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			cidr, err := GetServiceNetworkCIDRBlock(ctx, emptyClient)
			Expect(err).To(HaveOccurred())
			Expect(cidr).To(BeEmpty())
		})
	})

	Describe("GetInfrastructureCluster", func() {
		var infrastructure *configv1.Infrastructure

		BeforeEach(func() {
			infrastructure = &configv1.Infrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: configv1.InfrastructureStatus{
					InfrastructureName:     "test-cluster",
					APIServerInternalURL:   "https://internal.test.com",
				},
			}
		})

		It("should successfully retrieve infrastructure cluster", func() {
			clientWithInfra := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(infrastructure).
				Build()

			infra, err := GetInfrastructureCluster(ctx, clientWithInfra)
			Expect(err).NotTo(HaveOccurred())
			Expect(infra).NotTo(BeNil())
			Expect(infra.Status.InfrastructureName).To(Equal("test-cluster"))
		})

		It("should return error when no infrastructure cluster exists", func() {
			emptyClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			infra, err := GetInfrastructureCluster(ctx, emptyClient)
			Expect(err).To(HaveOccurred())
			Expect(infra).To(BeNil())
		})

		It("should return error when multiple infrastructure clusters exist", func() {
			infrastructure2 := infrastructure.DeepCopy()
			infrastructure2.Name = "cluster2"
			clientWithMultipleInfra := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(infrastructure, infrastructure2).
				Build()

			infra, err := GetInfrastructureCluster(ctx, clientWithMultipleInfra)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected 1 infrastructure cluster, got 2"))
			Expect(infra).To(BeNil())
		})
	})

	Describe("GetClusterName", func() {
		It("should successfully retrieve cluster name", func() {
			infrastructure := &configv1.Infrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: configv1.InfrastructureStatus{
					InfrastructureName: "test-cluster",
				},
			}
			clientWithInfra := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(infrastructure).
				Build()

			name, err := GetClusterName(ctx, clientWithInfra)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("test-cluster"))
		})
	})

	Describe("GetClusterAPIServerInternalURL", func() {
		It("should successfully retrieve API server internal URL", func() {
			infrastructure := &configv1.Infrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: configv1.InfrastructureStatus{
					APIServerInternalURL: "https://internal.test.com",
				},
			}
			clientWithInfra := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(infrastructure).
				Build()

			url, err := GetClusterAPIServerInternalURL(ctx, clientWithInfra)
			Expect(err).NotTo(HaveOccurred())
			Expect(url).To(Equal("https://internal.test.com"))
		})
	})
})