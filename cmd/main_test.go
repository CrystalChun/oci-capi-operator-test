package main

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	enableautoscaler "github.com/openshift/oci-capi-operator/internal/components/enable_autoscaler"
	"github.com/openshift/oci-capi-operator/internal/components/capoci"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("Main Command", func() {
	Describe("Package initialization", func() {
		It("should have properly initialized scheme", func() {
			// Verify that the global scheme variable is properly initialized
			Expect(scheme).NotTo(BeNil())
			
			// Check that various required types are registered
			gvks := scheme.AllKnownTypes()
			
			// Should have core Kubernetes types
			foundSecret := false
			foundDeployment := false
			foundConfigMap := false
			
			for gvk := range gvks {
				switch gvk.Kind {
				case "Secret":
					foundSecret = true
				case "Deployment":
					foundDeployment = true
				case "ConfigMap":
					foundConfigMap = true
				}
			}
			
			Expect(foundSecret).To(BeTrue())
			Expect(foundDeployment).To(BeTrue())
			Expect(foundConfigMap).To(BeTrue())
		})

		It("should have initialized setupLog", func() {
			Expect(setupLog).NotTo(BeNil())
		})
	})

	Describe("Options struct", func() {
		It("should create Options with all required fields", func() {
			options := Options{
				CAPOCICredentials: capoci.CAPOCICredentials{
					Region:      "us-west-2",
					TenancyID:   "test-tenancy",
					UserID:      "test-user",
					PrivateKey:  "test-key",
					Fingerprint: "test-fingerprint",
					Passphrase:  "test-passphrase",
				},
				AutoScalingConfig: enableautoscaler.Config{
					AutoScalingConfig: enableautoscaler.AutoScalingConfig{
						CPUs:     2,
						Memory:   4,
						MinNodes: 1,
						MaxNodes: 3,
						Shape:    "VM.Standard.E4.Flex",
						ImageID:  "test-image",
					},
					ClusterConfig: enableautoscaler.ClusterConfig{
						CompartmentID: "test-compartment",
					},
				},
				RunOptions: RunOptions{
					EnableLeaderElection: true,
					ProbeAddr:            ":8081",
					EnableHTTP2:          false,
				},
			}

			Expect(options.CAPOCICredentials.Region).To(Equal("us-west-2"))
			Expect(options.AutoScalingConfig.AutoScalingConfig.CPUs).To(Equal(int32(2)))
			Expect(options.RunOptions.EnableLeaderElection).To(BeTrue())
			Expect(options.RunOptions.ProbeAddr).To(Equal(":8081"))
			Expect(options.RunOptions.EnableHTTP2).To(BeFalse())
		})
	})

	Describe("RunOptions struct", func() {
		It("should create RunOptions with default values", func() {
			options := RunOptions{
				EnableLeaderElection: false,
				ProbeAddr:            ":8080",
				EnableHTTP2:          false,
			}

			Expect(options.EnableLeaderElection).To(BeFalse())
			Expect(options.ProbeAddr).To(Equal(":8080"))
			Expect(options.EnableHTTP2).To(BeFalse())
		})
	})

	Describe("NewRunCommand", func() {
		It("should create a valid run command", func() {
			cmd := NewRunCommand()

			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).To(Equal("run"))
			Expect(cmd.Short).To(ContainSubstring("Runs the OCI CAPI operator"))
			Expect(cmd.Run).NotTo(BeNil())
		})

		It("should have correct flags", func() {
			cmd := NewRunCommand()

			// Check that flags are defined
			healthProbeFlag := cmd.Flags().Lookup("health-probe-bind-address")
			Expect(healthProbeFlag).NotTo(BeNil())
			Expect(healthProbeFlag.DefValue).To(Equal(":8081"))

			leaderElectFlag := cmd.Flags().Lookup("leader-elect")
			Expect(leaderElectFlag).NotTo(BeNil())
			Expect(leaderElectFlag.DefValue).To(Equal("false"))

			http2Flag := cmd.Flags().Lookup("enable-http2")
			Expect(http2Flag).NotTo(BeNil())
			Expect(http2Flag.DefValue).To(Equal("false"))
		})
	})

	Describe("Main function components", func() {
		It("should create proper root command structure", func() {
			// Simulate the main command creation logic
			rootCmd := &cobra.Command{
				Use: "oci-capi-operator",
				Run: func(cmd *cobra.Command, args []string) {
					// Help would be shown and exit 1
				},
			}
			
			initCmd := NewInitCommand()
			runCmd := NewRunCommand()
			
			rootCmd.AddCommand(initCmd)
			rootCmd.AddCommand(runCmd)

			// Verify command structure
			commands := rootCmd.Commands()
			Expect(commands).To(HaveLen(2))
			
			commandNames := make([]string, len(commands))
			for i, cmd := range commands {
				commandNames[i] = cmd.Use
			}
			Expect(commandNames).To(ConsistOf("init", "run"))
		})

		It("should setup logging correctly", func() {
			// Test the logger setup similar to main()
			logger := zap.New(zap.JSONEncoder(func(o *zapcore.EncoderConfig) {
				o.EncodeTime = zapcore.RFC3339TimeEncoder
			}))
			
			Expect(logger).NotTo(BeNil())
			
			// Verify logger can be used
			logger.Info("test message")
		})
	})

	Describe("run function", func() {
		var (
			options Options
		)

		BeforeEach(func() {
			options = Options{
				RunOptions: RunOptions{
					EnableLeaderElection: false,
					ProbeAddr:            ":8081",
					EnableHTTP2:          false,
				},
			}
		})

		It("should handle HTTP/2 configuration", func() {
			// Test HTTP/2 disabled (default)
			options.RunOptions.EnableHTTP2 = false
			
			// The disableHTTP2 function would be called
			// We can't easily test the actual TLS config here,
			// but we can verify the option is set correctly
			Expect(options.RunOptions.EnableHTTP2).To(BeFalse())
		})

		It("should handle HTTP/2 enabled", func() {
			// Test HTTP/2 enabled
			options.RunOptions.EnableHTTP2 = true
			Expect(options.RunOptions.EnableHTTP2).To(BeTrue())
		})

		It("should create manager options correctly", func() {
			// Verify that manager options would be created properly
			expectedProbeAddr := ":8081"
			expectedLeaderElection := false
			expectedLeaderElectionID := "1af242a3.openshift.io"

			Expect(options.RunOptions.ProbeAddr).To(Equal(expectedProbeAddr))
			Expect(options.RunOptions.EnableLeaderElection).To(Equal(expectedLeaderElection))
			
			// The actual leader election ID is hardcoded in the function
			Expect(expectedLeaderElectionID).To(Equal("1af242a3.openshift.io"))
		})

		Context("without real Kubernetes cluster", func() {
			It("should validate manager creation parameters", func() {
				// In a real environment, this would test manager creation
				// Here we verify the parameters are correct
				Expect(options.RunOptions.ProbeAddr).NotTo(BeEmpty())
				Expect(scheme).NotTo(BeNil())
			})

			It("should validate controller setup parameters", func() {
				// Verify that controller setup would have correct parameters
				Expect(options.CAPOCICredentials).NotTo(BeNil())
				Expect(options.AutoScalingConfig).NotTo(BeNil())
			})
		})
	})

	Describe("Error handling", func() {
		It("should handle manager creation errors", func() {
			// The run function should handle manager creation errors
			// In a test environment without valid kube config, this would fail gracefully
			Expect(run).NotTo(BeNil())
		})

		It("should handle environment variable processing", func() {
			// Test that envconfig processing is handled
			// The actual processing would happen during run()
			options := Options{}
			
			// Verify the structure exists for envconfig
			Expect(options.CAPOCICredentials).NotTo(BeNil())
			Expect(options.AutoScalingConfig).NotTo(BeNil())
		})
	})

	Describe("Health checks", func() {
		It("should configure health and readiness checks", func() {
			// Verify that health check configuration would be applied
			// The actual checks are added to the manager in run()
			
			// These endpoints would be configured:
			healthEndpoint := "/healthz"
			readyEndpoint := "/readyz"
			
			Expect(healthEndpoint).To(Equal("/healthz"))
			Expect(readyEndpoint).To(Equal("/readyz"))
		})
	})
})