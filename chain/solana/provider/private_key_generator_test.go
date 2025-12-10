package provider

import (
	"testing"

	"github.com/gagliardetto/solana-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PrivateKeyFromRaw(t *testing.T) {
	t.Parallel()

	// Generate a random private key for testing
	privateKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	tests := []struct {
		name           string
		givePrivateKey string
		wantAddr       string
		wantErr        string
	}{
		{
			name:           "valid private key",
			givePrivateKey: privateKey.String(),
			wantAddr:       privateKey.PublicKey().String(),
		},
		{
			name:           "invalid private key",
			givePrivateKey: "invalid_private_key",
			wantErr:        "failed to parse private key",
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
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.givePrivateKey, got.String())
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

			gen := PrivateKeyRandom()
			got, err := gen.Generate()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.NotEmpty(t, got.String())
				assert.True(t, got.IsValid())
			}
		})
	}
}
