package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

const (
	query_CONTRACT_METADATA_BY_ID = `
		SELECT chain_selector, address, metadata FROM contract_metadata
		WHERE chain_selector = $1 AND address = $2`
	query_ALL_CONTRACT_METADATA = `
		SELECT chain_selector, address, metadata FROM contract_metadata`
	query_ADD_CONTRACT_METADATA = `
		INSERT INTO contract_metadata (chain_selector, address, metadata)
		VALUES ($1, $2, $3)`
	query_UPSERT_CONTRACT_METADATA = query_ADD_CONTRACT_METADATA + `
		ON CONFLICT ON CONSTRAINT contract_metadata_pkey
			DO UPDATE SET metadata = excluded.metadata`
	query_UPDATE_CONTRACT_METADATA = `
		UPDATE contract_metadata SET metadata = $3
		WHERE chain_selector = $1 AND address = $2`
	query_DELETE_CONTRACT_METADATA = `
		DELETE FROM contract_metadata
		WHERE chain_selector = $1 AND address = $2`
)

type memoryContractMetadataStore struct {
	t      *testing.T
	config MemoryDataStoreConfig
	db     *dbController
}

// Ensure memoryContractMetadataStore implements the V2 interface
var _ datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] = &memoryContractMetadataStore{}

func newCatalogContractMetadataStore(t *testing.T, config MemoryDataStoreConfig, db *dbController) *memoryContractMetadataStore {
	t.Helper()

	return &memoryContractMetadataStore{
		t:      t,
		config: config,
		db:     db,
	}
}

func (s *memoryContractMetadataStore) Get(_ context.Context, key datastore.ContractMetadataKey, options ...datastore.GetOption) (datastore.ContractMetadata, error) {
	ignoreTransactions := false
	for _, option := range options {
		switch option {
		case datastore.IgnoreTransactionsGetOption:
			ignoreTransactions = true
		}
	}
	var db DB
	if ignoreTransactions {
		db = s.db.base
	} else {
		db = s.db
	}

	rows, err := db.Query(query_CONTRACT_METADATA_BY_ID, key.ChainSelector(), key.Address())
	defer func(rows *sql.Rows) {
		if rows != nil {
			_ = rows.Close()
		}
	}(rows)
	if err != nil {
		return datastore.ContractMetadata{}, err
	}

	count := 0
	row := &datastore.ContractMetadata{}
	for rows.Next() {
		count++
		var metadataJSON sql.NullString
		err = rows.Scan(&row.ChainSelector, &row.Address, &metadataJSON)
		if err != nil {
			return datastore.ContractMetadata{}, err
		}

		// Parse metadata JSON if present
		if metadataJSON.Valid && metadataJSON.String != "" {
			var metadata any
			if unmarshalErr := json.Unmarshal([]byte(metadataJSON.String), &metadata); unmarshalErr != nil {
				return datastore.ContractMetadata{}, fmt.Errorf("failed to unmarshal metadata JSON: %w", unmarshalErr)
			}
			row.Metadata = metadata
		}
	}

	switch count {
	case 0:
		return *row, datastore.ErrContractMetadataNotFound
	case 1:
		return *row, nil
	default:
		err = fmt.Errorf("expected a single row, got %d", count)
		return *row, err
	}
}

// Fetch returns a copy of all ContractMetadata in the catalog.
func (s *memoryContractMetadataStore) Fetch(_ context.Context) ([]datastore.ContractMetadata, error) {
	rows, err := s.db.Query(query_ALL_CONTRACT_METADATA)
	defer func(rows *sql.Rows) {
		if rows != nil {
			_ = rows.Close()
		}
	}(rows)

	if err != nil {
		return nil, err
	}
	var records []datastore.ContractMetadata

	for rows.Next() {
		row := &datastore.ContractMetadata{}
		var metadataJSON sql.NullString
		err = rows.Scan(&row.ChainSelector, &row.Address, &metadataJSON)
		if err != nil {
			return records, err
		}

		// Parse metadata JSON if present
		if metadataJSON.Valid && metadataJSON.String != "" {
			var metadata any
			if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
				return records, fmt.Errorf("failed to unmarshal metadata JSON: %w", err)
			}
			row.Metadata = metadata
		}

		records = append(records, *row)
	}

	return records, nil
}

