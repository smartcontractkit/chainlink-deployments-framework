package sui_test

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
)

func TestNewSignerFromHexPrivateKey(t *testing.T) {
	t.Parallel()
	validHexKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	validHexKeyWith0x := "0x" + validHexKey

	t.Run("creates signer with valid hex private key", func(t *testing.T) {
		t.Parallel()
		signer, err := sui.NewSignerFromHexPrivateKey(validHexKey)
		require.NoError(t, err)
		assert.NotNil(t, signer)
	})

	t.Run("creates signer with valid hex private key with 0x prefix", func(t *testing.T) {
		t.Parallel()
		signer, err := sui.NewSignerFromHexPrivateKey(validHexKeyWith0x)
		require.NoError(t, err)
		require.NotNil(t, signer)
	})

	t.Run("fails with invalid hex characters", func(t *testing.T) {
		t.Parallel()
		invalidHexKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdeG" // G is invalid
		signer, err := sui.NewSignerFromHexPrivateKey(invalidHexKey)
		require.Error(t, err)
		assert.Nil(t, signer)
		assert.Contains(t, err.Error(), "invalid hex private key")
	})

	t.Run("fails with incorrect length - too short", func(t *testing.T) {
		t.Parallel()
		shortHexKey := "1234567890abcdef" // 16 chars instead of 64
		signer, err := sui.NewSignerFromHexPrivateKey(shortHexKey)
		require.Error(t, err)
		assert.Nil(t, signer)
		assert.Contains(t, err.Error(), "hex private key must be exactly 64 characters")
	})

	t.Run("fails with incorrect length - too long", func(t *testing.T) {
		t.Parallel()
		longHexKey := validHexKey + "ab" // 66 chars instead of 64
		signer, err := sui.NewSignerFromHexPrivateKey(longHexKey)
		require.Error(t, err)
		assert.Nil(t, signer)
		assert.Contains(t, err.Error(), "hex private key must be exactly 64 characters")
	})

	t.Run("fails with empty string", func(t *testing.T) {
		t.Parallel()
		signer, err := sui.NewSignerFromHexPrivateKey("")
		require.Error(t, err)
		assert.Nil(t, signer)
		assert.Contains(t, err.Error(), "hex private key must be exactly 64 characters")
	})

	t.Run("fails with only 0x prefix", func(t *testing.T) {
		t.Parallel()
		signer, err := sui.NewSignerFromHexPrivateKey("0x")
		require.Error(t, err)
		assert.Nil(t, signer)
		assert.Contains(t, err.Error(), "hex private key must be exactly 64 characters")
	})
}

func TestSuiSigner_Integration(t *testing.T) {
	t.Run("hex private key to signer workflow", func(t *testing.T) {
		t.Parallel()
		// Generate a random hex private key
		privateKeyBytes := make([]byte, 32)
		_, err := rand.Read(privateKeyBytes)
		require.NoError(t, err)

		hexPrivateKey := hex.EncodeToString(privateKeyBytes)

		// Create signer from hex private key
		signer, err := sui.NewSignerFromHexPrivateKey(hexPrivateKey)
		require.NoError(t, err)

		// Get address
		address, err := signer.GetAddress()
		require.NoError(t, err)
		assert.NotEmpty(t, address)

		// Sign a message
		message := []byte("integration test message")
		signatures, err := signer.Sign(message)
		require.NoError(t, err)
		assert.Len(t, signatures, 1)
		assert.NotEmpty(t, signatures[0])
	})

	t.Run("seed to signer workflow", func(t *testing.T) {
		t.Parallel()
		// Generate a random seed
		seed := make([]byte, 32)
		_, err := rand.Read(seed)
		require.NoError(t, err)

		// Create signer from seed
		signer, err := sui.NewSignerFromSeed(seed)
		require.NoError(t, err)

		// Get address
		address, err := signer.GetAddress()
		require.NoError(t, err)
		assert.NotEmpty(t, address)

		// Sign a message
		message := []byte("integration test message")
		signatures, err := signer.Sign(message)
		require.NoError(t, err)
		assert.Len(t, signatures, 1)
		assert.NotEmpty(t, signatures[0])
	})
}

func TestSuiSigner_DeterministicSignature(t *testing.T) {
	t.Parallel()
	t.Run("produces expected signature for known seed and message", func(t *testing.T) {
		t.Parallel()
		// Use deterministic seed - exactly 32 bytes
		seed := []byte("deterministic_test_seed_32bytes!")
		message := []byte("test message for signature capture")

		signer, err := sui.NewSignerFromSeed(seed)
		require.NoError(t, err)

		// Test address
		address, err := signer.GetAddress()
		require.NoError(t, err)
		expectedAddress := "0x523ab0c832f9b0eec27512190999827222efddb4b537458efd4f26c317a9b350"
		assert.Equal(t, expectedAddress, address)

		// Test signature
		signatures, err := signer.Sign(message)
		require.NoError(t, err)
		require.Len(t, signatures, 1)

		expectedSignature := "ADuBDPC+jHqozHBQ+OyJ+tvuARPZskHGXPDJaH172+G+Xf1FDxLjPNq87w7m2eAf8Ow61oyTw6IGd7kCYzKsIgcUWFegJRByHtBZ86VlbhBe4QOlzA7IVc1xHI4r/8pb3Q=="
		assert.Equal(t, expectedSignature, signatures[0])
	})
}
