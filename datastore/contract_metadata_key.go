package datastore

import (
	"fmt"
	"strconv"
	"strings"
)

// ContractMetadataKey is an interface that represents a key for ContractMetadata records.
// It is used to uniquely identify a record in the ContractMetadataStore.
type ContractMetadataKey interface {
	Comparable[ContractMetadataKey]
	fmt.Stringer

	// Address returns the address of the contract on the chain.
	Address() string
	// ChainSelector returns the chain-selector of the chain where the contract is deployed.
	ChainSelector() uint64
}

// contractMetadataKey implements the ContractMetadataKey interface.
var _ ContractMetadataKey = contractMetadataKey{}

// contractMetadataKey is a struct that implements the ContractMetadataKey interface.
// It is used to uniquely identify a record in the ContractMetadataStore.
type contractMetadataKey struct {
	chainSelector uint64
	address       string
}

// ChainSelector returns the chain-selector of the chain where the contract is deployed.
func (c contractMetadataKey) ChainSelector() uint64 { return c.chainSelector }

// Address returns the address of the contract on the chain.
func (c contractMetadataKey) Address() string { return c.address }

// Equals returns true if the two ContractMetadataKey instances are equal, false otherwise.
func (c contractMetadataKey) Equals(other ContractMetadataKey) bool {
	return c.chainSelector == other.ChainSelector() &&
		c.address == other.Address()
}

// String returns a string representation of the ContractMetadataKey.
func (c contractMetadataKey) String() string {
	return fmt.Sprintf("%d_%s", c.chainSelector, c.address)
}

// NewContractMetadataKeyFromString creates a new ContractMetadataKey instance from a string representation.
func NewContractMetadataKeyFromString(key string) (ContractMetadataKey, error) {
	parts := strings.Split(key, "_")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid contract metadata key: %s", key)
	}
	chainSelector, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chain selector: %w", err)
	}

	return NewContractMetadataKey(chainSelector, parts[1]), nil
}

// NewContractMetadataKey creates a new ContractMetadataKey instance.
func NewContractMetadataKey(chainSelector uint64, address string) ContractMetadataKey {
	return contractMetadataKey{
		chainSelector: chainSelector,
		address:       address,
	}
}
