package gas

import "math/big"

// Config holds per-chain default gas settings applied to the chain deployer at load time.
// These defaults apply to any code path that uses chain.DeployerKey (bind, gobindings, etc.).
//
// zkSync VM chains: defaults are applied to DeployerKey (bind/gobind writes, etc.).
// Contract deploys on zkSync use DeployerKeyZkSyncVM with SDK fee estimation instead,
// so gas_config does not affect the operations contract deploy path on those chains.
//
// Configure under a network entry's metadata block in .config/networks/*.yaml:
//
//	metadata:
//	  gas_config:
//	    default_gas_limit: 7500000
//	    default_gas_price_wei: 210000000000
//	    default_gas_tip_cap_wei: 40000000000  # optional EIP-1559 tip override
//	    max_tx_gas_limit: 16777216           # optional cap on deployer limit and eth_estimateGas (e.g. EIP-7825)
type Config struct {
	// DefaultGasLimit sets opts.GasLimit on the chain deployer when non-zero.
	DefaultGasLimit uint64 `yaml:"default_gas_limit,omitempty" json:"defaultGasLimit,omitempty"`
	// DefaultGasPriceWei sets legacy gasPrice on pre-EIP-1559 chains, or maxFeePerGas
	// (GasFeeCap) on EIP-1559 chains, in wei.
	DefaultGasPriceWei uint64 `yaml:"default_gas_price_wei,omitempty" json:"defaultGasPriceWei,omitempty"`
	// DefaultGasTipCapWei sets maxPriorityFeePerGas (GasTipCap) on EIP-1559 chains only, in wei.
	// Only applied when DefaultGasPriceWei is also set. When unset, the node suggestion is used.
	// The tip is capped so feeCap >= baseFee + tip.
	DefaultGasTipCapWei uint64 `yaml:"default_gas_tip_cap_wei,omitempty" json:"defaultGasTipCapWei,omitempty"`
	// MaxTxGasLimit caps gas when non-zero: min(value, max_tx_gas_limit) on default_gas_limit
	// (deployer key) and on eth_estimateGas results from the chain client.
	MaxTxGasLimit uint64 `yaml:"max_tx_gas_limit,omitempty" json:"maxTxGasLimit,omitempty"`
}

func weiToBigInt(wei uint64) *big.Int {
	if wei == 0 {
		return nil
	}

	return new(big.Int).SetUint64(wei)
}
