package evm

import (
	"testing"

	"github.com/stretchr/testify/require"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

func TestGetVerificationStrategyForNetwork(t *testing.T) {
	t.Parallel()

	require.Equal(t, StrategyUnknown, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeMainnet,
	}))

	eth := cfgnet.Network{
		Type: cfgnet.NetworkTypeMainnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type:   "etherscan",
			APIKey: "key",
		},
	}
	require.Equal(t, StrategyEtherscan, GetVerificationStrategyForNetwork(eth))

	snow := cfgnet.Network{
		Type: cfgnet.NetworkTypeTestnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type:   "snowtrace",
			APIKey: "k",
			URL:    "https://api-testnet.snowtrace.io/api",
		},
	}
	require.Equal(t, StrategyEtherscan, GetVerificationStrategyForNetwork(snow))

	require.Equal(t, StrategyUnknown, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeMainnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type: "etherscan",
		},
	}))

	require.Equal(t, StrategyEtherscan, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeMainnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type: "etherscan",
			URL:  "https://polygonscan.com/api",
		},
	}))

	require.Equal(t, StrategyBlockscout, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeMainnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type: "blockscout",
			URL:  "https://explorer.example/api",
		},
	}))

	require.Equal(t, StrategyRoutescan, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeTestnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type: "routescan",
		},
	}))

	require.Equal(t, StrategyUnknown, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: "",
		BlockExplorer: cfgnet.BlockExplorer{
			Type: "routescan",
		},
	}))

	require.Equal(t, StrategyOkLink, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeMainnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type:   "oklink",
			APIKey: "k",
			Slug:   "XLAYER",
		},
	}))

	require.Equal(t, StrategyOkLink, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeMainnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type: "oklink",
			Slug: "XLAYER",
		},
	}))

	require.Equal(t, StrategySocialScan, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeTestnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type:   "socialscan",
			APIKey: "k",
			Slug:   "pharos-testnet",
		},
	}))

	require.Equal(t, StrategySocialScan, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeTestnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type: "socialscan",
			Slug: "pharos-testnet",
		},
	}))

	require.Equal(t, StrategySourcify, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeMainnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type: "sourcify",
			URL:  "https://sourcify.dev/server",
		},
	}))

	require.Equal(t, StrategyEtherscan, GetVerificationStrategyForNetwork(cfgnet.Network{
		Type: cfgnet.NetworkTypeMainnet,
		BlockExplorer: cfgnet.BlockExplorer{
			Type:   "wemixscan",
			APIKey: "k",
			URL:    "https://api.wemixscan.com",
		},
	}))
}
