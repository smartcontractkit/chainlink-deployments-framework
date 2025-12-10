package solana

import (
	"fmt"

	sollib "github.com/gagliardetto/solana-go"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

// AddressToBytes converts a Solana address string to bytes.
// Solana addresses are base58-encoded public keys (32 bytes).
func AddressToBytes(address string) ([]byte, error) {
	pubkey, err := sollib.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Solana address format: %s, error: %w", address, err)
	}

	return pubkey.Bytes(), nil
}

// AddressConverter implements address conversion for Solana chains.
// This struct implements the AddressConverter strategy interface.
type AddressConverter struct{}

// ConvertToBytes converts a Solana address string to bytes.
func (s AddressConverter) ConvertToBytes(address string) ([]byte, error) {
	return AddressToBytes(address)
}

// Supports returns true if this converter supports the given chain family.
func (s AddressConverter) Supports(family string) bool {
	return family == chain_selectors.FamilySolana
}
