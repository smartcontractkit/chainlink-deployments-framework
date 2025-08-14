package provider

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
)

// SignerGenerator is an interface for signing TRON transactions.
type SignerGenerator interface {
	// Sign signs the given transaction hash and returns the signature bytes.
	Sign(ctx context.Context, txHash []byte) ([]byte, error)
	// GetAddress returns the TRON address associated with this signer generator.
	GetAddress() (address.Address, error)
}

var (
	_ SignerGenerator = (*signerGenCTFDefault)(nil)
	_ SignerGenerator = (*signerGenPrivateKey)(nil)
	_ SignerGenerator = (*signerRandom)(nil)
	_ SignerGenerator = (*signerGenKMS)(nil)
)

// signerGenCTFDefault is a default signer generator for CTF (Chainlink Testing Framework).
type signerGenCTFDefault struct {
	signerGenPrivateKey
}

// SignerGenCTFDefault creates a new instance of signerGenCTFDefault. It uses the default
// TRON account and private key from the blockchain package.
func SignerGenCTFDefault() *signerGenCTFDefault {
	return &signerGenCTFDefault{
		signerGenPrivateKey: signerGenPrivateKey{
			PrivateKey: blockchain.TRONAccounts.PrivateKeys[0],
		},
	}
}

// signerGenPrivateKey is a signer generator that creates a signer from the private key.
type signerGenPrivateKey struct {
	// PrivateKey is the hex formatted private key used to generate the Tron account.
	PrivateKey string

	// Lazy initialization fields
	once      sync.Once
	privKey   *ecdsa.PrivateKey
	address   address.Address
	initError error
}

// SignerGenPrivateKey creates a new instance of signerGenPrivateKey with the provided private key.
// Initialization is performed lazily on first Sign() or GetAddress() call using sync.Once.
func SignerGenPrivateKey(privateKey string) *signerGenPrivateKey {
	return &signerGenPrivateKey{
		PrivateKey: privateKey,
	}
}

// initialize parses the private key and derives the TRON address.
func (g *signerGenPrivateKey) initialize() error {
	// Parse the hex-encoded private key string directly to *ecdsa.PrivateKey
	privKey, err := crypto.HexToECDSA(g.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	g.privKey = privKey
	g.address = address.PubkeyToAddress(privKey.PublicKey)

	return nil
}

// Sign signs the given transaction hash using the private key.
// Initializes the private key lazily on first call.
func (g *signerGenPrivateKey) Sign(ctx context.Context, txHash []byte) ([]byte, error) {
	// Lazy initialization using sync.Once
	g.once.Do(func() {
		g.initError = g.initialize()
	})

	if g.initError != nil {
		return nil, fmt.Errorf("account generator initialization failed: %w", g.initError)
	}

	return crypto.Sign(txHash, g.privKey)
}

// GetAddress returns the TRON address associated with this signer generator.
// Initializes the private key lazily if not already initialized.
func (g *signerGenPrivateKey) GetAddress() (address.Address, error) {
	// Lazy initialization using sync.Once
	g.once.Do(func() {
		g.initError = g.initialize()
	})

	if g.initError != nil {
		return address.Address(""), fmt.Errorf("private key signer initialization failed: %w", g.initError)
	}

	return g.address, nil
}

// SignerRandom creates a new instance of the signerRandom generator.
// Initialization is performed lazily on first Sign() or GetAddress() call.
func SignerRandom() *signerRandom {
	return &signerRandom{}
}

// signerRandom is a TRON signer generator created with a random account.
type signerRandom struct {
	// Lazy initialization fields
	once      sync.Once
	privKey   *ecdsa.PrivateKey
	address   address.Address
	initError error
}

// initialize generates a new random TRON private key and derives the address.
func (g *signerRandom) initialize() error {
	// Generate a new random private key
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate a random private key: %w", err)
	}

	g.privKey = privKey
	g.address = address.PubkeyToAddress(privKey.PublicKey)

	return nil
}

// Sign signs the given transaction hash using the private key.
func (g *signerRandom) Sign(ctx context.Context, txHash []byte) ([]byte, error) {
	// Lazy initialization using sync.Once
	g.once.Do(func() {
		g.initError = g.initialize()
	})

	if g.initError != nil {
		return nil, fmt.Errorf("account generator initialization failed: %w", g.initError)
	}

	return crypto.Sign(txHash, g.privKey)
}

// GetAddress returns the TRON address associated with this signer generator.
func (g *signerRandom) GetAddress() (address.Address, error) {
	// Lazy initialization using sync.Once
	g.once.Do(func() {
		g.initError = g.initialize()
	})

	if g.initError != nil {
		return address.Address(""), fmt.Errorf("random signer initialization failed: %w", g.initError)
	}

	return g.address, nil
}

// signerGenKMS is a signer generator that uses AWS KMS for signing.
type signerGenKMS struct {
	signer *kmsSigner
}

// SignerGenKMS creates a new instance of signerGenKMS with the provided KMS configuration.
// Initialization is performed lazily on first Sign() or GetAddress() call using sync.Once.
func SignerGenKMS(keyID, keyRegion, awsProfile string) *signerGenKMS {
	return &signerGenKMS{
		signer: newKMSSigner(keyID, keyRegion, awsProfile),
	}
}

// Sign signs the given transaction hash using the KMS signer.
func (g *signerGenKMS) Sign(ctx context.Context, txHash []byte) ([]byte, error) {
	return g.signer.Sign(txHash)
}

// GetAddress returns the TRON address associated with this signer generator.
func (g *signerGenKMS) GetAddress() (address.Address, error) {
	return g.signer.GetAddress()
}
