package provider

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
)

// AccountGenerator is an interface for generating Tron accounts.
type AccountGenerator interface {
	Generate() (*keystore.KeyStore, keystore.Account, error)
}

var (
	_ AccountGenerator = (*accountGenPrivateKey)(nil)
)

// accountGenPrivateKey is an account generator that creates an account from the private key.
type accountGenPrivateKey struct {
	// PrivateKey is the hex formatted private key used to generate the Aptos account.
	PrivateKey string
}

// AccountGenPrivateKey creates a new instance of accountGenPrivateKey with the provided private key.
func AccountGenPrivateKey(privateKey string) *accountGenPrivateKey {
	return &accountGenPrivateKey{
		PrivateKey: privateKey,
	}
}

// Generate generates an Aptos account from the provided private key. It returns an error if the
// private key string cannot be parsed.
func (g *accountGenPrivateKey) Generate() (*keystore.KeyStore, keystore.Account, error) {
	// Decode the hex-encoded private key string
	privBytes, err := hex.DecodeString(g.PrivateKey)
	if err != nil {
		return nil, keystore.Account{}, fmt.Errorf("failed to decode hex-encoded private key: %w", err)
	}

	// Parse the bytes into an *ecdsa.PrivateKey
	privKey, err := crypto.ToECDSA(privBytes)
	if err != nil {
		return nil, keystore.Account{}, fmt.Errorf("failed to parse private key bytes: %w", err)
	}

	ks := keystore.NewKeyStore("./wallet", keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.ImportECDSA(privKey, "")
	if err != nil {
		return nil, keystore.Account{}, fmt.Errorf("failed to import ECDSA private key: %w", err)
	}

	return ks, acc, err
}
