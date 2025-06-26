package provider

import (
	"crypto/ecdsa"
	"encoding/asn1"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms"
)

// Defines a general test EVM address
var (
	testAddr1 = common.HexToAddress("0xc1d6fEcd5D09Ad67cF5E0FC9633D89759DD84271")
)

// Defines standard variables for a test chain.
var (
	testChainID    = chain_selectors.TEST_1000.EvmChainID // Defines a standard test EVM chain ID
	testChainIDBig = new(big.Int).SetUint64(testChainID)  // Defines the testChainID in *big.Int format
)

// Variables used for testing the KMS provider.
var (
	testAWSProfile     = "default"
	testKMSKeyID       = "1234567-1234-1234-1234-123456789012"
	testKMSKeyRegion   = "ap-southeast-1"
	testKMSKeyIDAWSStr = aws.String(testKMSKeyID)
	// testKMSPublicKeyHex is a sample KMS public key in hex format. This is returned as the public key
	// from the KMS service when calling GetPublicKey for the given kmsKeyID.
	testKMSPublicKeyHex = "3056301006072a8648ce3d020106052b8104000a034200043f20652b1dd7e8d448a1c9068247fae8940b70599df714a3947106c2411a7f1442ef26f3bb4ac7c5721177ea4a5c855317a25a4a01ae2d10f623c9f42de5d171"
)

// testKMSPublicKey returns the KMS public key in bytes.
func testKMSPublicKey(t *testing.T) []byte {
	t.Helper()

	b, err := hex.DecodeString(testKMSPublicKeyHex)
	require.NoError(t, err)

	return b
}

// testECDSAPublicKey returns the ECDSA public key from the KMS public key.
func testECDSAPublicKey(t *testing.T) *ecdsa.PublicKey {
	t.Helper()

	var spki kms.SPKI
	_, err := asn1.Unmarshal(testKMSPublicKey(t), &spki)
	require.NoError(t, err)

	pubKey, err := crypto.UnmarshalPubkey(spki.SubjectPublicKey.Bytes)
	require.NoError(t, err)

	return pubKey
}

// testEVMAddr returns the EVM address derived from the KMS public key.
func testEVMAddr(t *testing.T) common.Address {
	t.Helper()

	pubKey := testECDSAPublicKey(t)
	evmAddr := crypto.PubkeyToAddress(*pubKey)

	return evmAddr
}
