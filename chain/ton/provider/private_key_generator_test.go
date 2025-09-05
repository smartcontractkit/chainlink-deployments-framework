package provider

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
