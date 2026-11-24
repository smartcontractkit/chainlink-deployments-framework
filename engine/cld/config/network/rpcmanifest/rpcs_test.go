package rpcmanifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

func Test_rpcsForChain(t *testing.T) {
	t.Parallel()

	entry := chainEntry{
		Nodes: []node{
			{Provider: "LinkPool", ID: nodeID{Numerical: "linkpool1", Stable: "linkpool-test"}},
			{Provider: "NoNumericalID", ID: nodeID{Stable: "stable-only"}},
			{Provider: "Traversal", ID: nodeID{Numerical: "../x"}},
			{Provider: "Slash", ID: nodeID{Numerical: "a/b"}},
			{Provider: "Alchemy", ID: nodeID{Numerical: "alchemy_1-b"}},
		},
	}

	want := []cfgnet.RPC{
		{
			RPCName:            "LinkPool (linkpool1)",
			PreferredURLScheme: "http",
			HTTPURL:            "https://rpcs.example.com/42/linkpool1",
		},
		{
			RPCName:            "Alchemy (alchemy_1-b)",
			PreferredURLScheme: "http",
			HTTPURL:            "https://rpcs.example.com/42/alchemy_1-b",
		},
	}

	assert.Equal(t, want, rpcsForChain("https://rpcs.example.com", 42, entry, logger.Test(t)))
}

func Test_rpcsForChain_NoNodes(t *testing.T) {
	t.Parallel()

	require.Empty(t, rpcsForChain("https://rpcs.example.com", 42, chainEntry{}, logger.Test(t)))
}
