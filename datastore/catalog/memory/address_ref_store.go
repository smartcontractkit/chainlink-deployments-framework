package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

const (
	query_ADDRESS_REFERENCE_BY_ID = `
		SELECT * from address_references
		WHERE chain_selector = $1
		  AND contract_type = $2
		  AND version = $3
		  AND qualifier = $4`
	query_ALL_ADDRESS_REFERENCES = `
		SELECT chain_selector, contract_type, version, qualifier, address, label_set
          FROM address_references`
	query_ADD_ADDRESS_REFERENCE = `
		INSERT INTO address_references (chain_selector, contract_type, version, qualifier, address, label_set)
		VALUES ($1, $2, $3, $4, $5, $6)`
	query_UPSERT_ADDRESS_REFERENCE = query_ADD_ADDRESS_REFERENCE + `
		ON CONFLICT ON CONSTRAINT address_references_pkey
			DO UPDATE SET address = excluded.address, label_set = excluded.label_set`
	query_UPDATE_ADDRESS_REFERENCE = `
		UPDATE address_references SET
			address = $5,
			label_set = $6
        WHERE chain_selector = $1
			AND contract_type = $2
			AND version = $3
			AND qualifier = $4`
)

type memoryAddressRefStore struct {
	t      *testing.T
	config MemoryDataStoreConfig
	db     *dbController
}

// Ensure memoryAddressRefStore implements the V2 interface
var _ datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] = &memoryAddressRefStore{}

func newCatalogAddressRefStore(t *testing.T, config MemoryDataStoreConfig, db *dbController) *memoryAddressRefStore {
	return &memoryAddressRefStore{
		t:      t,
		config: config,
		db:     db,
	}
}

func (s *memoryAddressRefStore) Get(_ context.Context, key datastore.AddressRefKey, options ...datastore.GetOption) (datastore.AddressRef, error) {
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
	rows, err := db.Query(query_ADDRESS_REFERENCE_BY_ID, key.ChainSelector(), key.Type().String(), key.Version().String(), key.Qualifier())
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)
	if err != nil {
		return datastore.AddressRef{}, err
	}

	count := 0
	row := &datastore.AddressRef{}
	for rows.Next() {
		count++
		array := pq.StringArray{}
		err = rows.Scan(&row.ChainSelector, &row.Type, &row.Version, &row.Qualifier, &row.Address, &array)
		row.Labels.Add(array...)
		if err != nil {
			return datastore.AddressRef{}, err
		}
	}

	switch count {
	case 0:
		return *row, datastore.ErrAddressRefNotFound
	case 1:
		return *row, nil
	default:
		err = fmt.Errorf("expected a single row, got %d", count)
		return *row, err
	}
}

// Fetch returns a copy of all AddressRefs in the catalog.
func (s *memoryAddressRefStore) Fetch(_ context.Context) ([]datastore.AddressRef, error) {
	rows, err := s.db.Query(query_ALL_ADDRESS_REFERENCES)
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	if err != nil {
		return nil, err
	}
	var refs []datastore.AddressRef

	for rows.Next() {
		row := &datastore.AddressRef{}
		array := pq.StringArray{}
		err = rows.Scan(&row.ChainSelector, &row.Type, &row.Version, &row.Qualifier, &row.Address, &array)
		row.Labels.Add(array...)
		if err != nil {
			return refs, err
		}
		refs = append(refs, *row)
	}
	return refs, nil
}

// Filter returns a copy of all AddressRef in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *memoryAddressRefStore) Filter(
	ctx context.Context,
	filters ...datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef],
) ([]datastore.AddressRef, error) {
	// First, fetch all records from the catalog
	records, err := s.Fetch(ctx)
	if err != nil {
		// In case of error, return empty slice
		// In a more robust implementation, you might want to log this error
		return []datastore.AddressRef{}, fmt.Errorf("failed to fetch records: %w", err)
	}

	// Apply each filter in sequence
	for _, filter := range filters {
		records = filter(records)
	}

	return records, nil
}

func (s *memoryAddressRefStore) Add(ctx context.Context, r datastore.AddressRef) error {
	return s.edit(ctx, query_ADD_ADDRESS_REFERENCE, r)
}

func (s *memoryAddressRefStore) Upsert(ctx context.Context, r datastore.AddressRef) error {
	return s.edit(ctx, query_UPSERT_ADDRESS_REFERENCE, r)
}

func (s *memoryAddressRefStore) Update(ctx context.Context, r datastore.AddressRef) error {
	return s.edit(ctx, query_UPDATE_ADDRESS_REFERENCE, r)
}

func (s *memoryAddressRefStore) edit(_ context.Context, qry string, r datastore.AddressRef) error {
	result, err := s.db.Exec(
		qry,
		r.ChainSelector,
		r.Type.String(),
		r.Version.String(),
		r.Qualifier,
		r.Address,
		pq.StringArray(r.Labels.List()),
	)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		switch qry {
		case query_UPDATE_ADDRESS_REFERENCE:
			return datastore.ErrAddressRefNotFound
		default:
			return fmt.Errorf("expected 1 row affected, got %d", count)
		}
	}
	return nil
}

func (s *memoryAddressRefStore) Delete(_ context.Context, _ datastore.AddressRefKey) error {
	// The catalog API does not support delete operations
	// This is intentional as catalogs are typically immutable reference stores
	return errors.New("delete operation not supported by catalog API")
}
