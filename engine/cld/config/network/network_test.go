package network

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Network_ChainFamily(t *testing.T) {
	t.Parallel()

	network := Network{ChainSelector: chainsel.ETHEREUM_MAINNET.Selector}
	got, err := network.ChainFamily()
	require.NoError(t, err)

	assert.Equal(t, chainsel.FamilyEVM, got)
}

func Test_Network_ChainID(t *testing.T) {
	t.Parallel()

	network := Network{ChainSelector: chainsel.ETHEREUM_MAINNET.Selector}
	got, err := network.ChainID()
	require.NoError(t, err)

	assert.Equal(t, "1", got)
}

func Test_Network_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		giveFunc func(*Network)
		wantErr  string
	}{
		{
			name:     "valid network",
			giveFunc: func(n *Network) {},
		},
		{
			name:     "missing type",
			giveFunc: func(n *Network) { n.Type = "" },
			wantErr:  "type is required",
		},
		{
			name:     "missing chain selector",
			giveFunc: func(n *Network) { n.ChainSelector = 0 },
			wantErr:  "chain selector is required",
		},
		{
			name:     "missing RPCs",
			giveFunc: func(n *Network) { n.RPCs = []RPC{} },
			wantErr:  "at least one RPC is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			network := &Network{
				Type:          NetworkTypeMainnet,
				ChainSelector: chainsel.ETHEREUM_MAINNET.Selector,
				RPCs:          []RPC{{RPCName: "test_rpc", HTTPURL: "https://test.rpc", WSURL: "wss://test.rpc"}},
			}

			tt.giveFunc(network)

			err := network.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
