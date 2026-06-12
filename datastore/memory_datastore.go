package datastore

import (
	"errors"
	"fmt"
)

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

// WriteMetadata writes address refs and upserts contract and chain metadata and sets env metadata.
// Address refs use Add by default; pass WithUpsertAddressRefs to insert or replace by key.
func (s *MemoryDataStore) WriteMetadata(bundle MetadataBundle, opts ...WriteMetadataOption) error {
	return WriteMetadataToDataStore(s, bundle, opts...)
}

// Merge applies records and staged deletions from other onto this MemoryDataStore.
//
// Staged deletions (other.<Store>.DeletedRemoteKeys) are propagated by appending the
// key to s.<Store>.DeletedRemoteKeys via RemoteDelete and then removing the record
// from s.<Store>.Records via Delete (tolerating the per-store NotFound sentinel).
// The DeletedRemoteKeys append is what lets chained operations preserve delete intent
// across intermediate Merges.
//
// NotFound on the Delete step is tolerated because the source's staged key may
// legitimately not be present in the destination's Records (RemoteDelete is allowed
// to stage deletes for records that exist only in the remote backing store), and
// because we want repeated Merges of the same source to be idempotent.
//
// Precedence: a live record in other.<Store>.Records overrides a staged delete in
// s.<Store>.DeletedRemoteKeys for the same key. This is a side-effect of Upsert,
// which clears the key from DeletedRemoteKeys whenever a record is upserted (so a
// direct `RemoteDelete(k); Upsert(rec-with-k)` cancels the staged delete on the
// same store). Staged deletes are only "sticky" on the source side. Callers that
// need a destination-side staged delete to survive Merge must ensure the source
// either omits the live record or stages the delete itself.
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
		for _, dk := range src.DeletedRemoteKeys {
			key, keyErr := NewAddressRefKeyFromString(dk)
			if keyErr != nil {
				return fmt.Errorf("failed to parse address ref key: %w", keyErr)
			}
			if err = s.AddressRefStore.RemoteDelete(key); err != nil {
				return err
			}
			if err = s.AddressRefStore.Delete(key); err != nil && !errors.Is(err, ErrAddressRefNotFound) {
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
		for _, dk := range src.DeletedRemoteKeys {
			key, keyErr := NewChainMetadataKeyFromString(dk)
			if keyErr != nil {
				return fmt.Errorf("failed to parse chain metadata key: %w", keyErr)
			}
			if err = s.ChainMetadataStore.RemoteDelete(key); err != nil {
				return err
			}
			if err = s.ChainMetadataStore.Delete(key); err != nil && !errors.Is(err, ErrChainMetadataNotFound) {
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
		for _, dk := range src.DeletedRemoteKeys {
			key, keyErr := NewContractMetadataKeyFromString(dk)
			if keyErr != nil {
				return fmt.Errorf("failed to parse contract metadata key: %w", keyErr)
			}
			if err = s.ContractMetadataStore.RemoteDelete(key); err != nil {
				return err
			}
			if err = s.ContractMetadataStore.Delete(key); err != nil && !errors.Is(err, ErrContractMetadataNotFound) {
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
