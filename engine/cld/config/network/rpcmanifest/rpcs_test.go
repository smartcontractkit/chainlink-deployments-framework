package rpcmanifest_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network/rpcmanifest"
)

func TestRPCsForChain_IncludesAllNamedNodes(t *testing.T) {
	t.Parallel()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	entry, ok := manifest.Chain(9999999999999999999)
	require.True(t, ok)

	rpcs := rpcmanifest.RPCsForChain(9999999999999999999, entry)
	require.Len(t, rpcs, 1)
	assert.Equal(t, "Public (public1)", rpcs[0].RPCName)
	assert.Equal(t, "wss://rpcs.cldev.sh/9999999999999999999/public1", rpcs[0].WSURL)
}

func TestRPCsForChain_SkipsNodesWithoutNumericalID(t *testing.T) {
	t.Parallel()

	entry := rpcmanifest.ChainEntry{
		ChainSelector: "42",
		Nodes: []rpcmanifest.Node{
			{Provider: "NoNumericalID", ID: rpcmanifest.NodeID{Stable: "stable-only"}},
			{Provider: "HasNumericalID", ID: rpcmanifest.NodeID{Numerical: "node1", Stable: "stable-node1"}},
		},
	}

	rpcs := rpcmanifest.RPCsForChain(42, entry)

	// Only the node with a numerical ID — the other has no proxy path segment to template.
	require.Len(t, rpcs, 1)
	assert.Equal(t, "HasNumericalID (node1)", rpcs[0].RPCName)
	assert.Equal(t, "https://rpcs.cldev.sh/42/node1", rpcs[0].HTTPURL)
}

func TestMergeInto_ChainWithNoNodesFails(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "rpcs.json")
	require.NoError(t, os.WriteFile(path, []byte(`[{"ChainID":"1","ChainSelector":"42","Nodes":[]}]`), 0o600))
	manifest, err := rpcmanifest.Load(context.Background(), path)
	require.NoError(t, err)

	yamlCfg := cfgnet.NewConfig([]cfgnet.Network{{Type: cfgnet.NetworkTypeTestnet, ChainSelector: 42}})
	_, err = rpcmanifest.MergeInto(yamlCfg, manifest, logger.Test(t))
	require.ErrorContains(t, err, "no provider nodes in rpc manifest")
}
