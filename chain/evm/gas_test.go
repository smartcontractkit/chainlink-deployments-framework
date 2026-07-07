package evm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type stubGasLimitBufferClient struct {
	bufferBps uint64
}

func (c stubGasLimitBufferClient) GasLimitBufferBps() uint64 {
	return c.bufferBps
}

func TestApplyGasLimitBuffer(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(0), ApplyGasLimitBuffer(0, 2500))
	require.Equal(t, uint64(1_000_000), ApplyGasLimitBuffer(1_000_000, 0))
	require.Equal(t, uint64(1_250_000), ApplyGasLimitBuffer(1_000_000, 2500))
	require.Equal(t, uint64(1_300_000), ApplyGasLimitBuffer(1_000_000, 3000))
}

func TestGasLimitBufferBpsFromClient(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint64(0), GasLimitBufferBpsFromClient(nil))
	require.Equal(t, uint64(2500), GasLimitBufferBpsFromClient(stubGasLimitBufferClient{bufferBps: 2500}))
}
