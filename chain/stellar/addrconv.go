package stellar

import (
	"fmt"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stellar/go-stellar-sdk/strkey"
)

// AddressToBytes converts a Stellar address string to its 32-byte raw form.
// Stellar account addresses are strkey-encoded ed25519 public keys ("G…") and
// Soroban contract IDs are strkey-encoded ("C…"); both decode to 32 raw bytes.
func AddressToBytes(address string) ([]byte, error) {
	// Try account (G…) first, then contract (C…).
	if b, err := strkey.Decode(strkey.VersionByteAccountID, address); err == nil {
		return b, nil
	}
	if b, err := strkey.Decode(strkey.VersionByteContract, address); err == nil {
		return b, nil
	}
	return nil, fmt.Errorf("invalid Stellar address format: %s (expected account G… or contract C… strkey)", address)
}

// AddressConverter implements addrconv.Converter for Stellar chains.
type AddressConverter struct{}

// ConvertToBytes converts a Stellar address string to its 32-byte raw form.
func (s AddressConverter) ConvertToBytes(address string) ([]byte, error) {
	return AddressToBytes(address)
}

// Supports returns true if this converter supports the given chain family.
func (s AddressConverter) Supports(family string) bool {
	return family == chain_selectors.FamilyStellar
}
