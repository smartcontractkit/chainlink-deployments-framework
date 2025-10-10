package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

const (
	// envMetadataID is the fixed ID used for environment metadata (singleton pattern)
	envMetadataID = 1

	query_ENV_METADATA_GET = `
		SELECT metadata FROM environment_metadata
		WHERE id = $1`
	query_ENV_METADATA_SET = `
		INSERT INTO environment_metadata (id, metadata)
		VALUES ($1, $2)
		ON CONFLICT ON CONSTRAINT environment_metadata_pkey
			DO UPDATE SET metadata = excluded.metadata`
)

type memoryEnvMetadataStore struct {
	db *dbController
}

// Ensure memoryEnvMetadataStore implements the V2 interface
var _ datastore.MutableUnaryStoreV2[datastore.EnvMetadata] = &memoryEnvMetadataStore{}

func newCatalogEnvMetadataStore(db *dbController) *memoryEnvMetadataStore {
	return &memoryEnvMetadataStore{
		db: db,
	}
}

func (s *memoryEnvMetadataStore) Get(ctx context.Context, options ...datastore.GetOption) (datastore.EnvMetadata, error) {
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

	rows, err := db.QueryContext(ctx, query_ENV_METADATA_GET, envMetadataID)
	defer func(rows *sql.Rows) {
		if rows != nil {
			_ = rows.Close()
		}
	}(rows)
	if err != nil {
		return datastore.EnvMetadata{}, err
	}

	count := 0
	row := &datastore.EnvMetadata{}
	for rows.Next() {
		count++
		var metadataJSON sql.NullString
		err = rows.Scan(&metadataJSON)
		if err != nil {
			return datastore.EnvMetadata{}, err
		}

		// Parse metadata JSON if present
		if metadataJSON.Valid && metadataJSON.String != "" {
			var metadata any
			if unmarshalErr := json.Unmarshal([]byte(metadataJSON.String), &metadata); unmarshalErr != nil {
				return datastore.EnvMetadata{}, fmt.Errorf("failed to unmarshal metadata JSON: %w", unmarshalErr)
			}
			row.Metadata = metadata
		}
	}

	switch count {
	case 0:
		return *row, datastore.ErrEnvMetadataNotSet
	case 1:
		return *row, nil
	default:
		err = fmt.Errorf("expected at most one row, got %d", count)
		return *row, err
	}
}

func (s *memoryEnvMetadataStore) Set(ctx context.Context, metadata any, opts ...datastore.UpdateOption) error {
	// Build options with defaults
	options := &datastore.UpdateOptions{
		Updater: datastore.IdentityUpdaterF, // default updater
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get current record for merging (if it exists)
	currentRecord, err := s.Get(ctx)
	if err != nil && !errors.Is(err, datastore.ErrEnvMetadataNotSet) {
		return fmt.Errorf("failed to get current record for set: %w", err)
	}

	var finalMetadata any
	if errors.Is(err, datastore.ErrEnvMetadataNotSet) {
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
	record := datastore.EnvMetadata{
		Metadata: finalMetadata,
	}

	return s.edit(ctx, record)
}

func (s *memoryEnvMetadataStore) edit(ctx context.Context, r datastore.EnvMetadata) error {
	// Serialize metadata to JSON
	var metadataJSON string
	if r.Metadata != nil {
		metadataBytes, err := json.Marshal(r.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata to JSON: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	result, err := s.db.ExecContext(ctx, query_ENV_METADATA_SET, envMetadataID, metadataJSON)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("expected 1 row affected, got %d", count)
	}

	return nil
}
