package runtime

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	chainselectors "github.com/smartcontractkit/chain-selectors"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fchainevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/internal/mcmsutils"
)

func TestSignProposalTask_ID(t *testing.T) {
	t.Parallel()

	// Generate a test private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	proposalID := "test-proposal-1"
	task := SignProposalTask(proposalID, privateKey)

	id1 := task.ID()
	assert.Equal(t, id1, task.ID())
}

func TestSignProposalTask_Run(t *testing.T) {
	t.Parallel()

	// Generate a test private key
	privateKey1, err := crypto.GenerateKey()
	require.NoError(t, err)

	privateKey2, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := []struct {
		name           string
		setupState     func(t *testing.T) (*State, string)
		signingKeys    []*ecdsa.PrivateKey
		wantErr        string
		validateResult func(t *testing.T, state *State, taskID string)
	}{
		{
			name: "successfully signs MCMS proposal",
			setupState: func(t *testing.T) (*State, string) {
				t.Helper()

				// Create a test MCMS proposal
				proposal := createTestMCMSProposalForSigning(t)
				propState, err := newMCMSProposalState(&proposal)
				require.NoError(t, err)

				state := newState()
				state.Proposals = []ProposalState{propState}

				return state, propState.ID
			},
			signingKeys: []*ecdsa.PrivateKey{privateKey1},
			validateResult: func(t *testing.T, state *State, taskID string) {
				t.Helper()

				// Find the updated proposal
				require.Len(t, state.Proposals, 1)
				updatedProp := state.Proposals[0]

				// Decode and verify the proposal was signed
				prop, err := mcmsutils.DecodeProposal(updatedProp.JSON)
				require.NoError(t, err)
				assert.Len(t, prop.Signatures, 1)
			},
		},
		{
			name: "successfully signs MCMS proposal with multiple signing keys",
			setupState: func(t *testing.T) (*State, string) {
				t.Helper()

				// Create a test MCMS proposal
				proposal := createTestMCMSProposalForSigning(t)
				propState, err := newMCMSProposalState(&proposal)
				require.NoError(t, err)

				state := newState()
				state.Proposals = []ProposalState{propState}

				return state, propState.ID
			},
			signingKeys: []*ecdsa.PrivateKey{privateKey1, privateKey2},
			validateResult: func(t *testing.T, state *State, taskID string) {
				t.Helper()

				// Find the updated proposal
				require.Len(t, state.Proposals, 1)
				updatedProp := state.Proposals[0]

				// Decode and verify the proposal was signed
				prop, err := mcmsutils.DecodeProposal(updatedProp.JSON)
				require.NoError(t, err)
				assert.Len(t, prop.Signatures, 2)
				// Ensure the signatures are different
				assert.NotEqual(t, prop.Signatures[0].R, prop.Signatures[1].R)
				assert.NotEqual(t, prop.Signatures[0].S, prop.Signatures[1].S)
			},
		},
		{
			name: "successfully signs Timelock proposal",
			setupState: func(t *testing.T) (*State, string) {
				t.Helper()

				// Create a test Timelock proposal
				proposal := createTestTimelockProposalForSigning(t)
				propState, err := newTimelockProposalState(&proposal)
				require.NoError(t, err)

				state := newState()
				state.Proposals = []ProposalState{propState}

				return state, propState.ID
			},
			signingKeys: []*ecdsa.PrivateKey{privateKey1},
			validateResult: func(t *testing.T, state *State, taskID string) {
				t.Helper()

				// Find the updated proposal
				require.Len(t, state.Proposals, 1)
				updatedProp := state.Proposals[0]

				// Decode and verify the proposal was signed
				prop, err := mcmsutils.DecodeTimelockProposal(updatedProp.JSON)
				require.NoError(t, err)
				assert.Len(t, prop.Signatures, 1)
			},
		},
		{
			name: "fails when proposal not found",
			setupState: func(t *testing.T) (*State, string) {
				t.Helper()

				state := newState()
				// Don't add any proposals to the state

				return state, ""
			},
			signingKeys: []*ecdsa.PrivateKey{privateKey1},
			wantErr:     "proposal not found",
		},
		{
			name: "fails with invalid proposal JSON",
			setupState: func(t *testing.T) (*State, string) {
				t.Helper()

				// Create a proposal state with invalid JSON
				propState := ProposalState{
					ID:   "test-proposal-id",
					JSON: "invalid-json",
				}

				state := newState()
				state.Proposals = []ProposalState{propState}

				return state, propState.ID
			},
			signingKeys: []*ecdsa.PrivateKey{privateKey1},
			wantErr:     "invalid character", // JSON parsing error
		},
		{
			name: "fails with unsupported proposal kind",
			setupState: func(t *testing.T) (*State, string) {
				t.Helper()

				// Create a proposal state with unsupported kind
				invalidProposal := map[string]interface{}{
					"kind":    "UnsupportedKind",
					"version": "v1",
				}
				jsonBytes, err := json.Marshal(invalidProposal)
				require.NoError(t, err)

				propState := ProposalState{
					ID:   "test-proposal-id",
					JSON: string(jsonBytes),
				}

				state := newState()
				state.Proposals = []ProposalState{propState}

				return state, propState.ID
			},
			signingKeys: []*ecdsa.PrivateKey{privateKey1},
			wantErr:     "unsupported proposal kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state, proposalID := tt.setupState(t)
			env := createTestEnvironment(t)

			task := SignProposalTask(proposalID, tt.signingKeys...)
			taskID := task.ID()

			err := task.Run(env, state)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err, "Task should succeed")

				if tt.validateResult != nil {
					tt.validateResult(t, state, taskID)
				}
			}
		})
	}
}

