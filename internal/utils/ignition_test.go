package utils

import (
	"context"
	"encoding/json"

	"github.com/coreos/ignition/v2/config/v3_2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Ignition Utils", func() {
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

		// Create test infrastructure
		infrastructure := &configv1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Status: configv1.InfrastructureStatus{
				APIServerInternalURL: "https://api.test-cluster.example.com:6443",
			},
		}

		// Create test machine config CA secret
		machineConfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "machine-config-server-tls",
				Namespace: "openshift-machine-config-operator",
			},
			Data: map[string][]byte{
				"tls.crt": []byte("test-ca-cert"),
			},
		}

		k8sClient = fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(infrastructure, machineConfigSecret).
			Build()
	})

	Context("GenerateIgnitionConfig", func() {
		It("should generate valid ignition config", func() {
			ignitionConfigStr, err := GenerateIgnitionConfig(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			// Parse the generated config to verify its structure
			var ignitionConfig types.Config
			Expect(json.Unmarshal([]byte(ignitionConfigStr), &ignitionConfig)).To(Succeed())

			// Verify ignition version
			Expect(ignitionConfig.Ignition.Version).To(Equal("3.2.0"))

			// Verify systemd unit
			Expect(ignitionConfig.Systemd.Units).To(HaveLen(1))
			unit := ignitionConfig.Systemd.Units[0]
			Expect(unit.Name).To(Equal("set-hostname-oci.service"))
			Expect(*unit.Enabled).To(BeTrue())
			Expect(*unit.Contents).To(ContainSubstring("Description=Set hostname from OCI metadata"))

			// Verify files
			Expect(ignitionConfig.Storage.Files).To(HaveLen(1))
			file := ignitionConfig.Storage.Files[0]
			Expect(file.Node.Path).To(Equal("/usr/local/bin/set-hostname-oci.sh"))
			Expect(*file.Mode).To(Equal(493)) // 0755 in decimal

			// Verify TLS config
			Expect(ignitionConfig.Ignition.Security.TLS.CertificateAuthorities).To(HaveLen(1))
			certAuth := ignitionConfig.Ignition.Security.TLS.CertificateAuthorities[0]
			Expect(*certAuth.Source).To(ContainSubstring("data:text/plain;charset=utf-8;base64,"))

			// Verify merge config
			Expect(ignitionConfig.Ignition.Config.Merge).To(HaveLen(1))
			merge := ignitionConfig.Ignition.Config.Merge[0]
			Expect(*merge.Source).To(Equal("https://api.test-cluster.example.com:22623/config/worker"))
		})

		It("should fail when infrastructure cluster is not found", func() {
			// Delete the infrastructure object
			infrastructure := &configv1.Infrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
			}
			Expect(k8sClient.Delete(ctx, infrastructure)).To(Succeed())

			_, err := GenerateIgnitionConfig(ctx, k8sClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get cluster API server internal URL"))
		})

		It("should fail when machine config CA secret is not found", func() {
			// Delete the machine config secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-config-server-tls",
					Namespace: "openshift-machine-config-operator",
				},
			}
			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())

			_, err := GenerateIgnitionConfig(ctx, k8sClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get machine config CA"))
		})
	})
})
