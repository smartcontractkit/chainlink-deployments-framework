package evm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type stubMaxTxGasLimitClient struct{ max uint64 }

func (c stubMaxTxGasLimitClient) MaxTxGasLimit() uint64 {
	return c.max
}

func TestApplyGasLimitBuffer(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(0), ApplyGasLimitBuffer(0, 2500))
	require.Equal(t, uint64(1_000_000), ApplyGasLimitBuffer(1_000_000, 0))
	require.Equal(t, uint64(1_250_000), ApplyGasLimitBuffer(1_000_000, 2500))
	require.Equal(t, uint64(1_300_000), ApplyGasLimitBuffer(1_000_000, 3000))
	require.Equal(t, uint64(12_500_000_000_000_000), ApplyGasLimitBuffer(10_000_000_000_000_000, 2500))
	require.Equal(t, ^uint64(0), ApplyGasLimitBuffer(^uint64(0), 2500))
}

func TestCapGasLimit(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(1_000_000), CapGasLimit(1_000_000, 0))
	require.Equal(t, uint64(1_000_000), CapGasLimit(1_000_000, 2_000_000))
	require.Equal(t, EIP7825MaxTxGasLimit, CapGasLimit(20_000_000, EIP7825MaxTxGasLimit))
}

func TestMaxTxGasLimitFromClient(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(0), MaxTxGasLimitFromClient(nil))
	require.Equal(t, EIP7825MaxTxGasLimit, MaxTxGasLimitFromClient(stubMaxTxGasLimitClient{max: EIP7825MaxTxGasLimit}))
}
