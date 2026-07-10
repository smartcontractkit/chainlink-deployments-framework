package contract

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations/optest"
)

func TestNextBoostedGas(t *testing.T) {
	t.Parallel()

	limit, price := nextBoostedGas(GasBoostConfig{}, 0, 0, 0)
	require.Equal(t, uint64(5_000_000), limit)
	require.Equal(t, uint64(20_000_000_000), price)

	limit, price = nextBoostedGas(GasBoostConfig{}, 2, 0, 0)
	require.Equal(t, uint64(6_000_000), limit)
	require.Equal(t, uint64(40_000_000_000), price)

	limit, price = nextBoostedGas(GasBoostConfig{
		InitialGasLimit:   1_000_000,
		GasLimitIncrement: ptrUint64(100_000),
		InitialGasPrice:   5_000_000_000,
		GasPriceIncrement: ptrUint64(1_000_000_000),
	}, 3, 0, 0)
	require.Equal(t, uint64(1_300_000), limit)
	require.Equal(t, uint64(8_000_000_000), price)

	limit, price = nextBoostedGas(GasBoostConfig{}, 5, 4_000_000, 30_000_000_000)
	require.Equal(t, uint64(4_500_000), limit)
	require.Equal(t, uint64(40_000_000_000), price)

	zeroInc := uint64(0)
	limit, price = nextBoostedGas(GasBoostConfig{
		GasLimitIncrement: &zeroInc,
		GasPriceIncrement: &zeroInc,
	}, 1, 2_000_000, 10_000_000_000)
	require.Equal(t, uint64(2_000_000), limit)
	require.Equal(t, uint64(10_000_000_000), price)
}

func TestIsGasRetryableError(t *testing.T) {
	t.Parallel()

	require.False(t, isGasRetryableError(nil))
	require.False(t, isGasRetryableError(errors.New("invalid constructor args")))

	// String fallback for RPC / simulation messages.
	require.True(t, isGasRetryableError(errors.New("out of gas: gas required exceeds allowance")))
	require.True(t, isGasRetryableError(errors.New("transaction underpriced")))

	// geth sentinels via errors.Is.
	require.True(t, isGasRetryableError(vm.ErrOutOfGas))
	require.True(t, isGasRetryableError(fmt.Errorf("send failed: %w", txpool.ErrUnderpriced)))
	require.True(t, isGasRetryableError(fmt.Errorf("deploy failed: %w", core.ErrFeeCapTooLow)))
}

func TestTransactOptsWithGasOverrides(t *testing.T) {
	t.Parallel()

	base := &bind.TransactOpts{
		GasLimit:  21_000,
		GasPrice:  big.NewInt(1),
		GasFeeCap: big.NewInt(100),
		GasTipCap: big.NewInt(2),
	}

	require.Nil(t, transactOptsWithGasOverrides(nil, 1, 1, 0))
	require.Same(t, base, transactOptsWithGasOverrides(base, 0, 0, 0))

	limitOnly := transactOptsWithGasOverrides(base, 500_000, 0, 0)
	require.Equal(t, uint64(500_000), limitOnly.GasLimit)
	require.Equal(t, big.NewInt(1), limitOnly.GasPrice)
	require.Equal(t, big.NewInt(100), limitOnly.GasFeeCap)

	priceOnly := transactOptsWithGasOverrides(base, 0, 30_000_000_000, 0)
	require.Equal(t, uint64(21_000), priceOnly.GasLimit)
	require.Equal(t, uint64(30_000_000_000), priceOnly.GasPrice.Uint64())
	require.Nil(t, priceOnly.GasFeeCap)
	require.Nil(t, priceOnly.GasTipCap)

	withGas := transactOptsWithGasOverrides(base, 500_000, 30_000_000_000, 0)
	require.Equal(t, uint64(500_000), withGas.GasLimit)
	require.Equal(t, uint64(30_000_000_000), withGas.GasPrice.Uint64())
	require.Nil(t, withGas.GasFeeCap)
	require.Nil(t, withGas.GasTipCap)
	require.Equal(t, uint64(21_000), base.GasLimit, "must not mutate base opts")
	require.Equal(t, big.NewInt(100), base.GasFeeCap, "must not mutate base fee cap")

	withinCap := transactOptsWithGasOverrides(base, 14_000_000, 0, evm.EIP7825MaxTxGasLimit)
	require.Equal(t, uint64(14_000_000), withinCap.GasLimit)

	capped := transactOptsWithGasOverrides(base, 20_000_000, 0, evm.EIP7825MaxTxGasLimit)
	require.Equal(t, evm.EIP7825MaxTxGasLimit, capped.GasLimit)
}