func TestSignProposal(t *testing.T) {
	t.Parallel()

	// Generate a test private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tests := []struct {
		name           string
		setupPropState func(t *testing.T) ProposalState
		wantErr        string
		validateResult func(t *testing.T, result string)
	}{
		{
			name: "successfully signs valid MCMS proposal",
			setupPropState: func(t *testing.T) ProposalState {
				t.Helper()

				proposal := createTestMCMSProposalForSigning(t)
				propState, err := newMCMSProposalState(&proposal)
				require.NoError(t, err)

				return propState
			},
			validateResult: func(t *testing.T, result string) {
				t.Helper()

				// Decode the result and verify it's signed
				prop, err := mcmsutils.DecodeProposal(result)
				require.NoError(t, err)
				assert.Len(t, prop.Signatures, 1, "Proposal should have exactly one signature")

				// Verify signature components
				sig := prop.Signatures[0]
				assert.NotEmpty(t, sig.R, "Signature R component should not be empty")
				assert.NotEmpty(t, sig.S, "Signature S component should not be empty")
			},
		},
		{
			name: "fails with invalid proposal JSON",
			setupPropState: func(t *testing.T) ProposalState {
				t.Helper()

				return ProposalState{
					ID:   "test-proposal-id",
					JSON: "invalid-json-content",
				}
			},
			wantErr: "failed to decode proposal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			propState := tt.setupPropState(t)

			// Create signer
			signer, err := mcmsutils.NewSigner()
			require.NoError(t, err)

			result, err := signProposal(t.Context(), propState.JSON, privateKey, signer)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, result, "Result should be empty when error occurs")
			} else {
				require.NoError(t, err, "Signing should succeed")
				assert.NotEmpty(t, result, "Result should not be empty")

				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}
		})
	}
}

