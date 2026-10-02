package deployment

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestWaitForDeployedCode(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0x01")

	newBundle := func(t *testing.T) operations.Bundle {
		t.Helper()
		lggr, err := logger.New()
		require.NoError(t, err)

		return operations.NewBundle(
			func() context.Context { return t.Context() },
			lggr,
			operations.NewMemoryReporter(),
		)
	}

	t.Run("returns immediately when code is already visible", func(t *testing.T) { //nolint:paralleltest
		client := cldf_evm.NewMockOnchainClient(t)
		client.EXPECT().CodeAt(mock.Anything, addr, (*big.Int)(nil)).
			Return([]byte{0xDE, 0xAD}, nil).Once()

		require.NoError(t, WaitForDeployedCode(newBundle(t), cldf_evm.Chain{Client: client}, addr))
	})

	t.Run("no-ops when chain.Client is nil", func(t *testing.T) { //nolint:paralleltest
		require.NoError(t, WaitForDeployedCode(newBundle(t), cldf_evm.Chain{}, addr))
	})

	t.Run("retries while the endpoint reports empty code", func(t *testing.T) { //nolint:paralleltest
		client := cldf_evm.NewMockOnchainClient(t)
		var calls int
		client.EXPECT().CodeAt(mock.Anything, addr, (*big.Int)(nil)).
			RunAndReturn(func(context.Context, common.Address, *big.Int) ([]byte, error) {
				calls++
				if calls < 3 {
					return nil, nil
				}

				return []byte{0xDE, 0xAD}, nil
			})

		require.NoError(t, WaitForDeployedCode(newBundle(t), cldf_evm.Chain{Client: client}, addr))
		require.Equal(t, 3, calls)
	})

	t.Run("retries after a transient RPC error then succeeds", func(t *testing.T) { //nolint:paralleltest
		origDelay := codeCheckInitialDelay
		t.Cleanup(func() { codeCheckInitialDelay = origDelay })
		codeCheckInitialDelay = time.Millisecond

		client := cldf_evm.NewMockOnchainClient(t)
		var calls int
		client.EXPECT().CodeAt(mock.Anything, addr, (*big.Int)(nil)).
			RunAndReturn(func(context.Context, common.Address, *big.Int) ([]byte, error) {
				calls++
				if calls < 2 {
					return nil, errors.New("transient rpc error")
				}

				return []byte{0xDE, 0xAD}, nil
			})

		require.NoError(t, WaitForDeployedCode(newBundle(t), cldf_evm.Chain{Client: client}, addr))
		require.Equal(t, 2, calls)
	})

	t.Run("gives up after the attempt budget is exhausted", func(t *testing.T) { //nolint:paralleltest
		origAttempts, origDelay := codeCheckMaxAttempts, codeCheckInitialDelay
		t.Cleanup(func() { codeCheckMaxAttempts, codeCheckInitialDelay = origAttempts, origDelay })
		codeCheckMaxAttempts, codeCheckInitialDelay = 3, time.Millisecond

		client := cldf_evm.NewMockOnchainClient(t)
		client.EXPECT().CodeAt(mock.Anything, addr, (*big.Int)(nil)).Return(nil, nil)

		err := WaitForDeployedCode(newBundle(t), cldf_evm.Chain{Client: client}, addr)
		require.ErrorContains(t, err, "empty bytecode")
	})

	t.Run("returns context error when the context is done", func(t *testing.T) { //nolint:paralleltest
		origAttempts := codeCheckMaxAttempts
		t.Cleanup(func() { codeCheckMaxAttempts = origAttempts })
		codeCheckMaxAttempts = 5

		client := cldf_evm.NewMockOnchainClient(t)
		client.EXPECT().CodeAt(mock.Anything, addr, (*big.Int)(nil)).Return(nil, nil)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		lggr, err := logger.New()
		require.NoError(t, err)
		bundle := operations.NewBundle(func() context.Context { return ctx }, lggr, operations.NewMemoryReporter())

		err = WaitForDeployedCode(bundle, cldf_evm.Chain{Client: client}, addr)
		require.ErrorIs(t, err, context.Canceled)
	})
}
