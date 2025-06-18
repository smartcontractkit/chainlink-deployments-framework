package provider

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_TransactorGenerator(t *testing.T) {
	t.Parallel()

	// Setup the private key
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	privKeyBytes := crypto.FromECDSA(privKey)

	// Convert the private and public keys to hex strings
	hexPrivKey := hex.EncodeToString(privKeyBytes)
	privKeyHex := crypto.PubkeyToAddress(privKey.PublicKey).Hex()

	chainID := new(big.Int).SetUint64(chain_selectors.TEST_1000.EvmChainID)

	tests := []struct {
		name        string
		givePrivKey string
		giveChainID *big.Int
		want        string
		wantErr     string
	}{
		{
			name:        "valid default account and private key",
			givePrivKey: hexPrivKey,
			giveChainID: chainID,
			want:        privKeyHex,
		},
		{
			name:        "invalid private key",
			givePrivKey: "invalid",
			giveChainID: chainID,
			wantErr:     "failed to convert private key to ECDSA",
		},
		{
			name:        "invalid chain ID",
			givePrivKey: hexPrivKey,
			giveChainID: nil, // nil chain ID should trigger an error
			wantErr:     "no chain id specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := TransactorFromRaw(tt.givePrivKey)

			got, err := gen.Generate(tt.giveChainID)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got.From.Hex())
			}
		})
	}
}

func Test_TransactorRandom(t *testing.T) {
	t.Parallel()

	chainID := new(big.Int).SetUint64(chain_selectors.TEST_1000.EvmChainID)

	tests := []struct {
		name        string
		giveChainID *big.Int
		wantErr     string
	}{
		{
			name:        "valid",
			giveChainID: chainID,
		},
		{
			name:        "invalid chain ID",
			giveChainID: nil, // nil chain ID should trigger an error
			wantErr:     "no chain id specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := TransactorRandom()

			got, err := gen.Generate(tt.giveChainID)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, got.From)
			}
		})
	}
}
