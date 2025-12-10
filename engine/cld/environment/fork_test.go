package environment

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func Test_LoadForkedEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		domain       fdomain.Domain
		env          string
		blockNumbers map[uint64]*big.Int
		options      []LoadEnvironmentOption
		expectError  string
	}{
		{
			name:   "Invalid Environment",
			domain: fdomain.NewDomain("dummy", "test"),
			env:    "non_existent_env",
			blockNumbers: map[uint64]*big.Int{
				1: big.NewInt(1000),
			},
			expectError: "failed to load config",
		},
		{
			name:   "Address Book Failure",
			domain: setupTest(t, setupTestConfig),
			env:    "staging",
			blockNumbers: map[uint64]*big.Int{
				1: big.NewInt(1000),
			},
			expectError: "failed to load address book",
		},
		{
			name:         "Empty Block Numbers",
			domain:       setupTest(t, setupTestConfig, setupAddressbook),
			env:          "staging",
			blockNumbers: map[uint64]*big.Int{},
			expectError:  "failed to create anvil chains",
		},
		{
			name:   "Invalid Nodes File",
			domain: setupTest(t, setupTestConfig, setupAddressbook),
			env:    "staging",
			blockNumbers: map[uint64]*big.Int{
				16015286601757825753: big.NewInt(1000),
			},
			expectError: "failed to load nodes",
		},
		{
			name:   "OffchainClient Failure",
			domain: setupTest(t, setupTestConfig, setupAddressbook, setupNodes),
			env:    "staging",
			blockNumbers: map[uint64]*big.Int{
				16015286601757825753: big.NewInt(1000),
			},
			expectError: "failed to load offchain client",
		},
		{
			name:   "No Error",
			domain: setupTest(t, setupTestConfig, setupAddressbook, setupNodes),
			env:    "staging",
			blockNumbers: map[uint64]*big.Int{
				16015286601757825753: big.NewInt(1000),
			},
			options: []LoadEnvironmentOption{WithoutJD()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			forkEnv, err := LoadFork(t.Context(), tt.domain, tt.env, tt.blockNumbers, tt.options...)

			if tt.expectError != "" {
				require.ErrorContains(t, err, tt.expectError)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, "fork", forkEnv.Name)
		})
	}
}

func Test_ApplyChangesetOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		changesetOutput fdeployment.ChangesetOutput
		forkClients     map[uint64]ForkedOnchainClient
		blockNumbers    map[uint64]*big.Int
		expectError     string
	}{
		{
			name: "Timelock Proposal - No TimeLock Address",
			changesetOutput: fdeployment.ChangesetOutput{
				MCMSTimelockProposals: []mcms.TimelockProposal{
					createMCMSTimelockProposal(t, 123, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)),
				},
			},
			forkClients: nil,
			blockNumbers: map[uint64]*big.Int{
				16015286601757825753: big.NewInt(1000),
			},
			expectError: "no timelock address defined for chain selector",
		},
		{
			name: "Timelock Proposal - No Fork Client",
			changesetOutput: fdeployment.ChangesetOutput{
				MCMSTimelockProposals: []mcms.TimelockProposal{
					createMCMSTimelockProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)),
				},
			},
			forkClients: nil,
			blockNumbers: map[uint64]*big.Int{
				16015286601757825753: big.NewInt(1000),
			},
			expectError: "no fork client defined for chain selector",
		},
		{
			name: "Timelock Proposal - Failed Transaction",
			changesetOutput: fdeployment.ChangesetOutput{
				MCMSTimelockProposals: []mcms.TimelockProposal{
					createMCMSTimelockProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)),
				},
			},
			forkClients: map[uint64]ForkedOnchainClient{
				uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): MockForkedOnchainClient{returnError: true},
			},
			blockNumbers: map[uint64]*big.Int{
				uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): big.NewInt(1000),
			},
			expectError: "failed to send transaction on chain",
		},
		{
			name: "Timelock Proposal - No Error",
			changesetOutput: fdeployment.ChangesetOutput{
				MCMSTimelockProposals: []mcms.TimelockProposal{
					createMCMSTimelockProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)),
				},
			},
			forkClients: map[uint64]ForkedOnchainClient{
				uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): MockForkedOnchainClient{},
			},
			blockNumbers: map[uint64]*big.Int{
				uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): big.NewInt(1000),
			},
		},
		{
			name: "Base Proposal - No Fork Client",
			changesetOutput: fdeployment.ChangesetOutput{
				MCMSProposals: []mcms.Proposal{
					createBaseProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)),
				},
			},
			forkClients: nil,
			blockNumbers: map[uint64]*big.Int{
				16015286601757825753: big.NewInt(1000),
			},
			expectError: "no fork client defined for chain selector",
		},
		{
			name: "Base Proposal - Failed Transaction",
			changesetOutput: fdeployment.ChangesetOutput{
				MCMSProposals: []mcms.Proposal{
					createBaseProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)),
				},
			},
			forkClients: map[uint64]ForkedOnchainClient{
				uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): MockForkedOnchainClient{returnError: true},
			},
			blockNumbers: map[uint64]*big.Int{
				uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): big.NewInt(1000),
			},
			expectError: "failed to send transaction on chain",
		},
		{
			name: "Base Proposal - No Error",
			changesetOutput: fdeployment.ChangesetOutput{
				MCMSProposals: []mcms.Proposal{
					createBaseProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)),
				},
			},
			forkClients: map[uint64]ForkedOnchainClient{
				uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): MockForkedOnchainClient{},
			},
			blockNumbers: map[uint64]*big.Int{
				uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): big.NewInt(1000),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)

			forkEnv, err := LoadFork(t.Context(), domain, "staging", tt.blockNumbers, WithoutJD())
			require.NoError(t, err)

			if tt.forkClients != nil {
				forkEnv.ForkClients = tt.forkClients
			}

			_, err = forkEnv.ApplyChangesetOutput(t.Context(), tt.changesetOutput)

			if tt.expectError != "" {
				require.ErrorContains(t, err, tt.expectError)
				return
			}

			require.NoError(t, err)
		})
	}
}

