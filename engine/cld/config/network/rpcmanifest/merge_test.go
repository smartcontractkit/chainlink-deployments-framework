package rpcmanifest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network/rpcmanifest"
)

const (
	sepoliaSelector = uint64(16015286601757825753)
	// noNodesSelector is in testdata/rpcs.json with an empty node list.
	noNodesSelector = uint64(42)
	// unknownSelector is not in testdata/rpcs.json.
	unknownSelector = uint64(123)
)

func loadTestManifest(t *testing.T) *rpcmanifest.Manifest {
	t.Helper()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	return manifest
}

func TestMergeInto(t *testing.T) {
	t.Parallel()

	var (
		own       = cfgnet.RPC{RPCName: "Mine", PreferredURLScheme: "http", HTTPURL: "https://mine.example.com"}
		linkpool  = "LinkPool (linkpool1)"
		alchemy   = "Alchemy (alchemy1)"
		linkpoolH = "https://rpcs.cldev.sh/16015286601757825753/linkpool1"
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
			name: "yaml rpcs are kept in replace mode",
			give: cfgnet.Network{
				ChainSelector: sepoliaSelector, RPCs: []cfgnet.RPC{own}, RPCsMode: cfgnet.RPCsModeReplace,
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
			wantErr: "no provider nodes in rpc manifest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.give.Type = cfgnet.NetworkTypeTestnet
			tt.give.BlockExplorer = cfgnet.BlockExplorer{Type: "etherscan", URL: "https://example.com/api"}
			tt.give.Metadata = map[string]any{"custom_field": "custom_value"}

			merged, err := rpcmanifest.MergeInto(
				cfgnet.NewConfig([]cfgnet.Network{tt.give}), loadTestManifest(t), logger.Test(t),
			)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
			require.Len(t, merged.Networks(), 1)

			got, err := merged.NetworkBySelector(tt.give.ChainSelector)
			require.NoError(t, err)

			names := make([]string, 0, len(got.RPCs))
			for _, rpc := range got.RPCs {
				names = append(names, rpc.RPCName)
			}
			assert.Equal(t, tt.wantRPCNames, names)
			assert.Empty(t, got.RPCsMode)

			// Only rpcs and rpcs_mode may differ from the input.
			got.RPCs, got.RPCsMode = tt.give.RPCs, tt.give.RPCsMode
			assert.Equal(t, tt.give, got)
		})
	}
}

func TestMergeInto_NilArguments(t *testing.T) {
	t.Parallel()

	_, err := rpcmanifest.MergeInto(nil, loadTestManifest(t), logger.Test(t))
	require.ErrorContains(t, err, "yaml network config is nil")

	_, err = rpcmanifest.MergeInto(cfgnet.NewConfig(nil), nil, logger.Test(t))
	require.ErrorContains(t, err, "rpc manifest is nil")
}
