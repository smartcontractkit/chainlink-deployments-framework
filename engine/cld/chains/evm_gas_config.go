package chains

import (
	"math/big"

	chainsel "github.com/smartcontractkit/chain-selectors"

	fevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	evmprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
	evmclient "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

const (
	estimateGasBufferBps      = uint64(2500)
	hederaDeployerGasPriceWei = uint64(1_500_000_000_000)
)

type evmGasConfig struct {
	gasLimitBufferBps uint64
	maxTxGasLimit     uint64
	deployerGasLimit  uint64
	deployerGasPrice  uint64
}

func withEIP7825Cap(cfg evmGasConfig) evmGasConfig {
	cfg.maxTxGasLimit = fevm.EIP7825MaxTxGasLimit

	return cfg
}

func builtInEVMGasConfig(selector uint64) evmGasConfig {
	switch selector {
	case chainsel.ETHEREUM_MAINNET_BASE_1.Selector, chainsel.ETHEREUM_TESTNET_SEPOLIA_BASE_1.Selector,
		chainsel.ETHEREUM_MAINNET_OPTIMISM_1.Selector, chainsel.ETHEREUM_TESTNET_SEPOLIA_OPTIMISM_1.Selector:
		return withEIP7825Cap(evmGasConfig{gasLimitBufferBps: estimateGasBufferBps})
	case chainsel.METAL_MAINNET.Selector, chainsel.METAL_TESTNET.Selector:
		return withEIP7825Cap(evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: 5_000_000,
		})
	case chainsel.HEDERA_MAINNET.Selector, chainsel.HEDERA_TESTNET.Selector:
		return evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: hederaDeployerGasPriceWei,
		}
	case chainsel.BITCOIN_MAINNET_BOB_1.Selector, chainsel.BITCOIN_TESTNET_SEPOLIA_BOB_1.Selector:
		return withEIP7825Cap(evmGasConfig{
			deployerGasLimit: 7_500_000,
			deployerGasPrice: 2_000_000,
		})
	case chainsel.WEMIX_MAINNET.Selector, chainsel.WEMIX_TESTNET.Selector:
		return evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: 120_000_000_000,
		}
	case chainsel.MEGAETH_MAINNET.Selector, chainsel.MEGAETH_TESTNET.Selector, chainsel.MEGAETH_TESTNET_2.Selector:
		return evmGasConfig{
			deployerGasLimit: 400_000_000,
			deployerGasPrice: 1_000_000,
		}
	case chainsel.EDGE_MAINNET.Selector, chainsel.EDGE_TESTNET.Selector:
		return evmGasConfig{deployerGasLimit: 25_000_000}
	case chainsel.BITTENSOR_MAINNET.Selector, chainsel.BITTENSOR_TESTNET.Selector:
		return evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: 10_000_000_000,
		}
	case chainsel.MIND_MAINNET.Selector:
		return evmGasConfig{deployerGasLimit: 8_000_000}
	case chainsel.RONIN_MAINNET.Selector:
		return evmGasConfig{deployerGasPrice: 100_000_000_000}
	case chainsel.RONIN_TESTNET_SAIGON.Selector, chainsel.ETHEREUM_TESTNET_SEPOLIA_RONIN_1.Selector:
		return evmGasConfig{deployerGasPrice: 50_000_000_000}
	case chainsel.GNOSIS_CHAIN_TESTNET_CHIADO.Selector:
		return evmGasConfig{deployerGasLimit: 10_000_000}
	case chainsel.INK_TESTNET_SEPOLIA.Selector:
		return withEIP7825Cap(evmGasConfig{deployerGasLimit: 7_500_000})
	case chainsel.ZORA_MAINNET.Selector, chainsel.ZORA_TESTNET.Selector:
		return withEIP7825Cap(evmGasConfig{deployerGasLimit: 7_500_000})
	default:
		return evmGasConfig{}
	}
}

func evmClientOptsFromGasConfig(cfg evmGasConfig) []func(*evmclient.MultiClient) {
	return []func(*evmclient.MultiClient){
		evmclient.WithGasLimitBufferBps(cfg.gasLimitBufferBps),
		evmclient.WithMaxTxGasLimit(cfg.maxTxGasLimit),
	}
}

func evmSignerWithGasConfig(gen evmprov.SignerGenerator, cfg evmGasConfig) evmprov.SignerGenerator {
	var gasPrice *big.Int
	if cfg.deployerGasPrice > 0 {
		gasPrice = new(big.Int).SetUint64(cfg.deployerGasPrice)
	}

	gasLimit := fevm.CapGasLimit(cfg.deployerGasLimit, cfg.maxTxGasLimit)

	return evmprov.WrapSignerWithGasOverrides(gen, gasLimit, gasPrice)
}
