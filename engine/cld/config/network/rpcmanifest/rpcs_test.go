package rpcmanifest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network/rpcmanifest"
)

func TestRPCsForChain_IncludesAllNamedNodes(t *testing.T) {
	t.Parallel()

	manifest, err := rpcmanifest.Load(context.Background(), "testdata/rpcs.json")
	require.NoError(t, err)

	entry, ok := manifest.Chain(9999999999999999999)
	require.True(t, ok)

	rpcs := rpcmanifest.RPCsForChain(9999999999999999999, entry)
	require.Len(t, rpcs, 2)
	assert.Equal(t, "CLL Proxy", rpcs[0].RPCName)
	assert.Equal(t, "wss://rpcs.cldev.sh/9999999999999999999", rpcs[0].WSURL)
	assert.Equal(t, "Public (public1)", rpcs[1].RPCName)
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

	// Only the CLL Proxy entry plus the one node with a numerical ID — the node with no
	// numerical ID has no proxy path segment to template and must be skipped.
	require.Len(t, rpcs, 2)
	assert.Equal(t, "CLL Proxy", rpcs[0].RPCName)
	assert.Equal(t, "HasNumericalID (node1)", rpcs[1].RPCName)
	assert.Equal(t, "https://rpcs.cldev.sh/42/node1", rpcs[1].HTTPURL)
}
