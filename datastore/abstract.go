package datastore

import "context"

// Comparable provides an Equals() method which returns true if the two instances are equal, false otherwise.
type Comparable[T any] interface {
	// Equals()	returns true if the two instances are equal, false otherwise.
	Equals(T) bool
}

// Fetcher provides a Fetch() method which is used to complete a read query from a Store.
type Fetcher[R any] interface {
	// Fetch() returns a slice of records representing the entire data set. The returned slice
	// will be a newly allocated slice (not a reference to an existing one), and each record should
	// be a copy of the corresponding stored data. Modifying the returned slice or its records must
	// not affect the underlying data.
	Fetch() ([]R, error)
}

// Getter provides a Get() method which is used to complete a read by key query from a Store.
type Getter[K Comparable[K], R UniqueRecord[K, R]] interface {
	// Get() returns the record with the given key, or an error if no such record exists.
	Get(K) (R, error)
}

// PrimaryKeyHolder is an interface for types that can provide a unique identifier key for themselves.
type PrimaryKeyHolder[K Comparable[K]] interface {
	// Key() returns the primary key for the implementing type.
	Key() K
}

// UniqueRecord represents a data entry that is uniquely identifiable by its primary key.
type UniqueRecord[K Comparable[K], R PrimaryKeyHolder[K]] interface {
	PrimaryKeyHolder[K]
}

// FilterFunc is a function that filters a slice of records.
type FilterFunc[K Comparable[K], R UniqueRecord[K, R]] func([]R) []R

// Filterable provides a Filter() method which is used to complete a filtered query with from a Store.
type Filterable[K Comparable[K], R UniqueRecord[K, R]] interface {
	Filter(filters ...FilterFunc[K, R]) []R
}

// Store is an interface that represents an immutable set of records.
type Store[K Comparable[K], R UniqueRecord[K, R]] interface {
	Fetcher[R]
	Getter[K, R]
	Filterable[K, R]
}

// MutableStore is an interface that represents a mutable set of records.
type MutableStore[K Comparable[K], R UniqueRecord[K, R]] interface {
	Store[K, R]

	// Add inserts a new record into the MutableStore.
	Add(record R) error

	// Upsert behaves like Add where there is not already a record with the same composite primary key as the
	// supplied record, otherwise it behaves like an update.
	Upsert(record R) error

	// Update edits an existing record whose fields match the primary key elements of the supplied AddressRecord, with
	// the non-primary-key values of the supplied AddressRecord.
	Update(record R) error

	// Delete deletes record whose primary key elements match the supplied key, returning an error if no
	// such record exists to be deleted
	Delete(key K) error
}

// UnaryStore is an interface that represents a read-only store that is limited to a single record.
type UnaryStore[R any] interface {
	// Get returns the record or an error.
	// if the record exists, the error should be nil.
	// If the record does not exist, the error should not be nil.
	Get() (R, error)
}

// MutableUnaryStore is an interface that represents a mutable store that contains a single record.
type MutableUnaryStore[R any] interface {
	// Get returns a copy of the record or an error.
	// If the record exists, the error should be nil.
	// If the record does not exist, the error should not be nil.
	Get() (R, error)

	// Set sets the record in the store.
	// If the record already exists, it should be replaced.
	// If the record does not exist, it should be added.
	Set(record R) error
}

// V2 interfaces
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

// WithUpdater sets a custom metadata updater for update operations
func WithUpdater(updater MetadataUpdaterF) UpdateOption {
	return func(opts *UpdateOptions) {
		opts.Updater = updater
	}
}

// Fetcher provides a Fetch() method which is used to complete a read query from a Store.
type FetcherV2[R any] interface {
	// Fetch() returns a slice of records representing the entire data set. The returned slice
	// will be a newly allocated slice (not a reference to an existing one), and each record should
	// be a copy of the corresponding stored data. Modifying the returned slice or its records must
	// not affect the underlying data.
	Fetch(context.Context) ([]R, error)
}

// Filterable provides a Filter() method which is used to complete a filtered query with from a Store.
type FilterableV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	Filter(context.Context, ...FilterFunc[K, R]) ([]R, error)
}

// Getter provides a Get() method which is used to complete a read by key query from a Store.
type GetterV2[K Comparable[K], R UniqueRecord[K, R]] interface {
	// Get() returns the record with the given key, or an error if no such record exists.
	Get(context.Context, K) (R, error)
}

// MutableStore is an interface that represents a mutable set of records.
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

// MutableUnaryStore is an interface that represents a mutable store that contains a single record.
type MutableUnaryStoreV2[R any] interface {
	// Get returns a copy of the record or an error.
	// If the record exists, the error should be nil.
	// If the record does not exist, the error should not be nil.
	Get(context.Context) (R, error)

	// Set sets the record in the store.
	// If the record already exists, it should be replaced.
	// If the record does not exist, it should be added.
	// Options can be provided to customize the behavior (e.g., custom updater function).
	Set(ctx context.Context, metadata any, opts ...UpdateOption) error
}
