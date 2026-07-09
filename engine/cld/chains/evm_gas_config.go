package chains

import (
	"math/big"

	chainsel "github.com/smartcontractkit/chain-selectors"

	evmprov "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider"
	evmclient "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

// BaseGasLimitBufferBps is the proactive gas limit buffer applied to Base mainnet and testnet (+25%).
const BaseGasLimitBufferBps = uint64(2500)

// HederaDeployerGasPriceWei is the fixed legacy gas price for Hedera mainnet and testnet (1500 gwei).
const HederaDeployerGasPriceWei = uint64(1_500_000_000_000)

type evmGasConfig struct {
	gasLimitBufferBps uint64
	deployerGasLimit  uint64
	deployerGasPrice  *uint64
}

func builtInEVMGasConfig(selector uint64) evmGasConfig {
	switch selector {
	case chainsel.ETHEREUM_MAINNET_BASE_1.Selector, chainsel.ETHEREUM_TESTNET_SEPOLIA_BASE_1.Selector:
		return evmGasConfig{gasLimitBufferBps: BaseGasLimitBufferBps}
	case chainsel.METAL_MAINNET.Selector, chainsel.METAL_TESTNET.Selector:
		return evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: new(uint64(5_000_000)),
		}
	case chainsel.HEDERA_MAINNET.Selector, chainsel.HEDERA_TESTNET.Selector:
		return evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: new(HederaDeployerGasPriceWei),
		}
	case chainsel.BITCOIN_MAINNET_BOB_1.Selector, chainsel.BITCOIN_TESTNET_SEPOLIA_BOB_1.Selector:
		return evmGasConfig{
			deployerGasLimit: 7_500_000,
			deployerGasPrice: new(uint64(2_000_000)),
		}
	case chainsel.WEMIX_MAINNET.Selector:
		return evmGasConfig{
			deployerGasLimit: 7_500_000,
			deployerGasPrice: new(uint64(120_000_000_000)),
		}
	case chainsel.WEMIX_TESTNET.Selector:
		return evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: new(uint64(120_000_000_000)),
		}
	case chainsel.MEGAETH_MAINNET.Selector, chainsel.MEGAETH_TESTNET.Selector, chainsel.MEGAETH_TESTNET_2.Selector:
		return evmGasConfig{
			deployerGasLimit: 400_000_000,
			deployerGasPrice: new(uint64(1_000_000)),
		}
	case chainsel.EDGE_MAINNET.Selector, chainsel.EDGE_TESTNET.Selector:
		return evmGasConfig{deployerGasLimit: 25_000_000}
	case chainsel.BITTENSOR_MAINNET.Selector, chainsel.BITTENSOR_TESTNET.Selector:
		return evmGasConfig{
			deployerGasLimit: 10_000_000,
			deployerGasPrice: new(uint64(10_000_000_000)),
		}
	case chainsel.MIND_MAINNET.Selector:
		return evmGasConfig{deployerGasLimit: 8_000_000}
	case chainsel.RONIN_MAINNET.Selector:
		return evmGasConfig{
			deployerGasPrice: new(uint64(100_000_000_000)),
		}
	case chainsel.RONIN_TESTNET_SAIGON.Selector, chainsel.ETHEREUM_TESTNET_SEPOLIA_RONIN_1.Selector:
		return evmGasConfig{
			deployerGasPrice: new(uint64(50_000_000_000)),
		}
	case chainsel.GNOSIS_CHAIN_TESTNET_CHIADO.Selector:
		return evmGasConfig{deployerGasLimit: 10_000_000}
	case chainsel.INK_TESTNET_SEPOLIA.Selector:
		return evmGasConfig{deployerGasLimit: 7_500_000}
	case chainsel.ZORA_TESTNET.Selector:
		return evmGasConfig{deployerGasLimit: 7_500_000}
	default:
		return evmGasConfig{}
	}
}

func evmClientOptsFromGasConfig(cfg evmGasConfig) []func(*evmclient.MultiClient) {
	return []func(*evmclient.MultiClient){
		evmclient.WithGasLimitBufferBps(cfg.gasLimitBufferBps),
	}
}

func evmSignerWithGasConfig(gen evmprov.SignerGenerator, cfg evmGasConfig) evmprov.SignerGenerator {
	var gasPrice *big.Int
	if cfg.deployerGasPrice != nil {
		gasPrice = new(big.Int).SetUint64(*cfg.deployerGasPrice)
	}

	return evmprov.WrapSignerWithGasOverrides(gen, cfg.deployerGasLimit, gasPrice)
}
