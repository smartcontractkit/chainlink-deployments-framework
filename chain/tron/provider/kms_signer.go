package provider

import (
	"crypto/ecdsa"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"

	"github.com/aws/aws-sdk-go/aws"
	kmslib "github.com/aws/aws-sdk-go/service/kms"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms"
)

// kmsSigner handles TRON transaction signing using AWS KMS.
type kmsSigner struct {
	client         kms.Client
	kmsKeyID       string
	ecdsaPublicKey *ecdsa.PublicKey
	address        address.Address
}

// newKMSSigner creates a new KMS signer with the provided configuration.
// It initializes the KMS client and retrieves the address for the configured KMS key.
func newKMSSigner(keyID, keyRegion, awsProfile string) (*kmsSigner, error) {
	client, err := kms.NewClient(kms.ClientConfig{
		KeyID:      keyID,
		KeyRegion:  keyRegion,
		AWSProfile: awsProfile,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client: %w", err)
	}

	return newKMSSignerWithClient(keyID, client)
}

// newKMSSignerWithClient creates a new KMS signer with the provided KMS client.
// This constructor allows for dependency injection of the KMS client, which is useful for testing.
func newKMSSignerWithClient(keyID string, client kms.Client) (*kmsSigner, error) {
	signer := &kmsSigner{
		client:   client,
		kmsKeyID: keyID,
	}

	if err := signer.initializeWithClient(); err != nil {
		return nil, err
	}

	return signer, nil
}

// initializeWithClient initializes the KMS signer using the provided client.
// It retrieves the public key from KMS and derives the TRON address.
func (s *kmsSigner) initializeWithClient() error {
	// Get the public key from KMS to derive the TRON address
	pubKeyOutput, err := s.client.GetPublicKey(&kmslib.GetPublicKeyInput{
		KeyId: aws.String(s.kmsKeyID),
	})
	if err != nil {
		return fmt.Errorf("failed to get public key from KMS: %w", err)
	}

	// Parse the DER-encoded public key
	ecdsaPublicKey, err := crypto.UnmarshalPubkey(pubKeyOutput.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to parse ECDSA public key: %w", err)
	}

	s.ecdsaPublicKey = ecdsaPublicKey

	// Generate TRON address from the public key
	s.address = address.PubkeyToAddress(*ecdsaPublicKey)

	return nil
}

// Sign signs the given transaction hash using the KMS key.
func (s *kmsSigner) Sign(txHash []byte) ([]byte, error) {
	var (
		mType = kmslib.MessageTypeDigest
		algo  = kmslib.SigningAlgorithmSpecEcdsaSha256
	)

	// Sign the transaction hash using KMS.
	out, err := s.client.Sign(&kmslib.SignInput{
		KeyId:            &s.kmsKeyID,
		Message:          txHash,
		MessageType:      &mType,
		SigningAlgorithm: &algo,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction hash with KMS: %w", err)
	}

	// Convert KMS signature to TRON-compatible format
	tronSig, err := kmsToTronSig(out.Signature, s.ecdsaPublicKey, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to convert KMS signature to TRON format: %w", err)
	}

	return tronSig, nil
}

// GetAddress returns the TRON address associated with this KMS signer.
func (s *kmsSigner) GetAddress() (address.Address, error) {
	return s.address, nil
}

// kmsToTronSig converts a KMS signature to a TRON-compatible signature.
// This is similar to the EVM conversion but adapted for TRON's signature format.
func kmsToTronSig(kmsSig []byte, pubKey *ecdsa.PublicKey, hash []byte) ([]byte, error) {
	var ecdsaSig kms.ECDSASig
	if _, err := asn1.Unmarshal(kmsSig, &ecdsaSig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KMS signature: %w", err)
	}

	// Ensure R and S are 32 bytes each
	rBytes := make([]byte, 32)
	sBytes := make([]byte, 32)

	rBuf := ecdsaSig.R.Bytes
	sBuf := ecdsaSig.S.Bytes

	copy(rBytes[32-len(rBuf):], rBuf)
	copy(sBytes[32-len(sBuf):], sBuf)

	// Try both recovery IDs (0 and 1) to find the correct one
	for recoveryID := range 2 {
		// Create signature with recovery ID
		sig := make([]byte, 0, 65)
		sig = append(sig, rBytes...)
		sig = append(sig, sBytes...)
		sig = append(sig, byte(recoveryID))

		// Verify this signature recovers to the correct public key
		if isValidRecovery(sig, hash, pubKey) {
			return sig, nil
		}
	}

	return nil, errors.New("failed to find valid recovery ID for TRON signature")
}

// isValidRecovery checks if the signature with recovery ID recovers to the expected public key.
func isValidRecovery(sig []byte, hash []byte, expectedPubKey *ecdsa.PublicKey) bool {
	if len(sig) != 65 {
		return false
	}

	// Extract r, s
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])

	// Recover public key
	recoveredPub, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return false
	}

	// Check if the recovered public key matches the expected one
	return recoveredPub.X.Cmp(expectedPubKey.X) == 0 && recoveredPub.Y.Cmp(expectedPubKey.Y) == 0 &&
		r.Sign() > 0 && s.Sign() > 0
}
