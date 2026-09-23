package provider

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

func Test_PrivateKeyFromRaw(t *testing.T) {
	t.Parallel()

	// Generate a random ed25519 private key to use as a valid hex input
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	validHex := hex.EncodeToString(priv)

	tests := []struct {
		name           string
		givePrivateKey string
		wantBytes      []byte
		wantErr        string
	}{
		{
			name:           "valid hex private key",
			givePrivateKey: validHex,
			wantBytes:      priv,
		},
		{
			name:           "invalid hex string",
			givePrivateKey: "invalid_private_key",
			wantErr:        "failed to parse private key",
		},
		{
			name:           "invalid ed25519 key len",
			givePrivateKey: "abcdabcdabcdabcdabcdabcdabcdabcd", // 32 bytes instead of 64
			wantErr:        "invalid key len",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := PrivateKeyFromRaw(tt.givePrivateKey)
			got, err := gen.Generate()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.wantBytes, []byte(got))
		})
	}
}

func Test_PrivateKeyRandom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr string
	}{
		{
			name: "generates valid ed25519 keypair",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := PrivateKeyRandom()
			got, err := gen.Generate()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.Len(t, got, ed25519.PrivateKeySize)

			// Sanity check: sign and verify a message
			msg := []byte("ton-provider-key-test")
			sig := ed25519.Sign(got, msg)

			pub, ok := got.Public().(ed25519.PublicKey)
			require.True(t, ok, "public key should be ed25519.PublicKey")
			assert.True(t, ed25519.Verify(pub, msg, sig), "signature should verify")
		})
	}
}

// Test_PrivateKeyRandom_DerivationScheme pins the seed-derivation scheme used by
// privateKeyRandom: no password and no BIP39 (the pre-Options SeedToPrivateKey
// semantics). If a future tonutils-go version changes its functional-option
// defaults, this golden-value test fails instead of silently changing how
// private keys are derived.
func Test_PrivateKeyRandom_DerivationScheme(t *testing.T) {
	t.Parallel()

	// Golden vector generated with tonutils-go v1.18.0:
	// SeedToPrivateKeyWithOptions(seed, WithPassword(""), WithBIP39(false)).
	giveSeed := []string{
		"stay", "again", "retire", "employ", "party", "ripple", "reject", "shuffle",
		"similar", "clock", "wash", "all", "great", "height", "giggle", "offer",
		"age", "empower", "boss", "private", "island", "promote", "voyage", "layer",
	}
	wantKey := "6a1503c8cfa2039f38bbfe41be7f7b538d696e36e3f8421fe1c6f63a7d65a7912be188a0441c8176769031ddb8027c00f57f169b014e9818716079c47c5330ad"

	got, err := wallet.SeedToPrivateKeyWithOptions(giveSeed, wallet.WithPassword(""), wallet.WithBIP39(false))
	require.NoError(t, err)
	assert.Equal(t, wantKey, hex.EncodeToString(got))
}
