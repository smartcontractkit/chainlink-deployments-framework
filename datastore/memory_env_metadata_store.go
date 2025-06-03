package datastore

import (
	"sync"
)

// EnvMetadataStore is an interface that defines the methods for a store that manages environment metadata.
type EnvMetadataStore interface {
	UnaryStore[EnvMetadata]
}

// MutableEnvMetadataStore is an interface that defines the methods for a mutable store that manages environment metadata.
type MutableEnvMetadataStore interface {
	MutableUnaryStore[EnvMetadata]
}

// MemoryEnvMetadataStore is a concrete implementation of the EnvMetadataStore interface.
type MemoryEnvMetadataStore struct {
	mu     sync.RWMutex
	Record *EnvMetadata `json:"record"`
}

// MemoryEnvMetadataStore implements EnvMetadataStore interface.
var _ EnvMetadataStore = &MemoryEnvMetadataStore{}

// MemoryEnvMetadataStore implements MutableEnvMetadataStore interface.
var _ MutableEnvMetadataStore = &MemoryEnvMetadataStore{}

// NewMemoryEnvMetadataStore creates a new MemoryEnvMetadataStore instance.
func NewMemoryEnvMetadataStore() *MemoryEnvMetadataStore {
	return &MemoryEnvMetadataStore{Record: nil}
}

// Get returns a copy of the stored EnvMetadata record if it exists or an error if any occurred.
// If no record exist, it returns an empty EnvMetadata and ErrEnvMetadataNotSet.
// If the record exists, it returns a copy of the record and a nil error.
func (s *MemoryEnvMetadataStore) Get() (EnvMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Record == nil {
		return EnvMetadata{}, ErrEnvMetadataNotSet
	}

	return s.Record.Clone()
}

// Set sets the EnvMetadata record in the store. If the record already exists, it will be replaced.
func (s *MemoryEnvMetadataStore) Set(record EnvMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Record = &record

	return nil
}
