package evm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

// AddressToBytes converts an EVM address string to bytes.
// EVM addresses are hex strings (with or without 0x prefix) representing 20 bytes.
func AddressToBytes(address string) ([]byte, error) {
	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid EVM address format: %s", address)
	}

	addr := common.HexToAddress(address)

	return addr.Bytes(), nil
}

// AddressConverter implements address conversion for EVM-compatible chains.
// This struct implements the AddressConverter strategy interface.
type AddressConverter struct{}

// ConvertToBytes converts an EVM address string to bytes.
func (e AddressConverter) ConvertToBytes(address string) ([]byte, error) {
	return AddressToBytes(address)
}

// Supports returns true if this converter supports the given chain family.
func (e AddressConverter) Supports(family string) bool {
	return family == chain_selectors.FamilyEVM
}
