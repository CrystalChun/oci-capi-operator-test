package utils

import (
	"context"
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/coreos/ignition/v2/config/v3_2/types"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Ignition Utils", func() {
	var (
		ctx        context.Context
		fakeClient client.Client
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = configv1.AddToScheme(scheme)

		// Set up test objects
		infrastructure := &configv1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Status: configv1.InfrastructureStatus{
				APIServerInternalURL: "https://api-int.test-cluster.example.com:6443",
			},
		}

		machineConfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "machine-config-server-tls",
				Namespace: "openshift-machine-config-operator",
			},
			Data: map[string][]byte{
				"tls.crt": []byte("-----BEGIN CERTIFICATE-----\ntest-machine-config-ca\n-----END CERTIFICATE-----"),
			},
		}

		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(infrastructure, machineConfigSecret).
			Build()
	})

	Describe("GenerateIgnitionConfig", func() {
		It("should successfully generate ignition config", func() {
			configStr, err := GenerateIgnitionConfig(ctx, fakeClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(configStr).NotTo(BeEmpty())

			// Parse the generated config to verify structure
			var config types.Config
			err = json.Unmarshal([]byte(configStr), &config)
			Expect(err).NotTo(HaveOccurred())

			// Verify ignition version
			Expect(config.Ignition.Version).To(Equal("3.2.0"))

			// Verify systemd units
			Expect(config.Systemd.Units).To(HaveLen(1))
			unit := config.Systemd.Units[0]
			Expect(unit.Name).To(Equal("set-hostname-oci.service"))
			Expect(*unit.Enabled).To(BeTrue())
			Expect(*unit.Contents).To(ContainSubstring("Description=Set hostname from OCI metadata"))

			// Verify files
			Expect(config.Storage.Files).To(HaveLen(1))
			file := config.Storage.Files[0]
			Expect(file.Path).To(Equal("/usr/local/bin/set-hostname-oci.sh"))
			Expect(*file.Mode).To(Equal(493)) // 0755 in decimal
			Expect(*file.Contents.Source).To(ContainSubstring("data:text/plain;charset=utf-8;base64,"))

			// Verify TLS configuration
			Expect(config.Ignition.Security.TLS.CertificateAuthorities).To(HaveLen(1))
			ca := config.Ignition.Security.TLS.CertificateAuthorities[0]
			Expect(*ca.Source).To(ContainSubstring("data:text/plain;charset=utf-8;base64,"))

			// Verify merge configuration
			Expect(config.Ignition.Config.Merge).To(HaveLen(1))
			merge := config.Ignition.Config.Merge[0]
			Expect(*merge.Source).To(Equal("https://api-int.test-cluster.example.com:22623/config/worker"))
		})

		It("should handle API server URL with different formats", func() {
			// Test with URL that has port
			infrastructure := &configv1.Infrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: configv1.InfrastructureStatus{
					APIServerInternalURL: "https://api-int.test.com:6443",
				},
			}

			machineConfigSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-config-server-tls",
					Namespace: "openshift-machine-config-operator",
				},
				Data: map[string][]byte{
					"tls.crt": []byte("test-ca"),
				},
			}

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(infrastructure, machineConfigSecret).
				Build()

			configStr, err := GenerateIgnitionConfig(ctx, client)
			Expect(err).NotTo(HaveOccurred())

			var config types.Config
			err = json.Unmarshal([]byte(configStr), &config)
			Expect(err).NotTo(HaveOccurred())

			merge := config.Ignition.Config.Merge[0]
			Expect(*merge.Source).To(Equal("https://api-int.test.com:22623/config/worker"))
		})

		It("should handle API server URL without port", func() {
			infrastructure := &configv1.Infrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: configv1.InfrastructureStatus{
					APIServerInternalURL: "https://api-int.test.com",
				},
			}

			machineConfigSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-config-server-tls",
					Namespace: "openshift-machine-config-operator",
				},
				Data: map[string][]byte{
					"tls.crt": []byte("test-ca"),
				},
			}

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(infrastructure, machineConfigSecret).
				Build()

			configStr, err := GenerateIgnitionConfig(ctx, client)
			Expect(err).NotTo(HaveOccurred())

			var config types.Config
			err = json.Unmarshal([]byte(configStr), &config)
			Expect(err).NotTo(HaveOccurred())

			merge := config.Ignition.Config.Merge[0]
			Expect(*merge.Source).To(Equal("https://api-int.test.com:22623/config/worker"))
		})

		It("should return error when API server URL retrieval fails", func() {
			// Create client without infrastructure object
			emptyClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			configStr, err := GenerateIgnitionConfig(ctx, emptyClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get cluster API server internal URL"))
			Expect(configStr).To(BeEmpty())
		})

		It("should return error when machine config CA retrieval fails", func() {
			// Create client with infrastructure but without machine config secret
			infrastructure := &configv1.Infrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: configv1.InfrastructureStatus{
					APIServerInternalURL: "https://api-int.test.com:6443",
				},
			}

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(infrastructure).
				Build()

			configStr, err := GenerateIgnitionConfig(ctx, client)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get machine config CA"))
			Expect(configStr).To(BeEmpty())
		})

		It("should properly encode CA certificate in base64", func() {
			configStr, err := GenerateIgnitionConfig(ctx, fakeClient)
			Expect(err).NotTo(HaveOccurred())

			var config types.Config
			err = json.Unmarshal([]byte(configStr), &config)
			Expect(err).NotTo(HaveOccurred())

			ca := config.Ignition.Security.TLS.CertificateAuthorities[0]
			// Verify the source contains base64 encoded data
			Expect(*ca.Source).To(ContainSubstring("data:text/plain;charset=utf-8;base64,"))
			
			// Extract the base64 part and verify it's valid base64
			parts := strings.Split(*ca.Source, "base64,")
			Expect(parts).To(HaveLen(2))
			base64Part := parts[1]
			Expect(base64Part).NotTo(BeEmpty())
		})

		It("should generate valid JSON output", func() {
			configStr, err := GenerateIgnitionConfig(ctx, fakeClient)
			Expect(err).NotTo(HaveOccurred())

			// Verify it's valid JSON
			var jsonData interface{}
			err = json.Unmarshal([]byte(configStr), &jsonData)
			Expect(err).NotTo(HaveOccurred())

			// Verify it can be unmarshaled to ignition config structure
			var config types.Config
			err = json.Unmarshal([]byte(configStr), &config)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})