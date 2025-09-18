package runtime

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// mockChangeset implements fdeployment.ChangeSetV2 for testing
type mockChangeset[C any] struct {
	applyFunc               func(e fdeployment.Environment, config C) (fdeployment.ChangesetOutput, error)
	verifyPreconditionsFunc func(e fdeployment.Environment, config C) error
}

func (m *mockChangeset[C]) Apply(e fdeployment.Environment, config C) (fdeployment.ChangesetOutput, error) {
	if m.applyFunc != nil {
		return m.applyFunc(e, config)
	}

	return fdeployment.ChangesetOutput{}, nil
}

func (m *mockChangeset[C]) VerifyPreconditions(e fdeployment.Environment, config C) error {
	if m.verifyPreconditionsFunc != nil {
		return m.verifyPreconditionsFunc(e, config)
	}

	return nil
}

// mockChangesetConfig is a simple config type for testing
type mockChangesetConfig struct {
	Value string
}

func TestChangeTask(t *testing.T) {
	t.Parallel()

	t.Run("creates task with unique ID", func(t *testing.T) {
		t.Parallel()

		changeset := &mockChangeset[mockChangesetConfig]{}
		config := mockChangesetConfig{Value: "test"}

		task := ChangesetTask(changeset, config)

		// Verify task has a non-empty ID
		assert.NotEmpty(t, task.ID())

		// Verify config and changeset are set
		assert.Equal(t, config, task.config)
		assert.Equal(t, changeset, task.changeset)
	})

	t.Run("generates different IDs for different tasks", func(t *testing.T) {
		t.Parallel()

		changeset := &mockChangeset[mockChangesetConfig]{}
		config := mockChangesetConfig{Value: "test"}

		task1 := ChangesetTask(changeset, config)
		task2 := ChangesetTask(changeset, config)

		assert.NotEqual(t, task1.ID(), task2.ID())
	})
}

func TestChangeTask_ID(t *testing.T) {
	t.Parallel()

	changeset := &mockChangeset[mockChangesetConfig]{}
	config := mockChangesetConfig{Value: "test"}
	task := ChangesetTask(changeset, config)

	// ID should be consistent across multiple calls
	id1 := task.ID()
	id2 := task.ID()
	assert.Equal(t, id1, id2)
	assert.NotEmpty(t, id1)
}

func TestChangeTask_Run(t *testing.T) {
	t.Parallel()

	var (
		ds = fdatastore.NewMemoryDataStore()
	)
	err := ds.AddressRefStore.Add(fdatastore.AddressRef{
		ChainSelector: 1,
		Address:       "0x123",
		Type:          fdatastore.ContractType("ERC20"),
		Version:       semver.MustParse("1.0.0"),
		Qualifier:     "1",
	})
	require.NoError(t, err)

	tests := []struct {
		name          string
		changesetFunc func() *mockChangeset[mockChangesetConfig]
		wantErr       string
		assertOutput  func(t *testing.T, state *State, output fdeployment.ChangesetOutput)
	}{
		{
			name: "successful execution",
			changesetFunc: func() *mockChangeset[mockChangesetConfig] {
				expectedOutput := fdeployment.ChangesetOutput{
					DataStore: ds,
				}

				return &mockChangeset[mockChangesetConfig]{
					applyFunc: func(e fdeployment.Environment, config mockChangesetConfig) (fdeployment.ChangesetOutput, error) {
						return expectedOutput, nil
					},
				}
			},
			assertOutput: func(t *testing.T, state *State, output fdeployment.ChangesetOutput) {
				t.Helper()

				// Ensure the datastore is updated in the output
				assert.Equal(t, ds, output.DataStore)

				// Ensure the datastore is updated in the state
				assert.Equal(t, output.DataStore.Seal(), state.DataStore)
			},
		},
		{
			name: "precondition verification failure",
			changesetFunc: func() *mockChangeset[mockChangesetConfig] {
				return &mockChangeset[mockChangesetConfig]{
					verifyPreconditionsFunc: func(e fdeployment.Environment, config mockChangesetConfig) error {
						return errors.New("precondition failed")
					},
				}
			},
			wantErr: "precondition failed",
		},
		{
			name: "changeset apply failure",
			changesetFunc: func() *mockChangeset[mockChangesetConfig] {
				return &mockChangeset[mockChangesetConfig]{
					applyFunc: func(e fdeployment.Environment, config mockChangesetConfig) (fdeployment.ChangesetOutput, error) {
						return fdeployment.ChangesetOutput{}, errors.New("apply failed")
					},
				}
			},
			wantErr: "apply failed",
		},
		{
			name: "state update with nil datastore",
			changesetFunc: func() *mockChangeset[mockChangesetConfig] {
				return &mockChangeset[mockChangesetConfig]{
					applyFunc: func(e fdeployment.Environment, config mockChangesetConfig) (fdeployment.ChangesetOutput, error) {
						return fdeployment.ChangesetOutput{
							DataStore: nil, // This might not cause an error
						}, nil
					},
				}
			},
			assertOutput: func(t *testing.T, state *State, output fdeployment.ChangesetOutput) {
				t.Helper()

				assert.NotNil(t, ds, state.DataStore)
				assert.Nil(t, output.DataStore)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			changeset := tt.changesetFunc()
			config := mockChangesetConfig{Value: "test"}
			task := ChangesetTask(changeset, config)

			env := fdeployment.Environment{}
			state := &State{
				AddressBook: fdeployment.NewMemoryAddressBook(),
				DataStore:   fdatastore.NewMemoryDataStore().Seal(),
				Outputs:     make(map[string]fdeployment.ChangesetOutput),
			}

			// Execute
			err := task.Run(env, state)

			// Verify error expectations
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.NotContains(t, state.Outputs, task.ID())
			} else {
				require.NoError(t, err)
				assert.Contains(t, state.Outputs, task.ID())

				if tt.assertOutput != nil {
					tt.assertOutput(t, state, state.Outputs[task.ID()])
				}
			}
		})
	}
}

