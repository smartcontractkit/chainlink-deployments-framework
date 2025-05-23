package datastore

import (
	"github.com/Masterminds/semver/v3"
)

// Cloneable provides a Clone() method which returns a semi-deep copy of the type.
type Cloneable[R any] interface {
	// Clone() returns a semi-deep copy of the type. The implementation should call Clone() on
	// any nested Cloneable fields, and perform a shallow copy of any slices or maps.
	// NOTE: Handling non-Cloneable references to Cloneable types is beyond the intended scope of this interface.
	Clone() R
}

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

// UniqueRecord represents a data entry that is both Cloneable and uniquely identifiable by its primary key.
type UniqueRecord[K Comparable[K], R PrimaryKeyHolder[K]] interface {
	Cloneable[R]
	PrimaryKeyHolder[K]
}

type Record[R any] interface {
	Cloneable[R]
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
type UnaryStore[R Record[R]] interface {
	// Get returns the record or an error.
	// if the record exists, the error should be nil.
	// If the record does not exist, the error should not be nil.
	Get() (R, error)
}

// MutableUnaryStore is an interface that represents a mutable store that contains a single record.
type MutableUnaryStore[R Record[R]] interface {
	// Get returns a copy of the record or an error.
	// If the record exists, the error should be nil.
	// If the record does not exist, the error should not be nil.
	Get() (R, error)

	// Set sets the record in the store.
	// If the record already exists, it should be replaced.
	// If the record does not exist, it should be added.
	Set(record R) error
}

// EnvMetadataStore is an interface that defines the methods for a store that manages environment metadata.
type EnvMetadataStore interface {
	UnaryStore[EnvMetadata]
}

// MutableEnvMetadataStore is an interface that defines the methods for a mutable store that manages environment metadata.
type MutableEnvMetadataStore interface {
	MutableUnaryStore[EnvMetadata]
}

// ContractMetadataStore is an interface that represents an immutable view over a set
// of ContractMetadata records identified by ContractMetadataKey.
type ContractMetadataStore interface {
	Store[ContractMetadataKey, ContractMetadata]
}

// MutableContractMetadataStore is an interface that represents a mutable ContractMetadataStore
// of ContractMetadata records identified by ContractMetadataKey.
type MutableContractMetadataStore interface {
	MutableStore[ContractMetadataKey, ContractMetadata]
}

// AddressRefStore is an interface that represents an immutable view over a set
// of AddressRef records  identified by AddressRefKey.
type AddressRefStore interface {
	Store[AddressRefKey, AddressRef]
}

// MutableAddressRefStore is an interface that represents a mutable AddressRefStore
// of AddressRef records identified by AddressRefKey.
type MutableAddressRefStore interface {
	MutableStore[AddressRefKey, AddressRef]
}

// ContractMetadataKey is an interface that represents a key for ContractMetadata records.
// It is used to uniquely identify a record in the ContractMetadataStore.
type ContractMetadataKey interface {
	Comparable[ContractMetadataKey]

	// Address returns the address of the contract on the chain.
	Address() string
	// ChainSelector returns the chain-selector of the chain where the contract is deployed.
	ChainSelector() uint64
}

// AddressRefKey is an interface that represents a key for AddressRef records.
// It is used to uniquely identify a record in the AddressRefStore.
type AddressRefKey interface {
	Comparable[AddressRefKey]

	// ChainSelector returns the chain-selector selector of the chain where the contract is deployed.
	ChainSelector() uint64
	// Type returns the contract type of the contract.
	// This is a simple string type for identifying contract
	Type() ContractType
	// Version returns the semantic version of the contract.
	Version() *semver.Version
	// Qualifier returns the optional qualifier for the contract.
	// This can be used to differentiate between different references of the same contract.
	Qualifier() string
}
