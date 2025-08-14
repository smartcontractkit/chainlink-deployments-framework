package provider

import (
	"crypto/ecdsa"
	"encoding/asn1"
	"math/big"
	"testing"

	kmslib "github.com/aws/aws-sdk-go/service/kms"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms"
)

// MockKMSClient is a mock implementation of kms.Client for testing
type MockKMSClient struct {
	mock.Mock
}

func (m *MockKMSClient) GetPublicKey(input *kmslib.GetPublicKeyInput) (*kmslib.GetPublicKeyOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*kmslib.GetPublicKeyOutput), args.Error(1)
}

func (m *MockKMSClient) Sign(input *kmslib.SignInput) (*kmslib.SignOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*kmslib.SignOutput), args.Error(1)
}

func TestNewKMSSigner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		keyID      string
		keyRegion  string
		awsProfile string
	}{
		{
			name:       "valid parameters",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			signer := newKMSSigner(tt.keyID, tt.keyRegion, tt.awsProfile)

			require.NotNil(t, signer)
			assert.Equal(t, tt.keyID, signer.KeyID)
			assert.Equal(t, tt.keyRegion, signer.KeyRegion)
			assert.Equal(t, tt.awsProfile, signer.AWSProfile)

			// Verify lazy initialization fields are in initial state
			assert.Nil(t, signer.client)
			assert.Empty(t, signer.kmsKeyID)
			assert.Nil(t, signer.ecdsaPublicKey)
			assert.Empty(t, signer.address)
			assert.NoError(t, signer.initError)
		})
	}
}

func TestKMSSigner_LazyInitialization(t *testing.T) {
	t.Parallel()

	// Generate a test private key
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	signer := newKMSSigner("test-key-id", "us-east-1", "test-profile")
	require.NotNil(t, signer)

	// Verify initial state - should not be initialized yet
	assert.Nil(t, signer.client)
	assert.Empty(t, signer.kmsKeyID)
	assert.Nil(t, signer.ecdsaPublicKey)
	assert.Empty(t, signer.address)
	require.NoError(t, signer.initError)

	t.Run("signature conversion with real signature", func(t *testing.T) {
		// Create a real signature for testing
		hash := crypto.Keccak256([]byte("test message"))
		realSig, err := crypto.Sign(hash, privateKey)
		require.NoError(t, err)

		// Extract r and s from the real signature
		r := new(big.Int).SetBytes(realSig[:32])
		s := new(big.Int).SetBytes(realSig[32:64])

		// Create KMS signature structure
		ecdsaSig := kms.ECDSASig{
			R: asn1.RawValue{Bytes: r.Bytes()},
			S: asn1.RawValue{Bytes: s.Bytes()},
		}

		kmsSignature, err := asn1.Marshal(ecdsaSig)
		require.NoError(t, err)

		// Test signature conversion
		tronSig, err := kmsToTronSig(kmsSignature, &privateKey.PublicKey, hash)

		// The conversion should succeed with a real signature
		require.NoError(t, err)
		require.Len(t, tronSig, 65) // 32 bytes r + 32 bytes s + 1 byte v

		// Verify the r and s components
		rResult := tronSig[:32]
		sResult := tronSig[32:64]
		recoveryID := tronSig[64]

		// Check that r and s are properly padded to 32 bytes
		assert.Len(t, rResult, 32)
		assert.Len(t, sResult, 32)
		assert.LessOrEqual(t, recoveryID, byte(1)) // Recovery ID should be 0 or 1
	})
}

