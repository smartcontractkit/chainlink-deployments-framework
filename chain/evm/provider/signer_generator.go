package provider

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
)

// SignerGenerator is an interface for generating geth's *bind.TransactOpts instances and
// providing hash signing capabilities. These instances are used to sign transactions using geth bindings,
// and the SignHash method allows signing of arbitrary hashes.
type SignerGenerator interface {
	Generate(chainID *big.Int) (*bind.TransactOpts, error)
	SignHash(hash []byte) ([]byte, error)
}

var (
	_ SignerGenerator = (*transactorFromRaw)(nil)
	_ SignerGenerator = (*transactorRandom)(nil)
	_ SignerGenerator = (*transactorFromKMSSigner)(nil)
)

// GeneratorOptions contains configuration options for the SignerGenerator.
type GeneratorOptions struct {
	gasLimit uint64
}

// GeneratorOption is a function that modifies GeneratorOptions.
type GeneratorOption func(*GeneratorOptions)

func WithGasLimit(gasLimit uint64) GeneratorOption {
	return func(opts *GeneratorOptions) {
		opts.gasLimit = gasLimit
	}
}

// TransactorFromRaw returns a generator which creates a transactor from a raw private key.
func TransactorFromRaw(privKey string, opts ...GeneratorOption) SignerGenerator {
	// load default options
	defaultOpts := &GeneratorOptions{
		gasLimit: 0,
	}
	// apply provided options
	for _, opt := range opts {
		opt(defaultOpts)
	}

	return &transactorFromRaw{
		privKey:  privKey,
		gasLimit: defaultOpts.gasLimit,
	}
}

// transactorFromRaw is a SignerGenerator that creates a transactor from a private key.
type transactorFromRaw struct {
	privKey  string
	gasLimit uint64
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
	if g.gasLimit > 0 {
		transactor.GasLimit = g.gasLimit
	}

	return transactor, nil
}

// SignHash signs a hash using the private key stored in the generator.
func (g *transactorFromRaw) SignHash(hash []byte) ([]byte, error) {
	privKey, err := crypto.HexToECDSA(g.privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key to ECDSA: %w", err)
	}

	sig, err := crypto.Sign(hash, privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %w", err)
	}

	return sig, nil
}

// TransactorRandom is a SignerGenerator that creates a transactor with a random private key.
// A random private key is generated the first time Generate() or SignHash is called, and the same key is used for subsequent calls.
func TransactorRandom() SignerGenerator {
	return &transactorRandom{}
}

// transactorRandom is a SignerGenerator that creates a transactor from a random keypair.
type transactorRandom struct {
	privKey *ecdsa.PrivateKey
}

// Generate generates a random key and returns the bind transactor options.
func (g *transactorRandom) Generate(chainID *big.Int) (*bind.TransactOpts, error) {
	if g.privKey == nil {
		privKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate random private key: %w", err)
		}
		g.privKey = privKey
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(g.privKey, chainID)
	if err != nil {
		return nil, err
	}

	return transactor, nil
}

// SignHash signs a hash using the same random private key generated in Generate().
// If Generate() hasn't been called yet, it will generate a new random key.
func (g *transactorRandom) SignHash(hash []byte) ([]byte, error) {
	if g.privKey == nil {
		privKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate random private key: %w", err)
		}
		g.privKey = privKey
	}

	sig, err := crypto.Sign(hash, g.privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %w", err)
	}

	return sig, nil
}

// TransactorFromKMS creates a SignerGenerator that uses a KMS key to sign transactions.
//
// It requires the KMS key ID, region, and optionally an AWS profile name. If the AWS profile
// name is not provided, it defaults to using the environment variables to determine the AWS
// profile.
func TransactorFromKMS(keyID, keyRegion, awsProfileName string) (SignerGenerator, error) {
	signer, err := NewKMSSigner(keyID, keyRegion, awsProfileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS signer: %w", err)
	}

	return &transactorFromKMSSigner{
		signer: signer,
	}, nil
}

// TransactorFromKMSSigner creates a SignerGenerator from an existing KMSSigner instance.
func TransactorFromKMSSigner(signer *KMSSigner) SignerGenerator {
	return &transactorFromKMSSigner{
		signer: signer,
	}
}

// transactorFromKMSSigner is a SignerGenerator that creates a transactor using a KMS signer.
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

// SignHash signs a hash using the KMS signer.
func (g *transactorFromKMSSigner) SignHash(hash []byte) ([]byte, error) {
	return g.signer.SignHash(hash)
}