// Filter returns a copy of all ContractMetadata in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *memoryContractMetadataStore) Filter(
	ctx context.Context,
	filters ...datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata],
) ([]datastore.ContractMetadata, error) {
	// First, fetch all records from the catalog
	records, err := s.Fetch(ctx)
	if err != nil {
		// In case of error, return empty slice
		return []datastore.ContractMetadata{}, fmt.Errorf("failed to fetch records: %w", err)
	}

	// Apply each filter in sequence
	for _, filter := range filters {
		records = filter(records)
	}

	return records, nil
}

func (s *memoryContractMetadataStore) Add(ctx context.Context, r datastore.ContractMetadata) error {
	return s.edit(ctx, query_ADD_CONTRACT_METADATA, r)
}

func (s *memoryContractMetadataStore) Upsert(ctx context.Context, key datastore.ContractMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
	// Build options with defaults
	options := &datastore.UpdateOptions{
		Updater: datastore.IdentityUpdaterF, // default updater
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get current record for merging (if it exists)
	currentRecord, err := s.Get(ctx, key)
	if err != nil && !errors.Is(err, datastore.ErrContractMetadataNotFound) {
		return fmt.Errorf("failed to get current record for upsert: %w", err)
	}

	var finalMetadata any
	if errors.Is(err, datastore.ErrContractMetadataNotFound) {
		// Record doesn't exist, use the provided metadata directly
		finalMetadata = metadata
	} else {
		// Record exists, apply the updater to merge with existing metadata
		finalMetadata, err = options.Updater(currentRecord.Metadata, metadata)
		if err != nil {
			return fmt.Errorf("failed to apply metadata updater: %w", err)
		}
	}

	// Create record with final metadata
	record := datastore.ContractMetadata{
		ChainSelector: key.ChainSelector(),
		Address:       key.Address(),
		Metadata:      finalMetadata,
	}

	return s.edit(ctx, query_UPSERT_CONTRACT_METADATA, record)
}

func (s *memoryContractMetadataStore) Update(ctx context.Context, key datastore.ContractMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
	// Build options with defaults
	options := &datastore.UpdateOptions{
		Updater: datastore.IdentityUpdaterF, // default updater
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get current record - it must exist for update
	currentRecord, err := s.Get(ctx, key)
	if err != nil {
		if errors.Is(err, datastore.ErrContractMetadataNotFound) {
			return datastore.ErrContractMetadataNotFound
		}

		return fmt.Errorf("failed to get current record for update: %w", err)
	}

	// Apply the updater (either default or custom)
	finalMetadata, updateErr := options.Updater(currentRecord.Metadata, metadata)
	if updateErr != nil {
		return fmt.Errorf("failed to apply metadata updater: %w", updateErr)
	}

	// Create record with final metadata
	record := datastore.ContractMetadata{
		ChainSelector: key.ChainSelector(),
		Address:       key.Address(),
		Metadata:      finalMetadata,
	}

	return s.edit(ctx, query_UPDATE_CONTRACT_METADATA, record)
}

func (s *memoryContractMetadataStore) Delete(_ context.Context, _ datastore.ContractMetadataKey) error {
	// The catalog API does not support delete operations
	// This is intentional as catalogs are typically immutable reference stores
	return errors.New("delete operation not supported for catalog contract metadata store")
}

func (s *memoryContractMetadataStore) edit(_ context.Context, qry string, r datastore.ContractMetadata) error {
	// Serialize metadata to JSON
	var metadataJSON string
	if r.Metadata != nil {
		metadataBytes, err := json.Marshal(r.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata to JSON: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	result, err := s.db.Exec(qry, r.ChainSelector, r.Address, metadataJSON)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		switch qry {
		case query_UPDATE_CONTRACT_METADATA:
			return datastore.ErrContractMetadataNotFound
		default:
			return fmt.Errorf("expected 1 row affected, got %d", count)
		}
	}

	return nil
}
