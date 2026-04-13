package proposalutils

import (
	"math/big"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionForChain(t *testing.T) {
	t.Parallel()

	evmSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
	solanaSelector := chainsel.SOLANA_DEVNET.Selector
	aptosSelector := chainsel.APTOS_TESTNET.Selector

	tests := []struct {
		name         string
		chain        uint64
		toAddress    string
		data         []byte
		value        *big.Int
		contractType string
		tags         []string

		wantErr     string
		wantTo      string
		wantType    string
		wantTags    []string
		wantDataLen int
	}{
		{
			name:         "evm happy path",
			chain:        evmSelector,
			toAddress:    "0x1234567890abcdef1234567890abcdef12345678",
			data:         []byte{0xde, 0xad},
			value:        big.NewInt(42),
			contractType: "Router",
			tags:         []string{"deploy", "v2"},
			wantTo:       "0x1234567890AbcdEF1234567890aBcdef12345678",
			wantType:     "Router",
			wantTags:     []string{"deploy", "v2"},
			wantDataLen:  2,
		},
		{
			name:         "evm nil value treated as zero",
			chain:        evmSelector,
			toAddress:    "0x0000000000000000000000000000000000000001",
			data:         nil,
			value:        nil,
			contractType: "Token",
			tags:         nil,
			wantTo:       "0x0000000000000000000000000000000000000001",
			wantType:     "Token",
			wantTags:     nil,
			wantDataLen:  0,
		},
		{
			name:         "evm invalid address",
			chain:        evmSelector,
			toAddress:    "not-a-hex-address",
			data:         []byte{0x01},
			value:        big.NewInt(0),
			contractType: "Test",
			wantErr:      "invalid EVM address",
		},
		{
			name:         "solana happy path",
			chain:        solanaSelector,
			toAddress:    "11111111111111111111111111111111",
			data:         []byte{0x01, 0x02},
			value:        big.NewInt(0),
			contractType: "Program",
			tags:         []string{"sol"},
			wantTo:       "11111111111111111111111111111111",
			wantType:     "Program",
			wantTags:     []string{"sol"},
			wantDataLen:  2,
		},
		{
			name:         "unsupported chain family (aptos)",
			chain:        aptosSelector,
			toAddress:    "0xAPTOS",
			data:         nil,
			value:        big.NewInt(0),
			contractType: "Module",
			wantErr:      "unsupported chain family",
		},
		{
			name:         "invalid chain selector",
			chain:        0,
			toAddress:    "0xabc",
			data:         nil,
			value:        big.NewInt(0),
			contractType: "Unknown",
			wantErr:      "failed to get chain family for chain 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tx, err := TransactionForChain(tt.chain, tt.toAddress, tt.data, tt.value, tt.contractType, tt.tags)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Equal(t, mcmstypes.Transaction{}, tx)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTo, tx.To)
			assert.Equal(t, tt.wantType, tx.ContractType)
			assert.Equal(t, tt.wantTags, tx.Tags)
			assert.Len(t, tx.Data, tt.wantDataLen)
		})
	}
}

func TestBatchOperationForChain(t *testing.T) {
	t.Parallel()

	evmSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector

	tests := []struct {
		name         string
		chain        uint64
		toAddress    string
		data         []byte
		value        *big.Int
		contractType string
		tags         []string

		wantErr string
	}{
		{
			name:         "wraps transaction in batch operation",
			chain:        evmSelector,
			toAddress:    "0x1234567890abcdef1234567890abcdef12345678",
			data:         []byte{0xca, 0xfe},
			value:        big.NewInt(100),
			contractType: "Router",
			tags:         []string{"batch"},
		},
		{
			name:         "propagates TransactionForChain error",
			chain:        0,
			toAddress:    "0xabc",
			data:         nil,
			value:        big.NewInt(0),
			contractType: "X",
			wantErr:      "failed to create transaction for chain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bop, err := BatchOperationForChain(tt.chain, tt.toAddress, tt.data, tt.value, tt.contractType, tt.tags)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Equal(t, mcmstypes.BatchOperation{}, bop)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, mcmstypes.ChainSelector(tt.chain), bop.ChainSelector)
			require.Len(t, bop.Transactions, 1)

			tx := bop.Transactions[0]
			assert.Equal(t, tt.contractType, tx.ContractType)
			assert.Equal(t, tt.tags, tx.Tags)
			assert.Equal(t, tt.data, tx.Data)
		})
	}
}
