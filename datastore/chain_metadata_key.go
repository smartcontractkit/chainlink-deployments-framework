package datastore

import (
	"fmt"
	"strconv"
)

// ChainMetadataKey is an interface that represents a key for ChainMetadata records.
// It is used to uniquely identify a record in the ChainMetadataStore.
type ChainMetadataKey interface {
	Comparable[ChainMetadataKey]
	fmt.Stringer

	// ChainSelector returns the chain-selector of the chain associated with the metadata.
	ChainSelector() uint64
}

// contractMetadataKey implements the ChainMetadataKey interface.
var _ ChainMetadataKey = chainMetadataKey{}

// chainMetadataKey is a struct that implements the ChainMetadataKey interface.
// It is used to uniquely identify a record in the ChainMetadataStore.
type chainMetadataKey struct {
	chainSelector uint64
}

// ChainSelector returns the chain-selector of the chain associated with the metadata.
func (c chainMetadataKey) ChainSelector() uint64 { return c.chainSelector }

// Equals returns true if the two ChainMetadataKey instances are equal, false otherwise.
func (c chainMetadataKey) Equals(other ChainMetadataKey) bool {
	return c.chainSelector == other.ChainSelector()
}

// String returns a string representation of the ChainMetadataKey.
func (c chainMetadataKey) String() string {
	return strconv.FormatUint(c.chainSelector, 10)
}

// NewChainMetadataKeyFromString creates a new ChainMetadataKey instance from a string representation.
func NewChainMetadataKeyFromString(key string) (ChainMetadataKey, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("invalid chain metadata key: %s", key)
	}
	chainSelector, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chain selector: %w", err)
	}

	return NewChainMetadataKey(chainSelector), nil
}

// NewChainMetadataKey creates a new ChainMetadataKey instance.
func NewChainMetadataKey(chainSelector uint64) ChainMetadataKey {
	return chainMetadataKey{
		chainSelector: chainSelector,
	}
}
