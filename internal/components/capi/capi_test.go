package capi

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	securityv1 "github.com/openshift/api/security/v1"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("CAPI", func() {
	var (
		ctx                   context.Context
		instance              *capiv1alpha1.OCIClusterAutoscaler
		scheme                *runtime.Scheme
		capiSystemNamespace   string
		capociSystemNamespace string
		capiServiceAccount    string
		capociServiceAccount  string
		clusterRoleBinding    string
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
		Expect(securityv1.AddToScheme(scheme)).To(Succeed())
		Expect(rbacv1.AddToScheme(scheme)).To(Succeed())

		capiSystemNamespace = "capi-system"
		capociSystemNamespace = "capoci-system"
		capiServiceAccount = "capi-sa"
		capociServiceAccount = "capoci-sa"
		clusterRoleBinding = "capi-admin"
	})

	Context("GetClusterctlComponents", func() {
		It("should generate and modify components correctly", func() {
			components, err := GetClusterctlComponents(ctx, "test-deployment", "test-sa", capiSystemNamespace, instance, "test-webhook-service", scheme)
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
			component := GetComponents(capiSystemNamespace, capociSystemNamespace, capiServiceAccount, capociServiceAccount, clusterRoleBinding, instance)
			Expect(component).NotTo(BeNil())
			Expect(component.Name).To(Equal("CAPI"))

			// Verify subcomponents
			Expect(component.Subcomponents).To(HaveLen(4))
			Expect(component.Subcomponents[0].Name).To(Equal("scc"))
			Expect(component.Subcomponents[1].Name).To(Equal("namespace"))
			Expect(component.Subcomponents[2].Name).To(Equal("clusterRoleBinding"))
			Expect(component.Subcomponents[3].Name).To(Equal("serviceAccountSecret"))

			// Verify each subcomponent has an object and mutate function
			for _, subcomponent := range component.Subcomponents {
				Expect(subcomponent.Object).NotTo(BeNil())
				Expect(subcomponent.MutateFn).NotTo(BeNil())
			}
		})
	})

	Context("CAPINamespace", func() {
		It("should create namespace with correct configuration", func() {
			namespaceObj, mutateFn := CAPINamespace(capiSystemNamespace, instance)
			Expect(namespaceObj).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify namespace configuration
			ns := namespaceObj.(*corev1.Namespace)
			Expect(ns.Name).To(Equal(capiSystemNamespace))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels after mutation
			labels := ns.GetLabels()
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue("capi.openshift.io/managed-by", "test-autoscaler"))
		})
	})

	Context("SecurityContextConstraints", func() {
		It("should create SCC with correct configuration", func() {
			scc, mutateFn := SecurityContextConstraints(capiSystemNamespace, capociSystemNamespace, capociServiceAccount, capiServiceAccount, instance)
			Expect(scc).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify SCC configuration
			s := scc.(*securityv1.SecurityContextConstraints)
			Expect(s.Name).To(Equal("oci-capi"))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration after mutation
			Expect(s.RunAsUser.Type).To(Equal(securityv1.RunAsUserStrategyRunAsAny))
			Expect(s.SELinuxContext.Type).To(Equal(securityv1.SELinuxStrategyRunAsAny))
			Expect(s.SeccompProfiles).To(Equal([]string{"runtime/default"}))
			Expect(s.Users).To(ConsistOf(
				fmt.Sprintf("system:serviceaccount:%s:%s", capociSystemNamespace, capociServiceAccount),
				fmt.Sprintf("system:serviceaccount:%s:%s", capiSystemNamespace, capiServiceAccount),
			))

			// Verify labels
			labels := s.GetLabels()
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue("capi.openshift.io/managed-by", "test-autoscaler"))
		})
	})

	Context("ClusterRoleBinding", func() {
		It("should create cluster role binding with correct configuration", func() {
			crb, mutateFn := ClusterRoleBinding(clusterRoleBinding, capiServiceAccount, capiSystemNamespace)
			Expect(crb).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify cluster role binding configuration
			binding := crb.(*rbacv1.ClusterRoleBinding)
			Expect(binding.Name).To(Equal(clusterRoleBinding))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration after mutation
			Expect(binding.RoleRef.Kind).To(Equal("ClusterRole"))
			Expect(binding.RoleRef.Name).To(Equal("cluster-admin"))
			Expect(binding.Subjects).To(HaveLen(1))
			Expect(binding.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(binding.Subjects[0].Name).To(Equal(capiServiceAccount))
			Expect(binding.Subjects[0].Namespace).To(Equal(capiSystemNamespace))
		})
	})

	Context("ServiceAccountSecret", func() {
		It("should create service account secret with correct configuration", func() {
			secret, mutateFn := ServiceAccountSecret(capiServiceAccount, capiSystemNamespace)
			Expect(secret).NotTo(BeNil())
			Expect(mutateFn).NotTo(BeNil())

			// Verify secret configuration
			s := secret.(*corev1.Secret)
			Expect(s.Name).To(Equal(fmt.Sprintf("%s-token", capiServiceAccount)))
			Expect(s.Namespace).To(Equal(capiSystemNamespace))

			// Apply mutation
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify configuration after mutation
			Expect(s.Type).To(Equal(corev1.SecretTypeServiceAccountToken))
			Expect(s.Annotations).To(HaveKeyWithValue("kubernetes.io/service-account.name", capiServiceAccount))
		})
	})
})
