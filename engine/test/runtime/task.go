package runtime

import (
	"github.com/segmentio/ksuid"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// Executable represents a task that can be executed by the runtime.
//
// Executables are used to manipulate the runtime state and environment. The most common
// implementation is to run a changeset, however this could be extended to perform offchain
// operations in the future.
type Executable interface {
	// ID returns a unique identifier for this executable task.
	ID() string

	// Run executes the task against the provided environment and updates the state.
	// The environment represents the current deployment state, and the state parameter
	// should be updated with any changes produced by the execution.
	Run(e fdeployment.Environment, state *State) error
}

var _ Executable = &changesetTask[any]{}

// ChangesetTask creates a new executable task that runs a changeset.
// It generates a unique ID for the task and returns a task that can be executed by the runtime.
func ChangesetTask[C any](changeset fdeployment.ChangeSetV2[C], config C) changesetTask[C] {
	return changesetTask[C]{
		id:        ksuid.New().String(),
		changeset: changeset,
		config:    config,
	}
}

// changesetTask is a generic executable task that wraps a changeset and its configuration. The
// generic type parameter C represents the configuration type required by the changeset.
//
// Implements the Executable interface.
type changesetTask[C any] struct {
	id        string                     // Unique identifier for this task
	changeset fdeployment.ChangeSetV2[C] // The changeset to execute
	config    C                          // Configuration data for the changeset
}

// ID returns the unique identifier for this task.
func (r changesetTask[C]) ID() string {
	return r.id
}

// Run executes the changeset with the given environment and updates the runtime state.
//
// Returns an error if the changeset execution fails or state update fails.
func (r changesetTask[C]) Run(e fdeployment.Environment, state *State) error {
	output, err := r.applyChangeset(e)
	if err != nil {
		return err
	}

	// Update the state with the output
	if err := state.MergeChangesetOutput(r.ID(), output); err != nil {
		return err
	}

	// TODO: Execute MCMS proposals

	return nil
}

// applyChangeset verifies preconditions and applies the changeset to the environment.
//
// Returns the changeset output or an error if preconditions fail or application fails.
func (r changesetTask[C]) applyChangeset(e fdeployment.Environment) (fdeployment.ChangesetOutput, error) {
	if err := r.changeset.VerifyPreconditions(e, r.config); err != nil {
		return fdeployment.ChangesetOutput{}, err
	}

	return r.changeset.Apply(e, r.config)
}
