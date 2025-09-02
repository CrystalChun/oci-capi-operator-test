package utils

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubeAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "KubeAPI Suite")
}

var _ = Describe("KubeAPI Utils", func() {
	var (
		ctx        context.Context
		k8sClient  client.Client
		testScheme *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		testScheme = runtime.NewScheme()
		Expect(scheme.AddToScheme(testScheme)).To(Succeed())
		Expect(configv1.AddToScheme(testScheme)).To(Succeed())
		k8sClient = fake.NewClientBuilder().WithScheme(testScheme).Build()
	})

	AfterEach(func() {
		// Clean up resources if needed
	})

	Context("Network functions", func() {
		var testNetwork *configv1.Network

		BeforeEach(func() {
			testNetwork = &configv1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: configv1.NetworkSpec{
					ClusterNetwork: []configv1.ClusterNetworkEntry{
						{
							CIDR: "10.128.0.0/14",
						},
					},
					ServiceNetwork: []string{"172.30.0.0/16"},
				},
			}
			Expect(k8sClient.Create(ctx, testNetwork)).To(Succeed())
		})

		AfterEach(func() {
			k8sClient.Delete(ctx, testNetwork) //originally had Expect(k8sClient.Delete(ctx, testNetwork)).To(Succeed())
		})

		It("should get cluster network CIDR block", func() {
			cidr, err := GetClusterNetworkCIDRBlock(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(cidr).To(Equal("10.128.0.0/14"))
		})

		It("should get service network CIDR block", func() {
			cidr, err := GetServiceNetworkCIDRBlock(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(cidr).To(Equal("172.30.0.0/16"))
		})

		It("should return error when network config does not exist", func() {
			// Delete the test network first
			Expect(k8sClient.Delete(ctx, testNetwork)).To(Succeed())

			_, err := GetClusterNetworkCIDRBlock(ctx, k8sClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get cluster network CIDR block"))

			_, err = GetServiceNetworkCIDRBlock(ctx, k8sClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get service network CIDR block"))
		})
	})

	Context("Cluster Info functions", func() {
		var testInfrastructure *configv1.Infrastructure

		BeforeEach(func() {
			testInfrastructure = &configv1.Infrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: configv1.InfrastructureStatus{
					InfrastructureName:     "test-cluster",
					APIServerInternalURL:   "https://api.test-cluster.example.com:6443",
					APIServerURL:           "https://api.test-cluster.example.com",
					ControlPlaneTopology:   configv1.HighlyAvailableTopologyMode,
					InfrastructureTopology: configv1.HighlyAvailableTopologyMode,
				},
			}
			Expect(k8sClient.Create(ctx, testInfrastructure)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, testInfrastructure)).To(Succeed())
		})

		It("should get infrastructure cluster", func() {
			cluster, err := GetInfrastructureCluster(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(cluster).NotTo(BeNil())
			Expect(cluster.Name).To(Equal("cluster"))
			Expect(cluster.Status.InfrastructureName).To(Equal("test-cluster"))
			Expect(cluster.Status.APIServerInternalURL).To(Equal("https://api.test-cluster.example.com:6443"))
		})

		It("should get cluster name", func() {
			name, err := GetClusterName(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("test-cluster"))
		})

		It("should get cluster API server internal URL", func() {
			url, err := GetClusterAPIServerInternalURL(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(url).To(Equal("https://api.test-cluster.example.com:6443"))
		})

		It("should get machine config CA", func() {
			// Create test secret for machine config CA
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-config-server-tls",
					Namespace: "openshift-machine-config-operator",
				},
				Data: map[string][]byte{
					"tls.crt": []byte("test-ca-cert"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			ca, err := GetMachineConfigCA(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(ca).To(Equal("test-ca-cert"))

			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
		})

		It("should get kubeconfig CA", func() {
			// Create test configmap for kubeconfig CA
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kube-root-ca.crt",
					Namespace: "kube-system",
				},
				Data: map[string]string{
					"ca.crt": "test-ca-cert",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())

			ca, err := GetKubeconfigCA(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(ca).To(Equal("test-ca-cert"))

			Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
		})
	})

	Context("Annotation functions", func() {
		It("should set OpenShift service cert annotation", func() {
			obj := &unstructured.Unstructured{}
			obj.SetName("test-service")

			err := SetOpenshiftServiceCertAnnotation(obj, "test-service")
			Expect(err).NotTo(HaveOccurred())

			annotations := obj.GetAnnotations()
			Expect(annotations).To(HaveLen(1))
			Expect(annotations).To(HaveKeyWithValue(OpenshiftServiceCertAnnotation, "test-service"))
		})

		It("should not set OpenShift service cert annotation when names don't match", func() {
			obj := &unstructured.Unstructured{}
			obj.SetName("different-name")

			err := SetOpenshiftServiceCertAnnotation(obj, "test-service")
			Expect(err).NotTo(HaveOccurred())

			annotations := obj.GetAnnotations()
			Expect(annotations).To(BeNil())
		})

		It("should merge OpenShift service cert annotation with existing annotations", func() {
			obj := &unstructured.Unstructured{}
			obj.SetName("test-service")
			obj.SetAnnotations(map[string]string{
				"existing-annotation":         "existing-value",
				CertManagerCAInjectAnnotation: "old-value",
			})

			err := SetOpenshiftServiceCertAnnotation(obj, "test-service")
			Expect(err).NotTo(HaveOccurred())

			annotations := obj.GetAnnotations()
			Expect(annotations).To(HaveLen(2))
			Expect(annotations).To(HaveKeyWithValue(OpenshiftServiceCertAnnotation, "test-service"))
			Expect(annotations).To(HaveKeyWithValue("existing-annotation", "existing-value"))
			Expect(annotations).NotTo(HaveKey(CertManagerCAInjectAnnotation))
		})

		It("should return error when object is nil for service cert annotation", func() {
			err := SetOpenshiftServiceCertAnnotation(nil, "test-service")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unstructured object is nil"))
		})

		It("should set OpenShift CA bundle annotation", func() {
			obj := &unstructured.Unstructured{}

			err := SetOpenshiftCABundleAnnotation(obj)
			Expect(err).NotTo(HaveOccurred())

			annotations := obj.GetAnnotations()
			Expect(annotations).To(HaveLen(1))
			Expect(annotations).To(HaveKeyWithValue(OpenshiftCABundleAnnotation, "true"))
		})

		It("should merge OpenShift CA bundle annotation with existing annotations", func() {
			obj := &unstructured.Unstructured{}
			obj.SetAnnotations(map[string]string{
				"existing-annotation":         "existing-value",
				CertManagerCAInjectAnnotation: "old-value",
			})

			err := SetOpenshiftCABundleAnnotation(obj)
			Expect(err).NotTo(HaveOccurred())

			annotations := obj.GetAnnotations()
			Expect(annotations).To(HaveLen(2))
			Expect(annotations).To(HaveKeyWithValue(OpenshiftCABundleAnnotation, "true"))
			Expect(annotations).To(HaveKeyWithValue("existing-annotation", "existing-value"))
			Expect(annotations).NotTo(HaveKey(CertManagerCAInjectAnnotation))
		})

		It("should return error when object is nil for CA bundle annotation", func() {
			err := SetOpenshiftCABundleAnnotation(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unstructured object is nil"))
		})
	})

	Context("Label functions", func() {
		It("should get default labels", func() {
			labels := GetDefaultLabels("test-instance")
			Expect(labels).To(HaveLen(2))
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue(ManagedByLabel, "test-instance"))
		})

		It("should set default labels on object with no existing labels", func() {
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-configmap",
					Namespace: "test-namespace",
				},
			}

			err := SetDefaultLabels(obj, "test-instance")
			Expect(err).NotTo(HaveOccurred())

			labels := obj.GetLabels()
			Expect(labels).To(HaveLen(2))
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue(ManagedByLabel, "test-instance"))
		})

		It("should merge default labels with existing labels", func() {
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-configmap",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"existing-label": "existing-value",
					},
				},
			}

			err := SetDefaultLabels(obj, "test-instance")
			Expect(err).NotTo(HaveOccurred())

			labels := obj.GetLabels()
			Expect(labels).To(HaveLen(3))
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue(ManagedByLabel, "test-instance"))
			Expect(labels).To(HaveKeyWithValue("existing-label", "existing-value"))
		})

		It("should return error when object is nil", func() {
			err := SetDefaultLabels(nil, "test-instance")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("object is nil"))
		})
	})

	Context("Deployment functions", func() {
		It("should get deployment condition when it exists", func() {
			conditions := []appsv1.DeploymentCondition{
				{
					Type:    appsv1.DeploymentAvailable,
					Status:  corev1.ConditionTrue,
					Reason:  "MinimumReplicasAvailable",
					Message: "Deployment has minimum availability.",
				},
				{
					Type:    appsv1.DeploymentProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "NewReplicaSetAvailable",
					Message: "ReplicaSet is progressing.",
				},
			}

			condition := GetDeploymentCondition(conditions, appsv1.DeploymentAvailable)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Type).To(Equal(appsv1.DeploymentAvailable))
			Expect(condition.Status).To(Equal(corev1.ConditionTrue))
			Expect(condition.Reason).To(Equal("MinimumReplicasAvailable"))
		})

		It("should return nil when deployment condition does not exist", func() {
			conditions := []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
			}

			condition := GetDeploymentCondition(conditions, appsv1.DeploymentProgressing)
			Expect(condition).To(BeNil())
		})

		It("should edit deployment certs successfully", func() {
			deployment := &appsv1.Deployment{
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
							},
						},
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			Expect(testScheme.Convert(deployment, unstructuredObj, nil)).To(Succeed())

			err := EditDeploymentCerts(testScheme, unstructuredObj, "new-secret")
			Expect(err).NotTo(HaveOccurred())

			// Convert back to deployment to verify changes
			updatedDeployment := &appsv1.Deployment{}
			Expect(testScheme.Convert(unstructuredObj, updatedDeployment, nil)).To(Succeed())

			// Verify the cert volume was updated
			certVolume := updatedDeployment.Spec.Template.Spec.Volumes[0]
			Expect(certVolume.Name).To(Equal("cert"))
			Expect(certVolume.Secret.SecretName).To(Equal("new-secret"))
		})

		It("should return error when unstructured object is nil", func() {
			err := EditDeploymentCerts(testScheme, nil, "new-secret")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unstructured object is nil"))
		})
	})

	Context("Secret functions", func() {
		var testSecret *corev1.Secret

		BeforeEach(func() {
			testSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"test-key": []byte("test-value"),
				},
			}
			Expect(k8sClient.Create(ctx, testSecret)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, testSecret)).To(Succeed())
		})

		It("should get an existing secret", func() {
			secret, err := GetSecret(ctx, k8sClient, "test-secret", "test-namespace")
			Expect(err).NotTo(HaveOccurred())
			Expect(secret).NotTo(BeNil())
			Expect(secret.Name).To(Equal("test-secret"))
			Expect(secret.Namespace).To(Equal("test-namespace"))
			Expect(secret.Data).To(HaveKeyWithValue("test-key", []byte("test-value")))
		})

		It("should return error when getting non-existent secret", func() {
			_, err := GetSecret(ctx, k8sClient, "non-existent", "test-namespace")
			Expect(err).To(HaveOccurred())
		})

		It("should get secret data for existing key", func() {
			data, err := GetSecretData(ctx, k8sClient, "test-secret", "test-namespace", "test-key")
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal([]byte("test-value")))
		})

		It("should return error when getting data for non-existent key", func() {
			_, err := GetSecretData(ctx, k8sClient, "test-secret", "test-namespace", "non-existent-key")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key non-existent-key not found in secret test-secret"))
		})
	})
})
