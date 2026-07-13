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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/gas"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations/optest"
)

func TestNextBoostedGas(t *testing.T) {
	t.Parallel()

	limit, price := gas.NextBoostedGas(gas.BoostConfig{}, 0, 0, 0)
	require.Equal(t, uint64(5_000_000), limit)
	require.Equal(t, uint64(20_000_000_000), price)

	limit, price = gas.NextBoostedGas(gas.BoostConfig{}, 2, 0, 0)
	require.Equal(t, uint64(6_000_000), limit)
	require.Equal(t, uint64(40_000_000_000), price)

	limit, price = gas.NextBoostedGas(gas.BoostConfig{
		InitialGasLimit:   1_000_000,
		GasLimitIncrement: ptrUint64(100_000),
		InitialGasPrice:   5_000_000_000,
		GasPriceIncrement: ptrUint64(1_000_000_000),
	}, 3, 0, 0)
	require.Equal(t, uint64(1_300_000), limit)
	require.Equal(t, uint64(8_000_000_000), price)

	limit, price = gas.NextBoostedGas(gas.BoostConfig{}, 5, 4_000_000, 30_000_000_000)
	require.Equal(t, uint64(4_500_000), limit)
	require.Equal(t, uint64(40_000_000_000), price)

	zeroInc := uint64(0)
	limit, price = gas.NextBoostedGas(gas.BoostConfig{
		GasLimitIncrement: &zeroInc,
		GasPriceIncrement: &zeroInc,
	}, 1, 2_000_000, 10_000_000_000)
	require.Equal(t, uint64(2_000_000), limit)
	require.Equal(t, uint64(10_000_000_000), price)
}

func TestIsGasRetryableError(t *testing.T) {
	t.Parallel()

	require.False(t, gas.IsRetryable(nil))
	require.False(t, gas.IsRetryable(errors.New("invalid constructor args")))
	require.True(t, gas.IsRetryable(errors.New("out of gas: gas required exceeds allowance")))
	require.True(t, gas.IsRetryable(errors.New("transaction underpriced")))
	require.True(t, gas.IsRetryable(vm.ErrOutOfGas))
	require.True(t, gas.IsRetryable(fmt.Errorf("send failed: %w", txpool.ErrUnderpriced)))
	require.True(t, gas.IsRetryable(fmt.Errorf("deploy failed: %w", core.ErrFeeCapTooLow)))
}

func TestTransactOptsWithGasOverrides(t *testing.T) {
	t.Parallel()

	base := &bind.TransactOpts{
		GasLimit:  21_000,
		GasPrice:  big.NewInt(1),
		GasFeeCap: big.NewInt(100),
		GasTipCap: big.NewInt(2),
	}
	mockClient := evm.NewMockOnchainClient(t)
	chain := evm.Chain{Client: mockClient}

	opts, err := transactOptsWithGasOverrides(t.Context(), chain, nil, 1, 1)
	require.Error(t, err)
	require.Nil(t, opts)

	same, err := transactOptsWithGasOverrides(t.Context(), chain, base, 0, 0)
	require.NoError(t, err)
	require.Same(t, base, same)

	limitOnly, err := transactOptsWithGasOverrides(t.Context(), chain, base, 500_000, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(500_000), limitOnly.GasLimit)
	require.Equal(t, big.NewInt(1), limitOnly.GasPrice)
	require.Equal(t, big.NewInt(100), limitOnly.GasFeeCap)

	mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
		Return(&types.Header{BaseFee: nil}, nil).Twice()

	legacyOnly, err := transactOptsWithGasOverrides(t.Context(), chain, base, 0, 30_000_000_000)
	require.NoError(t, err)
	require.Equal(t, uint64(21_000), legacyOnly.GasLimit)
	require.Equal(t, uint64(30_000_000_000), legacyOnly.GasPrice.Uint64())
	require.Nil(t, legacyOnly.GasFeeCap)
	require.Nil(t, legacyOnly.GasTipCap)

	withGas, err := transactOptsWithGasOverrides(t.Context(), chain, base, 500_000, 30_000_000_000)
	require.NoError(t, err)
	require.Equal(t, uint64(500_000), withGas.GasLimit)
	require.Equal(t, uint64(30_000_000_000), withGas.GasPrice.Uint64())
	require.Nil(t, withGas.GasFeeCap)
	require.Nil(t, withGas.GasTipCap)
	require.Equal(t, uint64(21_000), base.GasLimit, "must not mutate base opts")
	require.Equal(t, big.NewInt(100), base.GasFeeCap, "must not mutate base fee cap")
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

func TestShouldAutoGasBoost(t *testing.T) {
	t.Parallel()

	boost := &gas.BoostConfig{}
	chainWithBoost := evm.Chain{
		GasConfig: &gas.Config{Boost: boost},
	}

	require.True(t, shouldAutoGasBoost(chainWithBoost, 0, 0))
	require.False(t, shouldAutoGasBoost(chainWithBoost, 1, 0))
	require.False(t, shouldAutoGasBoost(chainWithBoost, 0, 1))
	require.False(t, shouldAutoGasBoost(evm.Chain{}, 0, 0))
	require.False(t, shouldAutoGasBoost(evm.Chain{GasConfig: &gas.Config{}}, 0, 0))
}

func TestRetryDeployWithGasBoostFromDeployerBaseline(t *testing.T) {
	t.Parallel()

	var gasLimits []uint64
	op := operations.NewOperation(
		"gas-boost-baseline",
		semver.MustParse("1.0.0"),
		"test",
		func(_ operations.Bundle, _ evm.Chain, input DeployInput[struct{}]) (uint64, error) {
			gasLimits = append(gasLimits, input.GasLimit)
			if len(gasLimits) == 1 {
				return 0, errors.New("out of gas")
			}

			return input.GasLimit, nil
		},
	)

	cfg := &GasBoostConfig{GasLimitIncrement: ptrUint64(500_000)}
	bundle := optest.NewBundle(t)
	res, err := operations.ExecuteOperation(
		bundle,
		op,
		evm.Chain{
			Selector: 1,
			DeployerKey: &bind.TransactOpts{
				GasLimit: 7_500_000,
				GasPrice: big.NewInt(210_000_000_000),
			},
		},
		DeployInput[struct{}]{},
		RetryDeployWithGasBoost[struct{}](cfg),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(8_000_000), res.Output)
	require.Equal(t, []uint64{0, 8_000_000}, gasLimits)
}
