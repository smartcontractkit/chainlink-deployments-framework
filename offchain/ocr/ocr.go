package ocr

import (
	"github.com/cosmos/go-bip39"
)

// NewBIP39Mnemonic generates a new BIP39 mnemonic phrase with the specified entropy size.
func NewBIP39Mnemonic(entropySize int) (string, error) {
	entropy, err := bip39.NewEntropy(entropySize)
	if err != nil {
		return "", err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}

	return mnemonic, nil
}
