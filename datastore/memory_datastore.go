package datastore

import "errors"

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
	// Fetch address ref records from the other data store
	addressRefs, err := other.Addresses().Fetch()
	if err != nil {
		return err
	}

	// Upsert address ref records into the current data store
	for _, addressRef := range addressRefs {
		if err = s.AddressRefStore.Upsert(addressRef); err != nil {
			return err
		}
	}

	// Propagate address ref deletions
	if src, ok := other.Addresses().(*MemoryAddressRefStore); ok {
		for _, dk := range src.DeletedKeys {
			if err = s.AddressRefStore.Delete(dk); err != nil && !errors.Is(err, ErrAddressRefNotFound) {
				return err
			}
		}
	}

	chainMetadataRecords, err := other.ChainMetadata().Fetch()
	if err != nil {
		return err
	}

	// Upsert chain metadata records into the current data store
	for _, record := range chainMetadataRecords {
		if err = s.ChainMetadataStore.Upsert(record); err != nil {
			return err
		}
	}

	// Propagate chain metadata deletions
	if src, ok := other.ChainMetadata().(*MemoryChainMetadataStore); ok {
		for _, dk := range src.DeletedKeys {
			if err = s.ChainMetadataStore.Delete(dk); err != nil && !errors.Is(err, ErrChainMetadataNotFound) {
				return err
			}
		}
	}

	// Fetch contract metadata records from the other data store
	contractMetadataRecords, err := other.ContractMetadata().Fetch()
	if err != nil {
		return err
	}

	// Upsert contract metadata records into the current data store
	for _, record := range contractMetadataRecords {
		if err = s.ContractMetadataStore.Upsert(record); err != nil {
			return err
		}
	}

	// Propagate contract metadata deletions
	if src, ok := other.ContractMetadata().(*MemoryContractMetadataStore); ok {
		for _, dk := range src.DeletedKeys {
			if err = s.ContractMetadataStore.Delete(dk); err != nil && !errors.Is(err, ErrContractMetadataNotFound) {
				return err
			}
		}
	}

	// Fetch env metadata record from the other data store
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
