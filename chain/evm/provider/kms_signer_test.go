package provider

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"testing"

	kmslib "github.com/aws/aws-sdk-go/service/kms"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	kmsmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms/mocks"
)

func Test_NewKMSigner(t *testing.T) {
	t.Parallel()

	signer, err := NewKMSSigner(testKMSKeyID, testKMSKeyRegion, testAWSProfile)
	require.NoError(t, err)
	require.NotNil(t, signer)
	require.Equal(t, testKMSKeyID, signer.kmsKeyID)
	require.NotNil(t, signer.client)
}

func Test_KMSSigner_GetECDSAPublicKey(t *testing.T) {
	t.Parallel()

	var (
		publicKeyBytes = testKMSPublicKey(t)
		ecdsaPublicKey = testECDSAPublicKey(t)
	)

	tests := []struct {
		name          string
		beforeFunc    func(c *kmsmocks.MockClient)
		giveCachedKey *ecdsa.PublicKey
		want          *ecdsa.PublicKey
		wantErr       string
	}{
		{
			name: "valid public key",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: publicKeyBytes,
					}, nil)
			},
			want: ecdsaPublicKey,
		},
		{
			name:          "cached public key",
			giveCachedKey: ecdsaPublicKey,
			want:          ecdsaPublicKey,
		},
		{
			name: "could not get KMS public key",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(mock.IsType(&kmslib.GetPublicKeyInput{})).
					Return(nil, assert.AnError)
			},
			wantErr: "cannot get public key from KMS",
		},
		{
			name: "could not unmarshal KMS public key",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: []byte("invalid"),
					}, nil)
			},
			wantErr: "cannot parse asn1 public key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				client = kmsmocks.NewMockClient(t)
				signer = &KMSSigner{
					client:   client,
					kmsKeyID: testKMSKeyID,
				}
			)

			if tt.giveCachedKey != nil {
				signer.ecdsaPublicKey = tt.giveCachedKey
			}

			if tt.beforeFunc != nil {
				tt.beforeFunc(client)
			}

			pubKey, err := signer.GetECDSAPublicKey()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, pubKey)
			}
		})
	}
}

func Test_KMSSigner_GetAddress(t *testing.T) {
	t.Parallel()

	var (
		kmsPublicKey = testKMSPublicKey(t)
		publicKey    = testECDSAPublicKey(t)
	)

	tests := []struct {
		name       string
		beforeFunc func(c *kmsmocks.MockClient)
		wantErr    string
	}{
		{
			name: "gets the address successfully",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: kmsPublicKey,
					}, nil)
			},
		},
		{
			name: "error fetching KMS public key",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(mock.IsType(&kmslib.GetPublicKeyInput{})).
					Return(nil, assert.AnError)
			},
			wantErr: "cannot get public key from KMS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				client = kmsmocks.NewMockClient(t)
				signer = &KMSSigner{
					client:   client,
					kmsKeyID: testKMSKeyID,
				}
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(client)
			}

			addr, err := signer.GetAddress()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, crypto.PubkeyToAddress(*publicKey), addr)
			}
		})
	}
}

func Test_KMSSigner_GetTransactOpts(t *testing.T) {
	t.Parallel()

	publicKeyBytes := testKMSPublicKey(t)

	tests := []struct {
		name        string
		beforeFunc  func(c *kmsmocks.MockClient)
		giveChainID *big.Int
		wantErr     string
	}{
		{
			name: "gets the transact opts successfully",
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
			name:        "chainID is nil",
			giveChainID: nil,
			wantErr:     "chainID is required",
		},
		{
			name: "error fetching KMS public key",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(mock.IsType(&kmslib.GetPublicKeyInput{})).
					Return(nil, assert.AnError)
			},
			giveChainID: testChainIDBig,
			wantErr:     "cannot get public key from KMS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				client = kmsmocks.NewMockClient(t)
				signer = &KMSSigner{
					client:   client,
					kmsKeyID: testKMSKeyID,
				}
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(client)
			}

			topts, err := signer.GetTransactOpts(t.Context(), tt.giveChainID)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, topts)
			}
		})
	}
}

