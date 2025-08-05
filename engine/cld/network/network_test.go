package network

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Network_ChainFamily(t *testing.T) {
	t.Parallel()

	network := Network{ChainSelector: chain_selectors.ETHEREUM_MAINNET.Selector}
	got, err := network.ChainFamily()
	require.NoError(t, err)

	assert.Equal(t, chain_selectors.FamilyEVM, got)
}

func Test_Network_ChainID(t *testing.T) {
	t.Parallel()

	network := Network{ChainSelector: chain_selectors.ETHEREUM_MAINNET.Selector}
	got, err := network.ChainID()
	require.NoError(t, err)

	assert.Equal(t, "1", got)
}

func Test_Network_Preferred_Endpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give RPC
		want string
	}{
		{
			name: "HTTP preferred",
			give: RPC{
				RPCName:            "test_rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test.rpc",
				WSURL:              "wss://test.rpc",
			},
			want: "https://test.rpc",
		},
		{
			name: "WS preferred",
			give: RPC{
				RPCName:            "test_rpc",
				PreferredURLScheme: "ws",
				HTTPURL:            "https://test.rpc",
				WSURL:              "wss://test.rpc",
			},
			want: "wss://test.rpc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.give.PreferredEndpoint())
		})
	}
}
