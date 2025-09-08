package sui

import (
	"encoding/hex"
	"fmt"
	"strings"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

// AddressToBytes converts a Sui address string to bytes.
// Sui addresses are hex strings typically prefixed with "0x" (32 bytes).
func AddressToBytes(address string) ([]byte, error) {
	// Remove "0x" prefix if present
	address = strings.TrimPrefix(address, "0x")

	// Sui addresses should be 64 hex characters (32 bytes)
	if len(address) != 64 {
		return nil, fmt.Errorf("invalid Sui address format: expected 64 hex characters, got %d", len(address))
	}

	addressBytes, err := hex.DecodeString(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Sui address format: %s, error: %w", address, err)
	}

	return addressBytes, nil
}

// AddressConverter implements address conversion for Sui chains.
// This struct implements the AddressConverter strategy interface.
type AddressConverter struct{}

// ConvertToBytes converts a Sui address string to bytes.
func (s AddressConverter) ConvertToBytes(address string) ([]byte, error) {
	return AddressToBytes(address)
}

// Supports returns true if this converter supports the given chain family.
func (s AddressConverter) Supports(family string) bool {
	return family == chain_selectors.FamilySui
}
