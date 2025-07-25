package provider

import (
	"crypto/ecdsa"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AccountGenPrivateKey(t *testing.T) {
	t.Parallel()

	// Generate a random private key for testing
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	privBytes := crypto.FromECDSA(privateKey)
	privKey := hex.EncodeToString(privBytes)

	// Convert to Tron-style Base58 address
	tronAddr := address.PubkeyToAddress(privateKey.PublicKey)

	tests := []struct {
		name           string
		givePrivateKey string
		wantAddr       string
		wantPrivateKey *ecdsa.PrivateKey
		wantErr        string
	}{
		{
			name:           "valid private key",
			givePrivateKey: privKey,
			wantAddr:       tronAddr.String(),
			wantPrivateKey: privateKey,
		},
		{
			name:           "invalid private key",
			givePrivateKey: "invalid_private_key",
			wantErr:        "failed to decode hex-encoded private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := AccountGenPrivateKey(tt.givePrivateKey)
			gotKs, gotAddr, err := gen.Generate()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, gotKs)
				assert.NotNil(t, gotAddr)
				assert.Equal(t, tt.wantAddr, gotAddr.String())
				assert.Equal(t, tt.wantPrivateKey, gotKs.Keys[gotAddr.String()])
			}
		})
	}
}

func Test_PrivateKeyRandom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantAddr string
		wantErr  string
	}{
		{
			name: "valid private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := AccountRandom()
			gotKs, gotAddr, err := gen.Generate()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, gotKs)
				assert.NotNil(t, gotAddr)
				require.Contains(t, gotKs.Keys, gotAddr.String())
			}
		})
	}
}
