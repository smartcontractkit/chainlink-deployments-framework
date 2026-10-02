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
	// strkey.Decode validates only the version byte and checksum, not the payload
	// length, so a syntactically valid strkey with a non-32-byte payload would
	// otherwise be accepted with the wrong length. Both account (G…) and contract
	// (C…) keys are exactly 32 bytes; enforce that invariant before returning.
	if b, err := strkey.Decode(strkey.VersionByteAccountID, address); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := strkey.Decode(strkey.VersionByteContract, address); err == nil && len(b) == 32 {
		return b, nil
	}

	return nil, fmt.Errorf("invalid Stellar address format: %s (expected 32-byte account G… or contract C… strkey)", address)
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
