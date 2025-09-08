package provider

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/xssnick/tonutils-go/ton/wallet"
)

// PrivateKeyGenerator is an interface for generating Ton keypairs.
type PrivateKeyGenerator interface {
	Generate() (ed25519.PrivateKey, error)
}

var (
	_ PrivateKeyGenerator = (*privateKeyFromRaw)(nil)
	_ PrivateKeyGenerator = (*privateKeyRandom)(nil)
)

// PrivateKeyFromRaw creates a new instance of the privateKeyFromRaw generator with the raw private
// key.
func PrivateKeyFromRaw(privateKey string) *privateKeyFromRaw {
	return &privateKeyFromRaw{
		privateKey: privateKey,
	}
}

type privateKeyFromRaw struct {
	// privateKey is the hex encoded private key used to generate the TON keypair.
	privateKey string
}

func (g *privateKeyFromRaw) Generate() (ed25519.PrivateKey, error) {
	privateKeyBytes, err := hex.DecodeString(g.privateKey)
	if err != nil {
		return ed25519.PrivateKey{}, fmt.Errorf("failed to parse private key: %w", err)
	}

	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return ed25519.PrivateKey{}, fmt.Errorf("invalid key len: %d, must be %d", len(privateKeyBytes), ed25519.PrivateKeySize)
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)

	return privateKey, nil
}

func PrivateKeyRandom() *privateKeyRandom {
	return &privateKeyRandom{}
}

type privateKeyRandom struct{}

func (g *privateKeyRandom) Generate() (ed25519.PrivateKey, error) {
	seed := wallet.NewSeed()
	privateKey, err := wallet.SeedToPrivateKey(seed /*password=*/, "" /*isBIP39=*/, false)

	if err != nil {
		return ed25519.PrivateKey{}, fmt.Errorf("failed to generate random private key: %w", err)
	}

	return privateKey, nil
}