func Test_KMSSigner_SignHash(t *testing.T) {
	t.Parallel()

	var (
		kmsPublicKey = testKMSPublicKey(t)
	)

	// Simple hash to sign
	h := sha256.New()
	h.Write([]byte("signme"))
	txHash := h.Sum(nil)

	// A valid KMS signature in ASN.1 format, which we will use as a mock return value.
	sigBytes, err := hex.DecodeString(testKMSSignatureHex)
	require.NoError(t, err)

	// The expected EVM signature bytes that we will compare against based on the sigBytes.
	wantEVMSigBytes, err := hex.DecodeString("6475584f9afacee823cd3479364ec7a2fc2a804739f87e48604d3fd2a74a5dbb47fd03ccec57903bbd5dc0c9df19b63573754b4003235d10356ef30f2247028e01")
	require.NoError(t, err)

	tests := []struct {
		name       string
		beforeFunc func(c *kmsmocks.MockClient)
		giveTxHash []byte
		want       []byte
		wantErr    string
	}{
		{
			name: "signs the hash successfully",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: kmsPublicKey,
					}, nil)

				c.EXPECT().
					Sign(mock.IsType(&kmslib.SignInput{})).
					Return(&kmslib.SignOutput{
						Signature: sigBytes,
					}, nil)
			},
			giveTxHash: txHash,
			want:       wantEVMSigBytes,
		},
		{
			name: "failed to get KMS public key",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(nil, assert.AnError)
			},
			giveTxHash: txHash,
			wantErr:    "cannot get public key from KMS",
		},
		{
			name: "failed to sign hash with KMS",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: kmsPublicKey,
					}, nil)

				c.EXPECT().
					Sign(mock.IsType(&kmslib.SignInput{})).
					Return(nil, assert.AnError)
			},
			giveTxHash: txHash,
			wantErr:    "call to kms.Sign() failed",
		},
		{
			name: "failed to convert KMS signature to Ethereum signature",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(&kmslib.GetPublicKeyInput{
						KeyId: testKMSKeyIDAWSStr,
					}).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: kmsPublicKey,
					}, nil)

				c.EXPECT().
					Sign(mock.IsType(&kmslib.SignInput{})).
					Return(&kmslib.SignOutput{
						Signature: []byte("invalid"),
					}, nil)
			},
			giveTxHash: txHash,
			wantErr:    "failed to convert KMS signature to Ethereum signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				client = kmsmocks.NewMockClient(t)
				signer = &KMSSigner{
					client:   client,
					kmsKeyID: testKMSKeyID,
				}
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(client)
			}

			got, err := signer.SignHash(tt.giveTxHash)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

