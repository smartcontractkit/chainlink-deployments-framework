package stellar

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	bindings "github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/kms"
)

// signTimeout bounds a single kms.Sign call.
const signTimeout = 30 * time.Second

// KmsSigner signs Stellar transactions with an Ed25519 key held in AWS KMS.
// The private key never leaves KMS. It implements bindings.Signer.
//
// The signer deliberately holds no context. It is built while the chain is
// loaded but signs for as long as the chain is in use, so retaining the loader's
// context would make every signature fail once that context is done.
type KmsSigner struct {
	signer  *kms.Ed25519KMSSigner
	pubKey  ed25519.PublicKey
	address string
}

var _ bindings.Signer = (*KmsSigner)(nil)

// NewKmsSigner builds a bindings.Signer backed by an AWS KMS Ed25519 key.
func NewKmsSigner(ctx context.Context, keyID, keyRegion string) (bindings.Signer, error) {
	client, err := kms.NewEd25519Client(ctx, keyRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS Ed25519 client: %w", err)
	}

	return newKmsSigner(ctx, keyID, client)
}

// newKmsSigner builds a KmsSigner from an existing KMS Ed25519 client. This
// constructor accepts the client directly so tests can inject a mock.
//
// ctx covers construction only, which is the one-off public key fetch. It is not
// retained for later signing.
func newKmsSigner(ctx context.Context, keyID string, client kms.Ed25519Client) (*KmsSigner, error) {
	signer, err := kms.NewEd25519KMSSigner(ctx, keyID, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS Ed25519 signer: %w", err)
	}

	pubKey, err := signer.GetPublicKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch KMS Ed25519 public key: %w", err)
	}

	address, err := strkey.Encode(strkey.VersionByteAccountID, pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Stellar account address: %w", err)
	}

	return &KmsSigner{
		signer:  signer,
		pubKey:  pubKey,
		address: address,
	}, nil
}

// Sign signs message with the KMS Ed25519 key.
//
// bindings.Signer.Sign takes no context, so Sign derives a fresh one per call
// instead of reusing the context that built the signer.
func (s *KmsSigner) Sign(message []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), signTimeout)
	defer cancel()

	return s.signer.Sign(ctx, message)
}

// SignDecorated signs message and returns a decorated signature in Stellar XDR
// format. The hint is the last four bytes of the public key.
func (s *KmsSigner) SignDecorated(message []byte) (xdr.DecoratedSignature, error) {
	sig, err := s.Sign(message)
	if err != nil {
		return xdr.DecoratedSignature{}, err
	}

	return xdr.NewDecoratedSignature(sig, s.hint()), nil
}

// hint returns the last four bytes of the public key.
func (s *KmsSigner) hint() [4]byte {
	var hint [4]byte
	copy(hint[:], s.pubKey[len(s.pubKey)-4:])

	return hint
}

// Address returns the Stellar account address derived from the public key.
func (s *KmsSigner) Address() string {
	return s.address
}

// KeypairFull returns nil. The private key lives in KMS, so no keypair.Full is
// available.
func (s *KmsSigner) KeypairFull() *keypair.Full {
	return nil
}
