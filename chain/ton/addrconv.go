package ton

import (
	"fmt"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/xssnick/tonutils-go/address"
)

// AddressToBytes converts a TON address string to bytes.
// TON addresses can be in various formats but are normalized to 32 bytes.
func AddressToBytes(addressStr string) ([]byte, error) {
	addr, err := address.ParseAddr(addressStr)
	if err != nil {
		return nil, fmt.Errorf("invalid TON address format: %s, error: %w", addressStr, err)
	}

	// Return the raw 32-byte address data
	return addr.Data(), nil
}

// AddressConverter implements address conversion for TON chains.
// This struct implements the AddressConverter strategy interface.
type AddressConverter struct{}

// ConvertToBytes converts a TON address string to bytes.
func (t AddressConverter) ConvertToBytes(address string) ([]byte, error) {
	return AddressToBytes(address)
}

// Supports returns true if this converter supports the given chain family.
func (t AddressConverter) Supports(family string) bool {
	return family == chain_selectors.FamilyTon
}
