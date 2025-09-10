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
							Source: swag.String("data:text/plain;charset=utf-8;base64,IyEvYmluL2Jhc2gKc2V0IC1ldW8gcGlwZWZhaWwKCiMgR2V0IGhvc3RuYW1lIGZyb20gT0NJIG1ldGFkYXRhCmhvc3RuYW1lPSQoY3VybCAtcyBodHRwOi8vMTY5LjI1NC4xNjkuMjU0L29wYy92Mi9pbnN0YW5jZS8gfCBqcSAtciAuZGlzcGxheU5hbWUpCgppZiBbIC1uICIkaG9zdG5hbWUiIF07IHRoZW4KICAgIGVjaG8gIlNldHRpbmcgaG9zdG5hbWUgdG8gJGhvc3RuYW1lIgogICAgaG9zdG5hbWVjdGwgc2V0LWhvc3RuYW1lICIkaG9zdG5hbWUiCmVsc2UKICAgIGVjaG8gIkZhaWxlZCB0byBnZXQgaG9zdG5hbWUgZnJvbSBPQ0kgbWV0YWRhdGEiCiAgICBleGl0IDEKZmkK"),
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
