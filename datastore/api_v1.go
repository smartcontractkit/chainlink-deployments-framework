package datastore

// Comparable provides an Equals() method which returns true if the two instances are equal, false otherwise.
type Comparable[T any] interface {
	// Equals	returns true if the two instances are equal, false otherwise.
	Equals(T) bool
}

// Fetcher provides a Fetch() method which is used to complete a read query from a Store.
type Fetcher[R any] interface {
	// Fetch returns a slice of records representing the entire data set. The returned slice
	// will be a newly allocated slice (not a reference to an existing one), and each record should
	// be a copy of the corresponding stored data. Modifying the returned slice or its records must
	// not affect the underlying data.
	Fetch() ([]R, error)
}

// Getter provides a Get() method which is used to complete a read by key query from a Store.
type Getter[K Comparable[K], R UniqueRecord[K, R]] interface {
	// Get returns the record with the given key, or an error if no such record exists.
	Get(K) (R, error)
}

// PrimaryKeyHolder is an interface for types that can provide a unique identifier key for themselves.
type PrimaryKeyHolder[K Comparable[K]] interface {
	// Key returns the primary key for the implementing type.
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

// BaseDataStore is an interface that defines the basic operations for a data store.
// It is parameterized by the type of address reference store, chain metadata, contract metadata store and
// env metadata store it uses.
type BaseDataStore[
	R AddressRefStore, CH ChainMetadataStore, CM ContractMetadataStore, EM EnvMetadataStore,
] interface {
	Addresses() R
	ChainMetadata() CH
	ContractMetadata() CM
	EnvMetadata() EM
}

// DataStore is an interface that defines the operations for a read-only data store.
type DataStore interface {
	BaseDataStore[
		AddressRefStore, ChainMetadataStore,
		ContractMetadataStore, EnvMetadataStore,
	]
}

// Merger is an interface that defines a method for merging two data stores.
type Merger[T any] interface {
	// Merge merges the given data into the current data store.
	// It should return an error if the merge fails.
	Merge(other T) error
}

// Sealer is an interface that defines a method for sealing a data store.
// A sealed data store cannot be modified further.
type Sealer[T any] interface {
	// Seal seals the data store, preventing further modifications.
	Seal() T
}

// MutableDataStore is an interface that defines the operations for a mutable data store.
type MutableDataStore interface {
	Merger[DataStore]
	Sealer[DataStore]

	BaseDataStore[
		MutableAddressRefStore, MutableChainMetadataStore,
		MutableContractMetadataStore, MutableEnvMetadataStore,
	]
}
