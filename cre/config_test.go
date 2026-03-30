package cre

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteProjectYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := ProjectConfig{
		"cld-deploy": {
			CreCLI: ProjectTargetCRECLI{DonFamily: "zone-a"},
			RPCs: []RPCEntry{
				{ChainName: "ethereum-testnet-sepolia-linea-1", URL: "https://rpc.example/rpc"},
			},
		},
	}
	path, err := WriteProjectYAML(dir, cfg)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "project.yaml"), path)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(b)
	require.Contains(t, s, "cld-deploy:")
	require.Contains(t, s, "don-family: zone-a")
	require.Contains(t, s, "chain-name: ethereum-testnet-sepolia-linea-1")
	require.Contains(t, s, "url: https://rpc.example/rpc")

	fi, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
}

func TestWriteWorkflowYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := WorkflowConfig{
		"cld-deploy": {
			UserWorkflow: UserWorkflow{
				DeploymentRegistry: "private",
				WorkflowName:       "my-wf",
			},
			WorkflowArtifacts: WorkflowArtifacts{
				WorkflowPath: ".",
				ConfigPath:   "./config.json",
			},
		},
	}
	path, err := WriteWorkflowYAML(dir, cfg)
	require.NoError(t, err)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(b)
	require.Contains(t, s, "deployment-registry: private")
	require.Contains(t, s, "workflow-name: my-wf")
	require.Contains(t, s, "workflow-path: .")
}

func TestWriteContextYAML_privateOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := ContextConfig{
		"PRODUCTION": {
			TenantID:   "cre-cll",
			DonFamily:  "zone-a",
			GatewayURL: "https://gw.example",
			Registries: []ContextRegistryEntry{
				{
					ID:               "private",
					Label:            "Private (Chainlink-hosted)",
					Type:             "off-chain",
					SecretsAuthFlows: []string{"browser"},
				},
			},
		},
	}
	path, err := WriteContextYAML(dir, cfg)
	require.NoError(t, err)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(b)
	require.Contains(t, s, "PRODUCTION:")
	require.Contains(t, s, "tenant_id: cre-cll")
	require.Contains(t, s, "secrets_auth_flows:")
	require.Contains(t, s, "type: off-chain")
}

func TestWriteContextYAML_onchainRegistry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := ContextConfig{
		"PRODUCTION": {
			TenantID:   "cre-mainline",
			DonFamily:  "zone-a",
			GatewayURL: "https://01.gateway.zone-a.cre.chain.link",
			Registries: []ContextRegistryEntry{
				{
					ID:               "onchain:ethereum-testnet-sepolia",
					Label:            "ethereum-testnet-sepolia (on-chain)",
					Type:             "on-chain",
					Address:          "0xaE55eB3EDAc48a1163EE2cbb1205bE1e90Ea1135",
					ChainName:        "ethereum-testnet-sepolia",
					SecretsAuthFlows: []string{"browser", "owner-key-signing"},
				},
			},
		},
	}
	path, err := WriteContextYAML(dir, cfg)
	require.NoError(t, err)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(b)
	require.Contains(t, s, "onchain:ethereum-testnet-sepolia")
	require.Contains(t, s, "chain_name: ethereum-testnet-sepolia")
	require.Contains(t, s, "0xaE55eB3EDAc48a1163EE2cbb1205bE1e90Ea1135")
}
