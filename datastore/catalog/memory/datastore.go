package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/rubenv/pgtest"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	_ "github.com/proullon/ramsql/driver"
)

var _ datastore.CatalogStore = &memoryDataStore{}

type memoryDataStore struct {
	t                     *testing.T
	config                MemoryDataStoreConfig
	db                    *dbController
	pg                    *pgtest.PG
	addressReferenceStore *memoryAddressRefStore
	chainMetadataStore    *memoryChainMetadataStore
	contractMetadataStore *memoryContractMetadataStore
}

type MemoryDataStoreConfig struct {
	Domain      string
	Environment string
}

// NewMemoryDataStore creates an in-memory version of the catalog datastore, which can be used
// in tests of changesets which require use of the catalog. This implementation does not store
// data, and any fixture must be provided to it at the start of the test. A new call to this
// function will create an entirely separate and new in-memory store, so changes will not be
// persisted.
//
// # You should call `store.Close()` between usages, unless you need to refer to shared test state
//
// This version is not threadsafe and could result in races when using transactions from multiple
// threads.
func NewMemoryDataStore(t *testing.T, config MemoryDataStoreConfig) *memoryDataStore {
	t.Helper()
	pgcfg := pgtest.New()
	pg, err := pgcfg.Start()
	require.NoError(t, err)
	ctrl := newDbController(pg.DB)

	require.NoError(t, ctrl.Fixture(sCHEMA_ADDRESS_REFERENCES))
	require.NoError(t, ctrl.Fixture(sCHEMA_CONTRACT_METADATA))
	require.NoError(t, ctrl.Fixture(sCHEMA_CHAIN_METADATA))
	require.NoError(t, ctrl.Fixture(sCHEMA_ENVIRONMENT_METADATA))
	return &memoryDataStore{
		t:                     t,
		config:                config,
		db:                    ctrl,
		pg:                    pg,
		addressReferenceStore: newCatalogAddressRefStore(t, config, ctrl),
		chainMetadataStore:    newCatalogChainMetadataStore(t, config, ctrl),
		contractMetadataStore: newCatalogContractMetadataStore(t, config, ctrl),
	}
}

// Close shuts down the in-process postgress instance.
func (m *memoryDataStore) Close() {
	require.NoError(m.t, m.pg.Stop())
}

func (m memoryDataStore) WithTransaction(ctx context.Context, fn datastore.TransactionLogic) (err error) {
	err = m.db.Begin()
	require.NoError(m.t, err)

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
	//TODO implement me
	panic("implement me")
}