// MockForkedOnchainClient is a mock implementation of ForkedOnchainClient
type MockForkedOnchainClient struct {
	returnError bool
}

func (m MockForkedOnchainClient) SendTransaction(ctx context.Context, from string, to string, data []byte) error {
	if m.returnError {
		return errors.New("mock error")
	}

	return nil
}

// createMCMSTimelockProposal creates a new MCMS timelock proposal for testing purposes.
func createMCMSTimelockProposal(t *testing.T, timelockAddress types.ChainSelector, operationsAddress types.ChainSelector) mcms.TimelockProposal {
	t.Helper()

	futureTime := time.Now().Add(time.Hour * 72).Unix()
	builder := mcms.NewTimelockProposalBuilder()
	builder.
		SetVersion("v1").
		SetAction(types.TimelockActionSchedule).
		// #nosec G115
		SetValidUntil(uint32(futureTime)).
		SetDescription("mcms timelock description").
		SetDelay(types.NewDuration(1 * time.Hour)).
		SetOverridePreviousRoot(true).
		SetChainMetadata(map[types.ChainSelector]types.ChainMetadata{
			operationsAddress: {
				StartingOpCount:  1,
				MCMAddress:       "0xMCMSAddress",
				AdditionalFields: nil,
			},
		}).
		SetTimelockAddresses(map[types.ChainSelector]string{
			timelockAddress: "0xTimelockAddress",
		}).
		SetOperations([]types.BatchOperation{
			{
				ChainSelector: operationsAddress,
				Transactions: []types.Transaction{
					{
						OperationMetadata: types.OperationMetadata{
							ContractType: "test",
						},
						To:               "0x123",
						Data:             []byte{1, 2, 3},
						AdditionalFields: json.RawMessage(`{"test": "test"}`),
					},
					{
						OperationMetadata: types.OperationMetadata{
							ContractType: "test2",
						},
						To:               "0x456",
						Data:             []byte{4, 5, 6},
						AdditionalFields: json.RawMessage(`{"test2": "test2"}`),
					},
				},
			},
		})

	proposal, err := builder.Build()
	require.NoError(t, err)

	return *proposal
}

// createBaseProposal creates a new MCMS timelock proposal for testing purposes.
func createBaseProposal(t *testing.T, metadataAddress types.ChainSelector, operationAddress types.ChainSelector) mcms.Proposal {
	t.Helper()

	futureTime := time.Now().Add(time.Hour * 72).Unix()
	builder := mcms.NewProposalBuilder()
	builder.
		SetVersion("v1").
		// #nosec G115
		SetValidUntil(uint32(futureTime)).
		SetDescription("mcms timelock description").
		SetOverridePreviousRoot(true).
		SetChainMetadata(map[types.ChainSelector]types.ChainMetadata{
			metadataAddress: {
				StartingOpCount:  1,
				MCMAddress:       "0xMCMSAddress",
				AdditionalFields: nil,
			},
		}).
		SetOperations([]types.Operation{
			{
				ChainSelector: operationAddress,
				Transaction: types.Transaction{
					OperationMetadata: types.OperationMetadata{
						ContractType: "test",
					},
					To:               "0x123",
					Data:             []byte{1, 2, 3},
					AdditionalFields: json.RawMessage(`{"test": "test"}`),
				},
			},
		})

	proposal, err := builder.Build()
	require.NoError(t, err)

	return *proposal
}
