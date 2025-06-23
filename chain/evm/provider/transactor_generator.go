package provider

import (
	"context"
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
	_ TransactorGenerator = (*transactorFromKMSSigner)(nil)
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

	transactor, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		return nil, err
	}

	return transactor, nil
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

	transactor, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		return nil, err
	}

	return transactor, nil
}

// TransactorFromKMS creates a TransactorGenerator that uses a KMS key to sign transactions.
//
// It requires the KMS key ID, region, and optionally an AWS profile name. If the AWS profile
// name is not provided, it defaults to using the environment variables to determine the AWS
// profile.
func TransactorFromKMS(keyID, keyRegion, awsProfileName string) (TransactorGenerator, error) {
	signer, err := NewKMSSigner(keyID, keyRegion, awsProfileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS signer: %w", err)
	}

	return &transactorFromKMSSigner{
		signer: signer,
	}, nil
}

// TransactorFromKMSSigner creates a TransactorGenerator from an existing KMSSigner instance.
func TransactorFromKMSSigner(signer *KMSSigner) TransactorGenerator {
	return &transactorFromKMSSigner{
		signer: signer,
	}
}

// transactorFromKMSSigner is a TransactorGenerator that creates a transactor using a KMS signer.
type transactorFromKMSSigner struct {
	signer *KMSSigner
}

// Generate uses KMS to create a bind.TransactOpts instance for signing transactions.
func (g *transactorFromKMSSigner) Generate(chainID *big.Int) (*bind.TransactOpts, error) {
	transactor, err := g.signer.GetTransactOpts(context.TODO(), chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transact opts from KMS signer: %w", err)
	}

	return transactor, nil
}
