package sui

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/block-vision/sui-go-sdk/signer"
	"golang.org/x/crypto/blake2b"
)

// TODO: Everything in this file should come from chainlink-sui when available
type SuiSigner interface {
	// Sign signs the given message and returns the serialized signature.
	Sign(message []byte) ([]string, error)

	// GetAddress returns the Sui address derived from the signer's public key
	GetAddress() (string, error)
}

type suiSigner struct {
	signer *signer.Signer
}

func NewSignerFromSeed(seed []byte) (SuiSigner, error) {
	sdkSigner := signer.NewSigner(seed)
	return &suiSigner{signer: sdkSigner}, nil
}

func NewSignerFromHexPrivateKey(hexPrivateKey string) (SuiSigner, error) {
	if len(hexPrivateKey) >= 2 && hexPrivateKey[:2] == "0x" {
		hexPrivateKey = hexPrivateKey[2:]
	}

	if len(hexPrivateKey) != 64 {
		return nil, fmt.Errorf("hex private key must be exactly 64 characters (32 bytes), got %d characters", len(hexPrivateKey))
	}

	privateKeyBytes, err := hex.DecodeString(hexPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex private key: %w", err)
	}

	return NewSignerFromSeed(privateKeyBytes)
}

func (s *suiSigner) Sign(message []byte) ([]string, error) {
	// Add intent scope for transaction data (0x00, 0x00, 0x00)
	intentMessage := append([]byte{0x00, 0x00, 0x00}, message...)

	// Hash the message with blake2b
	hash := blake2b.Sum256(intentMessage)

	// Sign the hash
	signature := ed25519.Sign(s.signer.PriKey, hash[:])

	// Get public key
	publicKey := s.signer.PriKey.Public().(ed25519.PublicKey)

	// Create serialized signature: flag + signature + pubkey
	serializedSig := make([]byte, 1+len(signature)+len(publicKey))
	serializedSig[0] = 0x00 // Ed25519 flag
	copy(serializedSig[1:], signature)
	copy(serializedSig[1+len(signature):], publicKey)

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(serializedSig)

	return []string{encoded}, nil
}

func (s *suiSigner) GetAddress() (string, error) {
	publicKey := s.signer.PriKey.Public().(ed25519.PublicKey)

	// For Ed25519, the signature scheme is 0x00
	const signatureScheme = 0x00

	// Create the data to hash: signature scheme byte || public key
	data := append([]byte{signatureScheme}, publicKey...)

	// Hash using Blake2b-256
	hash := blake2b.Sum256(data)

	// The Sui address is the hex representation of the hash
	return "0x" + hex.EncodeToString(hash[:]), nil
}

// PublicKeyBytes extracts the raw 32-byte ed25519 public key from a SuiSigner.
func PublicKeyBytes(s SuiSigner) ([]byte, error) {
	impl, ok := s.(*suiSigner)
	if !ok {
		return nil, fmt.Errorf("unsupported signer type %T", s)
	}
	priv := []byte(impl.signer.PriKey)
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("unexpected ed25519 key length: %d", len(priv))
	}
	pub := make([]byte, ed25519.PublicKeySize)
	copy(pub, priv[32:]) // last 32 bytes are the pubkey

	return pub, nil
}

// PrivateKey returns the underlying ed25519.PrivateKey (64 bytes = seed||pubkey)
// from a SuiSigner created in this package.
func PrivateKey(s SuiSigner) (ed25519.PrivateKey, error) {
	impl, ok := s.(*suiSigner)
	if !ok {
		return nil, fmt.Errorf("unsupported signer type %T", s)
	}

	return impl.signer.PriKey, nil
}
