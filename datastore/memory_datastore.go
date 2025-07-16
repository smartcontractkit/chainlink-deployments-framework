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
// It is parameterized by the type of address reference store, chain metadata, contract metadata store and
// env metadata store it uses.
type BaseDataStore[
	R AddressRefStore, CH ChainMetadataStore, CM ContractMetadataStore, EM EnvMetadataStore,
] interface {
	Addresses() R
	ChainMetadata() CH
	ContractMetadata() CM
	EnvMetadata() EM
}

// BaseDataStoreV2 is an interface that defines the basic operations for a data store using V2 interfaces.
type BaseDataStoreV2[
	R AddressRefStore, CH MutableStoreV2[ChainMetadataKey, ChainMetadata], CM ContractMetadataStore, EM MutableUnaryStoreV2[EnvMetadata],
] interface {
	Addresses() R
	ChainMetadata() CH
	ContractMetadata() CM
	EnvMetadata() EM
}

// DataStore is an interface that defines the operations for a read-only data store.
type DataStore interface {
	BaseDataStore[
		AddressRefStore, ChainMetadataStore,
		ContractMetadataStore, EnvMetadataStore,
	]
}

// MutableDataStore is an interface that defines the operations for a mutable data store.
type MutableDataStore interface {
	Merger[DataStore]
	Sealer[DataStore]

	BaseDataStore[
		MutableAddressRefStore, MutableChainMetadataStore,
		MutableContractMetadataStore, MutableEnvMetadataStore,
	]
}

type CatalogStore interface {
	BaseDataStoreV2[
		MutableAddressRefStore, MutableStoreV2[ChainMetadataKey, ChainMetadata],
		MutableContractMetadataStore, MutableUnaryStoreV2[EnvMetadata],
	]
}

// MemoryDataStore is a concrete implementation of the MutableDataStore interface.
var _ MutableDataStore = &MemoryDataStore{}

type MemoryDataStore struct {
	AddressRefStore       *MemoryAddressRefStore       `json:"addressRefStore"`
	ChainMetadataStore    *MemoryChainMetadataStore    `json:"chainMetadataStore"`
	ContractMetadataStore *MemoryContractMetadataStore `json:"contractMetadataStore"`
	EnvMetadataStore      *MemoryEnvMetadataStore      `json:"envMetadataStore"`
}

// NewMemoryDataStore creates a new instance of MemoryDataStore.
// NOTE: The instance returned is mutable and can be modified.
func NewMemoryDataStore() *MemoryDataStore {
	return &MemoryDataStore{
		AddressRefStore:       NewMemoryAddressRefStore(),
		ChainMetadataStore:    NewMemoryChainMetadataStore(),
		ContractMetadataStore: NewMemoryContractMetadataStore(),
		EnvMetadataStore:      NewMemoryEnvMetadataStore(),
	}
}

// Seal seals the MemoryDataStore, by returning a new instance of sealedMemoryDataStore.
func (s *MemoryDataStore) Seal() DataStore {
	return &sealedMemoryDataStore{
		AddressRefStore:       s.AddressRefStore,
		ChainMetadataStore:    s.ChainMetadataStore,
		ContractMetadataStore: s.ContractMetadataStore,
		EnvMetadataStore:      s.EnvMetadataStore,
	}
}

// Addresses returns the AddressRefStore of the MemoryDataStore.
func (s *MemoryDataStore) Addresses() MutableAddressRefStore {
	return s.AddressRefStore
}

// ChainMetadata returns the ChainMetadataStore of the MemoryDataStore.
func (s *MemoryDataStore) ChainMetadata() MutableChainMetadataStore {
	return s.ChainMetadataStore
}

// ContractMetadata returns the ContractMetadataStore of the MemoryDataStore.
func (s *MemoryDataStore) ContractMetadata() MutableContractMetadataStore {
	return s.ContractMetadataStore
}

// EnvMetadata returns the EnvMetadataStore of the MutableEnvMetadataStore.
func (s *MemoryDataStore) EnvMetadata() MutableEnvMetadataStore {
	return s.EnvMetadataStore
}

// Merge merges the given mutable data store into the current MemoryDataStore.
func (s *MemoryDataStore) Merge(other DataStore) error {
	addressRefs, err := other.Addresses().Fetch()
	if err != nil {
		return err
	}

	for _, addressRef := range addressRefs {
		if err = s.AddressRefStore.Upsert(addressRef); err != nil {
			return err
		}
	}

	chainMetadataRecords, err := other.ChainMetadata().Fetch()
	if err != nil {
		return err
	}

	for _, record := range chainMetadataRecords {
		if err = s.ChainMetadataStore.Upsert(record); err != nil {
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
var _ DataStore = &sealedMemoryDataStore{}

type sealedMemoryDataStore struct {
	AddressRefStore       *MemoryAddressRefStore       `json:"addressRefStore"`
	ChainMetadataStore    *MemoryChainMetadataStore    `json:"chainMetadataStore"`
	ContractMetadataStore *MemoryContractMetadataStore `json:"contractMetadataStore"`
	EnvMetadataStore      *MemoryEnvMetadataStore      `json:"envMetadataStore"`
}

// Addresses returns the AddressRefStore of the sealedMemoryDataStore.
// It implements the BaseDataStore interface.
func (s *sealedMemoryDataStore) Addresses() AddressRefStore {
	return s.AddressRefStore
}

func (s *sealedMemoryDataStore) ChainMetadata() ChainMetadataStore {
	return s.ChainMetadataStore
}

// ContractMetadata returns the ContractMetadataStore of the sealedMemoryDataStore.
func (s *sealedMemoryDataStore) ContractMetadata() ContractMetadataStore {
	return s.ContractMetadataStore
}

// EnvMetadata returns the EnvMetadataStore of the sealedMemoryDataStore.
func (s *sealedMemoryDataStore) EnvMetadata() EnvMetadataStore {
	return s.EnvMetadataStore
}
