package rpcmanifest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network/rpcmanifest"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	entry, ok := manifest.Chain(16015286601757825753)
	require.True(t, ok)
	assert.Equal(t, "11155111", entry.ChainID)
	assert.Len(t, entry.Nodes, 2)
}

func TestRPCsForChain(t *testing.T) {
	t.Parallel()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	entry, ok := manifest.Chain(16015286601757825753)
	require.True(t, ok)

	rpcs := rpcmanifest.RPCsForChain(16015286601757825753, entry)
	require.Len(t, rpcs, 3)
	assert.Equal(t, "CLL Proxy", rpcs[0].RPCName)
	assert.Equal(t, "https://rpcs.cldev.sh/16015286601757825753", rpcs[0].HTTPURL)
	assert.Equal(t, "wss://rpcs.cldev.sh/16015286601757825753", rpcs[0].WSURL)
	assert.Equal(t, "LinkPool (linkpool1)", rpcs[1].RPCName)
	assert.Equal(t, "https://rpcs.cldev.sh/16015286601757825753/linkpool1", rpcs[1].HTTPURL)
}

func TestMergeInto_PopulatesMissingRPCs(t *testing.T) {
	t.Parallel()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	yamlCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 16015286601757825753,
		},
	})

	merged, err := rpcmanifest.MergeInto(yamlCfg, manifest, logger.Test(t))
	require.NoError(t, err)

	net, err := merged.NetworkBySelector(16015286601757825753)
	require.NoError(t, err)
	require.Len(t, net.RPCs, 3)
	assert.Equal(t, "CLL Proxy", net.RPCs[0].RPCName)
}

func TestMergeInto_YAMLOverridesManifestRPCs(t *testing.T) {
	t.Parallel()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	yamlCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 16015286601757825753,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "Custom Override",
					PreferredURLScheme: "http",
					HTTPURL:            "https://example.com/rpc",
				},
			},
		},
	})

	merged, err := rpcmanifest.MergeInto(yamlCfg, manifest, logger.Test(t))
	require.NoError(t, err)

	net, err := merged.NetworkBySelector(16015286601757825753)
	require.NoError(t, err)
	require.Len(t, net.RPCs, 1)
	assert.Equal(t, "Custom Override", net.RPCs[0].RPCName)
}

func TestMergeInto_MissingManifestChainWithoutRPCs(t *testing.T) {
	t.Parallel()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	yamlCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 123,
		},
	})

	_, err = rpcmanifest.MergeInto(yamlCfg, manifest, logger.Test(t))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not present in rpc manifest")
}

func TestMergeIntoWithOptions_SkipMissingManifestChain(t *testing.T) {
	t.Parallel()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	yamlCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 123,
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 16015286601757825753,
		},
	})

	merged, err := rpcmanifest.MergeIntoWithOptions(yamlCfg, manifest, logger.Test(t), rpcmanifest.MergeOptions{
		SkipMissingManifestChains: true,
	})
	require.NoError(t, err)
	require.Len(t, merged.Networks(), 1)

	net, err := merged.NetworkBySelector(16015286601757825753)
	require.NoError(t, err)
	require.NotEmpty(t, net.RPCs)
}

func TestWriteConfigYAML_EmptyNetworks(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	err := rpcmanifest.WriteConfigYAML(cfgnet.NewConfig(nil), &buf)
	require.NoError(t, err)
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
	require.NotEqual(t, -1, posB)
	require.NotEqual(t, -1, posC)
	assert.True(t, posA < posB && posB < posC, "expected networks sorted by chain_selector, got:\n%s", out)
}