func TestSignTimelockProposal(t *testing.T) {
	t.Parallel()

	// Generate a test private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tests := []struct {
		name           string
		setupPropState func(t *testing.T) ProposalState
		wantErr        string
		validateResult func(t *testing.T, result string)
	}{
		{
			name: "successfully signs valid timelock proposal",
			setupPropState: func(t *testing.T) ProposalState {
				t.Helper()

				proposal := createTestTimelockProposalForSigning(t)
				propState, err := newTimelockProposalState(&proposal)
				require.NoError(t, err)

				return propState
			},
			validateResult: func(t *testing.T, result string) {
				t.Helper()

				// Decode the result and verify it's signed
				prop, err := mcmsutils.DecodeTimelockProposal(result)
				require.NoError(t, err)
				assert.Len(t, prop.Signatures, 1, "Timelock proposal should have exactly one signature")

				// Verify signature components
				sig := prop.Signatures[0]
				assert.NotEmpty(t, sig.R, "Signature R component should not be empty")
				assert.NotEmpty(t, sig.S, "Signature S component should not be empty")

				// Verify proposal properties are preserved
				assert.Equal(t, mcmstypes.TimelockActionSchedule, prop.Action)
				assert.NotEmpty(t, prop.Operations, "Operations should be preserved")
			},
		},
		{
			name: "fails with invalid proposal JSON",
			setupPropState: func(t *testing.T) ProposalState {
				t.Helper()

				return ProposalState{
					ID:   "test-timelock-proposal-id",
					JSON: "invalid-json-content",
				}
			},
			wantErr: "failed to decode timelock proposal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			propState := tt.setupPropState(t)

			// Create signer
			signer, err := mcmsutils.NewSigner()
			require.NoError(t, err)

			result, err := signTimelockProposal(t.Context(), propState.JSON, privateKey, signer)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, result, "Result should be empty when error occurs")
			} else {
				require.NoError(t, err, "Signing should succeed")
				assert.NotEmpty(t, result, "Result should not be empty")

				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}
		})
	}
}

func TestExecuteProposalTask_ID(t *testing.T) {
	t.Parallel()

	task := ExecuteProposalTask("test-proposal-id")
	assert.NotEmpty(t, task.ID())
}

func TestExecuteProposalTask_Run(t *testing.T) {
	t.Parallel()

	// helper function to seed the state with a proposal
	seedState := func(propState ProposalState) *State {
		state := newState()
		state.Proposals = []ProposalState{propState}

		return state
	}

	tests := []struct {
		name            string
		before          func(executor *mockProposalExecutor)
		setupState      func(t *testing.T) *State
		givePropStateID string
		wantErr         string
	}{
		{
			name: "successfully executes MCMS proposal",
			before: func(executor *mockProposalExecutor) {
				executor.EXPECT().ExecuteMCMS(mock.Anything, mock.Anything).Return(nil)
			},
			setupState: func(t *testing.T) *State {
				t.Helper()

				prop := createTestMCMSProposalForSigning(t)
				propState, err := newMCMSProposalState(&prop)
				require.NoError(t, err)

				propState.ID = "test-proposal-id" // Override the ID

				return seedState(propState)
			},
			givePropStateID: "test-proposal-id",
		},
		{
			name: "successfully executes Timelock proposal",
			before: func(executor *mockProposalExecutor) {
				executor.EXPECT().ExecuteTimelock(mock.Anything, mock.Anything).Return(nil)
			},
			setupState: func(t *testing.T) *State {
				t.Helper()

				prop := createTestTimelockProposalForSigning(t)
				propState, err := newTimelockProposalState(&prop)
				require.NoError(t, err)

				propState.ID = "test-proposal-id" // Override the ID

				return seedState(propState)
			},
			givePropStateID: "test-proposal-id",
		},
		{
			name: "fails to get proposal",
			setupState: func(t *testing.T) *State {
				t.Helper()

				return newState()
			},
			givePropStateID: "invalid",
			wantErr:         "proposal not found",
		},
		{
			name: "fails when proposal already executed",
			setupState: func(t *testing.T) *State {
				t.Helper()

				return seedState(ProposalState{
					ID:         "test-proposal-id",
					IsExecuted: true,
				})
			},
			givePropStateID: "test-proposal-id",
			wantErr:         "proposal already executed",
		},
		{
			name: "invalid proposal kind",
			setupState: func(t *testing.T) *State {
				t.Helper()

				propState := ProposalState{
					ID:   "test-proposal-id",
					JSON: `{`, // Invalid JSON will fail to decode the proposal kind
				}

				return seedState(propState)
			},
			givePropStateID: "test-proposal-id",
			wantErr:         "failed to get proposal kind",
		},
		{
			name: "invalid MCMS Proposal",
			setupState: func(t *testing.T) *State {
				t.Helper()

				return seedState(ProposalState{
					ID:   "test-proposal-id",
					JSON: `{"kind": "Proposal"}`,
				})
			},
			givePropStateID: "test-proposal-id",
			wantErr:         "failed to decode MCMS proposal (id: test-proposal-id)",
		},
		{
			name: "invalid Timelock Proposal",
			setupState: func(t *testing.T) *State {
				t.Helper()

				return seedState(ProposalState{
					ID:   "test-proposal-id",
					JSON: `{"kind": "TimelockProposal"}`},
				)
			},
			givePropStateID: "test-proposal-id",
			wantErr:         "failed to decode Timelock proposal (id: test-proposal-id)",
		},
		{
			name: "failed to execute MCMS proposal",
			before: func(executor *mockProposalExecutor) {
				executor.EXPECT().
					ExecuteMCMS(mock.Anything, mock.Anything).
					Return(errors.New("failed to execute MCMS proposal"))
			},
			setupState: func(t *testing.T) *State {
				t.Helper()

				prop := createTestMCMSProposalForSigning(t)
				propState, err := newMCMSProposalState(&prop)
				require.NoError(t, err)

				propState.ID = "test-proposal-id" // Override the ID

				return seedState(propState)
			},
			givePropStateID: "test-proposal-id",
			wantErr:         "failed to execute MCMS proposal (id: test-proposal-id)",
		},
		{
			name: "failed to execute Timelock proposal",
			before: func(executor *mockProposalExecutor) {
				executor.EXPECT().
					ExecuteTimelock(mock.Anything, mock.Anything).
					Return(errors.New("failed to execute Timelock proposal"))
			},
			setupState: func(t *testing.T) *State {
				t.Helper()

				prop := createTestTimelockProposalForSigning(t)
				propState, err := newTimelockProposalState(&prop)
				require.NoError(t, err)

				propState.ID = "test-proposal-id" // Override the ID

				return seedState(propState)
			},
			givePropStateID: "test-proposal-id",
			wantErr:         "failed to execute Timelock proposal (id: test-proposal-id)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor := newMockProposalExecutor(t)

			if tt.before != nil {
				tt.before(executor)
			}

			env := createTestEnvironment(t)
			state := tt.setupState(t)

			task := ExecuteProposalTask(tt.givePropStateID)
			task.newExecutor = func(e fdeployment.Environment) proposalExecutor {
				return executor
			}

			err := task.Run(env, state)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				updatedProp, err := state.GetProposal(tt.givePropStateID)
				require.NoError(t, err)
				assert.True(t, updatedProp.IsExecuted)
			}
		})
	}
}

