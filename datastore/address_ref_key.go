package datastore

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// AddressRefKey is an interface that represents a key for AddressRef records.
// It is used to uniquely identify a record in the AddressRefStore.
type AddressRefKey interface {
	Comparable[AddressRefKey]
	fmt.Stringer

	// ChainSelector returns the chain-selector selector of the chain where the contract is deployed.
	ChainSelector() uint64
	// Type returns the contract type of the contract.
	// This is a simple string type for identifying contract
	Type() ContractType
	// Version returns the semantic version of the contract.
	Version() *semver.Version
	// Qualifier returns the optional qualifier for the contract.
	// This can be used to differentiate between different references of the same contract.
	Qualifier() string
}

// addressRefKey implements the AddressRefKey interface.
var _ AddressRefKey = addressRefKey{}

// addressRefKey is a struct that implements the AddressRefKey interface.
// It is used to uniquely identify a record in the AddressRefStore.
type addressRefKey struct {
	chainSelector uint64
	contractType  ContractType
	version       *semver.Version
	qualifier     string
}

// CHainSelector returns the chain-selector selector of the chain where the contract is deployed.
func (a addressRefKey) ChainSelector() uint64 { return a.chainSelector }

// Type returns the contract type of the contract.
// This is a simple string type for identifying contract
func (a addressRefKey) Type() ContractType { return a.contractType }

// Version returns the semantic version of the contract.
func (a addressRefKey) Version() *semver.Version { return a.version }

// Qualifier returns the optional qualifier for the contract.
func (a addressRefKey) Qualifier() string { return a.qualifier }

// Equals returns true if the two AddressRefKey instances are equal, false otherwise.
func (a addressRefKey) Equals(other AddressRefKey) bool {
	return a.chainSelector == other.ChainSelector() &&
		a.contractType == other.Type() &&
		a.version.Equal(other.Version()) &&
		a.qualifier == other.Qualifier()
}

// String returns a string representation of the addressRefKey.
func (a addressRefKey) String() string {
	return fmt.Sprintf("%d_%s_%s_%s",
		a.chainSelector,
		a.contractType,
		a.version.String(),
		a.qualifier,
	)
}

// NewAddressRefKeyFromString creates a new AddressRefKey instance from a string representation.
func NewAddressRefKeyFromString(key string) (AddressRefKey, error) {
	parts := strings.Split(key, "_")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid address ref key: %s", key)
	}
	chainSelector, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chain selector: %w", err)
	}

	return NewAddressRefKey(chainSelector, ContractType(parts[1]), semver.MustParse(parts[2]), parts[3]), nil
}

// NewAddressRefKey creates a new AddressRefKey instance.
func NewAddressRefKey(chainSelector uint64, contractType ContractType, version *semver.Version, qualifier string) AddressRefKey {
	return addressRefKey{
		chainSelector: chainSelector,
		contractType:  contractType,
		version:       version,
		qualifier:     qualifier,
	}
}
