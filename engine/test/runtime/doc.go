// Package runtime provides an execution environment for executing changesets in tests.
//
// The runtime package is the core testing infrastructure that manages state accumulation
// across multiple task executions, ensuring that each task operates on a fresh environment
// that reflects the cumulative state changes from previous executions. This enables
// comprehensive integration testing of deployment workflows where multiple changesets
// or operations need to be executed in sequence.
//
// # Core Concepts
//
// The runtime operates on three main concepts:
//
//   - Runtime: The main orchestrator that manages task execution and state accumulation
//   - Executable: An interface that represents any task that can be executed by the runtime
//   - State: Internal state that accumulates changes from task executions
//
// # Thread Safety
//
// The runtime is thread-safe and ensures sequential execution of tasks through mutex
// protection. This guarantees that state consistency is maintained even when multiple
// goroutines attempt to execute tasks concurrently.
//
// # State Management
//
// Each task execution follows this process:
//  1. Execute the task against the current environment
//  2. Task is responsible for updating the runtime state
//  3. Generate a new environment incorporating the updated state
//  4. Proceed to the next task
//
// The state accumulates:
//   - Address book entries from changeset deployments
//   - DataStore updates including contract addresses and metadata
//   - Changeset outputs for debugging and verification
//
// # Basic Usage
//
// Here's a simple example of using the runtime to execute a changeset:
//
//		import (
//			"testing"
//
//			"github.com/stretchr/testify/require"
//
//			testenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/test/environment"
//			"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/runtime"
//		)
//
//		func TestMyDeployment(t *testing.T) {
//			// Create runtime instance with a simulated EVM blockchain
//			runtime := New(t.Context(), WithEnvOpts(
//	        	testenv.WithEVMSimulatedN(t, 1)
//	     	))
//
//			// Execute a changeset
//			task := ChangesetTask(myChangeset, MyChangesetConfig{
//				Parameter1: "value1",
//				Parameter2: 42,
//			})
//
//			err := runtime.Exec(task)
//			require.NoError(t, err)
//
//			// Verify deployment results
//			addrs, err := runtime.State().DataStore.Addresses().Fetch()
//			require.NoError(t, err)
//			assert.Len(t, addrs, 1)
//		}
//
// # Sequential Changeset Execution
//
// The runtime handles executing multiple changesets in sequence, where later changesets
// depend on the results of earlier ones. The environment provided to the later changesets
// will include the data of the previous changesets execution.
//
//	func TestMultiStepDeployment(t *testing.T) {
//		// Create runtime instance with a simulated EVM blockchain
//		runtime := New(t.Context(), WithEnvOpts(
//			testenv.WithEVMSimulatedN(t, 1)
//	    ))
//
//		// Define the first changeset
//		coreTask := ChangesetTask(coreChangeset, CoreConfig{})
//
//		// Define the dependent second changeset
//		// The environment provided to the dependent changeset will include the data of the previous changesets execution
//		dependentTask := ChangesetTask(dependentChangeset, DependentConfig{})
//
//		// Execute in sequence - each task sees the results of previous tasks
//		err := runtime.Exec(coreTask, dependentTask)
//		require.NoError(t, err)
//
//		// Verify final state contains all deployed contracts
//		addrs, err := runtime.State().DataStore.Addresses().Fetch()
//		require.NoError(t, err)
//		assert.Len(t, addrs, 1)
//	}
//
// # Environment Loading Options
//
// The engine/test/environment package provides various blockchain loading options:
//
//	// EVM simulated blockchains (fast, in-memory)
//	env, err := testenv.New(ctx, testenv.WithEVMSimulatedN(t, 2))
//
//	// Specific chain selectors
//	env, err := testenv.New(ctx, testenv.WithEVMSimulated(t, []uint64{
//		chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
//		chainsel.POLYGON_TESTNET_MUMBAI.Selector,
//	}))
//
//	// Multiple blockchain types
//	env, err := testenv.New(ctx,
//		testenv.WithEVMSimulatedN(t, 1),
//		testenv.WithSolanaContainerN(t, 1, "/path/to/programs", programIDs),
//		testenv.WithTonContainerN(t, 1),
//	)
//
//	// Custom EVM configuration
//	cfg := onchain.EVMSimLoaderConfig{
//		ChainID:    1337,
//		BlockTime:  time.Second,
//	}
//	env, err := testenv.New(t, testenv.WithEVMSimulatedWithConfigN(t, 1, cfg))
//
// # Error Handling
//
// Task execution is atomic - if any task fails, the runtime state remains unchanged
// and subsequent tasks are not executed:
//
//	func TestErrorHandling(t *testing.T) {
//		// Load test environment
//		env, err := testenv.New(t, testenv.WithEVMSimulatedN(t, 1))
//		require.NoError(t, err)
//
//		runtime := NewFromEnvironment(*env)
//
//		successTask := ChangesetTask(workingChangeset, WorkingConfig{})
//		failingTask := ChangesetTask(brokenChangeset, BrokenConfig{})
//		shouldNotRunTask := ChangesetTask(afterFailureChangeset, AfterFailureConfig{})
//
//		err := runtime.Exec(successTask, failingTask, shouldNotRunTask)
//		require.Error(t, err)
//		assert.Contains(t, err.Error(), "expected failure message")
//
//		// Only the successful task's output should be in state
//		outputs := runtime.State().Outputs
//		assert.Contains(t, outputs, "working-changeset-output")
//		assert.NotContains(t, outputs, "broken-changeset-output")
//		assert.NotContains(t, outputs, "after-failure-output")
//	}
//
// # State Inspection
//
// The runtime provides access to accumulated state for verification and debugging:
//
//	func TestStateInspection(t *testing.T) {
//		// Load test environment
//		env, err := testenv.New(t, testenv.WithEVMSimulatedN(t, 1))
//		require.NoError(t, err)
//
//		runtime := NewFromEnvironment(*env)
//
//		// Execute some tasks
//		err := runtime.Exec(task1, task2, task3)
//		require.NoError(t, err)
//
//		// Inspect final state
//		runtimeState := runtime.State()
//
//		// Check deployed addresses
//		addrs, err := runtimeState.DataStore.Addresses().Fetch()
//		require.NoError(t, err)
//
//		// Check address book entries (legacy)
//		addressBook := runtimeState.AddressBook
//
//		// Check task execution outputs
//		taskOutputs := runtimeState.Outputs
//		assert.Contains(t, taskOutputs, "task1-id")
//		assert.Contains(t, taskOutputs, "task2-id")
//		assert.Contains(t, taskOutputs, "task3-id")
//
//		// Access current environment (reflects all changes)
//		currentEnv := runtime.Environment()
//		finalAddrs, err := currentEnv.DataStore.Addresses().Fetch()
//		require.NoError(t, err)
//		assert.Equal(t, addrs, finalAddrs)
//	}
//
// # Integration with Testing Framework
//
// The runtime integrates seamlessly with Go's testing framework and popular assertion
// libraries like testify:
//
//	func TestIntegrationExample(t *testing.T) {
//		t.Parallel() // Runtime is thread-safe
//
//		// Subtests can each have their own runtime instance
//		t.Run("deployment_scenario_1", func(t *testing.T) {
//			env, err := testenv.New(ctx, testenv.WithEVMSimulatedN(t, 1))
//			require.NoError(t, err)
//
//			runtime := NewFromEnvironment(*env)
//			err = runtime.Exec(scenario1Tasks...)
//			require.NoError(t, err)
//			// scenario1-specific assertions
//		})
//
//		t.Run("deployment_scenario_2", func(t *testing.T) {
//			env, err := testenv.New(t, testenv.WithEVMSimulatedN(t, 2))
//			require.NoError(t, err)
//
//			runtime := NewFromEnvironment(*env)
//			err = runtime.Exec(scenario2Tasks...)
//			require.NoError(t, err)
//			// scenario2-specific assertions
//		})
//	}
//
// # MCMS Proposals
//
// The runtime provides specialized tasks for handling Multi-Chain Multi-Sig (MCMS) proposals.
//
// ## Available MCMS Tasks
//
// **SignProposalTask(proposalID, signingKeys...)** - Signs MCMS or Timelock proposals with
// one or more private keys. The task automatically detects the proposal type and applies
// the appropriate signing logic. Multiple signatures can be added in a single operation
// or accumulated across multiple signing tasks.
//
// **ExecuteProposalTask(proposalID)** - Executes a signed MCMS or Timelock proposal on
// the target blockchain networks. The task handles chain-specific configurations, sets
// merkle roots, executes operations sequentially, and confirms all transactions. For
// Timelock proposals, it manages scheduling and delay mechanisms.
//
// **SignAndExecuteProposalsTask(signingKeys)** - Processes all pending proposals in the
// runtime state by signing them with the provided keys and executing them in batch.
// This is useful for scenarios where multiple changesets generate proposals that need
// to be processed together or you want to just execute all pending proposals.
//
// ## Usage Pattern
//
// MCMS proposals follow a three-phase workflow:
//  1. **Generation**: Changesets create proposals and store them in runtime state. This is done by running a ChangesetTask.
//  2. **Signing**: Use SignProposalTask to add cryptographic signatures
//  3. **Execution**: Use ExecuteProposalTask to apply changes on target chains
//
// The runtime automatically manages proposal state, tracking which proposals are pending,
// signed, or executed. Both standard MCMS proposals and Timelock proposals are supported
// through the same task interface.
//
// # Best Practices
//
//   - Create a new runtime instance for each test to ensure isolation.
//   - Be wary when using containerized chains as they take a long time to start containers. It is better to write a
//     longer test with a single containerized chain than to write a shorter test with multiple containerized chains.
//   - Verify both intermediate and final state in multi-step tests
//   - Leverage the runtime's error handling for negative test cases
//   - Use SignAndExecuteProposalsTask for processing, avoiding the need to fetch the proposal ID from state.
//
// # Common Patterns
//
// ## Setup Environment
//
//	func setupTestEnvironment(t *testing.T) *fdeployment.Environment {
//		t.Helper()
//
//		loader := testenv.NewLoader()
//		env, err := loader.Load(t, testenv.WithEVMSimulatedN(t, 1))
//		require.NoError(t, err)
//
//		return env
//	}
//
// ## Task Factory Pattern
//
//	func DeployTokenTask(config TokenConfig) Executable {
//		return ChangesetTask(tokenChangeset, config)
//	}
package runtime
