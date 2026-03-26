package evm

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestNewVerifier_StrategyUnknown(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyUnknown, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "no verifier for unknown strategy", err.Error())
}

func TestNewVerifier_StrategyEtherscan_NoAPIKey(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyEtherscan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "etherscan API key not configured for chain ethereum-mainnet", err.Error())
}

func TestNewVerifier_StrategyEtherscan_WithAPIKey(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyEtherscan, VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{APIKey: "test-key"}},
		Address: "0x123",
		Metadata: SolidityContractMetadata{
			Version:  "0.8.19",
			Language: "Solidity",
			Name:     "Test",
		},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "Test 1.0.0 (0x123 on ethereum-mainnet)", v.String())
}

func TestNewVerifier_StrategyBlockscout_WithURL(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyBlockscout, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://blockscout.com/api"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.NotNil(t, v)
}

func TestNewVerifier_StrategyBlockscout_NoURL(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyBlockscout, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "blockscout API URL not configured")
}

func TestNewVerifier_StrategySourcify(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.HEDERA_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategySourcify, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://sourcify.dev/server"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "Test 1.0.0 (0x123 on hedera-mainnet)", v.String())
}

func TestNewVerifier_StrategySourcify_NoURL(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategySourcify, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "sourcify API URL not configured")
}

func TestNewVerifier_StrategyRoutescan(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.AVALANCHE_TESTNET_FUJI.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyRoutescan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{Type: cfgnet.NetworkTypeTestnet, ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{APIKey: "test"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "Test 1.0.0 (0x123 on avalanche-testnet-fuji)", v.String())
}

func TestNewVerifier_StrategyRoutescan_NoNetworkType(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyRoutescan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "routescan requires network type mainnet or testnet")
}

func TestNewVerifier_StrategyOkLink(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET_XLAYER_1.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyOkLink, VerifierConfig{
		Chain: chain,
		Network: cfgnet.Network{
			ChainSelector: chain.Selector,
			BlockExplorer: cfgnet.BlockExplorer{APIKey: "test-key", Slug: "XLAYER"},
		},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "Test 1.0.0 (0x123 on XLAYER)", v.String())
}

func TestNewVerifier_StrategyOkLink_NoAPIKey(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET_XLAYER_1.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyOkLink, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{Slug: "XLAYER"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "OKLink API key not configured")
}

func TestNewVerifier_StrategyBtrScan(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.BITCOIN_MAINNET_BITLAYER_1.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyBtrScan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://btrscan.io/api", APIKey: "test"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "Test 1.0.0 (0x123 on bitcoin-mainnet-bitlayer-1)", v.String())
}

func TestNewVerifier_StrategyBtrScan_NoAPIKey(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyBtrScan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://btrscan.io"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "btrscan API key not configured")
}

func TestNewVerifier_StrategyCoreDAO(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.CORE_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyCoreDAO, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://openapi.coredao.org", APIKey: "test"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "Test 1.0.0 (0x123 on core-mainnet)", v.String())
}

func TestNewVerifier_StrategyL2Scan(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.BITCOIN_MERLIN_MAINNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyL2Scan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://scan.merlinchain.io/api", APIKey: "test"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "Test 1.0.0 (0x123 on bitcoin-merlin-mainnet)", v.String())
}

func TestNewVerifier_StrategySocialScan(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.PHAROS_TESTNET.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategySocialScan, VerifierConfig{
		Chain: chain,
		Network: cfgnet.Network{
			ChainSelector: chain.Selector,
			BlockExplorer: cfgnet.BlockExplorer{APIKey: "test-key", Slug: "pharos-testnet"},
		},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{Version: "0.8.19", Name: "Test"},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "Test 1.0.0 (0x123 on pharos-testnet)", v.String())
}

func TestNewVerifier_StrategyDefault(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(VerificationStrategy(999), VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "no verifier for strategy 999", err.Error())
}
