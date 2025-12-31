package jd

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainTypeToFamily(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		chainType nodev1.ChainType
		want      string
		wantErr   string
	}{
		{
			name:      "EVM chain type",
			chainType: nodev1.ChainType_CHAIN_TYPE_EVM,
			want:      chain_selectors.FamilyEVM,
		},
		{
			name:      "Aptos chain type",
			chainType: nodev1.ChainType_CHAIN_TYPE_APTOS,
			want:      chain_selectors.FamilyAptos,
		},
		{
			name:      "Solana chain type",
			chainType: nodev1.ChainType_CHAIN_TYPE_SOLANA,
			want:      chain_selectors.FamilySolana,
		},
		{
			name:      "Starknet chain type",
			chainType: nodev1.ChainType_CHAIN_TYPE_STARKNET,
			want:      chain_selectors.FamilyStarknet,
		},
		{
			name:      "Tron chain type",
			chainType: nodev1.ChainType_CHAIN_TYPE_TRON,
			want:      chain_selectors.FamilyTron,
		},
		{
			name:      "TON chain type",
			chainType: nodev1.ChainType_CHAIN_TYPE_TON,
			want:      chain_selectors.FamilyTon,
		},
		{
			name:      "Sui chain type",
			chainType: nodev1.ChainType_CHAIN_TYPE_SUI,
			want:      chain_selectors.FamilySui,
		},
		{
			name:      "unspecified chain type returns error",
			chainType: nodev1.ChainType_CHAIN_TYPE_UNSPECIFIED,
			wantErr:   "chain type must be specified",
		},
		{
			name:      "invalid chain type returns error",
			chainType: nodev1.ChainType(9999),
			wantErr:   "unsupported chain type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ChainTypeToFamily(tt.chainType)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
