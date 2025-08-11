package sui

import (
	"encoding/hex"
	"fmt"

	"github.com/block-vision/sui-go-sdk/constant"
	"github.com/block-vision/sui-go-sdk/signer"
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
	if s.signer == nil {
		return nil, fmt.Errorf("signer is nil")
	}

	// Sign the message as a transaction message
	signedMsg, err := s.signer.SignMessage(string(message), constant.TransactionDataIntentScope)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return []string{signedMsg.Signature}, nil
}

func (s *suiSigner) GetAddress() (string, error) {
	if s.signer == nil {
		return "", fmt.Errorf("signer is nil")
	}

	return s.signer.Address, nil
}
