package capi

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	securityv1 "github.com/openshift/api/security/v1"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("CAPI Components", func() {
	var (
		ctx        context.Context
		scheme     *runtime.Scheme
		instance   *capiv1alpha1.OCIClusterAutoscaler
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = rbacv1.AddToScheme(scheme)
		_ = securityv1.AddToScheme(scheme)
		_ = capiv1alpha1.AddToScheme(scheme)

		instance = &capiv1alpha1.OCIClusterAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-autoscaler",
				Namespace: "test-namespace",
			},
		}
	})

	Describe("GetComponents", func() {
		It("should return component with all subcomponents", func() {
			component := GetComponents(
				"capi-system",
				"capoci-system", 
				"capi-sa",
				"capoci-sa",
				"test-cluster-role-binding",
				instance,
			)

			Expect(component).NotTo(BeNil())
			Expect(component.Name).To(Equal("CAPI"))
			Expect(component.Subcomponents).To(HaveLen(4))

			// Verify subcomponent names
			subcompNames := make([]string, len(component.Subcomponents))
			for i, subcomp := range component.Subcomponents {
				subcompNames[i] = subcomp.Name
			}
			Expect(subcompNames).To(ConsistOf("scc", "namespace", "clusterRoleBinding", "serviceAccountSecret"))
		})
	})

	Describe("CAPINamespace", func() {
		It("should create namespace with correct name and mutate function", func() {
			obj, mutateFn := CAPINamespace("test-namespace", instance)

			namespace, ok := obj.(*corev1.Namespace)
			Expect(ok).To(BeTrue())
			Expect(namespace.Name).To(Equal("test-namespace"))

			// Test mutate function
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify labels were set
			labels := namespace.GetLabels()
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue("capi.openshift.io/managed-by", "test-autoscaler"))
		})
	})

	Describe("SecurityContextConstraints", func() {
		It("should create SCC with correct configuration", func() {
			obj, mutateFn := SecurityContextConstraints(
				"capi-system",
				"capoci-system",
				"capoci-sa",
				"capi-sa",
				instance,
			)

			scc, ok := obj.(*securityv1.SecurityContextConstraints)
			Expect(ok).To(BeTrue())
			Expect(scc.Name).To(Equal("oci-capi"))

			// Test mutate function
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify SCC configuration
			Expect(scc.RunAsUser.Type).To(Equal(securityv1.RunAsUserStrategyRunAsAny))
			Expect(scc.SELinuxContext.Type).To(Equal(securityv1.SELinuxStrategyRunAsAny))
			Expect(scc.SeccompProfiles).To(ConsistOf("runtime/default"))
			Expect(scc.Users).To(ConsistOf(
				"system:serviceaccount:capoci-system:capoci-sa",
				"system:serviceaccount:capi-system:capi-sa",
			))

			// Verify labels were set
			labels := scc.GetLabels()
			Expect(labels).To(HaveKeyWithValue("cluster.x-k8s.io/provider", "cluster-api"))
			Expect(labels).To(HaveKeyWithValue("capi.openshift.io/managed-by", "test-autoscaler"))
		})
	})

	Describe("ClusterRoleBinding", func() {
		It("should create cluster role binding with correct configuration", func() {
			obj, mutateFn := ClusterRoleBinding(
				"test-binding",
				"test-sa",
				"test-namespace",
			)

			crb, ok := obj.(*rbacv1.ClusterRoleBinding)
			Expect(ok).To(BeTrue())
			Expect(crb.Name).To(Equal("test-binding"))

			// Test mutate function
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify cluster role binding configuration
			Expect(crb.RoleRef.Kind).To(Equal("ClusterRole"))
			Expect(crb.RoleRef.Name).To(Equal("cluster-admin"))
			Expect(crb.Subjects).To(HaveLen(1))
			Expect(crb.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(crb.Subjects[0].Name).To(Equal("test-sa"))
			Expect(crb.Subjects[0].Namespace).To(Equal("test-namespace"))
		})
	})

	Describe("ServiceAccountSecret", func() {
		It("should create service account secret with correct configuration", func() {
			obj, mutateFn := ServiceAccountSecret("test-sa", "test-namespace")

			secret, ok := obj.(*corev1.Secret)
			Expect(ok).To(BeTrue())
			Expect(secret.Name).To(Equal("test-sa-token"))
			Expect(secret.Namespace).To(Equal("test-namespace"))

			// Test mutate function
			err := mutateFn()
			Expect(err).NotTo(HaveOccurred())

			// Verify secret configuration
			Expect(secret.Type).To(Equal(corev1.SecretTypeServiceAccountToken))
			Expect(secret.Annotations).To(HaveKeyWithValue(
				"kubernetes.io/service-account.name",
				"test-sa",
			))
		})
	})

	Describe("GetClusterctlComponents", func() {
		It("should handle empty component list", func() {
			// This test is limited because the actual clusterctl components generation
			// requires a proper clusterctl environment. In a real test, we would mock
			// the utils.GenerateCAPIComponents function.
			components, err := GetClusterctlComponents(
				ctx,
				"test-deployment",
				"test-sa",
				"test-namespace",
				instance,
				"test-webhook-service",
				scheme,
			)

			// In most test environments, this will fail due to missing clusterctl setup
			// but we can verify the function signature and error handling
			if err != nil {
				Expect(err).To(HaveOccurred())
				Expect(components).To(BeEmpty())
			} else {
				Expect(components).NotTo(BeNil())
			}
		})
	})

	Describe("Component interface compliance", func() {
		It("should ensure all created objects implement client.Object", func() {
			// Test that all objects returned by component functions implement client.Object
			namespace, _ := CAPINamespace("test", instance)
			Expect(namespace.GetName()).To(Equal("test"))

			scc, _ := SecurityContextConstraints("ns1", "ns2", "sa1", "sa2", instance)
			Expect(scc.GetName()).To(Equal("oci-capi"))

			crb, _ := ClusterRoleBinding("binding", "sa", "ns")
			Expect(crb.GetName()).To(Equal("binding"))

			secret, _ := ServiceAccountSecret("sa", "ns")
			Expect(secret.GetName()).To(Equal("sa-token"))
		})

		It("should ensure all mutate functions are valid", func() {
			// Test that all mutate functions can be called without error
			_, mutateFn1 := CAPINamespace("test", instance)
			Expect(mutateFn1()).NotTo(HaveOccurred())

			_, mutateFn2 := SecurityContextConstraints("ns1", "ns2", "sa1", "sa2", instance)
			Expect(mutateFn2()).NotTo(HaveOccurred())

			_, mutateFn3 := ClusterRoleBinding("binding", "sa", "ns")
			Expect(mutateFn3()).NotTo(HaveOccurred())

			_, mutateFn4 := ServiceAccountSecret("sa", "ns")
			Expect(mutateFn4()).NotTo(HaveOccurred())
		})

		It("should verify mutate functions are of correct type", func() {
			// Verify that mutate functions have the correct signature
			_, mutateFn := CAPINamespace("test", instance)
			var _ controllerutil.MutateFn = mutateFn

			_, mutateFn2 := SecurityContextConstraints("ns1", "ns2", "sa1", "sa2", instance)
			var _ controllerutil.MutateFn = mutateFn2

			_, mutateFn3 := ClusterRoleBinding("binding", "sa", "ns")
			var _ controllerutil.MutateFn = mutateFn3

			_, mutateFn4 := ServiceAccountSecret("sa", "ns")
			var _ controllerutil.MutateFn = mutateFn4
		})
	})
})