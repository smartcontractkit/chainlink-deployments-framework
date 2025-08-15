package provider

import (
	"crypto/ecdsa"
	"encoding/asn1"
	"math/big"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	kmslib "github.com/aws/aws-sdk-go/service/kms"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms"
	kmsmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms/mocks"
)

func TestNewKMSSigner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		keyID      string
		keyRegion  string
		awsProfile string
		wantErr    bool
	}{
		{
			name:       "invalid parameters - will fail during initialization",
			keyID:      "test-key-id",
			keyRegion:  "us-east-1",
			awsProfile: "test-profile",
			wantErr:    true,
		},
		{
			name:       "empty aws profile - will fail during initialization",
			keyID:      "test-key-id",
			keyRegion:  "us-west-2",
			awsProfile: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			signer, err := newKMSSigner(tt.keyID, tt.keyRegion, tt.awsProfile)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, signer)
			} else {
				require.NoError(t, err)
				require.NotNil(t, signer)

				// Verify initialization fields are properly set
				assert.NotNil(t, signer.client)
				assert.Equal(t, tt.keyID, signer.kmsKeyID)
				assert.NotNil(t, signer.ecdsaPublicKey)
				assert.NotEmpty(t, signer.address)
			}
		})
	}
}

func TestNewKMSSignerWithClient_Constructor(t *testing.T) {
	t.Parallel()

	// Generate a test private key and derive public key
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Marshal the public key to DER format (as KMS would return it)
	pubKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	tests := []struct {
		name        string
		keyID       string
		setupMock   func(*kmsmocks.MockClient)
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful initialization with mock client",
			keyID: "test-key-id",
			setupMock: func(mockClient *kmsmocks.MockClient) {
				mockClient.EXPECT().GetPublicKey(&kmslib.GetPublicKeyInput{
					KeyId: aws.String("test-key-id"),
				}).Return(&kmslib.GetPublicKeyOutput{
					PublicKey: pubKeyBytes,
				}, nil)
			},
			wantErr: false,
		},
		{
			name:  "KMS GetPublicKey fails",
			keyID: "test-key-id",
			setupMock: func(mockClient *kmsmocks.MockClient) {
				mockClient.EXPECT().GetPublicKey(&kmslib.GetPublicKeyInput{
					KeyId: aws.String("test-key-id"),
				}).Return(nil, assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to get public key from KMS",
		},
		{
			name:  "invalid public key format",
			keyID: "test-key-id",
			setupMock: func(mockClient *kmsmocks.MockClient) {
				mockClient.EXPECT().GetPublicKey(&kmslib.GetPublicKeyInput{
					KeyId: aws.String("test-key-id"),
				}).Return(&kmslib.GetPublicKeyOutput{
					PublicKey: []byte("invalid-public-key"),
				}, nil)
			},
			wantErr:     true,
			errContains: "failed to parse ECDSA public key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := kmsmocks.NewMockClient(t)
			tt.setupMock(mockClient)

			signer, err := newKMSSignerWithClient(tt.keyID, mockClient)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, signer)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, signer)
				assert.Equal(t, mockClient, signer.client)
				assert.Equal(t, tt.keyID, signer.kmsKeyID)
				assert.NotNil(t, signer.ecdsaPublicKey)
				assert.NotEmpty(t, signer.address)

				// Verify the address is properly derived
				expectedAddress := address.PubkeyToAddress(privateKey.PublicKey)
				assert.Equal(t, expectedAddress, signer.address)
			}
		})
	}
}

func TestKMSSignerWithClient_Sign(t *testing.T) {
	t.Parallel()

	// Generate a test private key and derive public key
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Marshal the public key to DER format (as KMS would return it)
	pubKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)

	// Create test hash to sign
	testHash := crypto.Keccak256([]byte("test message"))

	// Create a real signature to use as mock KMS response
	realSig, err := crypto.Sign(testHash, privateKey)
	require.NoError(t, err)

	// Extract r and s from the real signature to create mock KMS signature
	r := new(big.Int).SetBytes(realSig[:32])
	s := new(big.Int).SetBytes(realSig[32:64])

	ecdsaSig := kms.ECDSASig{
		R: asn1.RawValue{Bytes: r.Bytes()},
		S: asn1.RawValue{Bytes: s.Bytes()},
	}

	mockKMSSignature, err := asn1.Marshal(ecdsaSig)
	require.NoError(t, err)

	tests := []struct {
		name        string
		keyID       string
		setupMock   func(*kmsmocks.MockClient)
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful signing",
			keyID: "test-key-id",
			setupMock: func(mockClient *kmsmocks.MockClient) {
				// Mock GetPublicKey for initialization
				mockClient.EXPECT().GetPublicKey(&kmslib.GetPublicKeyInput{
					KeyId: aws.String("test-key-id"),
				}).Return(&kmslib.GetPublicKeyOutput{
					PublicKey: pubKeyBytes,
				}, nil)

				// Mock Sign operation
				mockClient.EXPECT().Sign(&kmslib.SignInput{
					KeyId:            aws.String("test-key-id"),
					Message:          testHash,
					MessageType:      aws.String(kmslib.MessageTypeDigest),
					SigningAlgorithm: aws.String(kmslib.SigningAlgorithmSpecEcdsaSha256),
				}).Return(&kmslib.SignOutput{
					Signature: mockKMSSignature,
				}, nil)
			},
			wantErr: false,
		},
		{
			name:  "KMS Sign fails",
			keyID: "test-key-id",
			setupMock: func(mockClient *kmsmocks.MockClient) {
				// Mock GetPublicKey for initialization
				mockClient.EXPECT().GetPublicKey(&kmslib.GetPublicKeyInput{
					KeyId: aws.String("test-key-id"),
				}).Return(&kmslib.GetPublicKeyOutput{
					PublicKey: pubKeyBytes,
				}, nil)

				// Mock Sign operation failure
				mockClient.EXPECT().Sign(&kmslib.SignInput{
					KeyId:            aws.String("test-key-id"),
					Message:          testHash,
					MessageType:      aws.String(kmslib.MessageTypeDigest),
					SigningAlgorithm: aws.String(kmslib.SigningAlgorithmSpecEcdsaSha256),
				}).Return(nil, assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to sign transaction hash with KMS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := kmsmocks.NewMockClient(t)
			tt.setupMock(mockClient)

			signer, err := newKMSSignerWithClient(tt.keyID, mockClient)
			require.NoError(t, err)
			require.NotNil(t, signer)

			signature, err := signer.Sign(testHash)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, signature)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, signature)
				assert.Len(t, signature, 65) // 32 bytes r + 32 bytes s + 1 byte recovery ID

				// Verify signature format
				recoveryID := signature[64]
				assert.LessOrEqual(t, recoveryID, byte(1))

				// Verify the signature can recover to the correct public key
				assert.True(t, isValidRecovery(signature, testHash, &privateKey.PublicKey))
			}
		})
	}
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