func TestSignAndExecuteProposalsTask_ID(t *testing.T) {
	t.Parallel()

	task := SignAndExecuteProposalsTask([]*ecdsa.PrivateKey{})
	assert.NotEmpty(t, task.ID())
}

func TestSignAndExecuteProposalsTask_Run(t *testing.T) {
	t.Parallel()

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := []struct {
		name           string
		before         func(executor *mockProposalExecutor)
		setupState     func(t *testing.T) *State
		signingKeys    []*ecdsa.PrivateKey
		wantErr        string
		validateResult func(t *testing.T, state *State)
	}{
		{
			name: "successfully signs and executes MCMS proposal",
			before: func(executor *mockProposalExecutor) {
				executor.EXPECT().ExecuteMCMS(mock.Anything, mock.Anything).Return(nil)
			},
			setupState: func(t *testing.T) *State {
				t.Helper()

				// Create a test MCMS proposal
				proposal := createTestMCMSProposalForSigning(t)
				propState, err := newMCMSProposalState(&proposal)
				require.NoError(t, err)

				state := newState()
				state.Proposals = []ProposalState{
					propState,
					{
						ID:         "test-proposal-id-2",
						IsExecuted: true,
					},
				}

				return state
			},
			signingKeys: []*ecdsa.PrivateKey{privateKey},
			validateResult: func(t *testing.T, state *State) {
				t.Helper()

				require.Len(t, state.Proposals, 2)
				assert.Empty(t, state.GetPendingProposals())
			},
		},
		{
			name: "fails to sign MCMS proposal",
			setupState: func(t *testing.T) *State {
				t.Helper()

				state := newState()
				state.Proposals = []ProposalState{
					{
						ID:   "test-proposal-id",
						JSON: "invalid-json-content",
					},
				}

				return state
			},
			signingKeys: []*ecdsa.PrivateKey{privateKey},
			wantErr:     "failed to sign proposal",
		},
		{
			name: "fails to execute MCMS proposal",
			before: func(executor *mockProposalExecutor) {
				executor.EXPECT().ExecuteMCMS(mock.Anything, mock.Anything).Return(
					errors.New("failed to execute MCMS proposal"),
				)
			},
			setupState: func(t *testing.T) *State {
				t.Helper()

				proposal := createTestMCMSProposalForSigning(t)
				propState, err := newMCMSProposalState(&proposal)
				require.NoError(t, err)

				state := newState()
				state.Proposals = []ProposalState{propState}

				return state
			},
			signingKeys: []*ecdsa.PrivateKey{privateKey},
			wantErr:     "failed to execute MCMS proposal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tt.setupState(t)
			env := createTestEnvironment(t)
			executor := newMockProposalExecutor(t)

			if tt.before != nil {
				tt.before(executor)
			}

			task := SignAndExecuteProposalsTask(tt.signingKeys)
			task.newExecutor = func(e fdeployment.Environment) proposalExecutor {
				return executor
			}

			err := task.Run(env, state)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err, "Task should succeed")

				if tt.validateResult != nil {
					tt.validateResult(t, state)
				}
			}
		})
	}
}

