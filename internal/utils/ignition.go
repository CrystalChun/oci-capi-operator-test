package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/go-openapi/swag"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GenerateIgnitionConfig(ctx context.Context, client client.Client) (string, error) {
	apiServerInternalURL, err := GetClusterAPIServerInternalURL(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster API server internal URL: %w", err)
	}
	apiServerInternalURL = strings.TrimPrefix(apiServerInternalURL, "https://")
	apiServerInternalURL, _, _ = strings.Cut(apiServerInternalURL, ":")

	machineConfigCA, err := GetMachineConfigCA(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get machine config CA: %w", err)
	}
	machineConfigCAB64 := base64.StdEncoding.EncodeToString([]byte(machineConfigCA))

	ignitionConfig := &types.Config{
		Systemd: types.Systemd{
			Units: []types.Unit{
				{
					Name:     "set-hostname-oci.service",
					Enabled:  swag.Bool(true),
					Contents: swag.String("[Unit]\nDescription=Set hostname from OCI metadata\nAfter=network-online.target\nWants=network-online.target\n[Service]\nType=oneshot\nExecStart=/usr/local/bin/set-hostname-oci.sh\n[Install]\nWantedBy=multi-user.target\n"),
				},
			},
		},
		Storage: types.Storage{
			Files: []types.File{
				{
					Node: types.Node{
						Path: "/usr/local/bin/set-hostname-oci.sh",
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source: swag.String("data:text/plain;charset=utf-8;base64,IyEvYmluL2Jhc2ggLXgKCk9DSV9IT1NUTkFNRT0vZXRjL2hvc3RuYW1lLW9jaQplY2hvICJDdXJyZW50IGhvc3RuYW1lOiAkKGhvc3RuYW1lKSIKdW50aWwgW1sgLXMgJE9DSV9IT1NUTkFNRSBdXTsgZG8KICAgIC91c3IvYmluL2N1cmwgLXMgLUggIkF1dGhvcml6YXRpb246IEJlYXJlciBPcmFjbGUiIGh0dHA6Ly8xNjkuMjU0LjE2OS4yNTQvb3BjL3YyL2luc3RhbmNlL2hvc3RuYW1lIC1vICRPQ0lfSE9TVE5BTUUKZG9uZQoKZWNobyAiU2V0dGluZyBob3N0bmFtZSB0byAkKGNhdCAkT0NJX0hPU1ROQU1FKSIKCmNhdCAkT0NJX0hPU1ROQU1FID4gL3Byb2Mvc3lzL2tlcm5lbC9ob3N0bmFtZQo="),
						},
						Mode: swag.Int(493),
					},
				},
			},
		},
		Ignition: types.Ignition{
			Version: "3.2.0",
			Security: types.Security{
				TLS: types.TLS{
					CertificateAuthorities: []types.Resource{
						{
							Source: swag.String(fmt.Sprintf("data:text/plain;charset=utf-8;base64,%s", machineConfigCAB64)),
						},
					},
				},
			},
			Config: types.IgnitionConfig{
				Merge: []types.Resource{
					{
						Source: swag.String(fmt.Sprintf("https://%s:22623/config/worker", apiServerInternalURL)),
					},
				},
			},
		},
	}

	ignitionConfigBytes, err := json.Marshal(ignitionConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ignition config: %w", err)
	}
	return string(ignitionConfigBytes), nil
}
