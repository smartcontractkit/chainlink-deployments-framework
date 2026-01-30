package mcmsutils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
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

	t.Run("success with multiple MCMS deployments on same chain", func(t *testing.T) {
		t.Parallel()

		selector := stubEVMChain().Selector

		// Setup: Create environment with TWO MCMS deployments
		ds := fdatastore.NewMemoryDataStore()

		// First MCMS deployment (MCMS_EVM_1)
		qualifier1 := "MCMS_EVM_1"
		timelockAddr1 := "0x1111111111111111111111111111111111111111"
		callProxyAddr1 := "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

		err := ds.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: selector,
			Type:          "RBACTimelock",
			Version:       semver.MustParse("1.0.0"),
			Address:       timelockAddr1,
			Qualifier:     qualifier1,
		})
		require.NoError(t, err)

		err = ds.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: selector,
			Type:          "CallProxy",
			Version:       semver.MustParse("1.0.0"),
			Address:       callProxyAddr1,
			Qualifier:     qualifier1,
		})
		require.NoError(t, err)

		// Second MCMS deployment (MCMS_EVM_2)
		qualifier2 := "MCMS_EVM_2"
		timelockAddr2 := "0x2222222222222222222222222222222222222222"
		callProxyAddr2 := "0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"

		err = ds.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: selector,
			Type:          "RBACTimelock",
			Version:       semver.MustParse("1.0.0"),
			Address:       timelockAddr2,
			Qualifier:     qualifier2,
		})
		require.NoError(t, err)

		err = ds.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: selector,
			Type:          "CallProxy",
			Version:       semver.MustParse("1.0.0"),
			Address:       callProxyAddr2,
			Qualifier:     qualifier2,
		})
		require.NoError(t, err)

		env := fdeployment.Environment{
			DataStore:   ds.Seal(),
			BlockChains: fchain.NewBlockChainsFromSlice([]fchain.BlockChain{stubEVMChain()}),
		}

		// Create proposal for SECOND MCMS instance
		proposal := stubTimelockProposal(mcmstypes.TimelockActionSchedule)
		proposal.TimelockAddresses[mcmstypes.ChainSelector(selector)] = timelockAddr2

		// Setup mocks
		mockMCMSExecutable := newMockMcmsExecutable(t)
		mockTimelockExecutable := newMockTimelockExecutable(t)

		mockMCMSExecutable.EXPECT().
			SetRoot(t.Context(), mcmstypes.ChainSelector(selector)).
			Return(setRootTxResult, nil)

		mockMCMSExecutable.EXPECT().
			Execute(t.Context(), 0).
			Return(confirmTxResult, nil)

		mockTimelockExecutable.EXPECT().
			IsReady(t.Context()).
			Return(nil)

		mockTimelockExecutable.EXPECT().
			Execute(t.Context(), 0, mock.AnythingOfType("[]mcms.Option")).
			Return(confirmTxResult, nil)

		executor := newMockedExecutor(t, env, mockMCMSExecutable, mockTimelockExecutable)

		// Execute - should succeed and use CallProxy from MCMS_EVM_2, not MCMS_EVM_1
		err = executor.ExecuteTimelock(t.Context(), proposal)
		require.NoError(t, err, "ExecuteTimelock should succeed with multiple MCMS deployments")
	})

	t.Run("fails with unknown timelock address in multiple MCMS environment", func(t *testing.T) {
		t.Parallel()

		selector := stubEVMChain().Selector

		// Setup: Create environment with two MCMS deployments
		ds := fdatastore.NewMemoryDataStore()

		err := ds.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: selector,
			Type:          "RBACTimelock",
			Version:       semver.MustParse("1.0.0"),
			Address:       "0x1111111111111111111111111111111111111111",
			Qualifier:     "MCMS_EVM_1",
		})
		require.NoError(t, err)

		err = ds.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: selector,
			Type:          "CallProxy",
			Version:       semver.MustParse("1.0.0"),
			Address:       "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
			Qualifier:     "MCMS_EVM_1",
		})
		require.NoError(t, err)

		env := fdeployment.Environment{
			DataStore:   ds.Seal(),
			BlockChains: fchain.NewBlockChainsFromSlice([]fchain.BlockChain{stubEVMChain()}),
		}

		// Create proposal with UNKNOWN timelock address
		proposal := stubTimelockProposal(mcmstypes.TimelockActionSchedule)
		proposal.TimelockAddresses[mcmstypes.ChainSelector(selector)] = "0x9999999999999999999999999999999999999999"

		// Setup mocks - execution gets to the point of looking up CallProxy
		mockMCMSExecutable := newMockMcmsExecutable(t)
		mockTimelockExecutable := newMockTimelockExecutable(t)

		mockMCMSExecutable.EXPECT().
			SetRoot(t.Context(), mcmstypes.ChainSelector(selector)).
			Return(setRootTxResult, nil)

		mockMCMSExecutable.EXPECT().
			Execute(t.Context(), 0).
			Return(confirmTxResult, nil)

		executor := newMockedExecutor(t, env, mockMCMSExecutable, mockTimelockExecutable)

		// Execute - should fail because timelock address is not in datastore
		err = executor.ExecuteTimelock(t.Context(), proposal)
		require.Error(t, err)
		require.ErrorContains(t, err, "RBACTimelock address 0x9999999999999999999999999999999999999999 not found",
			"Should fail when timelock address is not registered in datastore")
	})
}

