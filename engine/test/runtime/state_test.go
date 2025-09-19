package runtime

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/Masterminds/semver/v3"
	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func Test_seedStateFromEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  fdeployment.Environment
	}{
		{
			name: "environment with all fields",
			env: fdeployment.Environment{
				ExistingAddresses: fdeployment.NewMemoryAddressBook(),
				DataStore:         fdatastore.NewMemoryDataStore().Seal(),
			},
		},
		{
			name: "environment with nil fields",
			env: fdeployment.Environment{
				ExistingAddresses: nil,
				DataStore:         nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := seedStateFromEnvironment(tt.env)

			assert.NotNil(t, got)
			assert.Equal(t, tt.env.ExistingAddresses, got.AddressBook) //nolint:staticcheck // SA1019 (Deprecated): We still need to support AddressBook for now
			assert.Equal(t, tt.env.DataStore, got.DataStore)
			assert.NotNil(t, got.Outputs)
			assert.Empty(t, got.Outputs)
		})
	}
}

func TestState_MergeChangesetOutput(t *testing.T) {
	t.Parallel()

	var (
		taskID = "test-changeset"
	)

	tests := []struct {
		name        string
		stateFunc   func() *State
		taskID      string
		output      fdeployment.ChangesetOutput
		wantErr     string
		assertState func(t *testing.T, s *State)
	}{
		{
			name: "successful merge with both datastore and address book",
			stateFunc: func() *State {
				return newState()
			},
			taskID: taskID,
			output: fdeployment.ChangesetOutput{
				DataStore:   stubTestDataStore(t),
				AddressBook: stubTestAddressBook(),
			},
			assertState: func(t *testing.T, s *State) {
				t.Helper()

				// Verify the output was stored
				assert.Contains(t, s.Outputs, taskID)

				// Verify both datastore and address book were updated
				addrs, err := s.DataStore.Addresses().Fetch()
				require.NoError(t, err)
				assert.Len(t, addrs, 1)
				assert.Equal(t, "0x1234567890123456789012345678901234567890", addrs[0].Address)

				bookAddrs, err := s.AddressBook.AddressesForChain(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector)
				require.NoError(t, err)
				assert.Len(t, bookAddrs, 1)
				assert.Equal(t, "0x1234567890123456789012345678901234567890", addrs[0].Address)
			},
		},
		{
			name: "merge with proposals",
			stateFunc: func() *State {
				return newState()
			},
			taskID: taskID,
			output: fdeployment.ChangesetOutput{
				MCMSProposals:         stubTestMCMSProposals(),
				MCMSTimelockProposals: stubTestTimelockProposals(),
			},
			assertState: func(t *testing.T, s *State) {
				t.Helper()

				// Verify the output was stored
				assert.Contains(t, s.Outputs, taskID)

				// Should have 3 proposals total (2 MCMS + 1 Timelock)
				require.Len(t, s.Proposals, 3)

				// Verify first proposal is standard MCMS proposal
				var proposal1 map[string]any
				err := json.Unmarshal([]byte(s.Proposals[0].JSON), &proposal1)
				require.NoError(t, err)
				assert.Equal(t, "Proposal", proposal1["kind"])
				assert.Equal(t, "Test MCMS Proposal 1", proposal1["description"])

				// Verify second proposal is standard MCMS proposal
				var proposal2 map[string]any
				err = json.Unmarshal([]byte(s.Proposals[1].JSON), &proposal2)
				require.NoError(t, err)
				assert.Equal(t, "Proposal", proposal2["kind"])
				assert.Equal(t, "Test MCMS Proposal 2", proposal2["description"])

				// Verify third proposal is timelock proposal
				var timelockProposal map[string]any
				err = json.Unmarshal([]byte(s.Proposals[2].JSON), &timelockProposal)
				require.NoError(t, err)
				assert.Equal(t, "TimelockProposal", timelockProposal["kind"])
				assert.Equal(t, "Test Timelock Proposal", timelockProposal["description"])
			},
		},
		{
			name: "merge with nil datastore and address book",
			stateFunc: func() *State {
				return newState()
			},
			taskID: taskID,
			output: fdeployment.ChangesetOutput{
				DataStore:   nil,
				AddressBook: nil,
			},
			assertState: func(t *testing.T, s *State) {
				t.Helper()

				// State should remain unchanged except for the output being stored
				assert.Contains(t, s.Outputs, taskID)

				dsAddrs := s.DataStore.Addresses().Filter()
				assert.Empty(t, dsAddrs)

				bookAddrs, err := s.AddressBook.Addresses()
				require.NoError(t, err)
				assert.Empty(t, bookAddrs)
			},
		},
		{
			name: "fail to merge address book",
			stateFunc: func() *State {
				return newState()
			},
			taskID: taskID,
			output: fdeployment.ChangesetOutput{
				AddressBook: createTestAddressBook(1, "0x1234567890123456789012345678901234567890"),
			},
			wantErr: "failed to update address book state",
		},
		// Note: There is no test for an error when merging the datastore because merging cannot actually fail, despite
		// the Merge method returning an error.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tt.stateFunc()

			err := state.MergeChangesetOutput(tt.taskID, tt.output)

			// Verify error expectations
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			// Validate state
			if tt.assertState != nil {
				tt.assertState(t, state)
			}
		})
	}
}

