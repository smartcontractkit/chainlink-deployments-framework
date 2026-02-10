package stellar

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// StellarSigner is an interface that provides signing capabilities for Stellar transactions.
type StellarSigner interface {
	// Sign signs the given message and returns the signature bytes.
	Sign(message []byte) ([]byte, error)

	// SignDecorated signs the given message and returns a decorated signature (XDR format).
	SignDecorated(message []byte) (xdr.DecoratedSignature, error)

	// Address returns the Stellar address derived from the signer's public key.
	Address() string
}

// stellarKeypairSigner implements StellarSigner using a keypair.Full from the Stellar SDK.
type stellarKeypairSigner struct {
	kp *keypair.Full
}

var _ StellarSigner = (*stellarKeypairSigner)(nil)

// NewStellarKeypairSigner creates a new StellarSigner from a keypair.Full.
func NewStellarKeypairSigner(kp *keypair.Full) StellarSigner {
	return &stellarKeypairSigner{kp: kp}
}

// Sign signs the given message and returns the signature bytes.
func (s *stellarKeypairSigner) Sign(message []byte) ([]byte, error) {
	return s.kp.Sign(message)
}

// SignDecorated signs the given message and returns a decorated signature.
func (s *stellarKeypairSigner) SignDecorated(message []byte) (xdr.DecoratedSignature, error) {
	return s.kp.SignDecorated(message)
}

// Address returns the Stellar address.
func (s *stellarKeypairSigner) Address() string {
	return s.kp.Address()
}

// KeypairFromHex creates a keypair.Full from a hex-encoded private key.
// The hex string can be with or without the "0x" prefix.
func KeypairFromHex(hexKey string) (*keypair.Full, error) {
	// Remove "0x" prefix if present
	hexKey = strings.TrimPrefix(hexKey, "0x")

	// Decode hex to bytes
	rawSeed, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}

	// Stellar keypairs use 32-byte seeds
	if len(rawSeed) != 32 {
		return nil, fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(rawSeed))
	}

	var seed [32]byte
	copy(seed[:], rawSeed)

	kp, err := keypair.FromRawSeed(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create keypair from seed: %w", err)
	}

	return kp, nil
}
