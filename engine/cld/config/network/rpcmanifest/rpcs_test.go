package rpcmanifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network/rpcmanifest"
)

func TestRPCsForChain(t *testing.T) {
	t.Parallel()

	entry := rpcmanifest.ChainEntry{
		Nodes: []rpcmanifest.Node{
			{Provider: "LinkPool", ID: rpcmanifest.NodeID{Numerical: "linkpool1", Stable: "linkpool-test"}},
			{Provider: "NoNumericalID", ID: rpcmanifest.NodeID{Stable: "stable-only"}},
			{Provider: "Alchemy", ID: rpcmanifest.NodeID{Numerical: "alchemy1"}},
		},
	}

	want := []cfgnet.RPC{
		{
			RPCName:            "LinkPool (linkpool1)",
			PreferredURLScheme: "http",
			HTTPURL:            "https://rpcs.cldev.sh/42/linkpool1",
			WSURL:              "wss://rpcs.cldev.sh/42/linkpool1",
		},
		{
			RPCName:            "Alchemy (alchemy1)",
			PreferredURLScheme: "http",
			HTTPURL:            "https://rpcs.cldev.sh/42/alchemy1",
			WSURL:              "wss://rpcs.cldev.sh/42/alchemy1",
		},
	}

	assert.Equal(t, want, rpcmanifest.RPCsForChain(42, entry))
}

func TestRPCsForChain_NoNodes(t *testing.T) {
	t.Parallel()

	require.Empty(t, rpcmanifest.RPCsForChain(42, rpcmanifest.ChainEntry{}))
}
