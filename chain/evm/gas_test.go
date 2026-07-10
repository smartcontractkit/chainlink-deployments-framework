package evm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type stubGasEstimateAdjusterClient struct {
	bufferBps     uint64
	maxTxGasLimit uint64
}

func (c stubGasEstimateAdjusterClient) GasLimitBufferBps() uint64 {
	return c.bufferBps
}

func (c stubGasEstimateAdjusterClient) MaxTxGasLimit() uint64 {
	return c.maxTxGasLimit
}

func TestApplyGasLimitBuffer(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(0), ApplyGasLimitBuffer(0, 2500))
	require.Equal(t, uint64(1_000_000), ApplyGasLimitBuffer(1_000_000, 0))
	require.Equal(t, uint64(1_250_000), ApplyGasLimitBuffer(1_000_000, 2500))
	require.Equal(t, uint64(1_300_000), ApplyGasLimitBuffer(1_000_000, 3000))
	require.Equal(t, ^uint64(0), ApplyGasLimitBuffer(^uint64(0), 2500))
}

func TestCapGasLimit(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(1_000_000), CapGasLimit(1_000_000, 0))
	require.Equal(t, uint64(1_000_000), CapGasLimit(1_000_000, 2_000_000))
	require.Equal(t, EIP7825MaxTxGasLimit, CapGasLimit(20_000_000, EIP7825MaxTxGasLimit))
}

func TestApplyGasLimitWithBufferAndCap(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(1_250_000), ApplyGasLimitWithBufferAndCap(1_000_000, 2500, 0))
	require.Equal(t, EIP7825MaxTxGasLimit, ApplyGasLimitWithBufferAndCap(14_000_000, 2500, EIP7825MaxTxGasLimit))
}

func TestGasLimitBufferBpsFromClient(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(0), GasLimitBufferBpsFromClient(nil))
	require.Equal(t, uint64(2500), GasLimitBufferBpsFromClient(stubGasEstimateAdjusterClient{bufferBps: 2500}))
}

func TestMaxTxGasLimitFromClient(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(0), MaxTxGasLimitFromClient(nil))
	require.Equal(t, EIP7825MaxTxGasLimit, MaxTxGasLimitFromClient(stubGasEstimateAdjusterClient{maxTxGasLimit: EIP7825MaxTxGasLimit}))
}