func TestWriteConfigYAML_RoundTrips(t *testing.T) {
	t.Parallel()

	cfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 100,
			BlockExplorer: cfgnet.BlockExplorer{Type: "etherscan", URL: "https://example.com/api"},
			RPCs: []cfgnet.RPC{
				{RPCName: "a", PreferredURLScheme: "http", HTTPURL: "https://a.example.com", WSURL: "wss://a.example.com"},
			},
		},
	})

	var buf strings.Builder
	require.NoError(t, rpcmanifest.WriteConfigYAML(cfg, &buf))

	var parsed cfgnet.Config
	require.NoError(t, yaml.Unmarshal([]byte(buf.String()), &parsed), "output must be valid, parseable networks yaml:\n%s", buf.String())

	net, err := parsed.NetworkBySelector(100)
	require.NoError(t, err)
	assert.Equal(t, cfgnet.NetworkTypeTestnet, net.Type)
	assert.Equal(t, "etherscan", net.BlockExplorer.Type)
	require.Len(t, net.RPCs, 1)
	assert.Equal(t, "a", net.RPCs[0].RPCName)
	assert.Equal(t, "http", net.RPCs[0].PreferredURLScheme)
	assert.Equal(t, "https://a.example.com", net.RPCs[0].HTTPURL)
	assert.Equal(t, "wss://a.example.com", net.RPCs[0].WSURL)
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
	assert.Equal(t, 2, strings.Count(out, "preferred_url_scheme: *preferred_url_scheme"),
		"expected both RPC entries to alias the anchor instead of repeating the literal value:\n%s", out)
	assert.NotContains(t, out, "preferred_url_scheme: http\n",
		"RPC entries should reference the anchor, not repeat the literal scheme:\n%s", out)
}

func TestWriteConfigYAML_NilConfig(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	err := rpcmanifest.WriteConfigYAML(nil, &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestWriteConfigYAML_NilWriter(t *testing.T) {
	t.Parallel()

	err := rpcmanifest.WriteConfigYAML(cfgnet.NewConfig(nil), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestMergeInto_MetadataIsAlwaysLeftUntouched(t *testing.T) {
	t.Parallel()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	// newMetadata is called twice per case — once to build the input, once to build the expected
	// value — so input and expected are always independent instances. Comparing a value against
	// itself (e.g. sharing a *cfgnet.AnvilConfig or a map[string]any between the two) would make
	// an in-place mutation invisible to assert.Equal, since both sides would change together.
	tests := []struct {
		name        string
		newMetadata func() any
	}{
		{name: "nil", newMetadata: func() any { return nil }},
		{name: "typed EVMMetadata", newMetadata: func() any {
			return cfgnet.EVMMetadata{AnvilConfig: &cfgnet.AnvilConfig{Image: "ghcr.io/foundry-rs/foundry:latest", Port: 8550}}
		}},
		{name: "pointer to EVMMetadata", newMetadata: func() any {
			return &cfgnet.EVMMetadata{AnvilConfig: &cfgnet.AnvilConfig{Image: "ghcr.io/foundry-rs/foundry:latest", Port: 8550}}
		}},
		{name: "map with anvil_config", newMetadata: func() any {
			return map[string]any{
				"custom_field": "custom_value",
				"anvil_config": map[string]any{"image": "ghcr.io/foundry-rs/foundry:latest", "port": 8550},
			}
		}},
		{name: "map without anvil_config", newMetadata: func() any {
			return map[string]any{"custom_field": "custom_value"}
		}},
		{name: "unrecognized scalar", newMetadata: func() any { return "not-a-map" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.newMetadata()
			yamlCfg := cfgnet.NewConfig([]cfgnet.Network{
				{
					Type:          cfgnet.NetworkTypeTestnet,
					ChainSelector: 16015286601757825753,
					Metadata:      input,
				},
			})

			merged, err := rpcmanifest.MergeInto(yamlCfg, manifest, logger.Test(t))
			require.NoError(t, err)

			net, err := merged.NetworkBySelector(16015286601757825753)
			require.NoError(t, err)

			expected := tt.newMetadata()
			assert.Equal(t, expected, net.Metadata, "MergeInto must never modify metadata — it only fills in rpcs")

			// This chain is in the manifest with an archive node (testdata/rpcs.json), so if
			// archive-URL filling were ever reintroduced, this would catch it directly rather
			// than relying only on the (aliasing-prone) assert.Equal above.
			switch md := net.Metadata.(type) {
			case cfgnet.EVMMetadata:
				if md.AnvilConfig != nil {
					assert.Empty(t, md.AnvilConfig.ArchiveHTTPURL)
				}
			case *cfgnet.EVMMetadata:
				if md != nil && md.AnvilConfig != nil {
					assert.Empty(t, md.AnvilConfig.ArchiveHTTPURL)
				}
			case map[string]any:
				if anvil, ok := md["anvil_config"].(map[string]any); ok {
					assert.NotContains(t, anvil, "archive_http_url")
				}
			}
		})
	}
}
