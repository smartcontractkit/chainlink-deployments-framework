package provider

import (
	"encoding/hex"
	"math/big"
	"testing"

	kmslib "github.com/aws/aws-sdk-go/service/kms"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	zkAccounts "github.com/zksync-sdk/zksync2-go/accounts"

	kmsmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms/mocks"
)

func Test_ZkSyncSignerFromRaw(t *testing.T) {
	t.Parallel()

	// Setup the private key
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	privKeyBytes := crypto.FromECDSA(privKey)

	// Convert the private and public keys to hex strings
	hexPrivKey := hex.EncodeToString(privKeyBytes)

	signer, err := zkAccounts.NewECDSASignerFromRawPrivateKey(privKeyBytes, testChainIDBig)
	require.NoError(t, err)

	tests := []struct {
		name        string
		givePrivKey string
		giveChainID *big.Int
		wantAddr    string
		wantChainID *big.Int
		wantErr     string
	}{
		{
			name:        "valid default account and private key",
			givePrivKey: hexPrivKey,
			giveChainID: testChainIDBig,
			wantAddr:    signer.Address().Hex(),
			wantChainID: testChainIDBig,
		},
		{
			name:        "invalid private key",
			givePrivKey: "invalid",
			giveChainID: testChainIDBig,
			wantErr:     "invalid raw private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := ZkSyncSignerFromRaw(tt.givePrivKey)

			got, err := gen.Generate(tt.giveChainID)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantAddr, got.Address().Hex())
				assert.Equal(t, tt.wantChainID.Uint64(), got.ChainID().Uint64())
			}
		})
	}
}

func Test_ZkSyncSignerFromKMS(t *testing.T) {
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

			got, err := ZkSyncSignerFromKMS(tt.giveKeyID, tt.giveKeyRegion, "")

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

func Test_ZkSyncSignerRandom(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := ZKSyncSignerRandom()

			got, err := gen.Generate(tt.giveChainID)
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

func Test_zkSyncSignerFromKMS_Generate(t *testing.T) {
	t.Parallel()

	publicKeyBytes := testKMSPublicKey(t)

	tests := []struct {
		name        string
		beforeFunc  func(*kmsmocks.MockClient)
		giveChainID *big.Int
		wantAddr    string
		wantErr     string
	}{
		{
			name: "produces a zksync signer",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: publicKeyBytes,
					}, nil)
			},
			wantAddr:    testEVMAddr(t).Hex(),
			giveChainID: testChainIDBig,
		},
		{
			name: "fails to get address from KMS signer",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(nil, assert.AnError)
			},
			giveChainID: testChainIDBig,
			wantErr:     "failed to get address from KMS signer",
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

			gen := zkSyncSignerFromKMS{
				signer: signer,
			}

			got, err := gen.Generate(tt.giveChainID)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.wantAddr, got.Address().Hex())
			}
		})
	}
}
