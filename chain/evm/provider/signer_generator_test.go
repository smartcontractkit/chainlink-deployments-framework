package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"testing"

	kmslib "github.com/aws/aws-sdk-go/service/kms"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kmsmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms/mocks"
)

func Test_TransactorFromRaw(t *testing.T) {
	t.Parallel()

	// Setup the private key
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	privKeyBytes := crypto.FromECDSA(privKey)

	// Convert the private and public keys to hex strings
	hexPrivKey := hex.EncodeToString(privKeyBytes)
	privKeyHex := crypto.PubkeyToAddress(privKey.PublicKey).Hex()

	tests := []struct {
		name        string
		givePrivKey string
		giveChainID *big.Int
		giveOpts    []GeneratorOption
		wantAddr    string
		wantGas     uint64
		wantErr     string
	}{
		{
			name:        "valid default account and private key (no gas limit)",
			givePrivKey: hexPrivKey,
			giveChainID: testChainIDBig,
			wantAddr:    privKeyHex,
			wantGas:     0,
		},
		{
			name:        "valid with custom gas limit",
			givePrivKey: hexPrivKey,
			giveChainID: testChainIDBig,
			giveOpts:    []GeneratorOption{WithGasLimit(123456)},
			wantAddr:    privKeyHex,
			wantGas:     123456,
		},
		{
			name:        "invalid private key",
			givePrivKey: "invalid",
			giveChainID: testChainIDBig,
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

			gen := TransactorFromRaw(tt.givePrivKey, tt.giveOpts...)

			got, err := gen.Generate(tt.giveChainID)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantAddr, got.From.Hex())
			assert.Equal(t, tt.wantGas, got.GasLimit)
		})
	}
}

func Test_TransactorRandom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		giveChainID *big.Int
		wantErr     string
	}{
		{
			name:        "valid",
			giveChainID: testChainIDBig,
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

// We only test the initialization here because this constructs an actual KMS client
// and we don't want to make real KMS calls in unit tests.
func Test_TransactorFromKMS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		giveKeyID     string
		giveKeyRegion string
		wantErr       string
	}{
		{
			name:          "valid KMS key ID and region",
			giveKeyID:     testKMSKeyID,
			giveKeyRegion: testKMSKeyRegion,
		},
		{
			name:          "error creating KMS Signer",
			giveKeyID:     "", // empty key ID should trigger an error
			giveKeyRegion: testKMSKeyRegion,
			wantErr:       "failed to create KMS signer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := TransactorFromKMS(tt.giveKeyID, tt.giveKeyRegion, "")

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_TransactorFromKMSSigner(t *testing.T) {
	t.Parallel()

	publicKeyBytes := testKMSPublicKey(t)

	tests := []struct {
		name        string
		beforeFunc  func(*kmsmocks.MockClient)
		giveChainID *big.Int
		wantErr     string
	}{
		{
			name: "produces a transactor",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: publicKeyBytes,
					}, nil)
			},
			giveChainID: testChainIDBig,
		},
		{
			name:        "fails to generate opts",
			giveChainID: nil, // nil chain ID should trigger an error
			wantErr:     "failed to get transact opts from KMS signer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := kmsmocks.NewMockClient(t)

			if tt.beforeFunc != nil {
				tt.beforeFunc(client)
			}

			signer := &KMSSigner{
				client:   client,
				kmsKeyID: testKMSKeyID,
			}

			transactor := TransactorFromKMSSigner(signer)

			got, err := transactor.Generate(tt.giveChainID)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
			}
		})
	}
}

func Test_TransactorFromRaw_SignHash(t *testing.T) {
	t.Parallel()

	// Setup the private key
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	privKeyBytes := crypto.FromECDSA(privKey)
	hexPrivKey := hex.EncodeToString(privKeyBytes)

	// Create a test hash to sign
	testHash := crypto.Keccak256([]byte("test message"))

	tests := []struct {
		name        string
		givePrivKey string
		giveHash    []byte
		wantErr     string
	}{
		{
			name:        "valid hash signing",
			givePrivKey: hexPrivKey,
			giveHash:    testHash,
		},
		{
			name:        "invalid private key",
			givePrivKey: "invalid",
			giveHash:    testHash,
			wantErr:     "failed to convert private key to ECDSA",
		},
		{
			name:        "empty hash",
			givePrivKey: hexPrivKey,
			giveHash:    []byte{},
			wantErr:     "failed to sign hash: hash is required to be exactly 32 bytes (0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := TransactorFromRaw(tt.givePrivKey)

			signature, err := gen.SignHash(tt.giveHash)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, signature)
				require.Len(t, signature, 65) // Standard Ethereum signature length

				// Verify the signature is consistent
				signature2, err := gen.SignHash(tt.giveHash)
				require.NoError(t, err)
				require.Equal(t, signature, signature2, "SignHash should be deterministic for the same hash")

				// If we have a valid private key, verify signature recovery
				if tt.wantErr == "" && tt.givePrivKey == hexPrivKey {
					recoveredPubKey, err := crypto.SigToPub(tt.giveHash, signature)
					require.NoError(t, err)
					expectedAddr := crypto.PubkeyToAddress(privKey.PublicKey)
					recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)
					require.Equal(t, expectedAddr, recoveredAddr, "Recovered address should match the original")
				}
			}
		})
	}
}

