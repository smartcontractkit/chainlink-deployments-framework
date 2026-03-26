package provider

import (
	"errors"
	"fmt"

	"github.com/stellar/go-stellar-sdk/keypair"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar"
)

// KeypairGenerator is an interface for generating Stellar keypairs.
type KeypairGenerator interface {
	Generate() (stellar.StellarSigner, error)
}

// keypairFromHex is a KeypairGenerator that generates a keypair from a hex-encoded private key.
type keypairFromHex struct {
	hexKey string
}

var _ KeypairGenerator = (*keypairFromHex)(nil)

// KeypairFromHex creates a KeypairGenerator that generates a keypair from a hex-encoded private key.
// The hex string can be with or without the "0x" prefix.
func KeypairFromHex(hexKey string) KeypairGenerator {
	return &keypairFromHex{hexKey: hexKey}
}

// Generate generates a Stellar keypair from the hex-encoded private key.
func (k *keypairFromHex) Generate() (stellar.StellarSigner, error) {
	if k.hexKey == "" {
		return nil, errors.New("hex key is empty")
	}

	kp, err := stellar.KeypairFromHex(k.hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create keypair from hex: %w", err)
	}

	return stellar.NewStellarKeypairSigner(kp), nil
}

// keypairRandom is a KeypairGenerator that generates a random keypair.
type keypairRandom struct{}

var _ KeypairGenerator = (*keypairRandom)(nil)

// KeypairRandom creates a KeypairGenerator that generates a random keypair.
func KeypairRandom() KeypairGenerator {
	return &keypairRandom{}
}

// Generate generates a random Stellar keypair.
func (k *keypairRandom) Generate() (stellar.StellarSigner, error) {
	kp, err := keypair.Random()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random keypair: %w", err)
	}

	return stellar.NewStellarKeypairSigner(kp), nil
}
