package gas_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/gas"
)

func TestNextBoostedGas(t *testing.T) {
	t.Parallel()

	limit, price := gas.NextBoostedGas(gas.BoostConfig{}, 0, 0, 0)
	require.Equal(t, uint64(5_000_000), limit)
	require.Equal(t, uint64(20_000_000_000), price)

	limit, price = gas.NextBoostedGas(gas.BoostConfig{}, 2, 0, 0)
	require.Equal(t, uint64(6_000_000), limit)
	require.Equal(t, uint64(40_000_000_000), price)

	inc := uint64(100_000)
	limit, price = gas.NextBoostedGas(gas.BoostConfig{
		InitialGasLimit:   1_000_000,
		GasLimitIncrement: &inc,
		InitialGasPrice:   5_000_000_000,
		GasPriceIncrement: ptrUint64(1_000_000_000),
	}, 3, 0, 0)
	require.Equal(t, uint64(1_300_000), limit)
	require.Equal(t, uint64(8_000_000_000), price)

	limit, price = gas.NextBoostedGas(gas.BoostConfig{}, 5, 4_000_000, 30_000_000_000)
	require.Equal(t, uint64(4_500_000), limit)
	require.Equal(t, uint64(40_000_000_000), price)
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	require.False(t, gas.IsRetryable(nil))
	require.False(t, gas.IsRetryable(errors.New("invalid constructor args")))
	require.True(t, gas.IsRetryable(errors.New("out of gas: gas required exceeds allowance")))
	require.True(t, gas.IsRetryable(errors.New("transaction underpriced")))
	require.True(t, gas.IsRetryable(vm.ErrOutOfGas))
	require.True(t, gas.IsRetryable(fmt.Errorf("send failed: %w", txpool.ErrUnderpriced)))
	require.True(t, gas.IsRetryable(fmt.Errorf("deploy failed: %w", core.ErrFeeCapTooLow)))
}

func TestApplyDefaults_LegacyChain(t *testing.T) {
	t.Parallel()
	mockClient := evm.NewMockOnchainClient(t)
	mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
		Return(&types.Header{BaseFee: nil}, nil)

	opts := &bind.TransactOpts{}
	err := gas.ApplyDefaults(t.Context(), mockClient, opts, gas.Config{
		DefaultGasPriceWei: 50_000_000_000,
		DefaultGasLimit:    5_000_000,
	})
	require.NoError(t, err)
	require.Equal(t, big.NewInt(50_000_000_000), opts.GasPrice)
	require.Equal(t, uint64(5_000_000), opts.GasLimit)
	require.Nil(t, opts.GasTipCap)
	require.Nil(t, opts.GasFeeCap)
}

func TestApplyDefaults_EIP1559Chain(t *testing.T) {
	t.Parallel()
	mockClient := evm.NewMockOnchainClient(t)
	mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
		Return(&types.Header{BaseFee: big.NewInt(100)}, nil)
	mockClient.EXPECT().SuggestGasTipCap(mock.Anything).
		Return(big.NewInt(1_000_000_000), nil)

	opts := &bind.TransactOpts{}
	err := gas.ApplyDefaults(t.Context(), mockClient, opts, gas.Config{
		DefaultGasPriceWei: 50_000_000_000,
		DefaultGasLimit:    5_000_000,
	})
	require.NoError(t, err)
	require.Nil(t, opts.GasPrice)
	require.Equal(t, big.NewInt(1_000_000_000), opts.GasTipCap)
	require.Equal(t, big.NewInt(50_000_000_000), opts.GasFeeCap)
	require.Equal(t, uint64(5_000_000), opts.GasLimit)
}