func TestKMSToTronSig(t *testing.T) {
	t.Parallel()

	// Generate a test private key and create a real signature
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	hash := crypto.Keccak256([]byte("test message"))

	// Create a real ECDSA signature using the private key
	realSig, err := crypto.Sign(hash, privateKey)
	require.NoError(t, err)
	require.Len(t, realSig, 65)

	// Extract r and s from the real signature
	r := new(big.Int).SetBytes(realSig[:32])
	s := new(big.Int).SetBytes(realSig[32:64])

	tests := []struct {
		name    string
		r       *big.Int
		s       *big.Int
		wantErr bool
	}{
		{
			name: "valid signature from real ECDSA",
			r:    r,
			s:    s,
			// This should work because we're using r,s from a real signature
		},
		{
			name:    "arbitrary values that likely won't recover",
			r:       big.NewInt(12345),
			s:       big.NewInt(67890),
			wantErr: true, // Expect this to fail recovery
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create KMS signature
			ecdsaSig := kms.ECDSASig{
				R: asn1.RawValue{Bytes: tt.r.Bytes()},
				S: asn1.RawValue{Bytes: tt.s.Bytes()},
			}

			kmsSignature, err := asn1.Marshal(ecdsaSig)
			require.NoError(t, err)

			tronSig, err := kmsToTronSig(kmsSignature, &privateKey.PublicKey, hash)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to find valid recovery ID for TRON signature")
			} else {
				require.NoError(t, err)
				assert.Len(t, tronSig, 65)

				// Verify signature format
				rResult := tronSig[:32]
				sResult := tronSig[32:64]
				recoveryID := tronSig[64]

				assert.Len(t, rResult, 32)
				assert.Len(t, sResult, 32)
				assert.LessOrEqual(t, recoveryID, byte(1))
			}
		})
	}
}

func TestIsValidRecovery(t *testing.T) {
	t.Parallel()

	// Generate a test private key and sign a message
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	hash := crypto.Keccak256([]byte("test message"))

	// Create a valid signature using Ethereum's signing
	signature, err := crypto.Sign(hash, privateKey)
	require.NoError(t, err)
	require.Len(t, signature, 65)

	tests := []struct {
		name     string
		sig      []byte
		hash     []byte
		pubKey   *ecdsa.PublicKey
		expected bool
	}{
		{
			name:     "valid signature",
			sig:      signature,
			hash:     hash,
			pubKey:   &privateKey.PublicKey,
			expected: true,
		},
		{
			name:     "wrong hash",
			sig:      signature,
			hash:     crypto.Keccak256([]byte("different message")),
			pubKey:   &privateKey.PublicKey,
			expected: false,
		},
		{
			name:     "wrong public key",
			sig:      signature,
			hash:     hash,
			pubKey:   func() *ecdsa.PublicKey { k, _ := crypto.GenerateKey(); return &k.PublicKey }(),
			expected: false,
		},
		{
			name:     "invalid signature length",
			sig:      signature[:64], // Too short
			hash:     hash,
			pubKey:   &privateKey.PublicKey,
			expected: false,
		},
		{
			name:     "corrupted signature",
			sig:      func() []byte { s := make([]byte, 65); copy(s, signature); s[0] = 0xFF; return s }(),
			hash:     hash,
			pubKey:   &privateKey.PublicKey,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isValidRecovery(tt.sig, tt.hash, tt.pubKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKMSSigner_ErrorScenarios(t *testing.T) {
	t.Parallel()

	t.Run("invalid asn1 signature", func(t *testing.T) {
		t.Parallel()

		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		hash := crypto.Keccak256([]byte("test message"))
		invalidKMSSignature := []byte("invalid asn1 data")

		_, err = kmsToTronSig(invalidKMSSignature, &privateKey.PublicKey, hash)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal KMS signature")
	})

	t.Run("no valid recovery id found", func(t *testing.T) {
		t.Parallel()

		// This test is tricky because we need to create a signature that doesn't
		// recover to the expected public key with either recovery ID
		// For now, we'll test with zero values which should fail
		zero := big.NewInt(0)

		ecdsaSig := kms.ECDSASig{
			R: asn1.RawValue{Bytes: zero.Bytes()},
			S: asn1.RawValue{Bytes: zero.Bytes()},
		}

		kmsSignature, err := asn1.Marshal(ecdsaSig)
		require.NoError(t, err)

		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		hash := crypto.Keccak256([]byte("test message"))

		_, err = kmsToTronSig(kmsSignature, &privateKey.PublicKey, hash)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find valid recovery ID for TRON signature")
	})
}
