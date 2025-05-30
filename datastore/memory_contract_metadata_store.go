package datastore

import (
	"sync"
)

// ContractMetadataStore is an interface that represents an immutable view over a set
// of ContractMetadata records identified by ContractMetadataKey.
type ContractMetadataStore interface {
	Store[ContractMetadataKey, ContractMetadata]
}

// MutableContractMetadataStore is an interface that represents a mutable ContractMetadataStore
// of ContractMetadata records identified by ContractMetadataKey.
type MutableContractMetadataStore interface {
	MutableStore[ContractMetadataKey, ContractMetadata]
}

// MemoryContractMetadataStore is an in-memory implementation of the ContractMetadataStore and
// MutableContractMetadataStore interfaces.
type MemoryContractMetadataStore struct {
	mu      sync.RWMutex
	Records []ContractMetadata `json:"records"`
}

// MemoryContractMetadataStore implements ContractMetadataStore interface.
var _ ContractMetadataStore = &MemoryContractMetadataStore{}

// MemoryContractMetadataStore implements MutableContractMetadataStore interface.
var _ MutableContractMetadataStore = &MemoryContractMetadataStore{}

// NewMemoryContractMetadataStore creates a new MemoryContractMetadataStore instance.
// It is a generic function that takes a type parameter M which must implement the Cloneable interface.
func NewMemoryContractMetadataStore() *MemoryContractMetadataStore {
	return &MemoryContractMetadataStore{Records: []ContractMetadata{}}
}

// Get returns the ContractMetadata for the provided key, or an error if no such record exists.
func (s *MemoryContractMetadataStore) Get(key ContractMetadataKey) (ContractMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx := s.indexOf(key)
	if idx == -1 {
		return ContractMetadata{}, ErrContractMetadataNotFound
	}

	return s.Records[idx].Clone()
}

// Fetch returns a copy of all ContractMetadata in the store.
func (s *MemoryContractMetadataStore) Fetch() ([]ContractMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := []ContractMetadata{}
	for _, record := range s.Records {
		record, err := record.Clone()
		if err != nil {
			return []ContractMetadata{}, err
		}

		records = append(records, record)
	}

	return records, nil
}

// Filter returns a copy of all ContractMetadata in the store that pass all of the provided filters.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *MemoryContractMetadataStore) Filter(filters ...FilterFunc[ContractMetadataKey, ContractMetadata]) []ContractMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := append([]ContractMetadata{}, s.Records...)
	for _, filter := range filters {
		records = filter(records)
	}

	return records
}

// indexOf returns the index of the record with the provided key, or -1 if no such record exists.
func (s *MemoryContractMetadataStore) indexOf(key ContractMetadataKey) int {
	for i, record := range s.Records {
		if record.Key().Equals(key) {
			return i
		}
	}

	return -1
}

// Add inserts a new record into the store.
// If a record with the same key already exists, an error is returned.
func (s *MemoryContractMetadataStore) Add(record ContractMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(record.Key())
	if idx != -1 {
		return ErrContractMetadataExists
	}
	s.Records = append(s.Records, record)

	return nil
}

// Upsert inserts a new record into the store if no record with the same key already exists.
// If a record with the same key already exists, it is updated.
func (s *MemoryContractMetadataStore) Upsert(record ContractMetadata) error {
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

// Update edits an existing record whose fields match the primary key elements of the supplied ContractMetadata, with
// the non-primary-key values of the supplied ContractMetadata.
// If no such record exists, an error is returned.
func (s *MemoryContractMetadataStore) Update(record ContractMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(record.Key())
	if idx == -1 {
		return ErrContractMetadataNotFound
	}
	s.Records[idx] = record

	return nil
}

// Delete deletes an existing record whose primary key elements match the supplied ContractMetadata, returning an error if no
// such record exists.
func (s *MemoryContractMetadataStore) Delete(key ContractMetadataKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(key)
	if idx == -1 {
		return ErrContractMetadataNotFound
	}
	s.Records = append(s.Records[:idx], s.Records[idx+1:]...)

	return nil
}
