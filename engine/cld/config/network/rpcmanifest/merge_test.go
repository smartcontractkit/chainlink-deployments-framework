package rpcmanifest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

const (
	sepoliaSelector = uint64(16015286601757825753)
	// noNodesSelector is in testdata/rpcs.json with an empty node list.
	noNodesSelector = uint64(42)
	// unknownSelector is not in testdata/rpcs.json.
	unknownSelector = uint64(123)
)

// loadTestManifest loads testdata/rpcs.json over HTTP and returns it with the server origin.
func loadTestManifest(t *testing.T) (*Manifest, string) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "testdata/rpcs.json")
	}))
	t.Cleanup(srv.Close)

	manifest, err := Load(context.Background(), srv.URL+"/files/rpcs.json")
	require.NoError(t, err)

	return manifest, srv.URL
}

func testNetwork(n cfgnet.Network) cfgnet.Network {
	n.Type = cfgnet.NetworkTypeTestnet
	n.BlockExplorer = cfgnet.BlockExplorer{Type: "etherscan", URL: "https://example.com/api"}
	n.Metadata = map[string]any{"custom_field": "custom_value"}

	return n
}

func TestMergeInto(t *testing.T) {
	t.Parallel()

	manifest, origin := loadTestManifest(t)

	var (
		own       = cfgnet.RPC{RPCName: "Mine", PreferredURLScheme: "http", HTTPURL: "https://mine.example.com"}
		linkpool  = "LinkPool (linkpool1)"
		alchemy   = "Alchemy (alchemy1)"
		linkpoolH = origin + "/16015286601757825753/linkpool1"
	)

	tests := []struct {
		name         string
		give         cfgnet.Network
		wantRPCNames []string
		wantErr      string
	}{
		{
			name:         "empty rpcs are filled from the manifest",
			give:         cfgnet.Network{ChainSelector: sepoliaSelector},
			wantRPCNames: []string{linkpool, alchemy},
		},
		{
			name:         "yaml rpcs are kept by default",
			give:         cfgnet.Network{ChainSelector: sepoliaSelector, RPCs: []cfgnet.RPC{own}},
			wantRPCNames: []string{"Mine"},
		},
		{
			name: "yaml rpcs are kept in fallback mode",
			give: cfgnet.Network{
				ChainSelector: sepoliaSelector, RPCs: []cfgnet.RPC{own}, RPCsMode: cfgnet.RPCsModeFallback,
			},
			wantRPCNames: []string{"Mine"},
		},
		{
			name:         "yaml rpcs are kept for a chain missing from the manifest",
			give:         cfgnet.Network{ChainSelector: unknownSelector, RPCs: []cfgnet.RPC{own}},
			wantRPCNames: []string{"Mine"},
		},
		{
			name: "append mode puts yaml rpcs first",
			give: cfgnet.Network{
				ChainSelector: sepoliaSelector, RPCs: []cfgnet.RPC{own}, RPCsMode: cfgnet.RPCsModeAppend,
			},
			wantRPCNames: []string{"Mine", linkpool, alchemy},
		},
		{
			name: "append mode skips manifest rpcs with a listed HTTP URL",
			give: cfgnet.Network{
				ChainSelector: sepoliaSelector,
				RPCs:          []cfgnet.RPC{{RPCName: "Pinned", HTTPURL: linkpoolH}},
				RPCsMode:      cfgnet.RPCsModeAppend,
			},
			wantRPCNames: []string{"Pinned", alchemy},
		},
		{
			name:         "append mode with empty rpcs is filled from the manifest",
			give:         cfgnet.Network{ChainSelector: sepoliaSelector, RPCsMode: cfgnet.RPCsModeAppend},
			wantRPCNames: []string{linkpool, alchemy},
		},
		{
			name: "append mode keeps yaml rpcs when the manifest chain has no nodes",
			give: cfgnet.Network{
				ChainSelector: noNodesSelector, RPCs: []cfgnet.RPC{own}, RPCsMode: cfgnet.RPCsModeAppend,
			},
			wantRPCNames: []string{"Mine"},
		},
		{
			name: "append mode fails for a chain missing from the manifest",
			give: cfgnet.Network{
				ChainSelector: unknownSelector, RPCs: []cfgnet.RPC{own}, RPCsMode: cfgnet.RPCsModeAppend,
			},
			wantErr: "chain selector 123 uses the rpc manifest",
		},
		{
			name: "unrecognized rpcs_mode fails",
			give: cfgnet.Network{
				ChainSelector: unknownSelector, RPCs: []cfgnet.RPC{own}, RPCsMode: "sideways",
			},
			wantErr: "rpcs_mode",
		},
		{
			name:    "empty rpcs fail for a chain missing from the manifest",
			give:    cfgnet.Network{ChainSelector: unknownSelector},
			wantErr: "chain selector 123 uses the rpc manifest",
		},
		{
			name:    "empty rpcs fail when the manifest chain has no nodes",
			give:    cfgnet.Network{ChainSelector: noNodesSelector},
			wantErr: "no usable provider nodes in rpc manifest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			give := testNetwork(tt.give)

			merged, err := MergeInto(cfgnet.NewConfig([]cfgnet.Network{give}), manifest, logger.Test(t))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
			require.Len(t, merged.Networks(), 1)

			got, err := merged.NetworkBySelector(give.ChainSelector)
			require.NoError(t, err)

			names := make([]string, 0, len(got.RPCs))
			for _, rpc := range got.RPCs {
				names = append(names, rpc.RPCName)
			}
			assert.Equal(t, tt.wantRPCNames, names)

			// Only rpcs may differ from the input.
			got.RPCs = give.RPCs
			assert.Equal(t, give, got)
		})
	}
}

