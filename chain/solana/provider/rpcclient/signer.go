package rpcclient

import (
	"context"
	"fmt"

	sollib "github.com/gagliardetto/solana-go"
)

// Signer signs Solana transaction messages on behalf of a single account.
//
// Implementations may hold the private key locally (see NewPrivateKeySigner) or delegate signing
// to a remote service such as AWS KMS, in which case the private key is never available.
type Signer interface {
	// PublicKey returns the public key of the signing account.
	PublicKey() sollib.PublicKey
	// Sign signs the serialized transaction message and returns the Ed25519 signature.
	Sign(ctx context.Context, message []byte) (sollib.Signature, error)
}

var _ Signer = (*privateKeySigner)(nil)

// privateKeySigner is a Signer backed by a local private key.
type privateKeySigner struct {
	key sollib.PrivateKey
}

// NewPrivateKeySigner returns a Signer backed by a local private key.
func NewPrivateKeySigner(key sollib.PrivateKey) Signer {
	return &privateKeySigner{key: key}
}

// PublicKey returns the public key of the private key.
func (s *privateKeySigner) PublicKey() sollib.PublicKey {
	return s.key.PublicKey()
}

// Sign signs the message with the private key.
func (s *privateKeySigner) Sign(_ context.Context, message []byte) (sollib.Signature, error) {
	return s.key.Sign(message)
}

// signTx signs every required signature of tx. The deployer signs for its own public key, and
// all other signers are looked up in the signers map.
func signTx(
	ctx context.Context,
	tx *sollib.Transaction,
	deployer Signer,
	signers map[sollib.PublicKey]sollib.PrivateKey,
) error {
	message, err := tx.Message.MarshalBinary()
	if err != nil {
		return fmt.Errorf("unable to encode message for signing: %w", err)
	}

	signerKeys := tx.Message.AccountKeys[:tx.Message.Header.NumRequiredSignatures]
	signatures := make([]sollib.Signature, len(signerKeys))
	for i, key := range signerKeys {
		var sig sollib.Signature
		if key.Equals(deployer.PublicKey()) {
			sig, err = deployer.Sign(ctx, message)
		} else if priv, ok := signers[key]; ok {
			sig, err = priv.Sign(message)
		} else {
			return fmt.Errorf("signer key %q not found. Ensure all the signer keys are provided", key)
		}
		if err != nil {
			return fmt.Errorf("failed to sign with key %q: %w", key, err)
		}

		signatures[i] = sig
	}

	tx.Signatures = signatures

	return nil
}
