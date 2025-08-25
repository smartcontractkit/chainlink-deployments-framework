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
	query_CHAIN_METADATA_BY_ID = `
		SELECT chain_selector, metadata FROM chain_metadata
		WHERE chain_selector = $1`
	query_ALL_CHAIN_METADATA = `
		SELECT chain_selector, metadata FROM chain_metadata`
	query_ADD_CHAIN_METADATA = `
		INSERT INTO chain_metadata (chain_selector, metadata)
		VALUES ($1, $2)`
	query_UPSERT_CHAIN_METADATA = query_ADD_CHAIN_METADATA + `
		ON CONFLICT ON CONSTRAINT chain_metadata_pkey
			DO UPDATE SET metadata = excluded.metadata`
	query_UPDATE_CHAIN_METADATA = `
		UPDATE chain_metadata SET metadata = $2
		WHERE chain_selector = $1`
	query_DELETE_CHAIN_METADATA = `
		DELETE FROM chain_metadata
		WHERE chain_selector = $1`
)

type memoryChainMetadataStore struct {
	t      *testing.T
	config MemoryDataStoreConfig
	db     *dbController
}

// Ensure memoryChainMetadataStore implements the V2 interface
var _ datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] = &memoryChainMetadataStore{}

func newCatalogChainMetadataStore(t *testing.T, config MemoryDataStoreConfig, db *dbController) *memoryChainMetadataStore {
	t.Helper()

	return &memoryChainMetadataStore{
		t:      t,
		config: config,
		db:     db,
	}
}

func (s *memoryChainMetadataStore) Get(_ context.Context, key datastore.ChainMetadataKey, options ...datastore.GetOption) (datastore.ChainMetadata, error) {
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

	rows, err := db.Query(query_CHAIN_METADATA_BY_ID, key.ChainSelector())
	defer func(rows *sql.Rows) {
		if rows != nil {
			_ = rows.Close()
		}
	}(rows)
	if err != nil {
		return datastore.ChainMetadata{}, err
	}

	count := 0
	row := &datastore.ChainMetadata{}
	for rows.Next() {
		count++
		var metadataJSON sql.NullString
		err = rows.Scan(&row.ChainSelector, &metadataJSON)
		if err != nil {
			return datastore.ChainMetadata{}, err
		}

		// Parse metadata JSON if present
		if metadataJSON.Valid && metadataJSON.String != "" {
			var metadata any
			if unmarshalErr := json.Unmarshal([]byte(metadataJSON.String), &metadata); unmarshalErr != nil {
				return datastore.ChainMetadata{}, fmt.Errorf("failed to unmarshal metadata JSON: %w", unmarshalErr)
			}
			row.Metadata = metadata
		}
	}

	switch count {
	case 0:
		return *row, datastore.ErrChainMetadataNotFound
	case 1:
		return *row, nil
	default:
		err = fmt.Errorf("expected a single row, got %d", count)
		return *row, err
	}
}

// Fetch returns a copy of all ChainMetadata in the catalog.
func (s *memoryChainMetadataStore) Fetch(_ context.Context) ([]datastore.ChainMetadata, error) {
	rows, err := s.db.Query(query_ALL_CHAIN_METADATA)
	defer func(rows *sql.Rows) {
		if rows != nil {
			_ = rows.Close()
		}
	}(rows)

	if err != nil {
		return nil, err
	}
	var records []datastore.ChainMetadata

	for rows.Next() {
		row := &datastore.ChainMetadata{}
		var metadataJSON sql.NullString
		err = rows.Scan(&row.ChainSelector, &metadataJSON)
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

// Filter returns a copy of all ChainMetadata in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *memoryChainMetadataStore) Filter(
	ctx context.Context,
	filters ...datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata],
) ([]datastore.ChainMetadata, error) {
	// First, fetch all records from the catalog
	records, err := s.Fetch(ctx)
	if err != nil {
		// In case of error, return empty slice
		return []datastore.ChainMetadata{}, fmt.Errorf("failed to fetch records: %w", err)
	}

	// Apply each filter in sequence
	for _, filter := range filters {
		records = filter(records)
	}

	return records, nil
}

func (s *memoryChainMetadataStore) Add(ctx context.Context, r datastore.ChainMetadata) error {
	return s.edit(ctx, query_ADD_CHAIN_METADATA, r)
}

func (s *memoryChainMetadataStore) Upsert(ctx context.Context, key datastore.ChainMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
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
	if err != nil && !errors.Is(err, datastore.ErrChainMetadataNotFound) {
		return fmt.Errorf("failed to get current record for upsert: %w", err)
	}

	var finalMetadata any
	if errors.Is(err, datastore.ErrChainMetadataNotFound) {
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
	record := datastore.ChainMetadata{
		ChainSelector: key.ChainSelector(),
		Metadata:      finalMetadata,
	}

	return s.edit(ctx, query_UPSERT_CHAIN_METADATA, record)
}

func (s *memoryChainMetadataStore) Update(ctx context.Context, key datastore.ChainMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
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
		if errors.Is(err, datastore.ErrChainMetadataNotFound) {
			return datastore.ErrChainMetadataNotFound
		}

		return fmt.Errorf("failed to get current record for update: %w", err)
	}

	// Apply the updater (either default or custom)
	finalMetadata, updateErr := options.Updater(currentRecord.Metadata, metadata)
	if updateErr != nil {
		return fmt.Errorf("failed to apply metadata updater: %w", updateErr)
	}

	// Create record with final metadata
	record := datastore.ChainMetadata{
		ChainSelector: key.ChainSelector(),
		Metadata:      finalMetadata,
	}

	return s.edit(ctx, query_UPDATE_CHAIN_METADATA, record)
}

func (s *memoryChainMetadataStore) Delete(_ context.Context, _ datastore.ChainMetadataKey) error {
	// The catalog API does not support delete operations
	// This is intentional as catalogs are typically immutable reference stores
	return errors.New("delete operation not supported for catalog chain metadata store")
}

func (s *memoryChainMetadataStore) edit(_ context.Context, qry string, r datastore.ChainMetadata) error {
	// Serialize metadata to JSON
	var metadataJSON string
	if r.Metadata != nil {
		metadataBytes, err := json.Marshal(r.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata to JSON: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	result, err := s.db.Exec(qry, r.ChainSelector, metadataJSON)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		switch qry {
		case query_UPDATE_CHAIN_METADATA:
			return datastore.ErrChainMetadataNotFound
		default:
			return fmt.Errorf("expected 1 row affected, got %d", count)
		}
	}

	return nil
}
