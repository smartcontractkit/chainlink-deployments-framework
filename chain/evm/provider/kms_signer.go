package provider

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/aws/aws-sdk-go/aws"
	kmslib "github.com/aws/aws-sdk-go/service/kms"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms"
)

// KMSSigner provides a signer for EVM transactions using a KMS key. It provides methods to
// convert KMS keys to EVM-compatible public keys, signatures, and geth bindings.
type KMSSigner struct {
	// client is the underlying KMS client used to sign transactions.
	client kms.Client
	// kmsKeyID is the ID of the KMS key used for signing. Required to store it on the struct
	// so we can use it later to sign transactions.
	kmsKeyID string
}

// NewKMSSigner creates a new KMSSigner instance using the provided KMS key ID, region, and
// AWS profile. If you prefer to use environment variables to define the AWS profile, you may
// set the awsProfile as an empty string.
func NewKMSSigner(keyID, keyRegion string, awsProfile string) (*KMSSigner, error) {
	client, err := kms.NewClient(kms.ClientConfig{
		KeyID:      keyID,
		KeyRegion:  keyRegion,
		AWSProfile: awsProfile,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize KMS Client: %w", err)
	}

	return &KMSSigner{
		client:   client,
		kmsKeyID: keyID,
	}, nil
}

// GetECDSAPublicKey retrieves the public key from KMS and converts it to its ECDSA representation.
func (s *KMSSigner) GetECDSAPublicKey() (*ecdsa.PublicKey, error) {
	out, err := s.client.GetPublicKey(&kmslib.GetPublicKeyInput{
		KeyId: aws.String(s.kmsKeyID),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get public key from KMS for KeyId=%s: %w", s.kmsKeyID, err)
	}

	// The public key is returned in ASN.1 format, which we need to decode into an SPKI structure.
	var spki kms.SPKI
	if _, err = asn1.Unmarshal(out.PublicKey, &spki); err != nil {
		return nil, fmt.Errorf("cannot parse asn1 public key for KeyId=%s: %w", s.kmsKeyID, err)
	}

	// Unmarshal the KMS public key bytes into an ECDSA public key.
	pubKey, err := crypto.UnmarshalPubkey(spki.SubjectPublicKey.Bytes)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal public key bytes: %w", err)
	}

	return pubKey, nil
}

// GetAddress returns the Ethereum address corresponding to the public key managed by KMS.
func (s *KMSSigner) GetAddress() (common.Address, error) {
	pubKey, err := s.GetECDSAPublicKey()
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get public key: %w", err)
	}

	return crypto.PubkeyToAddress(*pubKey), nil
}

// GetTransactOpts returns a *bind.TransactOpts configured to sign Ethereum transactions using the
// KMS-backed key.
//
// The returned TransactOpts uses the KMS key for signing and sets the correct sender address
// derived from the KMS public key.
func (s *KMSSigner) GetTransactOpts(
	ctx context.Context, chainID *big.Int,
) (*bind.TransactOpts, error) {
	if chainID == nil {
		return nil, errors.New("chainID is required")
	}

	// Construct the key's EVM Address from the public key
	pubKey, err := s.GetECDSAPublicKey()
	if err != nil {
		return nil, err
	}

	return &bind.TransactOpts{
		From:    crypto.PubkeyToAddress(*pubKey),
		Signer:  s.signerFunc(pubKey, chainID),
		Context: ctx,
	}, nil
}

// signerFunc returns a function that signs transactions using KMS. The returned function
// calls the KMS API to sign the transaction hash, converts the KMS signature to an
// Ethereum-compatible format, and applies the signature to the transaction.
func (s *KMSSigner) signerFunc(
	pubKey *ecdsa.PublicKey, chainID *big.Int,
) func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
	// Convert the public key to bytes and derive the EVM address from it.
	pubKeyBytes := secp256k1.S256().Marshal(pubKey.X, pubKey.Y)
	keyAddr := crypto.PubkeyToAddress(*pubKey)

	// Construct the EVM signer
	signer := types.LatestSignerForChainID(chainID)

	return func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
		if address != keyAddr {
			return nil, bind.ErrNotAuthorized
		}

		var (
			txHash = signer.Hash(tx).Bytes()
			mType  = kmslib.MessageTypeDigest
			algo   = kmslib.SigningAlgorithmSpecEcdsaSha256
		)

		// Sign the transaction hash using KMS.
		out, err := s.client.Sign(&kmslib.SignInput{
			KeyId:            &s.kmsKeyID,
			SigningAlgorithm: &algo,
			MessageType:      &mType,
			Message:          txHash,
		})
		if err != nil {
			return nil, fmt.Errorf("call to kms.Sign() failed on transaction: %w", err)
		}

		evmSig, err := kmsToEVMSig(out.Signature, pubKeyBytes, txHash)
		if err != nil {
			return nil, fmt.Errorf("failed to convert KMS signature to Ethereum signature: %w", err)
		}

		return tx.WithSignature(signer, evmSig)
	}
}