// Tests the SignerFunc that is inserted into the TransactOpts.
func Test_KMSSigner_signerFunc(t *testing.T) {
	t.Parallel()

	var (
		// tx is a sample transaction that we will use for testing. Changing this will impact the
		// expected signature values.
		tx = types.NewTransaction(
			1,                               // nonce
			common.HexToAddress("0xabc123"), // to address
			big.NewInt(1000000000000000000), // value: 1 ETH
			21000,                           // gas limit
			big.NewInt(20000000000),         // gas price: 20 Gwei
			[]byte{},                        // data
		)
		// kmsSigHex is a KMS signature of the tx. This has been generated using the KMS service to
		// use as a test value. If you change the tx, you will need to generate a new test signature.
		kmsSigHex = "3045022100997f61f1392a50a7e286775d8746c44227ce27c3842f08135a24e8741165b8c102203267d241f37d71ea4d450d436cfe257742d28259ec92ff8af16d547f50f3cf1d"
	)

	kmsSigBytes, err := hex.DecodeString(kmsSigHex)
	require.NoError(t, err)

	// These define the expected signature values that we will compare against.
	sigR, ok := new(big.Int).SetString("69428931383229605730800792997587575348728857812217229163580159879167166757057", 10)
	require.True(t, ok, "failed to parse sigR from string")
	sigS, ok := new(big.Int).SetString("22799078821607344335411375362755851548601990949454473972957949155543511977757", 10)
	require.True(t, ok, "failed to parse sigS from string")
	sigV := big.NewInt(2035)

	tests := []struct {
		name        string
		beforeFunc  func(c *kmsmocks.MockClient)
		giveKeyAddr common.Address
		giveTx      *types.Transaction
		wantSigR    *big.Int
		wantSigS    *big.Int
		wantSigV    *big.Int
		wantErr     string
	}{
		{
			name: "signs the transaction",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(mock.IsType(&kmslib.GetPublicKeyInput{})).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: testKMSPublicKey(t),
					}, nil)

				c.EXPECT().
					Sign(mock.IsType(&kmslib.SignInput{})).
					Return(&kmslib.SignOutput{
						Signature: kmsSigBytes,
					}, nil)
			},
			giveKeyAddr: testEVMAddr(t),
			giveTx:      tx,
			wantSigR:    sigR,
			wantSigS:    sigS,
			wantSigV:    sigV,
		},
		{
			name: "not authorized to sign",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(mock.IsType(&kmslib.GetPublicKeyInput{})).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: testKMSPublicKey(t),
					}, nil)
			},
			giveKeyAddr: common.HexToAddress("1234"), // Invalid address
			giveTx:      tx,
			wantErr:     bind.ErrNotAuthorized.Error(),
		},
		{
			name: "error signing transaction with KMS",
			beforeFunc: func(c *kmsmocks.MockClient) {
				c.EXPECT().
					GetPublicKey(mock.IsType(&kmslib.GetPublicKeyInput{})).
					Return(&kmslib.GetPublicKeyOutput{
						PublicKey: testKMSPublicKey(t),
					}, nil)

				c.EXPECT().
					Sign(mock.IsType(&kmslib.SignInput{})).
					Return(nil, assert.AnError)
			},
			giveKeyAddr: testEVMAddr(t),
			giveTx:      tx,
			wantErr:     "call to kms.Sign() failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				client = kmsmocks.NewMockClient(t)
				signer = &KMSSigner{
					client:   client,
					kmsKeyID: testKMSKeyID,
				}
				signerFunc = signer.signerFunc(testChainIDBig)
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(client)
			}

			got, err := signerFunc(tt.giveKeyAddr, tt.giveTx)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				got.RawSignatureValues()
			}
		})
	}
}