func TestRetryDeployWithGasBoostNilConfig(t *testing.T) {
	t.Parallel()

	failures := 2
	calls := 0
	op := operations.NewOperation(
		"gas-boost-nil",
		semver.MustParse("1.0.0"),
		"test",
		func(_ operations.Bundle, _ evm.Chain, _ DeployInput[struct{}]) (struct{}, error) {
			calls++
			if failures > 0 {
				failures--
				return struct{}{}, errors.New("out of gas")
			}

			return struct{}{}, nil
		},
	)

	bundle := optest.NewBundle(t)
	_, err := operations.ExecuteOperation(
		bundle,
		op,
		evm.Chain{Selector: 1},
		DeployInput[struct{}]{},
		RetryDeployWithGasBoost[struct{}](nil),
	)
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

func TestRetryDeployWithGasBoostRetriesOnGasError(t *testing.T) {
	t.Parallel()

	failures := 2
	var gasLimits []uint64
	op := operations.NewOperation(
		"gas-boost-retry",
		semver.MustParse("1.0.0"),
		"test",
		func(_ operations.Bundle, _ evm.Chain, input DeployInput[struct{}]) (uint64, error) {
			gasLimits = append(gasLimits, input.GasLimit)
			if failures > 0 {
				failures--
				return 0, errors.New("out of gas: gas required exceeds allowance")
			}

			return input.GasLimit, nil
		},
	)

	cfg := &GasBoostConfig{
		InitialGasLimit:   1_000_000,
		GasLimitIncrement: ptrUint64(100_000),
		InitialGasPrice:   5_000_000_000,
		GasPriceIncrement: ptrUint64(1_000_000_000),
	}
	bundle := optest.NewBundle(t)
	res, err := operations.ExecuteOperation(
		bundle,
		op,
		evm.Chain{Selector: 1},
		DeployInput[struct{}]{},
		RetryDeployWithGasBoost[struct{}](cfg),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1_100_000), res.Output)
	require.Equal(t, []uint64{0, 1_000_000, 1_100_000}, gasLimits)
}

func TestRetryDeployWithGasBoostSkipsNonGasErrors(t *testing.T) {
	t.Parallel()

	var gasLimits []uint64
	op := operations.NewOperation(
		"gas-boost-skip",
		semver.MustParse("1.0.0"),
		"test",
		func(_ operations.Bundle, _ evm.Chain, input DeployInput[struct{}]) (struct{}, error) {
			gasLimits = append(gasLimits, input.GasLimit)
			return struct{}{}, errors.New("invalid constructor args")
		},
	)

	bundle := optest.NewBundle(t)
	_, err := operations.ExecuteOperation(
		bundle,
		op,
		evm.Chain{Selector: 1},
		DeployInput[struct{}]{},
		RetryDeployWithGasBoost[struct{}](&GasBoostConfig{}),
	)
	require.Error(t, err)
	for _, limit := range gasLimits {
		require.Equal(t, uint64(0), limit)
	}
}

func TestRetryDeployWithGasBoostSkipsZkSync(t *testing.T) {
	t.Parallel()

	failures := 2
	var gasLimits []uint64
	op := operations.NewOperation(
		"gas-boost-zksync",
		semver.MustParse("1.0.0"),
		"test",
		func(_ operations.Bundle, _ evm.Chain, input DeployInput[struct{}]) (struct{}, error) {
			gasLimits = append(gasLimits, input.GasLimit)
			if failures > 0 {
				failures--
				return struct{}{}, errors.New("out of gas")
			}

			return struct{}{}, nil
		},
	)

	bundle := optest.NewBundle(t)
	_, err := operations.ExecuteOperation(
		bundle,
		op,
		evm.Chain{Selector: 1, IsZkSyncVM: true},
		DeployInput[struct{}]{},
		RetryDeployWithGasBoost[struct{}](&GasBoostConfig{}),
	)
	require.NoError(t, err)
	for _, limit := range gasLimits {
		require.Equal(t, uint64(0), limit)
	}
}