func TestMergeInto_ProviderURLs(t *testing.T) {
	t.Parallel()

	manifest, origin := loadTestManifest(t)

	merged, err := MergeInto(
		cfgnet.NewConfig([]cfgnet.Network{testNetwork(cfgnet.Network{ChainSelector: sepoliaSelector})}),
		manifest, logger.Test(t),
	)
	require.NoError(t, err)

	got, err := merged.NetworkBySelector(sepoliaSelector)
	require.NoError(t, err)
	assert.Equal(t, []cfgnet.RPC{
		{RPCName: "LinkPool (linkpool1)", PreferredURLScheme: "http", HTTPURL: origin + "/16015286601757825753/linkpool1"},
		{RPCName: "Alchemy (alchemy1)", PreferredURLScheme: "http", HTTPURL: origin + "/16015286601757825753/alchemy1"},
	}, got.RPCs)
}

func TestMergeInto_AppendIsIdempotent(t *testing.T) {
	t.Parallel()

	manifest, _ := loadTestManifest(t)
	cfg := cfgnet.NewConfig([]cfgnet.Network{testNetwork(cfgnet.Network{
		ChainSelector: sepoliaSelector,
		RPCs:          []cfgnet.RPC{{RPCName: "Mine", HTTPURL: "https://mine.example.com"}},
		RPCsMode:      cfgnet.RPCsModeAppend,
	})})

	once, err := MergeInto(cfg, manifest, logger.Test(t))
	require.NoError(t, err)

	twice, err := MergeInto(once, manifest, logger.Test(t))
	require.NoError(t, err)
	assert.Equal(t, once, twice)
}

func TestMergeInto_NilArguments(t *testing.T) {
	t.Parallel()

	_, err := MergeInto(nil, &Manifest{}, logger.Test(t))
	require.ErrorContains(t, err, "yaml network config is nil")

	_, err = MergeInto(cfgnet.NewConfig(nil), nil, logger.Test(t))
	require.ErrorContains(t, err, "rpc manifest is nil")
}