func TestChangeTask_ApplyChangeset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		changesetFunc func() *mockChangeset[mockChangesetConfig]
		config        mockChangesetConfig
		wantErr       string
		assertOutput  func(t *testing.T, output fdeployment.ChangesetOutput)
	}{
		{
			name: "successful application",
			changesetFunc: func() *mockChangeset[mockChangesetConfig] {
				expectedOutput := fdeployment.ChangesetOutput{
					DataStore: fdatastore.NewMemoryDataStore(),
				}

				return &mockChangeset[mockChangesetConfig]{
					applyFunc: func(e fdeployment.Environment, config mockChangesetConfig) (fdeployment.ChangesetOutput, error) {
						assert.Equal(t, "test-value", config.Value)
						return expectedOutput, nil
					},
				}
			},
			config: mockChangesetConfig{Value: "test-value"},
			assertOutput: func(t *testing.T, output fdeployment.ChangesetOutput) {
				t.Helper()
				assert.NotNil(t, output.DataStore)
			},
		},
		{
			name: "precondition failure",
			changesetFunc: func() *mockChangeset[mockChangesetConfig] {
				return &mockChangeset[mockChangesetConfig]{
					verifyPreconditionsFunc: func(e fdeployment.Environment, config mockChangesetConfig) error {
						return errors.New("precondition check failed")
					},
				}
			},
			config:  mockChangesetConfig{Value: "test"},
			wantErr: "precondition check failed",
			assertOutput: func(t *testing.T, output fdeployment.ChangesetOutput) {
				t.Helper()
				assert.Equal(t, fdeployment.ChangesetOutput{}, output)
			},
		},
		{
			name: "apply failure",
			changesetFunc: func() *mockChangeset[mockChangesetConfig] {
				return &mockChangeset[mockChangesetConfig]{
					applyFunc: func(e fdeployment.Environment, config mockChangesetConfig) (fdeployment.ChangesetOutput, error) {
						return fdeployment.ChangesetOutput{}, errors.New("apply operation failed")
					},
				}
			},
			config:  mockChangesetConfig{Value: "test"},
			wantErr: "apply operation failed",
			assertOutput: func(t *testing.T, output fdeployment.ChangesetOutput) {
				t.Helper()
				assert.Equal(t, fdeployment.ChangesetOutput{}, output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			changeset := tt.changesetFunc()
			task := ChangesetTask(changeset, tt.config)
			env := fdeployment.Environment{}

			// Execute
			output, err := task.applyChangeset(env)

			// Verify error expectations
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			// Validate output
			if tt.assertOutput != nil {
				tt.assertOutput(t, output)
			}
		})
	}
}
