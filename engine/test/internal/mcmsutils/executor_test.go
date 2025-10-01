package mcmsutils

import (
	"context"
	"errors"
	"testing"
	"time"

	gethtypes "github.com/ethereum/go-ethereum/core/types"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestExecutor_ExecuteMCMS(t *testing.T) {
	t.Parallel()

	// Fake transaction results
	gethTx := gethtypes.NewTx(&gethtypes.LegacyTx{})
	setRootTxResult := mcmstypes.TransactionResult{
		Hash:    "0x123",
		RawData: gethTx,
	}
	confirmTxResult := mcmstypes.TransactionResult{
		Hash:    "0x456",
		RawData: gethTx,
	}

	tests := []struct {
		name     string
		before   func(mcmsExecutable *mockMcmsExecutable)
		env      func(env *fdeployment.Environment) // Allow modification of the environment
		proposal func() *mcmslib.Proposal           // Generate the proposal
		wantErr  string
	}{
		{
			name: "success",
			before: func(mcmsExecutable *mockMcmsExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(setRootTxResult, nil)

				mcmsExecutable.EXPECT().
					Execute(t.Context(), 0).
					Return(confirmTxResult, nil)
			},
			proposal: func() *mcmslib.Proposal {
				return stubMCMSProposal()
			},
		},
		{
			name: "fails to validate proposal",
			proposal: func() *mcmslib.Proposal {
				return &mcmslib.Proposal{}
			},
			wantErr: "failed to validate MCMS proposal",
		},
		{
			name: "fails to get blockchains from environment for metadata",
			proposal: func() *mcmslib.Proposal {
				prop := stubMCMSProposal()
				prop.ChainMetadata[mcmstypes.ChainSelector(stubAptosChain().Selector)] = mcmstypes.ChainMetadata{ // Aptos chain is not configured in the environment
					StartingOpCount: 0,
					MCMAddress:      "0x0000000000000000000000000000000000000000",
				}

				return prop
			},
			wantErr: "failed to get blockchains from environment",
		},
		{
			name: "fails to set root",
			before: func(mcmsExecutable *mockMcmsExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(mcmstypes.TransactionResult{}, errors.New("failed to set root"))
			},
			proposal: func() *mcmslib.Proposal {
				return stubMCMSProposal()
			},
			wantErr: "failed to set root",
		},
		{
			name: "fails to confirm set root transaction",
			before: func(mcmsExecutable *mockMcmsExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(setRootTxResult, nil)
			},
			env: func(env *fdeployment.Environment) {
				c := stubEVMChain()
				c.Confirm = func(tx *gethtypes.Transaction) (uint64, error) {
					return 0, errors.New("failed to confirm transaction")
				}

				env.BlockChains = fchain.NewBlockChainsFromSlice(
					[]fchain.BlockChain{c},
				)
			},
			proposal: func() *mcmslib.Proposal {
				return stubMCMSProposal()
			},
			wantErr: "failed to confirm transaction",
		},
		{
			name: "fails to execute",
			before: func(mcmsExecutable *mockMcmsExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(setRootTxResult, nil)

				mcmsExecutable.EXPECT().
					Execute(t.Context(), 0).
					Return(confirmTxResult, errors.New("failed to execute"))
			},
			proposal: func() *mcmslib.Proposal {
				return stubMCMSProposal()
			},
			wantErr: "failed to execute",
		},
		{
			name: "fails to confirm execute transaction",
			before: func(mcmsExecutable *mockMcmsExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(setRootTxResult, nil)

				mcmsExecutable.EXPECT().
					Execute(t.Context(), 0).
					Return(confirmTxResult, nil)
			},
			env: func(env *fdeployment.Environment) {
				i := 0
				c := stubEVMChain()
				c.Confirm = func(tx *gethtypes.Transaction) (uint64, error) {
					// First call should be successful for set root
					if i == 0 {
						i++
						return 0, nil
					}

					// Second call should fail for execute
					return 0, errors.New("failed to confirm transaction")
				}

				env.BlockChains = fchain.NewBlockChainsFromSlice(
					[]fchain.BlockChain{c},
				)
			},
			proposal: func() *mcmslib.Proposal {
				return stubMCMSProposal()
			},
			wantErr: "failed to confirm transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				mockMCMSExecutable     = newMockMcmsExecutable(t)
				mockTimelockExecutable = newMockTimelockExecutable(t)
				e                      = stubEnvironment()
			)

			if tt.before != nil {
				tt.before(mockMCMSExecutable)
			}

			if tt.env != nil {
				tt.env(&e)
			}

			executor := newMockedExecutor(t, e, mockMCMSExecutable, mockTimelockExecutable)

			err := executor.ExecuteMCMS(t.Context(), tt.proposal())

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExecutor_ExecuteTimelock(t *testing.T) {
	t.Parallel()

	// Fake transaction results
	gethTx := gethtypes.NewTx(&gethtypes.LegacyTx{})
	setRootTxResult := mcmstypes.TransactionResult{
		Hash:    "0x123",
		RawData: gethTx,
	}
	confirmTxResult := mcmstypes.TransactionResult{
		Hash:    "0x456",
		RawData: gethTx,
	}

	tests := []struct {
		name     string
		before   func(mcmsExecutable *mockMcmsExecutable, timelockExecutable *mockTimelockExecutable)
		env      func(env *fdeployment.Environment)
		proposal func() *mcmslib.TimelockProposal
		wantErr  string
	}{
		{
			name: "success with schedule action",
			before: func(mcmsExecutable *mockMcmsExecutable, timelockExecutable *mockTimelockExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(setRootTxResult, nil)

				mcmsExecutable.EXPECT().
					Execute(t.Context(), 0).
					Return(confirmTxResult, nil)

				timelockExecutable.EXPECT().
					IsReady(t.Context()).
					Return(nil)

				timelockExecutable.EXPECT().
					Execute(t.Context(), 0, mock.AnythingOfType("[]mcms.Option")).
					Return(confirmTxResult, nil)
			},
			proposal: func() *mcmslib.TimelockProposal {
				return stubTimelockProposal(mcmstypes.TimelockActionSchedule)
			},
		},
		{
			name: "success with cancel action",
			before: func(mcmsExecutable *mockMcmsExecutable, timelockExecutable *mockTimelockExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(setRootTxResult, nil)

				mcmsExecutable.EXPECT().
					Execute(t.Context(), 0).
					Return(confirmTxResult, nil)
			},
			proposal: func() *mcmslib.TimelockProposal {
				return stubTimelockProposal(mcmstypes.TimelockActionCancel)
			},
		},
		{
			name: "fails to validate proposal",
			proposal: func() *mcmslib.TimelockProposal {
				return &mcmslib.TimelockProposal{}
			},
			wantErr: "failed to validate MCMS proposal",
		},
		{
			name: "fails to get blockchains from environment for metadata",
			proposal: func() *mcmslib.TimelockProposal {
				prop := stubTimelockProposal(mcmstypes.TimelockActionSchedule)
				prop.ChainMetadata[mcmstypes.ChainSelector(stubAptosChain().Selector)] = mcmstypes.ChainMetadata{ // Aptos chain is not configured in the environment
					StartingOpCount: 0,
					MCMAddress:      "0x0000000000000000000000000000000000000000",
				}

				return prop
			},
			wantErr: "failed to get blockchains from environment",
		},
		{
			name: "fails to execute the converted MCMS proposal",
			before: func(mcmsExecutable *mockMcmsExecutable, timelockExecutable *mockTimelockExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(mcmstypes.TransactionResult{}, errors.New("failed to set root"))
			},
			proposal: func() *mcmslib.TimelockProposal {
				return stubTimelockProposal(mcmstypes.TimelockActionSchedule)
			},
			wantErr: "failed to execute MCMS proposal",
		},
		{
			name: "fails to find the call proxy address",
			before: func(mcmsExecutable *mockMcmsExecutable, timelockExecutable *mockTimelockExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(setRootTxResult, nil)

				mcmsExecutable.EXPECT().
					Execute(t.Context(), 0).
					Return(confirmTxResult, nil)
			},
			proposal: func() *mcmslib.TimelockProposal {
				return stubTimelockProposal(mcmstypes.TimelockActionSchedule)
			},
			env: func(env *fdeployment.Environment) {
				env.DataStore = fdatastore.NewMemoryDataStore().Seal() // Create an empty datastore
			},
			wantErr: "ensure CallProxy is deployed and configured in datastore",
		},
		{
			name: "timelock is not ready",
			before: func(mcmsExecutable *mockMcmsExecutable, timelockExecutable *mockTimelockExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(setRootTxResult, nil)

				mcmsExecutable.EXPECT().
					Execute(t.Context(), 0).
					Return(confirmTxResult, nil)

				timelockExecutable.EXPECT().
					IsReady(t.Context()).
					Return(errors.New("timelock is not ready"))
			},
			proposal: func() *mcmslib.TimelockProposal {
				return stubTimelockProposal(mcmstypes.TimelockActionSchedule)
			},
			wantErr: "timelock proposal is not ready for execution",
		},
		{
			name: "fails to execute the timelock proposal",
			before: func(mcmsExecutable *mockMcmsExecutable, timelockExecutable *mockTimelockExecutable) {
				mcmsExecutable.EXPECT().
					SetRoot(t.Context(), mcmstypes.ChainSelector(stubEVMChain().Selector)).
					Return(setRootTxResult, nil)

				mcmsExecutable.EXPECT().
					Execute(t.Context(), 0).
					Return(confirmTxResult, nil)

				timelockExecutable.EXPECT().
					IsReady(t.Context()).
					Return(nil)

				timelockExecutable.EXPECT().
					Execute(t.Context(), 0, mock.AnythingOfType("[]mcms.Option")).
					Return(mcmstypes.TransactionResult{}, errors.New("failed to execute"))
			},
			proposal: func() *mcmslib.TimelockProposal {
				return stubTimelockProposal(mcmstypes.TimelockActionSchedule)
			},
			wantErr: "failed to execute timelock operation 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				mockMCMSExecutable     = newMockMcmsExecutable(t)
				mockTimelockExecutable = newMockTimelockExecutable(t)
				e                      = stubEnvironment()
			)

			if tt.before != nil {
				tt.before(mockMCMSExecutable, mockTimelockExecutable)
			}

			if tt.env != nil {
				tt.env(&e)
			}

			executor := newMockedExecutor(t, e, mockMCMSExecutable, mockTimelockExecutable)

			err := executor.ExecuteTimelock(t.Context(), tt.proposal())

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// newMockedExecutor creates a new Executor with the mock executables by overriding the
// newExecutable and newTimelockExecutable functions. Retry attempts and delay are overridden
// to speed up the tests.
func newMockedExecutor(
	t *testing.T,
	e fdeployment.Environment,
	mockMCMSExecutable *mockMcmsExecutable,
	mockTimelockExecutable *mockTimelockExecutable,
) *Executor {
	t.Helper()

	executor := NewExecutor(e)
	executor.newExecutable = func(
		_ *mcmslib.Proposal, _ map[mcmstypes.ChainSelector]mcmssdk.Executor,
	) (mcmsExecutable, error) {
		return mockMCMSExecutable, nil
	}
	executor.newTimelockExecutable = func(
		_ context.Context, _ *mcmslib.TimelockProposal, _ map[mcmstypes.ChainSelector]mcmssdk.TimelockExecutor,
	) (timelockExecutable, error) {
		return mockTimelockExecutable, nil
	}
	executor.retryAttempts = 1
	executor.retryDelay = 1 * time.Millisecond

	return executor
}
