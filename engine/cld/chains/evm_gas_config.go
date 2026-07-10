package chains

import (
	"math/big"

	chainsel "github.com/smartcontractkit/chain-selectors"

	fevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	evmprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
	evmclient "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

// BaseGasLimitBufferBps is the proactive gas limit buffer applied to Base and Optimism mainnet and testnets (+25%).
const BaseGasLimitBufferBps = uint64(2500)

// HederaDeployerGasPriceWei is the fixed legacy gas price for Hedera mainnet and testnet (1500 gwei).
const HederaDeployerGasPriceWei = uint64(1_500_000_000_000)

type evmGasConfig struct {
	gasLimitBufferBps uint64
	maxTxGasLimit     uint64
	deployerGasLimit  uint64
	deployerGasPrice  *uint64
}

func (cfg evmGasConfig) hasOverrides() bool {
	return cfg.gasLimitBufferBps > 0 ||
		cfg.deployerGasLimit > 0 ||
		cfg.deployerGasPrice != nil
}

// eip7825ChainSelectors lists configured chains that enforce the EIP-7825 per-transaction gas cap.
var eip7825ChainSelectors = map[uint64]struct{}{
	chainsel.ETHEREUM_MAINNET_BASE_1.Selector:              {},
	chainsel.ETHEREUM_TESTNET_SEPOLIA_BASE_1.Selector:      {},
	chainsel.ETHEREUM_MAINNET_OPTIMISM_1.Selector:          {},
	chainsel.ETHEREUM_TESTNET_SEPOLIA_OPTIMISM_1.Selector:  {},
	chainsel.METAL_MAINNET.Selector:                        {},
	chainsel.METAL_TESTNET.Selector:                        {},
	chainsel.BITCOIN_MAINNET_BOB_1.Selector:                {},
	chainsel.BITCOIN_TESTNET_SEPOLIA_BOB_1.Selector:        {},
	chainsel.INK_TESTNET_SEPOLIA.Selector:                  {},
	chainsel.ZORA_TESTNET.Selector:                         {},
}

func usesEIP7825TxGasCap(selector uint64) bool {
	_, ok := eip7825ChainSelectors[selector]
	return ok
}

func estimateBufferGasConfig() evmGasConfig {
	return evmGasConfig{gasLimitBufferBps: BaseGasLimitBufferBps}
}

func applyEIP7825GasCapIfConfigured(selector uint64, cfg evmGasConfig) evmGasConfig {
	if !usesEIP7825TxGasCap(selector) || !cfg.hasOverrides() {
		return cfg
	}

	cfg.maxTxGasLimit = fevm.EIP7825MaxTxGasLimit
	if cfg.deployerGasLimit > 0 {
		cfg.deployerGasLimit = fevm.CapGasLimit(cfg.deployerGasLimit, cfg.maxTxGasLimit)
	}

	return cfg
}

func builtInEVMGasConfig(selector uint64) evmGasConfig {
	var cfg evmGasConfig

	switch selector {
	case chainsel.ETHEREUM_MAINNET_BASE_1.Selector, chainsel.ETHEREUM_TESTNET_SEPOLIA_BASE_1.Selector,
		chainsel.ETHEREUM_MAINNET_OPTIMISM_1.Selector, chainsel.ETHEREUM_TESTNET_SEPOLIA_OPTIMISM_1.Selector:
		cfg = estimateBufferGasConfig()
	case chainsel.METAL_MAINNET.Selector, chainsel.METAL_TESTNET.Selector:
		cfg = evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: new(uint64(5_000_000)),
		}
	case chainsel.HEDERA_MAINNET.Selector, chainsel.HEDERA_TESTNET.Selector:
		cfg = evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: new(HederaDeployerGasPriceWei),
		}
	case chainsel.BITCOIN_MAINNET_BOB_1.Selector, chainsel.BITCOIN_TESTNET_SEPOLIA_BOB_1.Selector:
		cfg = evmGasConfig{
			deployerGasLimit: 7_500_000,
			deployerGasPrice: new(uint64(2_000_000)),
		}
	case chainsel.WEMIX_MAINNET.Selector:
		cfg = evmGasConfig{
			deployerGasLimit: 7_500_000,
			deployerGasPrice: new(uint64(120_000_000_000)),
		}
	case chainsel.WEMIX_TESTNET.Selector:
		cfg = evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: new(uint64(120_000_000_000)),
		}
	case chainsel.MEGAETH_MAINNET.Selector, chainsel.MEGAETH_TESTNET.Selector, chainsel.MEGAETH_TESTNET_2.Selector:
		cfg = evmGasConfig{
			deployerGasLimit: 400_000_000,
			deployerGasPrice: new(uint64(1_000_000)),
		}
	case chainsel.EDGE_MAINNET.Selector, chainsel.EDGE_TESTNET.Selector:
		cfg = evmGasConfig{deployerGasLimit: 25_000_000}
	case chainsel.BITTENSOR_MAINNET.Selector, chainsel.BITTENSOR_TESTNET.Selector:
		cfg = evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: new(uint64(10_000_000_000)),
		}
	case chainsel.MIND_MAINNET.Selector:
		cfg = evmGasConfig{deployerGasLimit: 8_000_000}
	case chainsel.RONIN_MAINNET.Selector:
		cfg = evmGasConfig{
			deployerGasPrice: new(uint64(100_000_000_000)),
		}
	case chainsel.RONIN_TESTNET_SAIGON.Selector, chainsel.ETHEREUM_TESTNET_SEPOLIA_RONIN_1.Selector:
		cfg = evmGasConfig{
			deployerGasPrice: new(uint64(50_000_000_000)),
		}
	case chainsel.GNOSIS_CHAIN_TESTNET_CHIADO.Selector:
		cfg = evmGasConfig{deployerGasLimit: 10_000_000}
	case chainsel.INK_TESTNET_SEPOLIA.Selector:
		cfg = evmGasConfig{deployerGasLimit: 7_500_000}
	case chainsel.ZORA_TESTNET.Selector:
		cfg = evmGasConfig{deployerGasLimit: 7_500_000}
	default:
		return evmGasConfig{}
	}

	return applyEIP7825GasCapIfConfigured(selector, cfg)
}

func evmClientOptsFromGasConfig(cfg evmGasConfig) []func(*evmclient.MultiClient) {
	opts := make([]func(*evmclient.MultiClient), 0, 2)
	if cfg.gasLimitBufferBps > 0 {
		opts = append(opts, evmclient.WithGasLimitBufferBps(cfg.gasLimitBufferBps))
	}
	if cfg.maxTxGasLimit > 0 {
		opts = append(opts, evmclient.WithMaxTxGasLimit(cfg.maxTxGasLimit))
	}

	return opts
}

func evmSignerWithGasConfig(gen evmprov.SignerGenerator, cfg evmGasConfig) evmprov.SignerGenerator {
	var gasPrice *big.Int
	if cfg.deployerGasPrice != nil {
		gasPrice = new(big.Int).SetUint64(*cfg.deployerGasPrice)
	}

	gasLimit := fevm.CapGasLimit(cfg.deployerGasLimit, cfg.maxTxGasLimit)

	return evmprov.WrapSignerWithGasOverrides(gen, gasLimit, gasPrice)
}
