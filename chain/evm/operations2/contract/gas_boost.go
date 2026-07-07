package contract

import (
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

const (
	defaultInitialGasLimit   = uint64(5_000_000)
	defaultGasLimitIncrement = uint64(500_000)
	defaultInitialGasPrice   = uint64(20_000_000_000)
	defaultGasPriceIncrement = uint64(10_000_000_000)
)

// GasBoostConfig defines gas limit and price increments applied on deploy retries.
// Increment fields use pointers so zero is a valid configured value (no increment).
// A nil increment pointer uses the package default increment.
type GasBoostConfig struct {
	InitialGasLimit   uint64  `json:"initialGasLimit"`
	GasLimitIncrement *uint64 `json:"gasLimitIncrement,omitempty"`
	InitialGasPrice   uint64  `json:"initialGasPrice"`
	GasPriceIncrement *uint64 `json:"gasPriceIncrement,omitempty"`
}

// GasOverridable is implemented by operation inputs that carry optional gas overrides.
type GasOverridable[IN any] interface {
	GasBoostValues() (gasLimit, gasPrice uint64)
	WithGasBoost(gasLimit, gasPrice uint64) IN
}

func (in DeployInput[ARGS]) GasBoostValues() (uint64, uint64) {
	return in.GasLimit, in.GasPrice
}

func (in DeployInput[ARGS]) WithGasBoost(gasLimit, gasPrice uint64) DeployInput[ARGS] {
	in.GasLimit = gasLimit
	in.GasPrice = gasPrice

	return in
}

func (in FunctionInput[ARGS]) GasBoostValues() (uint64, uint64) {
	return in.GasLimit, in.GasPrice
}

func (in FunctionInput[ARGS]) WithGasBoost(gasLimit, gasPrice uint64) FunctionInput[ARGS] {
	in.GasLimit = gasLimit
	in.GasPrice = gasPrice

	return in
}

// RetryWithGasBoost enables the default operation retry policy and increases gas on EVM retries.
// The operation may retry on any failure (per the framework retry policy); gas limit and price are adjusted
// only when the prior attempt failed with a gas-related error.
// The first execution attempt uses the chain deployer's default gas settings.
// On ZkSync VM chains, gas fields are not adjusted.
// When cfg is nil, returns a no-op option and retry remains disabled (omit this option instead).
// Use operations.WithRetry for retry without gas adjustment.
// Input types must implement GasOverridable.
func RetryWithGasBoost[IN GasOverridable[IN]](cfg *GasBoostConfig) operations.ExecuteOption[IN, evm.Chain] {
	if cfg == nil {
		return func(*operations.ExecuteConfig[IN, evm.Chain]) {}
	}
	c := *cfg

	return operations.WithRetryInput(func(attempt uint, err error, in IN, deps evm.Chain) IN {
		if deps.IsZkSyncVM || !isGasRetryableError(err) {
			return in
		}

		gasLimit, gasPrice := in.GasBoostValues()
		gasLimit, gasPrice = nextBoostedGas(c, attempt, gasLimit, gasPrice)

		return in.WithGasBoost(gasLimit, gasPrice)
	})
}

// RetryDeployWithGasBoost enables RetryWithGasBoost for DeployInput.
// The first execution attempt uses the chain deployer's default gas settings (auto-estimation).
// On ZkSync VM chains, omit this option for ZkSync-only deploy flows.
func RetryDeployWithGasBoost[ARGS any](cfg *GasBoostConfig) operations.ExecuteOption[DeployInput[ARGS], evm.Chain] {
	return RetryWithGasBoost[DeployInput[ARGS]](cfg)
}

// RetryWriteWithGasBoost enables RetryWithGasBoost for FunctionInput.
// On ZkSync VM chains, omit this option for ZkSync-only write flows.
func RetryWriteWithGasBoost[ARGS any](cfg *GasBoostConfig) operations.ExecuteOption[FunctionInput[ARGS], evm.Chain] {
	return RetryWithGasBoost[FunctionInput[ARGS]](cfg)
}

func (cfg GasBoostConfig) gasLimitIncrement() uint64 {
	if cfg.GasLimitIncrement == nil {
		return defaultGasLimitIncrement
	}

	return *cfg.GasLimitIncrement
}

func (cfg GasBoostConfig) gasPriceIncrement() uint64 {
	if cfg.GasPriceIncrement == nil {
		return defaultGasPriceIncrement
	}

	return *cfg.GasPriceIncrement
}

func resolveInitialGasLimit(cfg GasBoostConfig) uint64 {
	if cfg.InitialGasLimit > 0 {
		return cfg.InitialGasLimit
	}

	return defaultInitialGasLimit
}

func resolveInitialGasPrice(cfg GasBoostConfig) uint64 {
	if cfg.InitialGasPrice > 0 {
		return cfg.InitialGasPrice
	}

	return defaultInitialGasPrice
}

func nextBoostedGas(cfg GasBoostConfig, attempt uint, previousLimit, previousPrice uint64) (gasLimit uint64, gasPrice uint64) {
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

func isGasRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, vm.ErrOutOfGas) ||
		errors.Is(err, vm.ErrCodeStoreOutOfGas) ||
		errors.Is(err, core.ErrIntrinsicGas) ||
		errors.Is(err, core.ErrFeeCapTooLow) ||
		errors.Is(err, core.ErrGasLimitReached) ||
		errors.Is(err, txpool.ErrUnderpriced) ||
		errors.Is(err, txpool.ErrReplaceUnderpriced) ||
		errors.Is(err, txpool.ErrTxGasPriceTooLow) ||
		errors.Is(err, txpool.ErrGasLimit) {
		return true
	}

	msg := strings.ToLower(err.Error())
	for _, pattern := range gasRetryableErrorPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	return false
}

// gasRetryableErrorPatterns covers RPC and wrapped errors that no longer carry geth sentinels.
var gasRetryableErrorPatterns = []string{
	"out of gas",
	"gas required exceeds allowance",
	"intrinsic gas too low",
	"underpriced",
	"replacement transaction underpriced",
	"max fee per gas less than block base fee",
	"exceeds block gas limit",
}

func transactOptsWithGasOverrides(base *bind.TransactOpts, gasLimit, gasPrice, bufferBps uint64) *bind.TransactOpts {
	if base == nil {
		return nil
	}
	if gasLimit == 0 && gasPrice == 0 {
		return base
	}

	opts := *base
	if gasLimit > 0 {
		opts.GasLimit = evm.ApplyGasLimitBuffer(gasLimit, bufferBps)
	}
	if gasPrice > 0 {
		opts.GasPrice = new(big.Int).SetUint64(gasPrice)
		opts.GasFeeCap = nil
		opts.GasTipCap = nil
	}

	return &opts
}
