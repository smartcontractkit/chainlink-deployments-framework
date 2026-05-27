package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	fcre "github.com/smartcontractkit/chainlink-deployments-framework/cre"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
)

func TestBuildContextConfig(t *testing.T) {
	t.Parallel()

	domainPrivate := []fcre.ContextRegistryEntry{
		{
			ID:               "private",
			Label:            "Private (Chainlink-hosted)",
			Type:             "off-chain",
			SecretsAuthFlows: []string{"browser", "owner-key-signing"},
		},
	}

	tests := []struct {
		name             string
		donFamily        string
		contextOverrides ContextOverrides
		cfg              cfgenv.CREConfig
		domainRegistries []fcre.ContextRegistryEntry
		wantErr          string
		check            func(t *testing.T, cfg ContextConfig)
	}{
		{
			name:      "falls back to CREConfig values",
			donFamily: "feeds-zone",
			cfg: cfgenv.CREConfig{
				CLIEnv: "STAGING", GatewayURL: "https://gw.example",
				Auth: cfgenv.CREAuthConfig{TenantID: "env-tenant"},
			},
			domainRegistries: domainPrivate,
			check: func(t *testing.T, cfg ContextConfig) {
				t.Helper()
				require.Len(t, cfg, 1)
				ce := cfg["STAGING"]
				require.Equal(t, "env-tenant", ce.TenantID)
				require.Equal(t, "feeds-zone", ce.DonFamily)
				require.Equal(t, "https://gw.example", ce.GatewayURL)
				require.Len(t, ce.Registries, 1)
				require.Equal(t, "private", ce.Registries[0].ID)
			},
		},
		{
			name:      "input overrides take precedence",
			donFamily: "feeds-zone",
			contextOverrides: ContextOverrides{
				TenantID: "yaml-tenant", GatewayURL: "https://gw.override",
			},
			cfg: cfgenv.CREConfig{
				CLIEnv: "STAGING", GatewayURL: "https://gw.example",
				Auth: cfgenv.CREAuthConfig{TenantID: "env-tenant"},
			},
			domainRegistries: domainPrivate,
			check: func(t *testing.T, cfg ContextConfig) {
				t.Helper()
				ce := cfg["STAGING"]
				require.Equal(t, "yaml-tenant", ce.TenantID)
				require.Equal(t, "https://gw.override", ce.GatewayURL)
			},
		},
		{
			name:             "defaults to PRODUCTION when CLIEnv is empty",
			donFamily:        "z",
			cfg:              cfgenv.CREConfig{},
			domainRegistries: domainPrivate,
			check: func(t *testing.T, cfg ContextConfig) {
				t.Helper()
				require.Contains(t, cfg, defaultEnvName)
			},
		},
		{
			name:      "context registries replace domain defaults",
			donFamily: "z",
			contextOverrides: ContextOverrides{
				Registries: []fcre.ContextRegistryEntry{
					{ID: "custom", Label: "Custom", Type: "on-chain"},
				},
			},
			cfg:              cfgenv.CREConfig{},
			domainRegistries: domainPrivate,
			check: func(t *testing.T, cfg ContextConfig) {
				t.Helper()
				ce := cfg[defaultEnvName]
				require.Len(t, ce.Registries, 1)
				require.Equal(t, "custom", ce.Registries[0].ID)
			},
		},
		{
			name:             "empty domain and empty context registries returns error",
			donFamily:        "z",
			cfg:              cfgenv.CREConfig{},
			domainRegistries: nil,
			wantErr:          "CRE context registries: empty after merge",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := BuildContextConfig(tc.donFamily, tc.contextOverrides, tc.cfg, tc.domainRegistries)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, got)

				return
			}
			require.NoError(t, err)
			tc.check(t, got)
		})
	}
}

func TestIsOnChainRegistry(t *testing.T) {
	t.Parallel()

	regs := []fcre.ContextRegistryEntry{
		{ID: "private", Label: "Private", Type: "off-chain"},
		{ID: "onchain-reg", Label: "On-Chain", Type: "on-chain"},
	}

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "on-chain match", id: "onchain-reg", want: true},
		{name: "off-chain match", id: "private", want: false},
		{name: "no match", id: "unknown", want: false},
		{name: "empty list", id: "private"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			list := regs
			if tc.name == "empty list" {
				list = nil
			}
			require.Equal(t, tc.want, IsOnChainRegistry(tc.id, list))
		})
	}
}

func TestFlatRegistries(t *testing.T) {
	t.Parallel()

	cfg := ContextConfig{
		defaultEnvName: {Registries: []fcre.ContextRegistryEntry{
			{ID: "a", Type: "on-chain"},
		}},
		"STAGING": {Registries: []fcre.ContextRegistryEntry{
			{ID: "b", Type: "off-chain"},
		}},
	}
	got := FlatRegistries(cfg)
	require.Len(t, got, 2)

	ids := map[string]bool{}
	for _, r := range got {
		ids[r.ID] = true
	}
	require.True(t, ids["a"])
	require.True(t, ids["b"])
}

func TestWriteContextYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := ContextConfig{
		defaultEnvName: {
			TenantID:   "t",
			DonFamily:  "zone-a",
			GatewayURL: "https://gw",
			Registries: []fcre.ContextRegistryEntry{{ID: "private", Label: "Private", Type: "off-chain"}},
		},
	}
	path, err := WriteContextYAML(dir, cfg)
	require.NoError(t, err)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var got ContextConfig
	require.NoError(t, yaml.Unmarshal(raw, &got))
	require.Equal(t, "t", got[defaultEnvName].TenantID)
	require.Len(t, got[defaultEnvName].Registries, 1)
}
