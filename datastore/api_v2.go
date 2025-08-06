package datastore

import "context"

type TransactionLogic func(ctx context.Context) error

// Transactional is an interface which supports keeping datastore operations within transactional
// boundaries.
type Transactional interface {
	BeginTransaction() error
	CommitTransaction() error
	RollbackTransaction() error
	WithTransaction(ctx context.Context, fn TransactionLogic) error
}

// BaseDataStoreV2 is an interface that defines the basic operations for a data store using V2 interfaces.
type BaseDataStoreV2[
	R MutableRefStoreV2[AddressRefKey, AddressRef],
	CH MutableStoreV2[ChainMetadataKey, ChainMetadata],
	CM MutableStoreV2[ContractMetadataKey, ContractMetadata],
	EM MutableUnaryStoreV2[EnvMetadata],
] interface {
	Addresses() R
	ChainMetadata() CH
	ContractMetadata() CM
	EnvMetadata() EM
}

// CatalogStore is a convenience interface which wraps up the various generics so they need not
// be repeatedly specified.
type CatalogStore interface {
	Transactional
	BaseDataStoreV2[
		MutableRefStoreV2[AddressRefKey, AddressRef], MutableStoreV2[ChainMetadataKey, ChainMetadata],
		MutableStoreV2[ContractMetadataKey, ContractMetadata], MutableUnaryStoreV2[EnvMetadata],
	]
}

// MetadataUpdaterF  characterises a change to some metadata as a sort of operational transform,
// which can be run against any conforming metadata object. It should be cautious - e.g. if it's
// adding to a slice in a sub-struct it should make sure the sub-struct was initialized, so the
// logic is universal to any version of that metadata struct it happens upon.
//
// This approach, used via WithUpdater, allows for conflict-free update logic to be applied,
// with automatic handling of any data races.
type MetadataUpdaterF func(latest any, incoming any) (any, error)

// IdentityUpdaterF is the default updater that simply replaces latest with incoming
func IdentityUpdaterF(latest any, incoming any) (any, error) {
	return incoming, nil
}

// UpdateOptions holds configuration for update operations
type UpdateOptions struct {
	Updater MetadataUpdaterF
}

// UpdateOption is a function that modifies UpdateOption
type UpdateOption func(*UpdateOptions)

// WithUpdater sets a custom metadata updater for update operations.
func WithUpdater(updater MetadataUpdaterF) UpdateOption {
	return func(opts *UpdateOptions) {
		opts.Updater = updater
	}
}

// FetcherV2 provides a Fetch() method which is used to complete a read query from a Store.
type FetcherV2[R any] interface {
	// Fetch returns a slice of records representing the entire data set. The returned slice
	// will be a newly allocated slice (not a reference to an existing one), and each record should
	// be a copy of the corresponding stored data. Modifying the returned slice or its records must
	// not affect the underlying data.
	Fetch() ([]R, error)
}

// FilterableV2 provides a Filter() method which is used to complete a filtered query with from a Store.
type FilterableV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	Filter(...FilterFunc[K, R]) ([]R, error)
}

// GetterV2 provides a Get() method which is used to complete a read by key query from a Store.
type GetterV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	// Get returns the record with the given key, or an error if no such record exists.
	Get(K) (R, error)
	// GetIgnoringTransactions returns the record with the given key, but does so from outside
	// the current (if any) transaction context. Any writes performed within uncommitted
	// transactions will not be reflected in the result.
	GetIgnoringTransactions(K) (R, error)
}

// MutableStoreV2 is an interface that represents a mutable set of records.
type MutableStoreV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	FetcherV2[R]
	GetterV2[K, R]
	FilterableV2[K, R]

	// Add inserts a new record into the MutableStore.
	Add(record R) error

	// Upsert behaves like Add where there is not already a record with the same composite primary key as the
	// supplied record, otherwise it behaves like an update.
	// Options can be provided to customize the behavior (e.g., custom updater function).
	Upsert(key K, metadata any, opts ...UpdateOption) error

	// Update edits an existing record whose fields match the primary key elements of the supplied AddressRecord, with
	// the non-primary-key values of the supplied AddressRecord.
	// Options can be provided to customize the behavior (e.g., custom updater function).
	Update(key K, metadata any, opts ...UpdateOption) error

	// Delete deletes record whose primary key elements match the supplied key, returning an error if no
	// such record exists to be deleted
	Delete(key K) error
}

type MutableRefStoreV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	FetcherV2[R]
	GetterV2[K, R]
	FilterableV2[K, R]

	Add(record R) error

	Update(record R) error

	Upsert(record R) error

	Delete(key K) error
}

// MutableUnaryStoreV2 is an interface that represents a mutable store that contains a single record.
type MutableUnaryStoreV2[R any] interface {
	// Get returns a copy of the record or an error.
	// If the record exists, the error should be nil.
	// If the record does not exist, the error should not be nil.
	Get() (R, error)

	// Set sets the record in the store.
	// If the record already exists, it should be replaced.
	// If the record does not exist, it should be added.
	// Options can be provided to customize the behavior (e.g., custom updater function).
	Set(metadata any, opts ...UpdateOption) error
}
