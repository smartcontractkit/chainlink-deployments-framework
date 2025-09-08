package provider

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"

	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
)

// AccountGenerator is an interface for generating Aptos accounts.
type AccountGenerator interface {
	Generate() (*aptoslib.Account, error)
}

var (
	_ AccountGenerator = (*accountGenCTFDefault)(nil)
	_ AccountGenerator = (*accountGenNewSingleSender)(nil)
	_ AccountGenerator = (*accountGenPrivateKey)(nil)
)

// accountGenCTFDefault is a default account generator for CTF (Chainlink Testing Framework).
type accountGenCTFDefault struct {
	// The account address string to use for generating the account.
	accountStr string
	// privateKeyStr is the private key string to use for generating the account.
	privateKeyStr string
}

// AccountGenCTFDefault creates a new instance of accountGenCTFDefault. It uses the default
// Aptos account and private key from the blockchain package.
func AccountGenCTFDefault() *accountGenCTFDefault {
	return &accountGenCTFDefault{
		accountStr:    blockchain.DefaultAptosAccount,
		privateKeyStr: blockchain.DefaultAptosPrivateKey,
	}
}

// Generate generates an Aptos account using the default address and private key from the
// blockchain package. It returns an error if the address or private key is invalid.
func (g *accountGenCTFDefault) Generate() (*aptoslib.Account, error) {
	var address aptoslib.AccountAddress

	if err := address.ParseStringRelaxed(g.accountStr); err != nil {
		return nil, fmt.Errorf("failed to parse account address %s: %w", g.accountStr, err)
	}

	privateKeyBytes, err := hex.DecodeString(strings.TrimPrefix(g.privateKeyStr, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	privateKey := ed25519.NewKeyFromSeed(privateKeyBytes)

	return aptoslib.NewAccountFromSigner(&crypto.Ed25519PrivateKey{Inner: privateKey}, address)
}

// accountGenNewSingleSender is an account generator that creates a new single sender account.
type accountGenNewSingleSender struct{}

// AccountGenNewSingleSender creates a new instance of accountGenNewSingleSender.
func AccountGenNewSingleSender() *accountGenNewSingleSender {
	return &accountGenNewSingleSender{}
}

// Generate generates a new Aptos account using the aptos library's single sender account creation
// method.
func (g *accountGenNewSingleSender) Generate() (*aptoslib.Account, error) {
	return aptoslib.NewEd25519SingleSenderAccount()
}

// accountGenPrivateKey is an account generator that creates an account from the private key.
type accountGenPrivateKey struct {
	// privateKey is the hex formatted private key used to generate the Aptos account.
	privateKey string
}

// AccountGenPrivateKey creates a new instance of accountGenPrivateKey with the provided private key.
func AccountGenPrivateKey(privateKey string) *accountGenPrivateKey {
	return &accountGenPrivateKey{
		privateKey: privateKey,
	}
}

// Generate generates an Aptos account from the provided private key. It returns an error if the
// private key string cannot be parsed.
func (g *accountGenPrivateKey) Generate() (*aptoslib.Account, error) {
	privateKey := &crypto.Ed25519PrivateKey{}
	if err := privateKey.FromHex(g.privateKey); err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return aptoslib.NewAccountFromSigner(privateKey)
}
