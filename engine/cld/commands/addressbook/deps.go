// Package addressbook provides CLI commands for address book management operations.
package addressbook

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// AddressBookMergerFunc merges a changeset's address book to the main address book.
type AddressBookMergerFunc func(envDir domain.EnvDir, name, timestamp string) error

// AddressBookMigratorFunc migrates the address book to the new datastore format.
type AddressBookMigratorFunc func(envDir domain.EnvDir) error

// AddressBookRemoverFunc removes a changeset's address book entries from the main address book.
type AddressBookRemoverFunc func(envDir domain.EnvDir, name, timestamp string) error

// defaultAddressBookMerger is the production implementation that merges address books.
func defaultAddressBookMerger(envDir domain.EnvDir, name, timestamp string) error {
	return envDir.MergeChangesetAddressBook(name, timestamp)
}

// defaultAddressBookMigrator is the production implementation that migrates address books.
func defaultAddressBookMigrator(envDir domain.EnvDir) error {
	return envDir.MigrateAddressBook()
}

// defaultAddressBookRemover is the production implementation that removes address book entries.
func defaultAddressBookRemover(envDir domain.EnvDir, name, timestamp string) error {
	return envDir.RemoveChangesetAddressBook(name, timestamp)
}

// Deps holds the injectable dependencies for addressbook commands.
// All fields are optional; nil values will use production defaults.
type Deps struct {
	// AddressBookMerger merges a changeset's address book to the main address book.
	// Default: envDir.MergeChangesetAddressBook
	AddressBookMerger AddressBookMergerFunc

	// AddressBookMigrator migrates the address book to the new datastore format.
	// Default: envDir.MigrateAddressBook
	AddressBookMigrator AddressBookMigratorFunc

	// AddressBookRemover removes a changeset's address book entries.
	// Default: envDir.RemoveChangesetAddressBook
	AddressBookRemover AddressBookRemoverFunc
}

// applyDefaults fills in nil dependencies with production defaults.
func (d *Deps) applyDefaults() {
	if d.AddressBookMerger == nil {
		d.AddressBookMerger = defaultAddressBookMerger
	}
	if d.AddressBookMigrator == nil {
		d.AddressBookMigrator = defaultAddressBookMigrator
	}
	if d.AddressBookRemover == nil {
		d.AddressBookRemover = defaultAddressBookRemover
	}
}
