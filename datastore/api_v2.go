package datastore

import "context"

type TransactionLogic func(ctx context.Context) error

// Transactional is an interface which supports keeping datastore operations within transactional
// boundaries.
type Transactional interface {
	// WithTransaction allows the caller to wrap their datastore operations in a transactional
	// boundary, such that any datastore operations will succeed or fail together. The caller
	// supplies a lambda containing the operations, which the calling context being plumbed
	// through. Starting and committing the transaction is automated, and if an error is
	// returned, the transaction is rolled-back instead.
	WithTransaction(ctx context.Context, fn TransactionLogic) error

	// These are not publicly available in the API yet, pending further discussion around
	// exposing them.
	/* BeginTransaction() error */
	/* CommitTransaction() error */
	/* RollbackTransaction() error */
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
	Fetch(context.Context) ([]R, error)
}

// FilterableV2 provides a Filter() method which is used to complete a filtered query with from a Store.
type FilterableV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	Filter(context.Context, ...FilterFunc[K, R]) ([]R, error)
}

// GetterV2 provides a Get() method which is used to complete a read by key query from a Store.
type GetterV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	// Get returns the record with the given key, or an error if no such record exists.
	Get(context.Context, K) (R, error)
	// GetIgnoringTransactions returns the record with the given key, but does so from outside
	// the current (if any) transaction context. Any writes performed within uncommitted
	// transactions will not be reflected in the result.
	GetIgnoringTransactions(context.Context, K) (R, error)
}

// MutableStoreV2 is an interface that represents a mutable set of records.
type MutableStoreV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	FetcherV2[R]
	GetterV2[K, R]
	FilterableV2[K, R]

	// Add inserts a new record into the MutableStore.
	Add(ctx context.Context, record R) error

	// Upsert behaves like Add where there is not already a record with the same composite primary key as the
	// supplied record, otherwise it behaves like an update.
	// Options can be provided to customize the behavior (e.g., custom updater function).
	Upsert(ctx context.Context, key K, metadata any, opts ...UpdateOption) error

	// Update edits an existing record whose fields match the primary key elements of the supplied AddressRecord, with
	// the non-primary-key values of the supplied AddressRecord.
	// Options can be provided to customize the behavior (e.g., custom updater function).
	Update(ctx context.Context, key K, metadata any, opts ...UpdateOption) error

	// Delete deletes record whose primary key elements match the supplied key, returning an error if no
	// such record exists to be deleted
	Delete(ctx context.Context, key K) error
}

type MutableRefStoreV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	FetcherV2[R]
	GetterV2[K, R]
	FilterableV2[K, R]

	Add(ctx context.Context, record R) error

	Update(ctx context.Context, record R) error

	Upsert(ctx context.Context, record R) error

	Delete(ctx context.Context, key K) error
}

// MutableUnaryStoreV2 is an interface that represents a mutable store that contains a single record.
type MutableUnaryStoreV2[R any] interface {
	// Get returns a copy of the record or an error.
	// If the record exists, the error should be nil.
	// If the record does not exist, the error should not be nil.
	Get(ctx context.Context) (R, error)

	// Set sets the record in the store.
	// If the record already exists, it should be replaced.
	// If the record does not exist, it should be added.
	// Options can be provided to customize the behavior (e.g., custom updater function).
	Set(ctx context.Context, metadata any, opts ...UpdateOption) error
}
