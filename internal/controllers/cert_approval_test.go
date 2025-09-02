package controllers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	certificatesv1client "k8s.io/client-go/kubernetes/typed/certificates/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Certificate Approval Controller", func() {
	var (
		ctx                  context.Context
		controllerReconciler *CertificateApprovalReconciler
		csrName              string
		csrNamespace         string
	)

	BeforeEach(func() {
		ctx = context.Background()
		csrName = "test-csr"
		csrNamespace = "capi-system"

		controllerReconciler = &CertificateApprovalReconciler{
			Client:    k8sClient,
			Scheme:    k8sClient.Scheme(),
			CSRClient: *certificatesv1client.NewForConfigOrDie(cfg),
		}

		// Create OCIMachine
		machine := &infrastructurev1beta2.OCIMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-worker-1",
				Namespace: csrNamespace,
			},
		}
		Expect(k8sClient.Create(ctx, machine)).To(Succeed())
	})

	Context("When reconciling a CSR", func() {
		var csr *certificatesv1.CertificateSigningRequest

		BeforeEach(func() {
			// Generate a test CSR
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			template := &x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName: "system:node:test-worker-1",
				},
			}

			csrBytes, err := x509.CreateCertificateRequest(rand.Reader, template, key)
			Expect(err).NotTo(HaveOccurred())

			csrPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE REQUEST",
				Bytes: csrBytes,
			})

			csr = &certificatesv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: csrName,
				},
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "kubernetes.io/kube-apiserver-client-kubelet",
					Request:    csrPEM,
					Username:   "system:node:test-worker-1",
					Groups: []string{
						"system:nodes",
						"system:authenticated",
					},
				},
			}
			Expect(k8sClient.Create(ctx, csr)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, csr)).To(Succeed())
		})

		It("should approve CSR for matching OCIMachine", func() {
			result, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name: csrName,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify CSR is approved
			updatedCSR := &certificatesv1.CertificateSigningRequest{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: csrName}, updatedCSR)
			Expect(err).NotTo(HaveOccurred())
			Expect(isCSRApproved(updatedCSR)).To(BeTrue())
		})

		It("should not approve CSR for non-matching OCIMachine", func() {
			csr.Spec.Username = "system:node:non-existent-worker"
			Expect(k8sClient.Update(ctx, csr)).To(Succeed())

			result, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name: csrName,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify CSR is not approved
			updatedCSR := &certificatesv1.CertificateSigningRequest{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: csrName}, updatedCSR)
			Expect(err).NotTo(HaveOccurred())
			Expect(isCSRApproved(updatedCSR)).To(BeFalse())
		})

		It("should not approve CSR with invalid signer name", func() {
			csr.Spec.SignerName = "invalid-signer"
			Expect(k8sClient.Update(ctx, csr)).To(Succeed())

			result, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name: csrName,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify CSR is not approved
			updatedCSR := &certificatesv1.CertificateSigningRequest{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: csrName}, updatedCSR)
			Expect(err).NotTo(HaveOccurred())
			Expect(isCSRApproved(updatedCSR)).To(BeFalse())
		})

		It("should not approve already approved CSR", func() {
			// Manually approve the CSR
			now := metav1.Now()
			csr.Status.Conditions = []certificatesv1.CertificateSigningRequestCondition{
				{
					Type:               certificatesv1.CertificateApproved,
					Status:             corev1.ConditionTrue,
					Reason:             "ManualApproval",
					Message:            "Manually approved",
					LastUpdateTime:     now,
					LastTransitionTime: now,
				},
			}
			Expect(k8sClient.Status().Update(ctx, csr)).To(Succeed())

			result, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name: csrName,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify CSR status hasn't changed
			updatedCSR := &certificatesv1.CertificateSigningRequest{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: csrName}, updatedCSR)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCSR.Status.Conditions).To(HaveLen(1))
			Expect(updatedCSR.Status.Conditions[0].Reason).To(Equal("ManualApproval"))
		})

		It("should not approve denied CSR", func() {
			// Manually deny the CSR
			now := metav1.Now()
			csr.Status.Conditions = []certificatesv1.CertificateSigningRequestCondition{
				{
					Type:               certificatesv1.CertificateDenied,
					Status:             corev1.ConditionTrue,
					Reason:             "ManualDenial",
					Message:            "Manually denied",
					LastUpdateTime:     now,
					LastTransitionTime: now,
				},
			}
			Expect(k8sClient.Status().Update(ctx, csr)).To(Succeed())

			result, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name: csrName,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify CSR status hasn't changed
			updatedCSR := &certificatesv1.CertificateSigningRequest{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: csrName}, updatedCSR)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCSR.Status.Conditions).To(HaveLen(1))
			Expect(updatedCSR.Status.Conditions[0].Type).To(Equal(certificatesv1.CertificateDenied))
		})
	})

	Context("CSR Hostname Extraction", func() {
		It("should extract hostname from client kubelet CSR", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "kubernetes.io/kube-apiserver-client-kubelet",
					Request:    generateCSRWithCN("system:node:test-worker-1"),
				},
			}
			hostname, err := getCSRHostname(csr)
			Expect(err).NotTo(HaveOccurred())
			Expect(hostname).To(Equal("test-worker-1"))
		})

		It("should extract hostname from serving CSR", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "kubernetes.io/kubelet-serving",
					Username:   "system:node:test-worker-1",
				},
			}
			hostname, err := getCSRHostname(csr)
			Expect(err).NotTo(HaveOccurred())
			Expect(hostname).To(Equal("test-worker-1"))
		})

		It("should handle invalid CSR", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "kubernetes.io/kube-apiserver-client-kubelet",
					Request:    []byte("invalid-csr"),
				},
			}
			_, err := getCSRHostname(csr)
			Expect(err).To(HaveOccurred())
		})
	})
})

// Helper function to generate a CSR with a given CN
func generateCSRWithCN(cn string) []byte {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil
	}

	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: cn,
		},
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})
}
