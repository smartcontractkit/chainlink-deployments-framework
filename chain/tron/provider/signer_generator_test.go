package provider

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SignerGenCTFDefault(t *testing.T) {
	t.Parallel()

	gen := SignerGenCTFDefault()
	require.NotNil(t, gen)

	addr, err := gen.GetAddress()
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	// Test that the address is consistent
	addr2, err := gen.GetAddress()
	require.NoError(t, err)
	require.Equal(t, addr, addr2)
}

func Test_SignerGenPrivateKey(t *testing.T) {
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
			wantErr:        "failed to parse private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := SignerGenPrivateKey(tt.givePrivateKey)
			gotAddr, err := gen.GetAddress()

			if tt.wantErr != "" {
				// For error cases, GetAddress should return an error
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, gotAddr)
				assert.Equal(t, tt.wantAddr, gotAddr.String())

				// Test consistency
				gotAddr2, err2 := gen.GetAddress()
				require.NoError(t, err2)
				assert.Equal(t, gotAddr, gotAddr2)
			}
		})
	}
}

func Test_SignerRandom(t *testing.T) {
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

			gen := SignerRandom()
			gotAddr, err := gen.GetAddress()

			if tt.wantErr != "" {
				// For error cases, GetAddress should return an error
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, gotAddr)

				// Test consistency
				gotAddr2, err2 := gen.GetAddress()
				require.NoError(t, err2)
				assert.Equal(t, gotAddr, gotAddr2)
			}
		})
	}
}

func Test_SignerGenKMS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		keyID      string
		keyRegion  string
		awsProfile string
		wantErr    string
	}{
		{
			name:       "valid configuration",
			keyID:      "test-key-id",
			keyRegion:  "us-east-1",
			awsProfile: "test-profile",
		},
		{
			name:       "empty aws profile",
			keyID:      "test-key-id",
			keyRegion:  "us-west-2",
			awsProfile: "",
		},
		{
			name:       "empty key id",
			keyID:      "",
			keyRegion:  "us-east-1",
			awsProfile: "test-profile",
			wantErr:    "failed to create KMS client", // This would happen during lazy init
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := SignerGenKMS(tt.keyID, tt.keyRegion, tt.awsProfile)
			require.NotNil(t, gen)
			require.NotNil(t, gen.signer)

			// Verify the KMS signer has the correct configuration
			assert.Equal(t, tt.keyID, gen.signer.KeyID)
			assert.Equal(t, tt.keyRegion, gen.signer.KeyRegion)
			assert.Equal(t, tt.awsProfile, gen.signer.AWSProfile)

			// Note: We can't easily test Sign() and GetAddress() without mocking
			// the AWS KMS service, which would require dependency injection or
			// more complex mocking setup. The KMS signer tests cover the core logic.
		})
	}
}

func Test_SignerGenerators_Signing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		signer SignerGenerator
	}{
		{
			name:   "SignerGenPrivateKey with CTF default key",
			signer: SignerGenCTFDefault(),
		},
		{
			name:   "SignerRandom with generated key",
			signer: SignerRandom(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test that we can get a valid address from the signer
			signerAddr, err := tt.signer.GetAddress()
			require.NoError(t, err, "Failed to get signer address")
			require.NotEmpty(t, signerAddr.String(), "Signer address should not be empty")

			// Test that we can sign a transaction hash
			// Use a sample transaction hash (32 bytes) for testing
			sampleTxHash := []byte("test_transaction_hash_32_bytes_!")
			require.Len(t, sampleTxHash, 32, "Sample hash should be 32 bytes")

			signature, err := tt.signer.Sign(context.Background(), sampleTxHash)
			require.NoError(t, err, "Failed to sign transaction hash")
			require.NotEmpty(t, signature, "Signature should not be empty")
			require.Len(t, signature, 65, "TRON signature should be 65 bytes (r+s+v)")

			// Verify the signature format is correct for TRON
			// The last byte should be the recovery ID in range [0, 1] (unlike Ethereum which uses [27, 30])
			recoveryID := signature[64]
			require.LessOrEqual(t, recoveryID, uint8(1),
				"Recovery ID should be in range [0, 1] for TRON, got: %d", recoveryID)

			// Test signature consistency - signing the same hash should produce the same result
			signature2, err := tt.signer.Sign(context.Background(), sampleTxHash)
			require.NoError(t, err, "Failed to sign transaction hash again")
			require.Equal(t, signature, signature2, "Signatures should be deterministic for the same input")
		})
	}
}
