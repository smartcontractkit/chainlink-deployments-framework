package provider

import (
	"context"
	"crypto/ecdsa"
	"fmt"

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
func SignerGenCTFDefault() (*signerGenCTFDefault, error) {
	privKeyGen, err := SignerGenPrivateKey(blockchain.TRONAccounts.PrivateKeys[0])
	if err != nil {
		return nil, fmt.Errorf("failed to create CTF default signer: %w", err)
	}

	return &signerGenCTFDefault{
		signerGenPrivateKey: *privKeyGen,
	}, nil
}

// signerGenPrivateKey is a signer generator that creates a signer from the private key.
type signerGenPrivateKey struct {
	// PrivateKey is the hex formatted private key used to generate the Tron account.
	PrivateKey string

	privKey *ecdsa.PrivateKey
	address address.Address
}

// SignerGenPrivateKey creates a new instance of signerGenPrivateKey with the provided private key.
func SignerGenPrivateKey(privateKey string) (*signerGenPrivateKey, error) {
	gen := &signerGenPrivateKey{
		PrivateKey: privateKey,
	}

	if err := gen.initialize(); err != nil {
		return nil, err
	}

	return gen, nil
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
func (g *signerGenPrivateKey) Sign(ctx context.Context, txHash []byte) ([]byte, error) {
	return crypto.Sign(txHash, g.privKey)
}

// GetAddress returns the TRON address associated with this signer generator.
func (g *signerGenPrivateKey) GetAddress() (address.Address, error) {
	return g.address, nil
}

// SignerRandom creates a new instance of the signerRandom generator.
func SignerRandom() (*signerRandom, error) {
	gen := &signerRandom{}

	if err := gen.initialize(); err != nil {
		return nil, err
	}

	return gen, nil
}

// signerRandom is a TRON signer generator created with a random account.
type signerRandom struct {
	privKey *ecdsa.PrivateKey
	address address.Address
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
	return crypto.Sign(txHash, g.privKey)
}

// GetAddress returns the TRON address associated with this signer generator.
func (g *signerRandom) GetAddress() (address.Address, error) {
	return g.address, nil
}

// signerGenKMS is a signer generator that uses AWS KMS for signing.
type signerGenKMS struct {
	signer *kmsSigner
}

// SignerGenKMS creates a new instance of signerGenKMS with the provided KMS configuration.

func SignerGenKMS(keyID, keyRegion, awsProfile string) (*signerGenKMS, error) {
	signer, err := newKMSSigner(keyID, keyRegion, awsProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS signer: %w", err)
	}

	return &signerGenKMS{
		signer: signer,
	}, nil
}

// Sign signs the given transaction hash using the KMS signer.
func (g *signerGenKMS) Sign(ctx context.Context, txHash []byte) ([]byte, error) {
	return g.signer.Sign(txHash)
}

// GetAddress returns the TRON address associated with this signer generator.
func (g *signerGenKMS) GetAddress() (address.Address, error) {
	return g.signer.GetAddress()
}