func ptrUint64(v uint64) *uint64 {
	return &v
}

func TestRetryWriteWithGasBoostRetriesOnGasError(t *testing.T) {
	t.Parallel()

	failures := 2
	var gasLimits []uint64
	op := operations.NewOperation(
		"write-gas-boost-retry",
		semver.MustParse("1.0.0"),
		"test",
		func(_ operations.Bundle, _ evm.Chain, input FunctionInput[struct{}]) (struct{}, error) {
			gasLimits = append(gasLimits, input.GasLimit)
			if failures > 0 {
				failures--
				return struct{}{}, errors.New("out of gas: gas required exceeds allowance")
			}

			return struct{}{}, nil
		},
	)

	cfg := &GasBoostConfig{
		InitialGasLimit:   1_000_000,
		GasLimitIncrement: ptrUint64(100_000),
	}
	bundle := optest.NewBundle(t)
	_, err := operations.ExecuteOperation(
		bundle,
		op,
		evm.Chain{Selector: 1},
		FunctionInput[struct{}]{},
		RetryWriteWithGasBoost[struct{}](cfg),
	)
	require.NoError(t, err)
	require.Equal(t, []uint64{0, 1_000_000, 1_100_000}, gasLimits)
}

func TestRetryWriteWithGasBoostNilConfig(t *testing.T) {
	t.Parallel()

	failures := 2
	calls := 0
	op := operations.NewOperation(
		"write-gas-boost-nil",
		semver.MustParse("1.0.0"),
		"test",
		func(_ operations.Bundle, _ evm.Chain, _ FunctionInput[struct{}]) (struct{}, error) {
			calls++
			if failures > 0 {
				failures--
				return struct{}{}, errors.New("out of gas")
			}

			return struct{}{}, nil
		},
	)

	bundle := optest.NewBundle(t)
	_, err := operations.ExecuteOperation(
		bundle,
		op,
		evm.Chain{Selector: 1},
		FunctionInput[struct{}]{},
		RetryWriteWithGasBoost[struct{}](nil),
	)
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

func TestRetryWriteWithGasBoostSkipsNonGasErrors(t *testing.T) {
	t.Parallel()

	var gasLimits []uint64
	op := operations.NewOperation(
		"write-gas-boost-skip",
		semver.MustParse("1.0.0"),
		"test",
		func(_ operations.Bundle, _ evm.Chain, input FunctionInput[struct{}]) (struct{}, error) {
			gasLimits = append(gasLimits, input.GasLimit)
			return struct{}{}, errors.New("revert: unauthorized")
		},
	)

	bundle := optest.NewBundle(t)
	_, err := operations.ExecuteOperation(
		bundle,
		op,
		evm.Chain{Selector: 1},
		FunctionInput[struct{}]{},
		RetryWriteWithGasBoost[struct{}](&GasBoostConfig{}),
	)
	require.Error(t, err)
	for _, limit := range gasLimits {
		require.Equal(t, uint64(0), limit)
	}
}

func TestRetryWriteWithGasBoostSkipsZkSync(t *testing.T) {
	t.Parallel()

	failures := 2
	var gasLimits []uint64
	op := operations.NewOperation(
		"write-gas-boost-zksync",
		semver.MustParse("1.0.0"),
		"test",
		func(_ operations.Bundle, _ evm.Chain, input FunctionInput[struct{}]) (struct{}, error) {
			gasLimits = append(gasLimits, input.GasLimit)
			if failures > 0 {
				failures--
				return struct{}{}, errors.New("out of gas")
			}

			return struct{}{}, nil
		},
	)

	bundle := optest.NewBundle(t)
	_, err := operations.ExecuteOperation(
		bundle,
		op,
		evm.Chain{Selector: 1, IsZkSyncVM: true},
		FunctionInput[struct{}]{},
		RetryWriteWithGasBoost[struct{}](&GasBoostConfig{}),
	)
	require.NoError(t, err)
	for _, limit := range gasLimits {
		require.Equal(t, uint64(0), limit)
	}
}