var (
	// secp256k1N is the N value of the secp256k1 curve, used to adjust the S value in signatures.
	secp256k1N = crypto.S256().Params().N
	// secp256k1HalfN is half of the secp256k1 N value, used to adjust the S value in signatures.
	secp256k1HalfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

// kmsToEVMSig converts a KMS signature to an Ethereum-compatible signature. This follows this
// example provided by AWS Guides.
//
// [AWS Guides]: https://aws.amazon.com/blogs/database/part2-use-aws-kms-to-securely-manage-ethereum-accounts/
func kmsToEVMSig(kmsSig, ecdsaPubKeyBytes, hash []byte) ([]byte, error) {
	var ecdsaSig kms.ECDSASig
	if _, err := asn1.Unmarshal(kmsSig, &ecdsaSig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KMS signature: %w", err)
	}

	rBytes := ecdsaSig.R.Bytes
	sBytes := ecdsaSig.S.Bytes

	// Adjust S value from signature to match EVM standard.
	//
	// After we extract r and s successfully, we have to test if the value of s is greater than
	// secp256k1n/2 as specified in EIP-2 and flip it if required.
	sBigInt := new(big.Int).SetBytes(sBytes)
	if sBigInt.Cmp(secp256k1HalfN) > 0 {
		sBytes = new(big.Int).Sub(secp256k1N, sBigInt).Bytes()
	}

	return recoverEVMSignature(ecdsaPubKeyBytes, hash, rBytes, sBytes)
}

// recoverEVMSignature attempts to reconstruct the EVM signature by trying both possible recovery
// IDs (v = 0 and v = 1). It compares the recovered public key with the expected public key bytes
// to determine the correct signature.
//
// Returns the valid EVM signature if successful, or an error if neither recovery ID matches.
func recoverEVMSignature(expectedPublicKey, txHash, r, s []byte) ([]byte, error) {
	// Ethereum signatures require r and s to be exactly 32 bytes each.
	rsSig := append(padTo32Bytes(r), padTo32Bytes(s)...)
	// Ethereum signatures have a 65th byte called the recovery ID (v), which can be 0 or 1.
	// Here we append 0 to the signature to start with for the first recovery attempt.
	evmSig := append(rsSig, []byte{0}...)

	recoveredPublicKey, err := crypto.Ecrecover(txHash, evmSig)
	if err != nil {
		return nil, fmt.Errorf("failed to recover signature with v=0: %w", err)
	}

	if hex.EncodeToString(recoveredPublicKey) != hex.EncodeToString(expectedPublicKey) {
		// If the first recovery attempt failed, we try with v=1.
		evmSig = append(rsSig, []byte{1}...)
		recoveredPublicKey, err = crypto.Ecrecover(txHash, evmSig)
		if err != nil {
			return nil, fmt.Errorf("failed to recover signature with v=1: %w", err)
		}

		if hex.EncodeToString(recoveredPublicKey) != hex.EncodeToString(expectedPublicKey) {
			return nil, errors.New("cannot reconstruct public key from sig")
		}
	}

	return evmSig, nil
}

// padTo32Bytes pads the given byte slice to 32 bytes by trimming leading zeros and prepending
// zeros.
func padTo32Bytes(buffer []byte) []byte {
	buffer = bytes.TrimLeft(buffer, "\x00")
	for len(buffer) < 32 {
		zeroBuf := []byte{0}
		buffer = append(zeroBuf, buffer...)
	}

	return buffer
}
