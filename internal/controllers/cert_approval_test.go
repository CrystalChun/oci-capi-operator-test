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
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Certificate Approval Controller", func() {
	var (
		ctx            context.Context
		fakeClient     client.Client
		scheme         *runtime.Scheme
		reconciler     *CertificateApprovalReconciler
		testOCIMachine *metav1.PartialObjectMetadata
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		_ = certificatesv1.AddToScheme(scheme)
		_ = corev1.AddToScheme(scheme)

		// Create a mock OCI machine
		testOCIMachine = &metav1.PartialObjectMetadata{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
				Kind:       "OCIMachine",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster-worker-1",
				Namespace: "capi-system",
			},
		}
		testOCIMachine.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "infrastructure.cluster.x-k8s.io",
			Version: "v1beta2",
			Kind:    "OCIMachine",
		})

		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(testOCIMachine).
			Build()

		reconciler = &CertificateApprovalReconciler{
			Client: fakeClient,
			Scheme: scheme,
			// Note: CSRClient would need to be mocked for full integration tests
		}
	})

	Describe("isKubeletCSR", func() {
		It("should return true for kube-apiserver-client-kubelet signer", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "kubernetes.io/kube-apiserver-client-kubelet",
				},
			}
			Expect(isKubeletCSR(csr)).To(BeTrue())
		})

		It("should return true for kubelet-serving signer", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "kubernetes.io/kubelet-serving",
				},
			}
			Expect(isKubeletCSR(csr)).To(BeTrue())
		})

		It("should return false for other signers", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "kubernetes.io/something-else",
				},
			}
			Expect(isKubeletCSR(csr)).To(BeFalse())
		})

		It("should return false for empty signer", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "",
				},
			}
			Expect(isKubeletCSR(csr)).To(BeFalse())
		})
	})

	Describe("isCSRApproved", func() {
		It("should return true when CSR has approved condition", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{
						{
							Type:   certificatesv1.CertificateApproved,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
			Expect(isCSRApproved(csr)).To(BeTrue())
		})

		It("should return false when CSR has no conditions", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{},
				},
			}
			Expect(isCSRApproved(csr)).To(BeFalse())
		})

		It("should return false when CSR has only non-approved conditions", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{
						{
							Type:   certificatesv1.CertificateDenied,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
			Expect(isCSRApproved(csr)).To(BeFalse())
		})
	})

	Describe("isCSRDenied", func() {
		It("should return true when CSR has denied condition", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{
						{
							Type:   certificatesv1.CertificateDenied,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
			Expect(isCSRDenied(csr)).To(BeTrue())
		})

		It("should return false when CSR has no denied condition", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{
						{
							Type:   certificatesv1.CertificateApproved,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
			Expect(isCSRDenied(csr)).To(BeFalse())
		})
	})

	Describe("hasMatchingOCIMachine", func() {
		It("should return true when hostname matches machine name", func() {
			result := reconciler.hasMatchingOCIMachine(ctx, "worker-1")
			Expect(result).To(BeTrue())
		})

		It("should return true when machine name contains hostname", func() {
			result := reconciler.hasMatchingOCIMachine(ctx, "test-cluster-worker-1")
			Expect(result).To(BeTrue())
		})

		It("should return false when no machine matches", func() {
			result := reconciler.hasMatchingOCIMachine(ctx, "nonexistent-worker")
			Expect(result).To(BeFalse())
		})

		It("should return false when hostname is empty", func() {
			result := reconciler.hasMatchingOCIMachine(ctx, "")
			Expect(result).To(BeFalse())
		})
	})

	Describe("getHostnameFromServingCSR", func() {
		It("should extract hostname from username with system:node prefix", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					Username: "system:node:worker-1",
				},
			}
			hostname, err := getHostnameFromServingCSR(csr)
			Expect(err).NotTo(HaveOccurred())
			Expect(hostname).To(Equal("worker-1"))
		})

		It("should extract hostname and remove domain suffix", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					Username: "system:node:worker-1.example.com",
				},
			}
			hostname, err := getHostnameFromServingCSR(csr)
			Expect(err).NotTo(HaveOccurred())
			Expect(hostname).To(Equal("worker-1"))
		})

		It("should return error for empty username", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					Username: "",
				},
			}
			hostname, err := getHostnameFromServingCSR(csr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CSR username is empty"))
			Expect(hostname).To(BeEmpty())
		})

		It("should return error for username without system:node prefix", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					Username: "user:worker-1",
				},
			}
			hostname, err := getHostnameFromServingCSR(csr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not have expected system:node: prefix"))
			Expect(hostname).To(BeEmpty())
		})

		It("should return error when hostname is empty after extraction", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					Username: "system:node:",
				},
			}
			hostname, err := getHostnameFromServingCSR(csr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("extracted hostname is empty"))
			Expect(hostname).To(BeEmpty())
		})
	})

	Describe("getHostnameFromClientKubeletCSR", func() {
		var validCSRBytes []byte

		BeforeEach(func() {
			// Generate a valid CSR for testing
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			template := x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName: "system:node:worker-1",
				},
			}

			csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
			Expect(err).NotTo(HaveOccurred())

			pemBlock := &pem.Block{
				Type:  "CERTIFICATE REQUEST",
				Bytes: csrBytes,
			}
			validCSRBytes = pem.EncodeToMemory(pemBlock)
		})

		It("should extract hostname from valid CSR with system:node CommonName", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					Request: validCSRBytes,
				},
			}
			hostname, err := getHostnameFromClientKubeletCSR(csr)
			Expect(err).NotTo(HaveOccurred())
			Expect(hostname).To(Equal("worker-1"))
		})

		It("should return error for empty CSR request", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					Request: []byte{},
				},
			}
			hostname, err := getHostnameFromClientKubeletCSR(csr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CSR request is empty"))
			Expect(hostname).To(BeEmpty())
		})

		It("should return error for invalid PEM", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					Request: []byte("invalid pem data"),
				},
			}
			hostname, err := getHostnameFromClientKubeletCSR(csr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to decode PEM block"))
			Expect(hostname).To(BeEmpty())
		})
	})

	Describe("getCSRHostname", func() {
		It("should route to client kubelet CSR handler", func() {
			// Create a valid CSR for client kubelet
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			template := x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName: "system:node:worker-1",
				},
			}

			csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
			Expect(err).NotTo(HaveOccurred())

			pemBlock := &pem.Block{
				Type:  "CERTIFICATE REQUEST",
				Bytes: csrBytes,
			}
			validCSRBytes := pem.EncodeToMemory(pemBlock)

			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "kubernetes.io/kube-apiserver-client-kubelet",
					Request:    validCSRBytes,
				},
			}
			hostname, err := getCSRHostname(csr)
			Expect(err).NotTo(HaveOccurred())
			Expect(hostname).To(Equal("worker-1"))
		})

		It("should route to serving CSR handler", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "kubernetes.io/kubelet-serving",
					Username:   "system:node:worker-1",
				},
			}
			hostname, err := getCSRHostname(csr)
			Expect(err).NotTo(HaveOccurred())
			Expect(hostname).To(Equal("worker-1"))
		})

		It("should return empty hostname for unknown signer", func() {
			csr := &certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "unknown.signer/name",
				},
			}
			hostname, err := getCSRHostname(csr)
			Expect(err).NotTo(HaveOccurred())
			Expect(hostname).To(BeEmpty())
		})
	})

	Describe("Reconcile", func() {
		It("should return no error for non-existent CSR", func() {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "non-existent-csr",
				},
			}
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})

		It("should skip already approved CSR", func() {
			approvedCSR := &certificatesv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "approved-csr",
				},
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{
						{
							Type:   certificatesv1.CertificateApproved,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}

			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(approvedCSR).
				Build()
			reconciler.Client = fakeClient

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "approved-csr",
				},
			}
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})

		It("should skip non-kubelet CSR", func() {
			nonKubeletCSR := &certificatesv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "non-kubelet-csr",
				},
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "some.other/signer",
				},
			}

			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(nonKubeletCSR).
				Build()
			reconciler.Client = fakeClient

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "non-kubelet-csr",
				},
			}
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
})
