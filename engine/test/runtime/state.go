package runtime

import (
	"fmt"
	"sync"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// State represents the mutable State of a test runtime environment. It tracks the accumulated
// results of changeset executions including address book updates, datastore changes, and
// changeset outputs. All State modifications are thread-safe through the use of a mutex.
//
// The State is updated after each changeset execution to reflect the changes made by that
// changeset, allowing subsequent changesets to build upon previous results.
type State struct {
	mu sync.Mutex // Protects all fields from concurrent access

	AddressBook fdeployment.AddressBook                // Legacy address book (deprecated)
	DataStore   fdatastore.DataStore                   // Datastore containing address references and metadata
	Outputs     map[string]fdeployment.ChangesetOutput // Changeset outputs keyed by changeset ID
}

// seedStateFromEnvironment creates a new state instance initialized with data from the given
// environment. This is used to bootstrap the runtime state with the initial environment data.
func seedStateFromEnvironment(e fdeployment.Environment) *State {
	return &State{
		AddressBook: e.ExistingAddresses, //nolint:staticcheck // SA1019 (Deprecated): We still need to support AddressBook for now
		DataStore:   e.DataStore,
		Outputs:     make(map[string]fdeployment.ChangesetOutput),
	}
}

// MergeChangesetOutput updates the state with the results of a changeset execution.
// This method is thread-safe and updates all relevant state components based on the
// changeset output.
//
// The update process includes:
// 1. Merging any address book changes from the changeset output
// 2. Merging any datastore changes from the changeset output
// 3. Storing the complete changeset output for future reference
//
// Returns an error if the address book or datastore merge operations fail.
func (s *State) MergeChangesetOutput(id string, out fdeployment.ChangesetOutput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.mergeAddressBook(out); err != nil {
		return fmt.Errorf("failed to update address book state: %w", err)
	}

	if err := s.mergeDataStore(out); err != nil {
		return fmt.Errorf("failed to update datastore state: %w", err)
	}

	s.updateOutput(id, out)

	return nil
}

// mergeAddressBook merges address book changes from a changeset output into the current state.
// This is a legacy operation as AddressBook is deprecated in favor of DataStore.
//
// If the changeset output contains no address book data, this method does nothing.
// Otherwise, it merges the output's address book into the current state's address book.
//
// Returns an error if the merge operation fails.
func (s *State) mergeAddressBook(out fdeployment.ChangesetOutput) error {
	// If the output does not contain an address book, do nothing.
	if out.AddressBook == nil { //nolint:staticcheck // SA1019 (Deprecated): We still need to support AddressBook for now
		return nil
	}

	return s.AddressBook.Merge(out.AddressBook) //nolint:staticcheck // SA1019 (Deprecated): We still need to support AddressBook for now
}

// mergeDataStore merges datastore changes from a changeset output into the current state.
// This method creates a new datastore that combines the existing state with the new changes.
//
// The merge process:
// 1. Creates a new memory datastore
// 2. Merges the current state's datastore into it
// 3. Merges the changeset output's datastore into it
// 4. Seals the result and updates the state
//
// If the changeset output contains no datastore data, this method does nothing.
//
// Returns an error if any of the merge operations fail.
func (s *State) mergeDataStore(out fdeployment.ChangesetOutput) error {
	// If the output does not contain a datastore, do nothing.
	if out.DataStore == nil {
		return nil
	}

	ds := fdatastore.NewMemoryDataStore()

	// Merge in existing datastore.
	if err := ds.Merge(s.DataStore); err != nil {
		return fmt.Errorf("failed to merge existing datastore: %w", err)
	}

	// Merge in output datastore
	if err := ds.Merge(out.DataStore.Seal()); err != nil {
		return fmt.Errorf("failed to merge output datastore: %w", err)
	}

	// Update the state with the new datastore.
	s.DataStore = ds.Seal()

	return nil
}

// updateOutput stores the changeset output in the state's outputs map for future reference.
// This allows the runtime to track all changeset executions and their results.
func (s *State) updateOutput(id string, out fdeployment.ChangesetOutput) {
	s.Outputs[id] = out
}