// createTestMCMSProposalForSigning creates a minimal MCMS proposal for testing signing functionality
func createTestMCMSProposalForSigning(t *testing.T) mcmslib.Proposal {
	t.Helper()

	selector := chainselectors.ETHEREUM_TESTNET_SEPOLIA.Selector

	return mcmslib.Proposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:    "v1",
			Kind:       mcmstypes.KindProposal,
			ValidUntil: 2004259681,
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				mcmstypes.ChainSelector(selector): {
					StartingOpCount: 0,
					MCMAddress:      "0x0000000000000000000000000000000000000000",
				},
			},
		},
		Operations: []mcmstypes.Operation{
			{
				ChainSelector: mcmstypes.ChainSelector(selector),
				Transaction: mcmstypes.Transaction{
					To:               "0x123",
					AdditionalFields: json.RawMessage(`{"value": 0}`),
					Data:             []byte{1, 2, 3},
					OperationMetadata: mcmstypes.OperationMetadata{
						ContractType: "test",
						Tags:         []string{"test"},
					},
				},
			},
		},
	}
}

// createTestTimelockProposalForSigning creates a minimal Timelock proposal for testing signing functionality
func createTestTimelockProposalForSigning(t *testing.T) mcmslib.TimelockProposal {
	t.Helper()

	selector := chainselectors.ETHEREUM_TESTNET_SEPOLIA.Selector

	return mcmslib.TimelockProposal{
		BaseProposal: mcmslib.BaseProposal{
			Version:    "v1",
			Kind:       mcmstypes.KindTimelockProposal,
			ValidUntil: 2004259681,
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				mcmstypes.ChainSelector(selector): {
					StartingOpCount: 0,
					MCMAddress:      "0x0000000000000000000000000000000000000000",
				},
			},
		},
		Action: mcmstypes.TimelockActionSchedule,
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			mcmstypes.ChainSelector(selector): "0x0000000000000000000000000000000000000000",
		},
		Operations: []mcmstypes.BatchOperation{
			{
				ChainSelector: mcmstypes.ChainSelector(selector),
				Transactions: []mcmstypes.Transaction{
					{
						To:               "0x123",
						AdditionalFields: json.RawMessage(`{"value": 0}`),
						Data:             []byte{1, 2, 3},
						OperationMetadata: mcmstypes.OperationMetadata{
							ContractType: "test",
							Tags:         []string{"test"},
						},
					},
				},
			},
		},
	}
}

// createTestEnvironment creates a test environment with necessary chains configured
func createTestEnvironment(t *testing.T) fdeployment.Environment {
	t.Helper()

	evmChain := fchainevm.Chain{
		Selector: chainselectors.ETHEREUM_TESTNET_SEPOLIA.Selector,
	}

	return fdeployment.Environment{
		BlockChains: fchain.NewBlockChainsFromSlice([]fchain.BlockChain{evmChain}),
		GetContext: func() context.Context {
			return t.Context()
		},
	}
}
