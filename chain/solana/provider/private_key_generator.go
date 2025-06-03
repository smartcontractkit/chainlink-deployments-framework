package provider

import (
	"fmt"

	sollib "github.com/gagliardetto/solana-go"
)

// PrivateKeyGenerator is an interface for generating Solana Keypairs.
type PrivateKeyGenerator interface {
	// Generate creates a new Solana keypair and returns the private key
	Generate() (sollib.PrivateKey, error)
}

var (
	_ PrivateKeyGenerator = (*privateKeyFromRaw)(nil)
	_ PrivateKeyGenerator = (*privateKeyRandom)(nil)
)

// PrivateKeyFromRaw creates a new instance of the privateKeyFromRaw generator with the raw private
// key.
func PrivateKeyFromRaw(privateKey string) *privateKeyFromRaw {
	return &privateKeyFromRaw{
		PrivateKey: privateKey,
	}
}

// privateKeyFromRaw is an Solana keypair generator that creates an keypair from a private key.
type privateKeyFromRaw struct {
	// PrivateKey is the base58 encoded private key used to generate the Solana keypair.
	PrivateKey string
}

// Generate generates a Solana keypair from the provided base58 encoded private key and returns the
// private key.
func (g *privateKeyFromRaw) Generate() (sollib.PrivateKey, error) {
	privKey, err := sollib.PrivateKeyFromBase58(g.PrivateKey)
	if err != nil {
		return sollib.PrivateKey{}, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privKey, nil
}

// PrivateKeyRandom creates a new instance of the privateKeyRandom generator.
func PrivateKeyRandom() *privateKeyRandom {
	return &privateKeyRandom{}
}

// privateKeyRandom is an Solana keypair generator that creates a random keypair.
type privateKeyRandom struct{}

// Generate generates a new random Solana keypair and returns the private key.
func (g *privateKeyRandom) Generate() (sollib.PrivateKey, error) {
	privKey, err := sollib.NewRandomPrivateKey()
	if err != nil {
		return sollib.PrivateKey{}, fmt.Errorf("failed to generate random private key: %w", err)
	}

	return privKey, nil
}
