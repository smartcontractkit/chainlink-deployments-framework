package datastore

import "errors"

// Merger is an interface that defines a method for merging two data stores.
type Merger[T any] interface {
	// Merge merges the given data into the current data store.
	// It should return an error if the merge fails.
	Merge(other T) error
}

// Sealer is an interface that defines a method for sealing a data store.
// A sealed data store cannot be modified further.
type Sealer[T any] interface {
	// Seal seals the data store, preventing further modifications.
	Seal() T
}

// BaseDataStore is an interface that defines the basic operations for a data store.
// It is parameterized by the type of address reference store and contract metadata store it uses.
type BaseDataStore[
	U any,
	R AddressRefStore, CM ContractMetadataStore, EM EnvMetadataStore[U],
] interface {
	Addresses() R
	ContractMetadata() CM
	EnvMetadata() EM
}

// DataStore is an interface that defines the operations for a read-only data store.
type DataStore[U any] interface {
	BaseDataStore[U, AddressRefStore, ContractMetadataStore, EnvMetadataStore[U]]
}

// MutableDataStore is an interface that defines the operations for a mutable data store.
type MutableDataStore[U any] interface {
	Merger[DataStore[U]]
	Sealer[DataStore[U]]

	BaseDataStore[U, MutableAddressRefStore, MutableContractMetadataStore, MutableEnvMetadataStore[U]]
}

// MemoryDataStore is a concrete implementation of the MutableDataStore interface.
var _ MutableDataStore[DefaultMetadata] = &MemoryDataStore[DefaultMetadata]{}

type MemoryDataStore[EM any] struct {
	AddressRefStore       *MemoryAddressRefStore       `json:"addressRefStore"`
	ContractMetadataStore *MemoryContractMetadataStore `json:"contractMetadataStore"`
	EnvMetadataStore      *MemoryEnvMetadataStore[EM]  `json:"envMetadataStore"`
}

// NewMemoryDataStore creates a new instance of MemoryDataStore.
// NOTE: The instance returned is mutable and can be modified.
func NewMemoryDataStore[EM any]() *MemoryDataStore[EM] {
	return &MemoryDataStore[EM]{
		AddressRefStore:       NewMemoryAddressRefStore(),
		ContractMetadataStore: NewMemoryContractMetadataStore(),
		EnvMetadataStore:      NewMemoryEnvMetadataStore[EM](),
	}
}

// Seal seals the MemoryDataStore, by returning a new instance of sealedMemoryDataStore.
func (s *MemoryDataStore[EM]) Seal() DataStore[EM] {
	return &sealedMemoryDataStore[EM]{
		AddressRefStore:       s.AddressRefStore,
		ContractMetadataStore: s.ContractMetadataStore,
		EnvMetadataStore:      s.EnvMetadataStore,
	}
}

// Addresses returns the AddressRefStore of the MemoryDataStore.
func (s *MemoryDataStore[EM]) Addresses() MutableAddressRefStore {
	return s.AddressRefStore
}

// ContractMetadata returns the ContractMetadataStore of the MemoryDataStore.
func (s *MemoryDataStore[EM]) ContractMetadata() MutableContractMetadataStore {
	return s.ContractMetadataStore
}

// EnvMetadata returns the EnvMetadataStore of the MutableEnvMetadataStore.
func (s *MemoryDataStore[EM]) EnvMetadata() MutableEnvMetadataStore[EM] {
	return s.EnvMetadataStore
}

// Merge merges the given mutable data store into the current MemoryDataStore.
func (s *MemoryDataStore[EM]) Merge(other DataStore[EM]) error {
	addressRefs, err := other.Addresses().Fetch()
	if err != nil {
		return err
	}

	for _, addressRef := range addressRefs {
		if err = s.AddressRefStore.Upsert(addressRef); err != nil {
			return err
		}
	}

	contractMetadataRecords, err := other.ContractMetadata().Fetch()
	if err != nil {
		return err
	}

	for _, record := range contractMetadataRecords {
		if err = s.ContractMetadataStore.Upsert(record); err != nil {
			return err
		}
	}

	envMetadata, err := other.EnvMetadata().Get()
	if err != nil {
		if errors.Is(err, ErrEnvMetadataNotSet) {
			// If the env metadata was not set in `other` data store, Get() will return
			// ErrEnvMetadataNotSet. In this case, we don't need to do anything because
			// since `other` doesn't contain any update to the env metadata, we can just
			// skip the env metadata update.
			return nil
		}

		return err
	}
	// If the env metadata was set, we need to update it in the current
	// data store.
	err = s.EnvMetadataStore.Set(envMetadata)
	if err != nil {
		return err
	}

	return nil
}

// SealedMemoryDataStore is a concrete implementation of the DataStore interface.
// It represents a sealed data store that cannot be modified further.
var _ DataStore[DefaultMetadata] = &sealedMemoryDataStore[DefaultMetadata]{}

type sealedMemoryDataStore[EM any] struct {
	AddressRefStore       *MemoryAddressRefStore       `json:"addressRefStore"`
	ContractMetadataStore *MemoryContractMetadataStore `json:"contractMetadataStore"`
	EnvMetadataStore      *MemoryEnvMetadataStore[EM]  `json:"envMetadataStore"`
}

// Addresses returns the AddressRefStore of the sealedMemoryDataStore.
// It implements the BaseDataStore interface.
func (s *sealedMemoryDataStore[EM]) Addresses() AddressRefStore {
	return s.AddressRefStore
}

// ContractMetadata returns the ContractMetadataStore of the sealedMemoryDataStore.
func (s *sealedMemoryDataStore[EM]) ContractMetadata() ContractMetadataStore {
	return s.ContractMetadataStore
}

// EnvMetadata returns the EnvMetadataStore of the sealedMemoryDataStore.
func (s *sealedMemoryDataStore[EM]) EnvMetadata() EnvMetadataStore[EM] {
	return s.EnvMetadataStore
}
