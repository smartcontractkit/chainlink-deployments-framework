package provider

import (
	"encoding/hex"
	"math/big"
	"testing"

	kmslib "github.com/aws/aws-sdk-go/service/kms"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kmsmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms/mocks"
)

func Test_TransactorGenerator(t *testing.T) {
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
		want        string
		wantErr     string
	}{
		{
			name:        "valid default account and private key",
			givePrivKey: hexPrivKey,
			giveChainID: testChainIDBig,
			want:        privKeyHex,
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

			gen := TransactorFromRaw(tt.givePrivKey)

			got, err := gen.Generate(tt.giveChainID)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got.From.Hex())
			}
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