func TestFindCallProxyAddressForTimelock(t *testing.T) {
	t.Parallel()

	selector := uint64(909606746561742123)
	ds := fdatastore.NewMemoryDataStore()

	t.Run("returns error when timelock not found", func(t *testing.T) { //nolint:paralleltest // shares datastore state
		addr, err := findCallProxyAddressForTimelock(ds.Addresses(), selector, "0xTimelock1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "RBACTimelock address 0xTimelock1 not found")
		require.Empty(t, addr)
	})

	// Add first MCMS deployment
	qualifier1 := "MCMS_EVM_1"
	timelockAddr1 := "0xTimelock1"
	callProxyAddr1 := "0xCallProxy1"

	err := ds.Addresses().Add(fdatastore.AddressRef{
		ChainSelector: selector,
		Type:          "RBACTimelock",
		Version:       semver.MustParse("1.0.0"),
		Address:       timelockAddr1,
		Qualifier:     qualifier1,
	})
	require.NoError(t, err)

	err = ds.Addresses().Add(fdatastore.AddressRef{
		ChainSelector: selector,
		Type:          "CallProxy",
		Version:       semver.MustParse("1.0.0"),
		Address:       callProxyAddr1,
		Qualifier:     qualifier1,
	})
	require.NoError(t, err)

	t.Run("finds correct CallProxy for first MCMS deployment", func(t *testing.T) { //nolint:paralleltest // shares datastore state
		addr, callProxyErr := findCallProxyAddressForTimelock(ds.Addresses(), selector, timelockAddr1)
		require.NoError(t, callProxyErr)
		require.Equal(t, callProxyAddr1, addr)
	})

	// Add second MCMS deployment
	qualifier2 := "MCMS_EVM_2"
	timelockAddr2 := "0xTimelock2"
	callProxyAddr2 := "0xCallProxy2"

	err = ds.Addresses().Add(fdatastore.AddressRef{
		ChainSelector: selector,
		Type:          "RBACTimelock",
		Version:       semver.MustParse("1.0.0"),
		Address:       timelockAddr2,
		Qualifier:     qualifier2,
	})
	require.NoError(t, err)

	err = ds.Addresses().Add(fdatastore.AddressRef{
		ChainSelector: selector,
		Type:          "CallProxy",
		Version:       semver.MustParse("1.0.0"),
		Address:       callProxyAddr2,
		Qualifier:     qualifier2,
	})
	require.NoError(t, err)

	t.Run("finds correct CallProxy for second MCMS deployment", func(t *testing.T) { //nolint:paralleltest // shares datastore state
		addr, callProxyErr := findCallProxyAddressForTimelock(ds.Addresses(), selector, timelockAddr2)
		require.NoError(t, callProxyErr)
		require.Equal(t, callProxyAddr2, addr)
	})

	t.Run("returns error when CallProxy not found for qualifier", func(t *testing.T) { //nolint:paralleltest // shares datastore state
		// Add a timelock without a corresponding CallProxy
		qualifier3 := "MCMS_EVM_3"
		timelockAddr3 := "0xTimelock3"

		err := ds.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: selector,
			Type:          "RBACTimelock",
			Version:       semver.MustParse("1.0.0"),
			Address:       timelockAddr3,
			Qualifier:     qualifier3,
		})
		require.NoError(t, err)

		addr, err := findCallProxyAddressForTimelock(ds.Addresses(), selector, timelockAddr3)
		require.Error(t, err)
		require.Contains(t, err.Error(), "CallProxy not found for qualifier")
		require.Empty(t, addr)
	})

	t.Run("empty qualifier acts as wildcard and fails with multiple CallProxies", func(t *testing.T) { //nolint:paralleltest // shares datastore state
		// Add MCMS deployment with empty qualifier
		// Empty qualifier should match ANY CallProxy (wildcard behavior)
		// Since we already have multiple CallProxies in the datastore, this should fail
		timelockAddr4 := "0xTimelock4"

		err := ds.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: selector,
			Type:          "RBACTimelock",
			Version:       semver.MustParse("1.0.0"),
			Address:       timelockAddr4,
			Qualifier:     "", // Empty = wildcard
		})
		require.NoError(t, err)

		// Should fail because empty qualifier matches ALL CallProxies (callProxyAddr1, callProxyAddr2)
		addr, err := findCallProxyAddressForTimelock(ds.Addresses(), selector, timelockAddr4)
		require.Error(t, err)
		require.Contains(t, err.Error(), "multiple CallProxy addresses found")
		require.Empty(t, addr)
	})

	t.Run("empty qualifier works when only one CallProxy exists", func(t *testing.T) { //nolint:paralleltest // parallel would need separate test
		// Create a fresh datastore with only ONE CallProxy to test wildcard success case
		freshDS := fdatastore.NewMemoryDataStore()
		freshSelector := selector

		timelockAddr := "0xTimelockSingle"
		callProxyAddr := "0xCallProxySingle"

		err := freshDS.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: freshSelector,
			Type:          "RBACTimelock",
			Version:       semver.MustParse("1.0.0"),
			Address:       timelockAddr,
			Qualifier:     "", // Empty = wildcard
		})
		require.NoError(t, err)

		err = freshDS.Addresses().Add(fdatastore.AddressRef{
			ChainSelector: freshSelector,
			Type:          "CallProxy",
			Version:       semver.MustParse("1.0.0"),
			Address:       callProxyAddr,
			Qualifier:     "some-qualifier", // Has a qualifier, but empty timelock qualifier acts as wildcard and matches any CallProxy
		})
		require.NoError(t, err)

		// Should succeed because there's only one CallProxy, and empty qualifier matches it
		addr, err := findCallProxyAddressForTimelock(freshDS.Addresses(), freshSelector, timelockAddr)
		require.NoError(t, err)
		require.Equal(t, callProxyAddr, addr)
	})
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
