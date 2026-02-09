package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeypairFromHex_Generate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hexKey  string
		wantErr string
	}{
		{
			name:   "valid hex key without prefix",
			hexKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		{
			name:   "valid hex key with 0x prefix",
			hexKey: "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		{
			name:    "empty hex key",
			hexKey:  "",
			wantErr: "hex key is empty",
		},
		{
			name:    "invalid hex",
			hexKey:  "not_valid_hex",
			wantErr: "failed to create keypair from hex",
		},
		{
			name:    "wrong length",
			hexKey:  "0123456789abcdef",
			wantErr: "failed to create keypair from hex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := KeypairFromHex(tt.hexKey)
			require.NotNil(t, gen)

			signer, err := gen.Generate()

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, signer)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, signer)

			// Verify the signer works
			address := signer.Address()
			assert.NotEmpty(t, address)

			message := []byte("test message")
			sig, err := signer.Sign(message)
			require.NoError(t, err)
			assert.NotNil(t, sig)
		})
	}
}

func TestKeypairFromHex_GenerateConsistent(t *testing.T) {
	t.Parallel()

	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	gen := KeypairFromHex(hexKey)

	// Generate twice
	signer1, err := gen.Generate()
	require.NoError(t, err)

	signer2, err := gen.Generate()
	require.NoError(t, err)

	// Both should produce the same address
	assert.Equal(t, signer1.Address(), signer2.Address())
}

func TestKeypairRandom_Generate(t *testing.T) {
	t.Parallel()

	gen := KeypairRandom()
	require.NotNil(t, gen)

	signer, err := gen.Generate()
	require.NoError(t, err)
	require.NotNil(t, signer)

	// Verify the signer works
	address := signer.Address()
	assert.NotEmpty(t, address)

	message := []byte("test message")
	sig, err := signer.Sign(message)
	require.NoError(t, err)
	assert.NotNil(t, sig)
}

func TestKeypairRandom_GenerateUnique(t *testing.T) {
	t.Parallel()

	gen := KeypairRandom()

	// Generate multiple times
	addresses := make(map[string]bool)
	for range 10 {
		signer, err := gen.Generate()
		require.NoError(t, err)

		address := signer.Address()
		assert.NotEmpty(t, address)

		// Each address should be unique (since they're random)
		assert.False(t, addresses[address], "address %s was generated more than once", address)
		addresses[address] = true
	}

	assert.Len(t, addresses, 10, "should have generated 10 unique addresses")
}

func TestKeypairGenerator_Interface(t *testing.T) {
	t.Parallel()

	// Verify both types implement KeypairGenerator
	var _ KeypairGenerator = (*keypairFromHex)(nil)
	var _ KeypairGenerator = (*keypairRandom)(nil)

	// Test interface methods
	hexGen := KeypairFromHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	gen1 := hexGen
	signer1, err := gen1.Generate()
	require.NoError(t, err)
	assert.NotNil(t, signer1)

	randomGen := KeypairRandom()
	gen2 := randomGen
	signer2, err := gen2.Generate()
	require.NoError(t, err)
	assert.NotNil(t, signer2)
}

func TestKeypairFromHex_GenerateWithDifferentKeys(t *testing.T) {
	t.Parallel()

	hexKey1 := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hexKey2 := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

	gen1 := KeypairFromHex(hexKey1)
	gen2 := KeypairFromHex(hexKey2)

	signer1, err := gen1.Generate()
	require.NoError(t, err)

	signer2, err := gen2.Generate()
	require.NoError(t, err)

	// Different keys should produce different addresses
	assert.NotEqual(t, signer1.Address(), signer2.Address())
}

func TestKeypairFromHex_GenerateSignAndVerify(t *testing.T) {
	t.Parallel()

	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	gen := KeypairFromHex(hexKey)
	signer, err := gen.Generate()
	require.NoError(t, err)

	message := []byte("test message for signing")
	sig, err := signer.Sign(message)
	require.NoError(t, err)
	assert.NotNil(t, sig)
	assert.NotEmpty(t, sig)

	// Generate another signer with the same key to verify cross-compatibility
	gen2 := KeypairFromHex(hexKey)
	signer2, err := gen2.Generate()
	require.NoError(t, err)

	assert.Equal(t, signer.Address(), signer2.Address(), "same key should produce same address")
}

func TestKeypairRandom_GenerateMultipleFromSameGen(t *testing.T) {
	t.Parallel()

	gen := KeypairRandom()

	// Generate from the same generator multiple times
	// Each call should produce a different random keypair
	signer1, err := gen.Generate()
	require.NoError(t, err)

	signer2, err := gen.Generate()
	require.NoError(t, err)

	// Should be different since they're random
	assert.NotEqual(t, signer1.Address(), signer2.Address())
}
