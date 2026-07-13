package gas

import "context"

// ExecuteWithBoost retries fn on gas-related errors, bumping gas limit and price between attempts.
// The first attempt uses zero overrides so the chain deployer's default settings apply.
// baselineLimit and baselinePriceWei should reflect DeployerKey gas after ApplyDefaults; they
// ensure the first retry increases from those values instead of reset boost.initial_* defaults.
// When cfg is nil or isZkSyncVM is true, fn is invoked once without gas adjustment.
func ExecuteWithBoost[T any](
	ctx context.Context,
	isZkSyncVM bool,
	cfg *BoostConfig,
	baselineLimit, baselinePriceWei uint64,
	fn func(gasLimit, gasPriceWei uint64) (T, error),
) (T, error) {
	var zero T
	if cfg == nil || isZkSyncVM {
		return fn(0, 0)
	}

	boostCfg := *cfg
	maxAttempts := boostCfg.maxAttempts()

	var (
		result      T
		err         error
		gasLimit    uint64
		gasPriceWei uint64
	)

	for attempt := range maxAttempts {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}

		result, err = fn(gasLimit, gasPriceWei)
		if err == nil {
			return result, nil
		}
		if !IsRetryable(err) {
			return zero, err
		}
		if attempt+1 >= maxAttempts {
			break
		}

		prevLimit, prevPrice := ResolveBoostPreviousGas(gasLimit, gasPriceWei, baselineLimit, baselinePriceWei)
		gasLimit, gasPriceWei = NextBoostedGas(boostCfg, attempt, prevLimit, prevPrice)
	}

	return zero, err
}
