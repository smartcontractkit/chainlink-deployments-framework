package datastore

import (
	"sync"
)

// ChainMetadataStore is an interface that represents an immutable view over a set
// of ChainMetadata records identified by ChainMetadataKey.
type ChainMetadataStore interface {
	Store[ChainMetadataKey, ChainMetadata]
}

// MutableChainMetadataStore is an interface that represents a mutable ChainMetadataStore
// of ChainMetadata records identified by ChainMetadataKey.
type MutableChainMetadataStore interface {
	MutableStore[ChainMetadataKey, ChainMetadata]
}

// MemoryChainMetadataStore is an in-memory implementation of the ChainMetadataStore and
// MutableChainMetadataStore interfaces.
type MemoryChainMetadataStore struct {
	mu      sync.RWMutex
	Records []ChainMetadata `json:"records"`
}

// MemoryChainMetadataStore implements ChainMetadataStore interface.
var _ ChainMetadataStore = &MemoryChainMetadataStore{}

// MemoryChainMetadataStore implements MutableChainMetadataStore interface.
var _ MutableChainMetadataStore = &MemoryChainMetadataStore{}

// NewMemoryChainMetadataStore creates a new MemoryChainMetadataStore instance.
func NewMemoryChainMetadataStore() *MemoryChainMetadataStore {
	return &MemoryChainMetadataStore{Records: []ChainMetadata{}}
}

// Get returns the ChainMetadata for the provided key, or an error if no such record exists.
// NOTE: The returned ChainMetadata will have an any type for the Metadata field.
// To convert it to a specific type, use the utility method As.
func (s *MemoryChainMetadataStore) Get(key ChainMetadataKey) (ChainMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx := s.indexOf(key)
	if idx == -1 {
		return ChainMetadata{}, ErrChainMetadataNotFound
	}

	return s.Records[idx].Clone()
}

// Fetch returns a copy of all ChainMetadata in the store.
// NOTE: The returned ChainMetadata will have an any type for the Metadata field.
// To convert it to a specific type, use the utility method As.
func (s *MemoryChainMetadataStore) Fetch() ([]ChainMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := []ChainMetadata{}
	for _, record := range s.Records {
		clone, err := record.Clone()
		if err != nil {
			return nil, err
		}
		records = append(records, clone)
	}

	return records, nil
}

// Filter returns a copy of all ChainMetadata in the store that pass all of the provided filters.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
// NOTE: The returned ChainMetadata will have an any type for the Metadata field.
// To convert it to a specific type, use the utility method As.
func (s *MemoryChainMetadataStore) Filter(filters ...FilterFunc[ChainMetadataKey, ChainMetadata]) []ChainMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := append([]ChainMetadata{}, s.Records...)
	for _, filter := range filters {
		records = filter(records)
	}

	return records
}

// indexOf returns the index of the record with the provided key, or -1 if no such record exists.
func (s *MemoryChainMetadataStore) indexOf(key ChainMetadataKey) int {
	for i, record := range s.Records {
		if record.Key().Equals(key) {
			return i
		}
	}

	return -1
}

// Add inserts a new record into the store.
// If a record with the same key already exists, an error is returned.
func (s *MemoryChainMetadataStore) Add(record ChainMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(record.Key())
	if idx != -1 {
		return ErrChainMetadataExists
	}
	s.Records = append(s.Records, record)

	return nil
}

// Upsert inserts a new record into the store if no record with the same key already exists.
// If a record with the same key already exists, it is updated.
func (s *MemoryChainMetadataStore) Upsert(record ChainMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(record.Key())
	if idx == -1 {
		s.Records = append(s.Records, record)
		return nil
	}
	s.Records[idx] = record

	return nil
}

// Update edits an existing record whose fields match the primary key elements of the supplied ChainMetadata, with
// the non-primary-key values of the supplied ChainMetadata.
// If no such record exists, an error is returned.
func (s *MemoryChainMetadataStore) Update(record ChainMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(record.Key())
	if idx == -1 {
		return ErrChainMetadataNotFound
	}
	s.Records[idx] = record

	return nil
}

// Delete deletes an existing record whose primary key elements match the supplied ChainMetadata, returning an error if no
// such record exists.
func (s *MemoryChainMetadataStore) Delete(key ChainMetadataKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(key)
	if idx == -1 {
		return ErrChainMetadataNotFound
	}
	s.Records = append(s.Records[:idx], s.Records[idx+1:]...)

	return nil
}
