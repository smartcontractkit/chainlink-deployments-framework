package gas

import (
	"math/big"
)

const (
	defaultInitialGasLimit   = uint64(5_000_000)
	defaultGasLimitIncrement = uint64(500_000)
	defaultInitialGasPrice   = uint64(20_000_000_000)
	defaultGasPriceIncrement = uint64(10_000_000_000)
	defaultMaxAttempts       = uint(10)
)

// Config holds per-chain default gas settings and optional retry boost configuration.
type Config struct {
	DefaultGasLimit     uint64       `yaml:"default_gas_limit,omitempty" json:"defaultGasLimit,omitempty"`
	DefaultGasPriceWei  uint64       `yaml:"default_gas_price_wei,omitempty" json:"defaultGasPriceWei,omitempty"`
	DefaultGasTipCapWei uint64       `yaml:"default_gas_tip_cap_wei,omitempty" json:"defaultGasTipCapWei,omitempty"`
	Boost               *BoostConfig `yaml:"boost,omitempty" json:"boost,omitempty"`
}

// BoostConfig defines gas limit and price increments applied on transaction retries.
// Increment fields use pointers so zero is a valid configured value (no increment).
type BoostConfig struct {
	InitialGasLimit   uint64  `yaml:"initial_gas_limit,omitempty" json:"initialGasLimit,omitempty"`
	GasLimitIncrement *uint64 `yaml:"gas_limit_increment,omitempty" json:"gasLimitIncrement,omitempty"`
	InitialGasPrice   uint64  `yaml:"initial_gas_price_wei,omitempty" json:"initialGasPrice,omitempty"`
	GasPriceIncrement *uint64 `yaml:"gas_price_increment,omitempty" json:"gasPriceIncrement,omitempty"`
	MaxAttempts       uint    `yaml:"max_attempts,omitempty" json:"maxAttempts,omitempty"`
}

func (cfg BoostConfig) gasLimitIncrement() uint64 {
	if cfg.GasLimitIncrement == nil {
		return defaultGasLimitIncrement
	}

	return *cfg.GasLimitIncrement
}

func (cfg BoostConfig) gasPriceIncrement() uint64 {
	if cfg.GasPriceIncrement == nil {
		return defaultGasPriceIncrement
	}

	return *cfg.GasPriceIncrement
}

func (cfg BoostConfig) maxAttempts() uint {
	if cfg.MaxAttempts > 0 {
		return cfg.MaxAttempts
	}

	return defaultMaxAttempts
}

func resolveInitialGasLimit(cfg BoostConfig) uint64 {
	if cfg.InitialGasLimit > 0 {
		return cfg.InitialGasLimit
	}

	return defaultInitialGasLimit
}

func resolveInitialGasPrice(cfg BoostConfig) uint64 {
	if cfg.InitialGasPrice > 0 {
		return cfg.InitialGasPrice
	}

	return defaultInitialGasPrice
}

// NextBoostedGas returns the gas limit and legacy gas price for the given retry attempt.
func NextBoostedGas(cfg BoostConfig, attempt uint, previousLimit, previousPrice uint64) (gasLimit uint64, gasPrice uint64) {
	initialGasLimit := resolveInitialGasLimit(cfg)
	gasLimitIncrement := cfg.gasLimitIncrement()
	initialGasPrice := resolveInitialGasPrice(cfg)
	gasPriceIncrement := cfg.gasPriceIncrement()

	if previousLimit > 0 {
		gasLimit = previousLimit + gasLimitIncrement
	} else {
		gasLimit = initialGasLimit + uint64(attempt)*gasLimitIncrement
	}

	if previousPrice > 0 {
		gasPrice = previousPrice + gasPriceIncrement
	} else {
		gasPrice = initialGasPrice + uint64(attempt)*gasPriceIncrement
	}

	return gasLimit, gasPrice
}

func weiToBigInt(wei uint64) *big.Int {
	if wei == 0 {
		return nil
	}

	return new(big.Int).SetUint64(wei)
}
