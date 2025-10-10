package memory

import (
	"context"
	"errors"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// transactionKey is a custom type for context keys to avoid collisions
type transactionKey struct{}

var _ datastore.CatalogStore = &memoryCatalogDataStore{}

type memoryCatalogDataStore struct {
	storage               *memoryStorage
	addressReferenceStore *memoryAddressRefStore
	chainMetadataStore    *memoryChainMetadataStore
	contractMetadataStore *memoryContractMetadataStore
	envMetadataStore      *memoryEnvMetadataStore
}

// NewMemoryCatalogDataStore creates an in-memory version of the catalog datastore.
// This implementation does not store data persistently.
// A new call to this function will create an entirely separate and new in-memory store, so changes will not be
// persisted.
//
// This version is not threadsafe and could result in races when using transactions from multiple
// threads.
func NewMemoryCatalogDataStore() (*memoryCatalogDataStore, error) {
	storage := newMemoryStorage()

	addressRefStore := newCatalogAddressRefStore(storage)
	chainMetadataStore := newCatalogChainMetadataStore(storage)
	contractMetadataStore := newCatalogContractMetadataStore(storage)
	envMetadataStore := newCatalogEnvMetadataStore(storage)

	return &memoryCatalogDataStore{
		storage:               storage,
		addressReferenceStore: addressRefStore,
		chainMetadataStore:    chainMetadataStore,
		contractMetadataStore: contractMetadataStore,
		envMetadataStore:      envMetadataStore,
	}, nil
}

// WithTransaction wraps the provided function in a transaction.
func (m memoryCatalogDataStore) WithTransaction(ctx context.Context, fn datastore.TransactionLogic) (err error) {
	tx := m.storage.beginTransaction()

	// Create a new context with the transaction
	txCtx := context.WithValue(ctx, transactionKey{}, tx)

	var txerr error
	defer func() {
		if r := recover(); r != nil {
			// rollback before re-panicking
			_ = m.storage.rollbackTransaction(tx)
			panic(r)
		} else if txerr != nil {
			// non panic error from the transaction logic itself
			err = errors.Join(err, m.storage.rollbackTransaction(tx))
		} else {
			// everything went fine
			err = m.storage.commitTransaction(tx)
		}
	}()

	txerr = fn(txCtx, m)

	return txerr
}

// Addresses returns the address reference store.
func (m memoryCatalogDataStore) Addresses() datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] {
	return m.addressReferenceStore
}

// ChainMetadata returns the chain metadata store.
func (m memoryCatalogDataStore) ChainMetadata() datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] {
	return m.chainMetadataStore
}

// ContractMetadata returns the contract metadata store.
func (m memoryCatalogDataStore) ContractMetadata() datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] {
	return m.contractMetadataStore
}

// EnvMetadata returns the environment metadata store.
func (m memoryCatalogDataStore) EnvMetadata() datastore.MutableUnaryStoreV2[datastore.EnvMetadata] {
	return m.envMetadataStore
}
