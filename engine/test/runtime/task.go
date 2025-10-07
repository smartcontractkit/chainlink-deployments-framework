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

type baseTask struct {
	id string
}

// newBaseTask creates a new base task with a unique ID.
func newBaseTask() *baseTask {
	return &baseTask{
		id: ksuid.New().String(),
	}
}

// ID returns the unique identifier for this task.
func (t *baseTask) ID() string {
	return t.id
}
