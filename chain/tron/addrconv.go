package tron

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

// AddressToBytes converts a Tron address string to bytes.
// Tron addresses can be in base58 format (like "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH").
func AddressToBytes(addressStr string) ([]byte, error) {
	addr, err := address.Base58ToAddress(addressStr)
	if err != nil {
		return nil, fmt.Errorf("invalid Tron address format: %s, error: %w", addressStr, err)
	}

	return addr.Bytes(), nil
}

// AddressConverter implements address conversion for Tron chains.
// This struct implements the AddressConverter strategy interface.
type AddressConverter struct{}

// ConvertToBytes converts a Tron address string to bytes.
func (t AddressConverter) ConvertToBytes(address string) ([]byte, error) {
	return AddressToBytes(address)
}

// Supports returns true if this converter supports the given chain family.
func (t AddressConverter) Supports(family string) bool {
	return family == chain_selectors.FamilyTron
}
