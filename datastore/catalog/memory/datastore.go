package memory

import (
	"context"
	"errors"
	"fmt"

	"github.com/rubenv/pgtest"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	_ "github.com/proullon/ramsql/driver"
)

var _ datastore.CatalogStore = &memoryDataStore{}

type memoryDataStore struct {
	config                MemoryDataStoreConfig
	db                    *dbController
	pg                    *pgtest.PG
	addressReferenceStore *memoryAddressRefStore
	chainMetadataStore    *memoryChainMetadataStore
	contractMetadataStore *memoryContractMetadataStore
	envMetadataStore      *memoryEnvMetadataStore
}

type MemoryDataStoreConfig struct {
	Domain      string
	Environment string
}

// NewMemoryDataStore creates an in-memory version of the catalog datastore.
// This implementation does not store data persistently, and any fixture must be provided to it at the start.
// A new call to this function will create an entirely separate and new in-memory store, so changes will not be
// persisted.
//
// # You should call `store.Close()` between usages, unless you need to refer to shared test state
//
// This version is not threadsafe and could result in races when using transactions from multiple
// threads.
func NewMemoryDataStore(config MemoryDataStoreConfig) (*memoryDataStore, error) {
	pgcfg := pgtest.New()
	pg, err := pgcfg.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres: %w", err)
	}

	ctrl := newDbController(pg.DB)
	ctx := context.Background()
	if err = ctrl.Fixture(ctx, sCHEMA_ADDRESS_REFERENCES); err != nil {
		_ = pg.Stop()
		return nil, fmt.Errorf("failed to create address references schema: %w", err)
	}
	if err = ctrl.Fixture(ctx, sCHEMA_CONTRACT_METADATA); err != nil {
		_ = pg.Stop()
		return nil, fmt.Errorf("failed to create contract metadata schema: %w", err)
	}
	if err = ctrl.Fixture(ctx, sCHEMA_CHAIN_METADATA); err != nil {
		_ = pg.Stop()
		return nil, fmt.Errorf("failed to create chain metadata schema: %w", err)
	}
	if err = ctrl.Fixture(ctx, sCHEMA_ENVIRONMENT_METADATA); err != nil {
		_ = pg.Stop()
		return nil, fmt.Errorf("failed to create environment metadata schema: %w", err)
	}

	addressRefStore := newCatalogAddressRefStore(config, ctrl)
	chainMetadataStore := newCatalogChainMetadataStore(config, ctrl)
	contractMetadataStore := newCatalogContractMetadataStore(config, ctrl)
	envMetadataStore := newCatalogEnvMetadataStore(config, ctrl)

	return &memoryDataStore{
		config:                config,
		db:                    ctrl,
		pg:                    pg,
		addressReferenceStore: addressRefStore,
		chainMetadataStore:    chainMetadataStore,
		contractMetadataStore: contractMetadataStore,
		envMetadataStore:      envMetadataStore,
	}, nil
}

// Close shuts down the in-process postgress instance.
func (m *memoryDataStore) Close() error {
	return m.pg.Stop()
}

func (m memoryDataStore) WithTransaction(ctx context.Context, fn datastore.TransactionLogic) (err error) {
	err = m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	var txerr error
	defer func() {
		if r := recover(); r != nil {
			// rollback before re-panicking
			_ = m.db.Rollback()
			panic(r)
		} else if txerr != nil {
			// non panic error from the transaction logic itself
			err = errors.Join(err, m.db.Rollback())
		} else {
			// everything went fine
			err = m.db.Commit()
		}
	}()

	txerr = fn(ctx, m)

	return txerr
}

func (m memoryDataStore) Addresses() datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] {
	return m.addressReferenceStore
}

func (m memoryDataStore) ChainMetadata() datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] {
	return m.chainMetadataStore
}

func (m memoryDataStore) ContractMetadata() datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] {
	return m.contractMetadataStore
}

func (m memoryDataStore) EnvMetadata() datastore.MutableUnaryStoreV2[datastore.EnvMetadata] {
	return m.envMetadataStore
}
