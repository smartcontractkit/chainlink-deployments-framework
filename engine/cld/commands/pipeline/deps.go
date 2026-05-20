package pipeline

import (
	"context"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

// EnvironmentLoaderFunc loads a deployment environment.
type EnvironmentLoaderFunc func(
	ctx context.Context,
	dom domain.Domain,
	envKey string,
	opts ...environment.LoadEnvironmentOption,
) (fdeployment.Environment, error)

// AddressBookMergerFunc merges a changeset's address book to the main address book.
type AddressBookMergerFunc func(envDir domain.EnvDir, name, timestamp string) error

// DataStoreMergerFunc merges a changeset's datastore to the main datastore.
type DataStoreMergerFunc func(envDir domain.EnvDir, name, timestamp string) error

// Deps holds optional dependencies that can be overridden for testing.
type Deps struct {
	// EnvironmentLoader loads a deployment environment. Default: environment.Load
	EnvironmentLoader EnvironmentLoaderFunc

	// AddressBookMerger merges a changeset's address book to the main address book.
	// Default: envDir.MergeChangesetAddressBook
	AddressBookMerger AddressBookMergerFunc

	// DataStoreMerger merges a changeset's datastore to the main datastore.
	// Default: envDir.MergeChangesetDataStore
	DataStoreMerger DataStoreMergerFunc
}

// DefaultEnvironmentLoader is used when Deps.EnvironmentLoader is nil.
// Tests can override this to inject a mock.
var DefaultEnvironmentLoader = environment.Load

// applyDefaults fills in nil fields with production implementations.
func (d *Deps) applyDefaults() {
	if d.EnvironmentLoader == nil {
		d.EnvironmentLoader = DefaultEnvironmentLoader
	}
	if d.AddressBookMerger == nil {
		d.AddressBookMerger = func(envDir domain.EnvDir, name, timestamp string) error {
			return envDir.MergeChangesetAddressBook(name, timestamp)
		}
	}
	if d.DataStoreMerger == nil {
		d.DataStoreMerger = func(envDir domain.EnvDir, name, timestamp string) error {
			return envDir.MergeChangesetDataStore(name, timestamp)
		}
	}
}
