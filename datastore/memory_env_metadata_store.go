package datastore

import (
	"errors"
	"sync"
)

var ErrEnvMetadataNotSet = errors.New("no environment metadata set")

// MemoryEnvMetadataStore implements EnvMetadataStore interface.
var _ EnvMetadataStore = &MemoryEnvMetadataStore{}

// MemoryEnvMetadataStore implements MutableEnvMetadataStore interface.
var _ MutableEnvMetadataStore = &MemoryEnvMetadataStore{}

// MemoryEnvMetadataStore is a concrete implementation of the EnvMetadataStore interface.
type MemoryEnvMetadataStore struct {
	mu     sync.RWMutex
	Record CustomMetadata `json:"record"`
}

// NewMemoryEnvMetadataStore creates a new MemoryEnvMetadataStore instance.
func NewMemoryEnvMetadataStore() *MemoryEnvMetadataStore {
	return &MemoryEnvMetadataStore{}
}

// Get returns a copy of the stored EnvMetadata record if it exists or an error if any occurred.
// If no record exist, it returns an empty EnvMetadata and ErrEnvMetadataNotSet.
// If the record exists, it returns a copy of the record and a nil error.
func (s *MemoryEnvMetadataStore) Get() (CustomMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Record == nil {
		return nil, ErrEnvMetadataNotSet
	}

	return s.Record.Clone(), nil
}

// Set sets the EnvMetadata record in the store. If the record already exists, it will be replaced.
func (s *MemoryEnvMetadataStore) Set(record CustomMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Record = record

	return nil
}
