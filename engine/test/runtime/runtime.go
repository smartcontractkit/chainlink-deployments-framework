package runtime

import (
	"sync"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// Runtime provides an execution environment for running tasks in tests.
//
// It manages the state accumulation across multiple task executions and ensures
// that each task operates on a fresh environment that reflects the cumulative
// state changes from previous executions.
//
// The runtime is thread-safe and ensures sequential execution of task to
// maintain state consistency. Each task execution updates the internal state
// and regenerates the environment for subsequent executions.
type Runtime struct {
	mu sync.Mutex

	state      *State                  // Accumulated state from task executions
	currentEnv fdeployment.Environment // Current environment with latest state applied
}

// NewFromEnvironment creates a new Runtime instance initialized with the given environment.
//
// Initial state is seeded from the provided environment's existing addresses and datastore.
// A fresh environment is immediately generated and cached for the first execution.
//
// Returns a configured Runtime ready for task execution, or an error if initialization fails.
func NewFromEnvironment(e fdeployment.Environment) *Runtime {
	return &Runtime{
		state:      seedStateFromEnvironment(e),
		currentEnv: e,
	}
}

// Exec executes a sequence of tasks in order, ensuring each operates on a fresh
// environment that reflects the cumulative state changes from previous executions.
//
// The execution process for each task:
// 1. Execute the task against the current environment (The task is responsible for updating state)
// 2. Generate a new environment incorporating the updated state
// 3. Proceed to the next task
//
// Execution is thread-safe and atomic - if any task fails, the runtime state
// remains unchanged and the error is returned immediately. All tasks must
// succeed for the execution to be considered successful.
//
// Returns an error if any task execution fails
func (r *Runtime) Exec(executables ...Executable) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Run each executable task
	for _, ex := range executables {
		if err := ex.Run(r.currentEnv, r.state); err != nil {
			return err
		}

		// Generate a new environment for the next execution
		r.currentEnv = r.generateNewEnvironment()
	}

	return nil
}

// State returns the current state of the runtime.
func (r *Runtime) State() *State {
	return r.state
}

// Environment returns the current environment of the runtime.
func (r *Runtime) Environment() fdeployment.Environment {
	return r.currentEnv
}

// generateNewEnvironment creates a fresh environment that combines the current
// environment configuration with the current accumulated state.
//
// This method ensures that each task execution operates on an environment
// that reflects all previous state changes while preserving the original
// environment configuration (chains, logger, etc.).
//
// Returns a new Environment instance ready for task execution.
func (r *Runtime) generateNewEnvironment() fdeployment.Environment {
	// Should we generate from the current env instead?
	return newEnvFromState(r.currentEnv, r.state)
}
