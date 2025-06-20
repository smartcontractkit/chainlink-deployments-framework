package provider

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
)

// TransactorGenerator is an interface for generating geth's *bind.TransactOpts instances. These
// instances are used to sign transactions using geth bindings.
type TransactorGenerator interface {
	Generate(chainID *big.Int) (*bind.TransactOpts, error)
}

var (
	_ TransactorGenerator = (*transactorFromRaw)(nil)
	_ TransactorGenerator = (*transactorRandom)(nil)
)

// TransactorFromRaw returns a generator which creates a transactor from a raw private key.
func TransactorFromRaw(privKey string) TransactorGenerator {
	return &transactorFromRaw{
		privKey: privKey,
	}
}

// transactorFromRaw is a TransactorGenerator that creates a transactor from a private key.
type transactorFromRaw struct {
	privKey string
}

// Generate parses the hex encoded private key and returns the bind transactor options.
func (g *transactorFromRaw) Generate(chainID *big.Int) (*bind.TransactOpts, error) {
	privKey, err := crypto.HexToECDSA(g.privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key to ECDSA: %w", err)
	}

	key, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// TransactorRandom is a TransactorGenerator that creates a transactor with a random private key.
func TransactorRandom() TransactorGenerator {
	return &transactorRandom{}
}

// transactorRandom is an TransactorGenerator that creates a transactor from a random keypair.
type transactorRandom struct{}

// Generate generates a random key and returns the bind transactor options.
func (g *transactorRandom) Generate(chainID *big.Int) (*bind.TransactOpts, error) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random private key: %w", err)
	}

	key, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		return nil, err
	}

	return key, nil
}