func Test_TransactorRandom_SignHash(t *testing.T) {
	t.Parallel()

	// Create a test hash to sign
	testHash := crypto.Keccak256([]byte("test message"))

	t.Run("valid hash signing", func(t *testing.T) {
		t.Parallel()

		gen := TransactorRandom()

		signature1, err := gen.SignHash(testHash)
		require.NoError(t, err)
		require.NotEmpty(t, signature1)
		require.Len(t, signature1, 65) // Standard Ethereum signature length

		// Since transactorRandom now stores the private key, subsequent calls should use the same key
		signature2, err := gen.SignHash(testHash)
		require.NoError(t, err)
		require.Equal(t, signature1, signature2, "SignHash should be consistent for the same generator instance")

		// Test with different hash should produce different signature
		differentHash := crypto.Keccak256([]byte("different message"))
		signature3, err := gen.SignHash(differentHash)
		require.NoError(t, err)
		require.NotEqual(t, signature1, signature3, "Different hashes should produce different signatures")
	})

	t.Run("empty hash", func(t *testing.T) {
		t.Parallel()

		gen := TransactorRandom()
		_, err := gen.SignHash([]byte{})
		require.ErrorContains(t, err, "failed to sign hash: hash is required to be exactly 32 bytes (0)")
	})

	t.Run("consistency between Generate and SignHash", func(t *testing.T) {
		t.Parallel()

		gen := TransactorRandom()

		// First call Generate to create the transactor
		transactor, err := gen.Generate(testChainIDBig)
		require.NoError(t, err)

		// Then call SignHash
		signature, err := gen.SignHash(testHash)
		require.NoError(t, err)

		// Verify that the signature was created by the same key as the transactor
		recoveredPubKey, err := crypto.SigToPub(testHash, signature)
		require.NoError(t, err)
		recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)
		require.Equal(t, transactor.From, recoveredAddr, "SignHash and Generate should use the same private key")
	})
}

func Test_TransactorFromKMSSigner_SignHash(t *testing.T) {
	t.Parallel()

	// Use the same hash and signature combination from the working KMS tests
	h := sha256.New()
	h.Write([]byte("signme"))
	testHash := h.Sum(nil)

	publicKeyBytes := testKMSPublicKey(t)

	// Valid KMS signature in ASN.1 format that matches the test hash and public key
	testKMSSignature, err := hex.DecodeString(testKMSSignatureHex)
	require.NoError(t, err)

	var (
		kmsDigestType       = kmslib.MessageTypeDigest
		kmsSigningAlgorithm = kmslib.SigningAlgorithmSpecEcdsaSha256
	)

	tests := []struct {
		name       string
		beforeFunc func(*kmsmocks.MockClient)
		giveHash   []byte
		wantErr    string
	}{
		{
			name: "successful hash signing",
			beforeFunc: func(c *kmsmocks.MockClient) {
				// Mock GetPublicKey call which is needed internally by SignHash
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: publicKeyBytes,
					}, nil)

				c.EXPECT().
					Sign(&kmslib.SignInput{
						KeyId:            testKMSKeyIDAWSStr,
						Message:          testHash,
						MessageType:      &kmsDigestType,
						SigningAlgorithm: &kmsSigningAlgorithm,
					}).
					Return(&kmslib.SignOutput{
						Signature: testKMSSignature,
					}, nil)
			},
			giveHash: testHash,
		},
		{
			name: "KMS signing error",
			beforeFunc: func(c *kmsmocks.MockClient) {
				// Mock GetPublicKey call which is needed internally by SignHash
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: publicKeyBytes,
					}, nil)

				c.EXPECT().
					Sign(&kmslib.SignInput{
						KeyId:            testKMSKeyIDAWSStr,
						Message:          testHash,
						MessageType:      &kmsDigestType,
						SigningAlgorithm: &kmsSigningAlgorithm,
					}).
					Return(nil, assert.AnError)
			},
			giveHash: testHash,
			wantErr:  "call to kms.Sign() failed",
		},
		{
			name: "GetPublicKey error",
			beforeFunc: func(c *kmsmocks.MockClient) {
				// Mock GetPublicKey call to return an error
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(nil, assert.AnError)
			},
			giveHash: testHash,
			wantErr:  "cannot get public key from KMS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := kmsmocks.NewMockClient(t)

			if tt.beforeFunc != nil {
				tt.beforeFunc(client)
			}

			signer := &KMSSigner{
				client:   client,
				kmsKeyID: testKMSKeyID,
			}

			gen := TransactorFromKMSSigner(signer)

			signature, err := gen.SignHash(tt.giveHash)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, signature)
			}
		})
	}
}
