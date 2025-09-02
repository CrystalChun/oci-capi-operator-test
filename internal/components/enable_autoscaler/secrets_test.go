package enableautoscaler

import (
	"context"
	"encoding/base64"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Secrets", func() {
	var (
		ctx               context.Context
		k8sClient         client.Client
		scheme            *runtime.Scheme
		instance          *ocicapioperatorv1alpha1.OCIClusterAutoscaler
		capiSystemNS      string
		clusterName       string
		serviceAccountSA  string
		serviceAccountKey string
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		instance = &ocicapioperatorv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}

		capiSystemNS = "capi-system"
		clusterName = "test-cluster"
		serviceAccountSA = "capi-sa"
		serviceAccountKey = fmt.Sprintf("%s-token", serviceAccountSA)

		// Create service account token secret
		saSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountKey,
				Namespace: capiSystemNS,
			},
			Data: map[string][]byte{
				"ca.crt": []byte("test-ca-cert"),
				"token":  []byte("test-token"),
			},
		}
		Expect(k8sClient.Create(ctx, saSecret)).To(Succeed())
	})

	Context("BootstrapConfigSecret", func() {
		It("should create bootstrap config secret with correct configuration", func() {
			secret, mutateFn := BootstrapConfigSecret(ctx, k8sClient, capiSystemNS, clusterName, instance)
			Expect(secret).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify secret configuration
			s := secret.(*corev1.Secret)
			Expect(s.Name).To(Equal(fmt.Sprintf("%s-bootstrap", clusterName)))
			Expect(s.Namespace).To(Equal(capiSystemNS))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels after mutation
			labels := s.GetLabels()
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue("capi.openshift.io/managed-by", clusterName))

			// Verify secret data
			Expect(s.Data).To(HaveKey("value"))
			Expect(s.Data).To(HaveKeyWithValue("format", []byte("ignition")))
		})
	})

	Context("KubeConfigSecret", func() {
		It("should create kubeconfig secret with correct configuration", func() {
			secret, mutateFn := KubeConfigSecret(ctx, k8sClient, capiSystemNS, clusterName, serviceAccountSA, instance)
			Expect(secret).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify secret configuration
			s := secret.(*corev1.Secret)
			Expect(s.Name).To(Equal(fmt.Sprintf("%s-kubeconfig", clusterName)))
			Expect(s.Namespace).To(Equal(capiSystemNS))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels after mutation
			labels := s.GetLabels()
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue("capi.openshift.io/managed-by", clusterName))
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/cluster-name", clusterName))
			Expect(labels).To(HaveKeyWithValue("clusterctl.cluster.x-k8s.io/move", ""))

			// Verify kubeconfig content
			kubeconfig := string(s.Data["value"])
			caCrt := base64.StdEncoding.EncodeToString([]byte("test-ca-cert"))
			Expect(kubeconfig).To(ContainSubstring(fmt.Sprintf("name: %s", clusterName)))
			Expect(kubeconfig).To(ContainSubstring("server: https://kubernetes.default.svc"))
			Expect(kubeconfig).To(ContainSubstring(fmt.Sprintf("certificate-authority-data: %s", caCrt)))
			Expect(kubeconfig).To(ContainSubstring("test-token"))
		})

		It("should fail when service account secret does not exist", func() {
			// Delete the service account secret
			saSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceAccountKey,
					Namespace: capiSystemNS,
				},
			}
			Expect(k8sClient.Delete(ctx, saSecret)).To(Succeed())

			_, mutateFn := KubeConfigSecret(ctx, k8sClient, capiSystemNS, clusterName, serviceAccountSA, instance)
			err := mutateFn()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get service account secret"))
		})
	})
})
