package gas_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/gas"
)

func TestCapGasLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		gas      uint64
		maxTxGas uint64
		want     uint64
	}{
		{name: "no cap", gas: 1_000_000, maxTxGas: 0, want: 1_000_000},
		{name: "below cap", gas: 12_000_000, maxTxGas: gas.EIP7825MaxTxGasLimit, want: 12_000_000},
		{name: "at cap", gas: gas.EIP7825MaxTxGasLimit, maxTxGas: gas.EIP7825MaxTxGasLimit, want: gas.EIP7825MaxTxGasLimit},
		{name: "above cap", gas: 20_000_000, maxTxGas: gas.EIP7825MaxTxGasLimit, want: gas.EIP7825MaxTxGasLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, gas.CapGasLimit(tt.gas, tt.maxTxGas))
		})
	}
}
