package capoci

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("CAPOCI", func() {
	var (
		ctx       context.Context
		instance  *capiv1alpha1.OCIClusterAutoscaler
		scheme    *runtime.Scheme
		auth      *CAPOCICredentials
		namespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		instance = &capiv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-autoscaler",
			},
		}

		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		auth = &CAPOCICredentials{
			TenancyID:            "test-tenancy",
			UserID:               "test-user",
			Region:               "test-region",
			Fingerprint:          "test-fingerprint",
			PrivateKey:           "test-key",
			UseInstancePrincipal: "false",
			Passphrase:           "test-passphrase",
		}

		namespace = "test-namespace"
	})

	Context("GetClusterctlComponents", func() {
		It("should generate and modify components correctly", func() {
			components, err := GetClusterctlComponents(ctx, "test-deployment", "test-sa", namespace, instance, "test-webhook-service", scheme)
			Expect(err).NotTo(HaveOccurred())
			Expect(components).To(BeAssignableToTypeOf([]unstructured.Unstructured{}))

			// Verify each component has the default labels
			for _, component := range components {
				labels := component.GetLabels()
				Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
				Expect(labels).To(HaveKeyWithValue("capi.openshift.io/managed-by", "test-autoscaler"))
			}
		})
	})

	Context("GetComponents", func() {
		It("should return component with correct subcomponents", func() {
			component := GetComponents(namespace, instance, auth)
			Expect(component).NotTo(BeNil())
			Expect(component.Name).To(Equal("CAPOCI"))

			// Verify subcomponents
			Expect(component.Subcomponents).To(HaveLen(2))
			Expect(component.Subcomponents[0].Name).To(Equal("namespace"))
			Expect(component.Subcomponents[1].Name).To(Equal("authConfigSecret"))

			// Verify each subcomponent has an object and mutate function
			for _, subcomponent := range component.Subcomponents {
				Expect(subcomponent.Object).NotTo(BeNil())
				Expect(subcomponent.MutateFn).NotTo(BeNil())
			}
		})
	})

	Context("Namespace", func() {
		It("should create namespace with correct configuration", func() {
			namespaceObj, mutateFn := Namespace(namespace, instance)
			Expect(namespaceObj).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify namespace configuration
			ns := namespaceObj.(*corev1.Namespace)
			Expect(ns.Name).To(Equal(namespace))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels after mutation
			labels := ns.GetLabels()
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue("capi.openshift.io/managed-by", "test-autoscaler"))
		})
	})

	Context("AuthConfigSecret", func() {
		It("should create secret with correct configuration", func() {
			secret, mutateFn := AuthConfigSecret(instance, namespace, auth)
			Expect(secret).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify secret configuration
			s := secret.(*corev1.Secret)
			Expect(s.Name).To(Equal("capoci-auth-config"))
			Expect(s.Namespace).To(Equal(namespace))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels after mutation
			labels := s.GetLabels()
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue("capi.openshift.io/managed-by", "test-autoscaler"))

			// Verify secret data
			Expect(s.Data).To(HaveKeyWithValue("tenancy", []byte("test-tenancy")))
			Expect(s.Data).To(HaveKeyWithValue("user", []byte("test-user")))
			Expect(s.Data).To(HaveKeyWithValue("region", []byte("test-region")))
			Expect(s.Data).To(HaveKeyWithValue("fingerprint", []byte("test-fingerprint")))
			Expect(s.Data).To(HaveKeyWithValue("key", []byte("test-key")))
			Expect(s.Data).To(HaveKeyWithValue("useInstancePrincipal", []byte("false")))
			Expect(s.Data).To(HaveKeyWithValue("passphrase", []byte("test-passphrase")))
		})
	})
})
