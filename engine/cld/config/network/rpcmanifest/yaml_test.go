package rpcmanifest_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network/rpcmanifest"
)

func TestWriteConfigYAML_EmptyNetworks(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	require.NoError(t, rpcmanifest.WriteConfigYAML(cfgnet.NewConfig(nil), &buf))
	assert.Contains(t, buf.String(), "networks: []")
}

func TestWriteConfigYAML_SortsBySelector(t *testing.T) {
	t.Parallel()

	cfg := cfgnet.NewConfig([]cfgnet.Network{
		{Type: cfgnet.NetworkTypeTestnet, ChainSelector: 300, RPCs: []cfgnet.RPC{{RPCName: "c", HTTPURL: "https://c.example.com"}}},
		{Type: cfgnet.NetworkTypeTestnet, ChainSelector: 100, RPCs: []cfgnet.RPC{{RPCName: "a", HTTPURL: "https://a.example.com"}}},
		{Type: cfgnet.NetworkTypeTestnet, ChainSelector: 200, RPCs: []cfgnet.RPC{{RPCName: "b", HTTPURL: "https://b.example.com"}}},
	})

	var buf strings.Builder
	require.NoError(t, rpcmanifest.WriteConfigYAML(cfg, &buf))

	out := buf.String()
	posA := strings.Index(out, "chain_selector: 100")
	posB := strings.Index(out, "chain_selector: 200")
	posC := strings.Index(out, "chain_selector: 300")
	require.NotEqual(t, -1, posA)
	assert.True(t, posA < posB && posB < posC, "networks not sorted by chain_selector:\n%s", out)
}

func TestWriteConfigYAML_RoundTrips(t *testing.T) {
	t.Parallel()

	want := cfgnet.Network{
		Type:          cfgnet.NetworkTypeTestnet,
		ChainSelector: 100,
		BlockExplorer: cfgnet.BlockExplorer{Type: "etherscan", URL: "https://example.com/api"},
		RPCs: []cfgnet.RPC{
			{RPCName: "a", PreferredURLScheme: "http", HTTPURL: "https://a.example.com", WSURL: "wss://a.example.com"},
		},
	}

	var buf strings.Builder
	require.NoError(t, rpcmanifest.WriteConfigYAML(cfgnet.NewConfig([]cfgnet.Network{want}), &buf))

	var parsed cfgnet.Config
	require.NoError(t, yaml.Unmarshal([]byte(buf.String()), &parsed), buf.String())

	got, err := parsed.NetworkBySelector(100)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestWriteConfigYAML_AliasesPreferredURLScheme(t *testing.T) {
	t.Parallel()

	cfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 100,
			RPCs: []cfgnet.RPC{
				{RPCName: "a", PreferredURLScheme: "http", HTTPURL: "https://a.example.com"},
				{RPCName: "b", PreferredURLScheme: "http", HTTPURL: "https://b.example.com"},
			},
		},
	})

	var buf strings.Builder
	require.NoError(t, rpcmanifest.WriteConfigYAML(cfg, &buf))

	out := buf.String()
	assert.Contains(t, out, `preferred_url_scheme: &preferred_url_scheme "http"`)
	assert.Equal(t, 2, strings.Count(out, "preferred_url_scheme: *preferred_url_scheme"), out)
}

func TestWriteConfigYAML_NilArguments(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	require.ErrorContains(t, rpcmanifest.WriteConfigYAML(nil, &buf), "network config is nil")
	require.ErrorContains(t, rpcmanifest.WriteConfigYAML(cfgnet.NewConfig(nil), nil), "writer is nil")
}
