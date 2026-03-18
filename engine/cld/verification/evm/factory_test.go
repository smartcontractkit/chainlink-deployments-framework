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

func TestNewVerifier_StrategyBlockscout(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyBlockscout, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://blockscout.com"}},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "chain ID 1 is not supported by the Blockscout API", err.Error())
}

func TestNewVerifier_StrategySourcify_UnsupportedChain(t *testing.T) {
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
	require.Contains(t, err.Error(), "not supported by Sourcify")
}

func TestNewVerifier_StrategySourcify(t *testing.T) {
	t.Parallel()

	chain := chainsel.Chain{
		EvmChainID: 295,
		Selector:   chainsel.HEDERA_MAINNET.Selector,
		Name:       "hedera-mainnet",
	}

	v, err := NewVerifier(StrategySourcify, VerifierConfig{
		Chain:   chain,
		Network: cfgnet.Network{ChainSelector: chain.Selector},
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
	require.Equal(t, "Test 1.0.0 (0x123 on hedera-mainnet)", v.String())
}

func TestNewVerifier_StrategyRoutescan(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.AVALANCHE_TESTNET_FUJI.Selector)
	require.True(t, ok)

	v, err := NewVerifier(StrategyRoutescan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector, BlockExplorer: cfgnet.BlockExplorer{URL: "https://testnet.snowtrace.io"}},
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

func TestNewVerifier_StrategyRoutescan_UnsupportedChain(t *testing.T) {
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
	require.Equal(t, "chain ID 1 is not supported by the Routescan API", err.Error())
}

func TestNewVerifier_StrategyOkLink(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET_XLAYER_1.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyOkLink, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "oklink verifier not yet implemented", err.Error())
}

func TestNewVerifier_StrategyBtrScan(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.BITCOIN_MAINNET_BITLAYER_1.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyBtrScan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "btrscan verifier not yet implemented", err.Error())
}

func TestNewVerifier_StrategyBtrScan_UnsupportedChain(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.ETHEREUM_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyBtrScan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "btrscan verifier not yet implemented", err.Error())
}

func TestNewVerifier_StrategyCoreDAO(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.CORE_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyCoreDAO, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "coredao verifier not yet implemented", err.Error())
}

func TestNewVerifier_StrategyL2Scan(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.BITCOIN_MERLIN_MAINNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategyL2Scan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "l2scan verifier not yet implemented", err.Error())
}

func TestNewVerifier_StrategySocialScan(t *testing.T) {
	t.Parallel()

	chain, ok := chainsel.ChainBySelector(chainsel.PHAROS_TESTNET.Selector)
	require.True(t, ok)

	_, err := NewVerifier(StrategySocialScan, VerifierConfig{
		Chain:        chain,
		Network:      cfgnet.Network{ChainSelector: chain.Selector},
		Address:      "0x123",
		Metadata:     SolidityContractMetadata{},
		ContractType: "Test",
		Version:      "1.0.0",
		Logger:       logger.Nop(),
	})
	require.Error(t, err)
	require.Equal(t, "socialscan verifier not yet implemented", err.Error())
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
