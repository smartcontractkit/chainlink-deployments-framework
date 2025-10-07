package runtime

import (
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

var _ Executable = &changesetTask[any]{}

// ChangesetTask creates a new executable task that runs a changeset.
// It generates a unique ID for the task and returns a task that can be executed by the runtime.
func ChangesetTask[C any](changeset fdeployment.ChangeSetV2[C], config C) changesetTask[C] {
	return changesetTask[C]{
		baseTask: newBaseTask(),

		changeset: changeset,
		config:    config,
	}
}

// changesetTask is a generic executable task that wraps a changeset and its configuration. The
// generic type parameter C represents the configuration type required by the changeset.
//
// Implements the Executable interface.
type changesetTask[C any] struct {
	*baseTask

	changeset fdeployment.ChangeSetV2[C] // The changeset to execute
	config    C                          // Configuration data for the changeset
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
