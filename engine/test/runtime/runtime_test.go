package runtime

import (
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/Masterminds/semver/v3"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/environment"
)

func TestNew(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)

	runtime, err := New(t.Context(), WithEnvOpts(
		environment.WithLogger(lggr),
	))
	require.NoError(t, err)
	require.NotNil(t, runtime)

	require.Equal(t, lggr, runtime.currentEnv.Logger)
}

func TestNewFromEnvironment(t *testing.T) {
	t.Parallel()

	env := fdeployment.Environment{
		Name:              "test-env",
		Logger:            logger.Test(t),
		ExistingAddresses: fdeployment.NewMemoryAddressBook(),
		DataStore:         fdatastore.NewMemoryDataStore().Seal(),
	}

	runtime := NewFromEnvironment(env)

	require.NotNil(t, runtime)
	require.NotNil(t, runtime.state)
	assert.Equal(t, env.ExistingAddresses, runtime.state.AddressBook) //nolint:staticcheck // SA1019 (Deprecated): We still need to support AddressBook for now
	assert.Equal(t, env.DataStore, runtime.state.DataStore)
	require.NotNil(t, runtime.state.Outputs)
	assert.Empty(t, runtime.state.Outputs)
	assert.Equal(t, env, runtime.currentEnv)
}

func TestRuntime_Exec(t *testing.T) {
	t.Parallel()

	var (
		taskID1 = "task-1"
		taskID2 = "task-2"
		taskID3 = "task-3"
	)

	tests := []struct {
		name            string
		runtimeFunc     func() *Runtime
		executablesFunc func(t *testing.T) []Executable
		wantErr         string
		assertRuntime   func(t *testing.T, r *Runtime)
	}{
		{
			name: "successful execution of single task",
			runtimeFunc: func() *Runtime {
				return NewFromEnvironment(fdeployment.Environment{
					Name:              "test-env",
					Logger:            logger.Nop(),
					ExistingAddresses: fdeployment.NewMemoryAddressBook(),
					DataStore:         fdatastore.NewMemoryDataStore().Seal(),
				})
			},
			executablesFunc: func(t *testing.T) []Executable {
				t.Helper()

				task1 := NewMockExecutable(t)
				task1.EXPECT().Run(
					mock.IsType(fdeployment.Environment{}), mock.IsType(&State{}),
				).RunAndReturn(func(e fdeployment.Environment, state *State) error {
					ds := fdatastore.NewMemoryDataStore()
					err := ds.Addresses().Add(fdatastore.AddressRef{
						ChainSelector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
						Address:       "0x1234567890123456789012345678901234567890",
						Type:          "TestContract",
						Version:       semver.MustParse("1.0.0"),
					})
					require.NoError(t, err)

					return state.MergeChangesetOutput(taskID1, fdeployment.ChangesetOutput{
						DataStore: ds,
					})
				})

				return []Executable{task1}
			},
			assertRuntime: func(t *testing.T, r *Runtime) {
				t.Helper()

				// Verify task output was stored
				assert.Contains(t, r.state.Outputs, taskID1)

				// Verify datastore was updated
				addrs, err := r.state.DataStore.Addresses().Fetch()
				require.NoError(t, err)
				assert.Len(t, addrs, 1)
				assert.Equal(t, "0x1234567890123456789012345678901234567890", addrs[0].Address)
			},
		},
		{
			name: "successful execution of multiple tasks",
			runtimeFunc: func() *Runtime {
				return NewFromEnvironment(fdeployment.Environment{
					Name:              "test-env",
					Logger:            logger.Nop(),
					ExistingAddresses: fdeployment.NewMemoryAddressBook(),
					DataStore:         fdatastore.NewMemoryDataStore().Seal(),
				})
			},
			executablesFunc: func(t *testing.T) []Executable {
				t.Helper()

				task1 := NewMockExecutable(t)
				task1.EXPECT().Run(
					mock.IsType(fdeployment.Environment{}), mock.IsType(&State{}),
				).RunAndReturn(func(e fdeployment.Environment, state *State) error {
					ds := fdatastore.NewMemoryDataStore()
					err := ds.Addresses().Add(fdatastore.AddressRef{
						ChainSelector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
						Address:       "0x1111111111111111111111111111111111111111",
						Type:          "TestContract",
						Version:       semver.MustParse("1.0.0"),
					})
					require.NoError(t, err)

					return state.MergeChangesetOutput(taskID1, fdeployment.ChangesetOutput{
						DataStore: ds,
					})
				})

				task2 := NewMockExecutable(t)
				task2.EXPECT().Run(
					// We expect the environment's datastore to have the address from task1
					mock.MatchedBy(func(e fdeployment.Environment) bool {
						addrs, err := e.DataStore.Addresses().Fetch()
						if err != nil {
							return false
						}

						return assert.Len(t, addrs, 1) && assert.Equal(t, "0x1111111111111111111111111111111111111111", addrs[0].Address)
					}),
					// mock.IsType(&State{}),
					// We expect the state to have the output from task1
					mock.MatchedBy(func(state *State) bool {
						_, ok := state.Outputs[taskID1]
						return ok
					}),
				).RunAndReturn(func(e fdeployment.Environment, state *State) error {
					ds := fdatastore.NewMemoryDataStore()
					err := ds.Addresses().Add(fdatastore.AddressRef{
						ChainSelector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
						Address:       "0x2222222222222222222222222222222222222222",
						Type:          "Contract2",
						Version:       semver.MustParse("1.0.0"),
					})
					require.NoError(t, err)

					return state.MergeChangesetOutput(taskID2, fdeployment.ChangesetOutput{
						DataStore: ds,
					})
				})

				return []Executable{task1, task2}
			},
			assertRuntime: func(t *testing.T, r *Runtime) {
				t.Helper()

				// Verify both task outputs were stored
				assert.Contains(t, r.state.Outputs, taskID1)
				assert.Contains(t, r.state.Outputs, taskID2)

				// Verify both addresses were added to datastore
				addrs, err := r.state.DataStore.Addresses().Fetch()
				require.NoError(t, err)
				assert.Len(t, addrs, 2)

				addresses := make([]string, len(addrs))
				for i, addr := range addrs {
					addresses[i] = addr.Address
				}
				assert.Contains(t, addresses, "0x1111111111111111111111111111111111111111")
				assert.Contains(t, addresses, "0x2222222222222222222222222222222222222222")
			},
		},
		{
			name: "task execution failure stops execution",
			runtimeFunc: func() *Runtime {
				return NewFromEnvironment(fdeployment.Environment{
					Name:   "test-env",
					Logger: logger.Nop(),
				})
			},
			executablesFunc: func(t *testing.T) []Executable {
				t.Helper()

				task1 := NewMockExecutable(t)
				task1.EXPECT().Run(
					mock.IsType(fdeployment.Environment{}), mock.IsType(&State{}),
				).RunAndReturn(func(e fdeployment.Environment, state *State) error {
					return state.MergeChangesetOutput(taskID1, fdeployment.ChangesetOutput{})
				})

				task2 := NewMockExecutable(t)
				task2.EXPECT().Run(
					mock.IsType(fdeployment.Environment{}), mock.IsType(&State{}),
				).Return(errors.New("task execution failed"))

				task3 := NewMockExecutable(t)
				task3.EXPECT().Run(
					mock.IsType(fdeployment.Environment{}), mock.IsType(&State{}),
				).RunAndReturn(func(e fdeployment.Environment, state *State) error {
					return state.MergeChangesetOutput(taskID3, fdeployment.ChangesetOutput{})
				}).Maybe() // We expect this task to not be executed, but leave this as Maybe in case it is executed and the assertion will catch the error.

				return []Executable{task1, task2, task3}
			},
			wantErr: "task execution failed",
			assertRuntime: func(t *testing.T, r *Runtime) {
				t.Helper()

				// Verify only the first task was executed
				assert.Contains(t, r.state.Outputs, taskID1)
				assert.NotContains(t, r.state.Outputs, taskID2)
				assert.NotContains(t, r.state.Outputs, taskID3)
			},
		},
		{
			name: "empty executables list",
			runtimeFunc: func() *Runtime {
				return NewFromEnvironment(fdeployment.Environment{
					Name:   "test-env",
					Logger: logger.Nop(),
				})
			},
			executablesFunc: func(t *testing.T) []Executable {
				t.Helper()

				return []Executable{}
			},
			assertRuntime: func(t *testing.T, r *Runtime) {
				t.Helper()

				// State should remain unchanged
				assert.Empty(t, r.state.Outputs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			runtime := tt.runtimeFunc()
			executables := tt.executablesFunc(t)

			// Execute
			err := runtime.Exec(executables...)

			// Verify error expectations
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			// Validate result
			if tt.assertRuntime != nil {
				tt.assertRuntime(t, runtime)
			}
		})
	}
}

func TestRuntime_Exec_Concurrent(t *testing.T) {
	t.Parallel()

	// Setup runtime
	env := fdeployment.Environment{
		Name:   "test-env",
		Logger: logger.Test(t),
	}
	runtime := NewFromEnvironment(env)

	// Number of concurrent operations
	numOps := 10
	var wg sync.WaitGroup
	errors := make([]error, numOps)

	// Pre-create all the executables to avoid issues in goroutines
	executables := make([]Executable, numOps)
	for i := range numOps {
		// These tasks create empty outputs so we can verify that the state was updated
		task := NewMockExecutable(t)
		task.EXPECT().Run(
			mock.IsType(fdeployment.Environment{}), mock.IsType(&State{}),
		).RunAndReturn(func(e fdeployment.Environment, state *State) error {
			return state.MergeChangesetOutput("task-"+strconv.Itoa(i), fdeployment.ChangesetOutput{})
		})

		executables[i] = task
	}

	// Run concurrent executions
	for i := range numOps {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Each goroutine executes its own task
			errors[idx] = runtime.Exec(executables[idx])
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify all operations succeeded (due to mutex protection)
	for i, err := range errors {
		require.NoError(t, err, "operation %d should not error", i)
	}

	require.Len(t, runtime.state.Outputs, numOps)
}

func TestRuntime_State(t *testing.T) {
	t.Parallel()

	runtime := NewFromEnvironment(fdeployment.Environment{
		Name:              "test-env",
		Logger:            logger.Nop(),
		ExistingAddresses: fdeployment.NewMemoryAddressBook(),
		DataStore:         fdatastore.NewMemoryDataStore().Seal(),
	})

	require.NotNil(t, runtime.State())
	assert.Equal(t, runtime.State(), runtime.state)
}

func TestRuntime_Environment(t *testing.T) {
	t.Parallel()

	runtime := NewFromEnvironment(fdeployment.Environment{
		Name:   "test-env",
		Logger: logger.Nop(),
	})

	require.NotNil(t, runtime.Environment())
	assert.Equal(t, runtime.Environment(), runtime.currentEnv)
}
