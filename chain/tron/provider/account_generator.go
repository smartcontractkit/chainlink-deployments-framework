package provider

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/keystore"
)

// AccountGenerator is an interface for generating Tron accounts.
type AccountGenerator interface {
	Generate() (*keystore.Keystore, address.Address, error)
}

var (
	_ AccountGenerator = (*accountGenCTFDefault)(nil)
	_ AccountGenerator = (*accountGenPrivateKey)(nil)
	_ AccountGenerator = (*accountRandom)(nil)
)

// accountGenCTFDefault is a default account generator for CTF (Chainlink Testing Framework).
type accountGenCTFDefault struct {
	accountGenPrivateKey
}

// AccountGenCTFDefault creates a new instance of accountGenCTFDefault. It uses the default
// TRON account and private key from the blockchain package.
func AccountGenCTFDefault() *accountGenCTFDefault {
	return &accountGenCTFDefault{
		accountGenPrivateKey: accountGenPrivateKey{
			PrivateKey: blockchain.TRONAccounts.PrivateKeys[0],
		},
	}
}

// accountGenPrivateKey is an account generator that creates an account from the private key.
type accountGenPrivateKey struct {
	// PrivateKey is the hex formatted private key used to generate the Tron account.
	PrivateKey string
}

// AccountGenPrivateKey creates a new instance of accountGenPrivateKey with the provided private key.
func AccountGenPrivateKey(privateKey string) *accountGenPrivateKey {
	return &accountGenPrivateKey{
		PrivateKey: privateKey,
	}
}

// Generate generates an Tron keystore account from the provided private key. It returns an error if the
// private key string cannot be parsed.
func (g *accountGenPrivateKey) Generate() (*keystore.Keystore, address.Address, error) {
	// Decode the hex-encoded private key string
	privBytes, err := hex.DecodeString(g.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode hex-encoded private key: %w", err)
	}

	// Parse the bytes into an *ecdsa.PrivateKey
	privKey, err := crypto.ToECDSA(privBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key bytes: %w", err)
	}

	ks, addr := keystore.NewKeystore(privKey)

	return ks, addr, err
}

// AccountRandom creates a new instance of the accountRandom generator.
func AccountRandom() *accountRandom {
	return &accountRandom{}
}

// accountRandom is an Tron keystore account pair generator created with a random account.
type accountRandom struct{}

// Generate generates a new random Tron keystore account pair and returns them.
func (g *accountRandom) Generate() (*keystore.Keystore, address.Address, error) {
	// Generate a new random private key
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode a random private key: %w", err)
	}

	ks, addr := keystore.NewKeystore(privKey)

	return ks, addr, err
}
