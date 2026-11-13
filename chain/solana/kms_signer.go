package solana

import (
	"context"
	"errors"
	"fmt"
	"time"

	sollib "github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana/provider/rpcclient"
)

// Signer signs Solana transaction messages on behalf of a single account.
type Signer = rpcclient.Signer

// NewPrivateKeySigner returns a Signer backed by a local private key.
func NewPrivateKeySigner(key sollib.PrivateKey) Signer {
	return rpcclient.NewPrivateKeySigner(key)
}

// kmsSignTimeout bounds a single kms.Sign call when the caller's context has no deadline.
const kmsSignTimeout = 30 * time.Second

// KMSSigner signs Solana transactions with an Ed25519 key held in AWS KMS. The private key never
// leaves KMS.
type KMSSigner struct {
	signer *kms.Ed25519KMSSigner
	pubKey sollib.PublicKey
}

var _ Signer = (*KMSSigner)(nil)

// NewKMSSigner builds a Signer backed by an AWS KMS Ed25519 key. The ctx is only used to fetch
// the public key and is not retained.
func NewKMSSigner(ctx context.Context, keyID, keyRegion string) (*KMSSigner, error) {
	// Check the key ID before building the client, so that a missing key ID is not masked by an
	// unrelated failure to load the AWS config.
	if keyID == "" {
		return nil, errors.New("KMS key ID is required")
	}

	client, err := kms.NewEd25519Client(ctx, keyRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS Ed25519 client: %w", err)
	}

	return newKMSSigner(ctx, keyID, client)
}

// newKMSSigner builds a KMSSigner from an existing KMS Ed25519 client. This constructor accepts
// the client directly so tests can inject a mock.
func newKMSSigner(ctx context.Context, keyID string, client kms.Ed25519Client) (*KMSSigner, error) {
	signer, err := kms.NewEd25519KMSSigner(ctx, keyID, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS Ed25519 signer: %w", err)
	}

	pubKey, err := signer.GetPublicKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch KMS Ed25519 public key: %w", err)
	}

	return &KMSSigner{
		signer: signer,
		pubKey: sollib.PublicKeyFromBytes(pubKey),
	}, nil
}

// PublicKey returns the Solana address of the KMS key.
func (s *KMSSigner) PublicKey() sollib.PublicKey {
	return s.pubKey
}

// Sign signs the message with the KMS Ed25519 key.
func (s *KMSSigner) Sign(ctx context.Context, message []byte) (sollib.Signature, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, kmsSignTimeout)
		defer cancel()
	}

	sig, err := s.signer.Sign(ctx, message)
	if err != nil {
		return sollib.Signature{}, err
	}

	return sollib.SignatureFromBytes(sig), nil
}