func TestApplyDefaults_GasLimitOnly(t *testing.T) {
	t.Parallel()
	mockClient := evm.NewMockOnchainClient(t)

	opts := &bind.TransactOpts{}
	err := gas.ApplyDefaults(t.Context(), mockClient, opts, gas.Config{
		DefaultGasLimit: 10_000_000,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(10_000_000), opts.GasLimit)
	require.Nil(t, opts.GasPrice)
}

func TestApplyDefaults_EIP1559WithTipOverride(t *testing.T) {
	t.Parallel()
	mockClient := evm.NewMockOnchainClient(t)
	mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
		Return(&types.Header{BaseFee: big.NewInt(100)}, nil)

	opts := &bind.TransactOpts{}
	err := gas.ApplyDefaults(t.Context(), mockClient, opts, gas.Config{
		DefaultGasPriceWei:  100_000_000_000,
		DefaultGasTipCapWei: 40_000_000_000,
	})
	require.NoError(t, err)
	require.Nil(t, opts.GasPrice)
	require.Equal(t, big.NewInt(40_000_000_000), opts.GasTipCap)
	require.Equal(t, big.NewInt(100_000_000_000), opts.GasFeeCap)
}

func TestExecuteWithBoost(t *testing.T) {
	t.Parallel()

	failures := 2
	var gasLimits []uint64
	cfg := gas.BoostConfig{
		InitialGasLimit:   1_000_000,
		GasLimitIncrement: ptrUint64(100_000),
	}

	result, err := gas.ExecuteWithBoost(t.Context(), false, &cfg, 0, 0,
		func(gasLimit, _ uint64) (uint64, error) {
			gasLimits = append(gasLimits, gasLimit)
			if failures > 0 {
				failures--
				return 0, errors.New("out of gas")
			}

			return gasLimit, nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1_100_000), result)
	require.Equal(t, []uint64{0, 1_000_000, 1_100_000}, gasLimits)
}

func TestExecuteWithBoostSkipsZkSync(t *testing.T) {
	t.Parallel()

	var gasLimits []uint64
	_, err := gas.ExecuteWithBoost(t.Context(), true, &gas.BoostConfig{}, 0, 0,
		func(gasLimit, _ uint64) (struct{}, error) {
			gasLimits = append(gasLimits, gasLimit)
			return struct{}{}, nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, []uint64{0}, gasLimits)
}

func TestBaselineFromTransactOpts(t *testing.T) {
	t.Parallel()

	limit, price := gas.BaselineFromTransactOpts(nil)
	require.Equal(t, uint64(0), limit)
	require.Equal(t, uint64(0), price)

	limit, price = gas.BaselineFromTransactOpts(&bind.TransactOpts{
		GasLimit: 7_500_000,
		GasPrice: big.NewInt(210_000_000_000),
	})
	require.Equal(t, uint64(7_500_000), limit)
	require.Equal(t, uint64(210_000_000_000), price)

	limit, price = gas.BaselineFromTransactOpts(&bind.TransactOpts{
		GasLimit:  5_000_000,
		GasFeeCap: big.NewInt(100_000_000_000),
		GasTipCap: big.NewInt(40_000_000_000),
	})
	require.Equal(t, uint64(5_000_000), limit)
	require.Equal(t, uint64(100_000_000_000), price)
}

func TestResolveBoostPreviousGas(t *testing.T) {
	t.Parallel()

	limit, price := gas.ResolveBoostPreviousGas(0, 0, 7_500_000, 210_000_000_000)
	require.Equal(t, uint64(7_500_000), limit)
	require.Equal(t, uint64(210_000_000_000), price)

	limit, price = gas.ResolveBoostPreviousGas(8_000_000, 220_000_000_000, 7_500_000, 210_000_000_000)
	require.Equal(t, uint64(8_000_000), limit)
	require.Equal(t, uint64(220_000_000_000), price)
}

func TestExecuteWithBoostFromBaseline(t *testing.T) {
	t.Parallel()

	var gasLimits []uint64
	cfg := gas.BoostConfig{
		GasLimitIncrement: ptrUint64(500_000),
		GasPriceIncrement: ptrUint64(10_000_000_000),
	}

	result, err := gas.ExecuteWithBoost(t.Context(), false, &cfg, 7_500_000, 210_000_000_000,
		func(gasLimit, _ uint64) (uint64, error) {
			gasLimits = append(gasLimits, gasLimit)
			if len(gasLimits) == 1 {
				return 0, errors.New("out of gas")
			}

			return gasLimit, nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, uint64(8_000_000), result)
	require.Equal(t, []uint64{0, 8_000_000}, gasLimits)
}

func TestApplyBoostOverrides_PreservesEIP1559Tip(t *testing.T) {
	t.Parallel()

	mockClient := evm.NewMockOnchainClient(t)
	mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
		Return(&types.Header{BaseFee: big.NewInt(100)}, nil)

	base := &bind.TransactOpts{
		GasLimit:  5_000_000,
		GasFeeCap: big.NewInt(100_000_000_000),
		GasTipCap: big.NewInt(40_000_000_000),
	}

	opts, err := gas.ApplyBoostOverrides(t.Context(), mockClient, base, 0, 110_000_000_000)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(40_000_000_000), opts.GasTipCap)
	require.Equal(t, big.NewInt(110_000_000_000), opts.GasFeeCap)
	require.Nil(t, opts.GasPrice)
}

func TestApplyBoostOverrides_NilClient(t *testing.T) {
	t.Parallel()

	base := &bind.TransactOpts{
		GasLimit:  21_000,
		GasFeeCap: big.NewInt(100),
		GasTipCap: big.NewInt(2),
	}

	opts, err := gas.ApplyBoostOverrides(t.Context(), nil, base, 750_000, 25_000_000_000)
	require.NoError(t, err)
	require.Equal(t, uint64(750_000), opts.GasLimit)
	require.Equal(t, uint64(25_000_000_000), opts.GasPrice.Uint64())
	require.Nil(t, opts.GasFeeCap)
	require.Nil(t, opts.GasTipCap)
}

func TestExecuteWithBoostRespectsContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	cfg := gas.BoostConfig{InitialGasLimit: 1_000_000}
	_, err := gas.ExecuteWithBoost(ctx, false, &cfg, 0, 0,
		func(_, _ uint64) (struct{}, error) {
			return struct{}{}, errors.New("out of gas")
		},
	)
	require.ErrorIs(t, err, context.Canceled)
}

func ptrUint64(v uint64) *uint64 {
	return &v
}
