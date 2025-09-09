package aptos

import (
	"fmt"

	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

// AddressToBytes converts an Aptos address string to bytes.
// Aptos addresses can be in various formats (short, long, with/without 0x prefix) but are normalized to 32 bytes.
func AddressToBytes(address string) ([]byte, error) {
	var addr aptoslib.AccountAddress
	err := addr.ParseStringRelaxed(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Aptos address format: %s, error: %w", address, err)
	}

	return addr[:], nil
}

// AddressConverter implements address conversion for Aptos chains.
// This struct implements the AddressConverter interface.
type AddressConverter struct{}

// ConvertToBytes converts an Aptos address string to bytes.
func (a AddressConverter) ConvertToBytes(address string) ([]byte, error) {
	return AddressToBytes(address)
}

// Supports returns true if this converter supports the given chain family.
func (a AddressConverter) Supports(family string) bool {
	return family == chain_selectors.FamilyAptos
}