func TestState_MergeChangesetOutput_Concurrent(t *testing.T) {
	t.Parallel()

	// Setup initial state
	state := newState()

	// Number of concurrent operations
	numOps := 10
	var wg sync.WaitGroup
	errors := make([]error, numOps)

	// Pre-create all the datastores outside the goroutines to avoid require.NoError in goroutines
	outputs := make([]fdeployment.ChangesetOutput, numOps)
	for i := range numOps {
		ds, err := createTestDataStore(t, fmt.Sprintf("0x%040d", i))
		require.NoError(t, err)
		outputs[i] = fdeployment.ChangesetOutput{
			DataStore: ds,
		}
	}

	// Run concurrent updates
	for i := range numOps {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			changesetID := fmt.Sprintf("changeset-%d", idx)
			errors[idx] = state.MergeChangesetOutput(changesetID, outputs[idx])
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify no errors occurred
	for i, err := range errors {
		require.NoError(t, err, "operation %d should not error", i)
	}

	// Verify all outputs were stored
	assert.Len(t, state.Outputs, numOps, "all changeset outputs should be stored")

	// Verify all changesets are present
	for i := range numOps {
		changesetID := fmt.Sprintf("changeset-%d", i)
		assert.Contains(t, state.Outputs, changesetID)
	}
}

func TestState_GetProposal(t *testing.T) {
	t.Parallel()

	state := newState()
	propState := ProposalState{
		ID:         "test-proposal",
		JSON:       `{}`,
		IsExecuted: false,
	}

	state.Proposals = append(state.Proposals, propState)

	got, err := state.GetProposal(propState.ID)
	require.NoError(t, err)
	require.Equal(t, propState, got)

	// Not found
	got, err = state.GetProposal("test-proposal-2")
	require.Error(t, err)
	require.EqualError(t, err, "proposal not found: test-proposal-2")
	require.Equal(t, ProposalState{}, got)
}

func TestState_UpdateProposalJSON(t *testing.T) {
	t.Parallel()

	state := newState()
	proposalID := "test-proposal"
	state.Proposals = append(state.Proposals, ProposalState{
		ID:         proposalID,
		JSON:       `{}`,
		IsExecuted: false,
	})

	proposalJSON := `{"key": "value"}`

	err := state.UpdateProposalJSON(proposalID, proposalJSON)
	require.NoError(t, err)

	state.Proposals[0].JSON = proposalJSON
	require.JSONEq(t, proposalJSON, state.Proposals[0].JSON)

	// Not found
	err = state.UpdateProposalJSON("test-proposal-2", proposalJSON)
	require.Error(t, err)
	require.EqualError(t, err, "proposal not found: test-proposal-2")
}

func TestState_MarkProposalExecuted(t *testing.T) {
	t.Parallel()

	state := newState()
	proposalID := "test-proposal"

	state.Proposals = append(state.Proposals, ProposalState{
		ID:         "test-proposal",
		IsExecuted: false,
	})

	err := state.MarkProposalExecuted(proposalID)
	require.NoError(t, err)
	assert.True(t, state.Proposals[0].IsExecuted)

	// Not found
	err = state.MarkProposalExecuted("test-proposal-2")
	require.EqualError(t, err, "proposal not found: test-proposal-2")
	require.Error(t, err)
}

// Helper functions for creating test data

// createTestDataStore creates a data store with the given address.
func createTestDataStore(t *testing.T, address string) (fdatastore.MutableDataStore, error) {
	t.Helper()

	ds := fdatastore.NewMemoryDataStore()
	err := ds.Addresses().Add(fdatastore.AddressRef{
		ChainSelector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
		Address:       address,
		Type:          "TestContract",
		Version:       semver.MustParse("1.0.0"),
	})

	return ds, err
}

// stubTestDataStore creates a data store with a single default entry for testing.
func stubTestDataStore(t *testing.T) fdatastore.MutableDataStore {
	t.Helper()

	ds, err := createTestDataStore(t, "0x1234567890123456789012345678901234567890")
	require.NoError(t, err)

	return ds
}

// createTestAddressBook creates an address book with the given selector and address.
func createTestAddressBook(selector uint64, addr string) fdeployment.AddressBook {
	tv := fdeployment.NewTypeAndVersion("TestContract", *semver.MustParse("1.0.0"))

	return fdeployment.NewMemoryAddressBookFromMap(map[uint64]map[string]fdeployment.TypeAndVersion{
		selector: {
			addr: tv,
		},
	})
}

// stubTestAddressBook creates an address book with a single default entry for testing.
func stubTestAddressBook() fdeployment.AddressBook {
	return createTestAddressBook(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, "0x1234567890123456789012345678901234567890")
}

// stubTestMCMSProposals stubs a slice of MCMS proposals for testing.
func stubTestMCMSProposals() []mcmslib.Proposal {
	return []mcmslib.Proposal{
		createTestMCMSProposal("Test MCMS Proposal 1"),
		createTestMCMSProposal("Test MCMS Proposal 2"),
	}
}

// createTestMCMSProposal creates a MCMS proposal for testing.
func createTestMCMSProposal(description string) mcmslib.Proposal {
	return mcmslib.Proposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:     "v1",
			Kind:        mcmstypes.KindProposal,
			Description: description,
		},
	}
}

// stubTestTimelockProposal stubs a timelock proposal for testing.
func stubTestTimelockProposals() []mcmslib.TimelockProposal {
	return []mcmslib.TimelockProposal{
		createTestTimelockProposal(),
	}
}

// createTestTimelockProposal creates a timelock proposal for testing.
func createTestTimelockProposal() mcmslib.TimelockProposal {
	return mcmslib.TimelockProposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:     "v1",
			Kind:        mcmstypes.KindTimelockProposal,
			Description: "Test Timelock Proposal",
		},
	}
}
