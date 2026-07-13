package contract

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/gas"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// GasBoostConfig defines gas limit and price increments applied on deploy retries.
// Increment fields use pointers so zero is a valid configured value (no increment).
// A nil increment pointer uses the package default increment.
type GasBoostConfig = gas.BoostConfig

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
		if deps.IsZkSyncVM || !gas.IsRetryable(err) {
			return in
		}

		gasLimit, gasPrice := in.GasBoostValues()
		baselineLimit, baselinePrice := gas.BaselineFromTransactOpts(deps.DeployerKey)
		prevLimit, prevPrice := gas.ResolveBoostPreviousGas(gasLimit, gasPrice, baselineLimit, baselinePrice)
		gasLimit, gasPrice = gas.NextBoostedGas(c, attempt, prevLimit, prevPrice)

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

func transactOptsWithGasOverrides(
	ctx context.Context,
	chain evm.Chain,
	base *bind.TransactOpts,
	gasLimit, gasPrice uint64,
) (*bind.TransactOpts, error) {
	if gasLimit == 0 && gasPrice == 0 {
		return base, nil
	}

	return gas.ApplyBoostOverrides(ctx, chain.Client, base, gasLimit, gasPrice)
}

func shouldAutoGasBoost(chain evm.Chain, gasLimit, gasPrice uint64) bool {
	// A non-nil Boost pointer (including boost: {} in YAML) enables the inner retry loop.
	// Use RetryDeployWithGasBoost for an additional outer operations-level retry; when both
	// are active, outer retries set input gas overrides and disable this auto path.
	return chain.GasConfig != nil &&
		chain.GasConfig.Boost != nil &&
		gasLimit == 0 &&
		gasPrice == 0
}