func Test_kmsToEthSign(t *testing.T) {
	t.Parallel()

	// Example values for testing
	kmsSigBytes, err := hex.DecodeString("304402206168865941bafcae3a8cf8b26edbb5693d62222b2e54d962c1aabbeaddf33b6802205edc7f597d2bf2d1eaa14fc514a6202bafcffe52b13ae3fec00674d92a874b73")
	require.NoError(t, err)
	ecdsaPublicKeyBytes, err := hex.DecodeString("04a735e9e3cb526f83be23b03f1f5ae7788a8654e3f0fcfb4f978290de07ebd47da30eeb72e904fdd4a81b46e320908ff4345e119148f89c1f04674c14a506e24b")
	require.NoError(t, err)
	txHashBytes, err := hex.DecodeString("a2f037301e90f58c084fe4bec2eef14b26e620d6b6cb46051037d03b29ab7d9a")
	require.NoError(t, err)
	wantEthSignBytes, err := hex.DecodeString("6168865941bafcae3a8cf8b26edbb5693d62222b2e54d962c1aabbeaddf33b685edc7f597d2bf2d1eaa14fc514a6202bafcffe52b13ae3fec00674d92a874b7300")
	require.NoError(t, err)

	tests := []struct {
		name                    string
		giveKMSSigBytes         []byte
		giveECDSAPublicKeyBytes []byte
		giveTxHashBytes         []byte
		want                    []byte
		wantErr                 string
	}{
		{
			name:                    "valid kms to eth sign conversion",
			giveKMSSigBytes:         kmsSigBytes,
			giveECDSAPublicKeyBytes: ecdsaPublicKeyBytes,
			giveTxHashBytes:         txHashBytes,
			want:                    wantEthSignBytes,
		},
		{
			name:                    "invalid kms signature bytes",
			giveKMSSigBytes:         []byte("invalid"),
			giveECDSAPublicKeyBytes: ecdsaPublicKeyBytes,
			giveTxHashBytes:         txHashBytes,
			wantErr:                 "failed to unmarshal KMS signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := kmsToEVMSig(
				tt.giveKMSSigBytes,
				tt.giveECDSAPublicKeyBytes,
				tt.giveTxHashBytes,
			)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_recoverEVMSignature(t *testing.T) {
	t.Parallel()

	txHashHex := "c40be743bc5558b716d388f90b753b259a1611734cd687440e60c53f596c38fc"
	txHash, err := hex.DecodeString(txHashHex)
	require.NoError(t, err)

	pubKey := testECDSAPublicKey(t)
	pubKeyBytes := secp256k1.S256().Marshal(pubKey.X, pubKey.Y)

	sigR, ok := new(big.Int).SetString("105277379219808013728922799499984494012712920083270162259481635005819358924647", 10)
	require.True(t, ok, "failed to parse sigR from string")
	sigS, ok := new(big.Int).SetString("32605629271737803816635902981832602576142754670751265099010581033821583724437", 10)
	require.True(t, ok, "failed to parse sigS from string")

	tests := []struct {
		name                       string
		giveExpectedPublicKeyBytes []byte
		giveTxHash                 []byte
		giveR                      []byte
		giveS                      []byte
		wantErr                    string
	}{
		{
			name:                       "successfully recovers the signature",
			giveExpectedPublicKeyBytes: pubKeyBytes,
			giveTxHash:                 txHash,
			giveR:                      sigR.Bytes(),
			giveS:                      sigS.Bytes(),
		},
		{
			name:                       "fails to recover with v=0",
			giveExpectedPublicKeyBytes: []byte{0x1, 0x2, 0x3}, // bad public key
			giveTxHash:                 []byte{0x4, 0x5, 0x6}, // bad tx hash
			wantErr:                    "failed to recover signature with v=0",
		},
		{
			name:                       "fails to recover with v=0 or v=1",
			giveExpectedPublicKeyBytes: []byte{}, // empty public key so it will not match
			giveTxHash:                 txHash,
			giveR:                      sigR.Bytes(),
			giveS:                      sigS.Bytes(),
			wantErr:                    "cannot reconstruct public key from sig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := recoverEVMSignature(tt.giveExpectedPublicKeyBytes, tt.giveTxHash, tt.giveR, tt.giveS)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_pad32Bytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give []byte
		want []byte
	}{
		{
			name: "already 32 bytes",
			give: make([]byte, 32),
			want: make([]byte, 32),
		},
		{
			name: "less than 32 bytes, leading zeros",
			give: []byte{0, 0, 1, 2, 3},
			want: append(make([]byte, 29), 1, 2, 3),
		},
		{
			name: "less than 32 bytes, no leading zeros",
			give: []byte{1, 2, 3},
			want: append(make([]byte, 29), 1, 2, 3),
		},
		{
			name: "more than 32 bytes, leading zeros",
			give: append([]byte{0, 0}, make([]byte, 32)...),
			want: make([]byte, 32),
		},
		{
			name: "empty input",
			give: []byte{},
			want: make([]byte, 32),
		},
		{
			name: "exactly 32 bytes, non-zero",
			give: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			want: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := padTo32Bytes(tt.give)
			require.Len(t, got, 32)
			require.Equal(t, tt.want, got)
		})
	}
}
