package environment

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func Test_LoadForkedEnvironment_InvalidEnvironment(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := cldf_domain.NewDomain("dummy", "test")

	lggr := logger.Test(t)
	blockNumbers := map[uint64]*big.Int{
		1: big.NewInt(1000),
	}

	_, err := LoadForkedEnvironment(context.Background(), lggr, "non_existent_env", domain, blockNumbers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func Test_LoadForkedEnvironment_AddressBookFailure(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig)

	lggr := logger.Test(t)
	blockNumbers := map[uint64]*big.Int{
		1: big.NewInt(1000),
	}

	_, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load address book")
}

func Test_LoadForkedEnvironment_EmptyBlockNumbers(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook)

	lggr := logger.Test(t)
	blockNumbers := map[uint64]*big.Int{}

	_, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create anvil chains")
}

func Test_LoadForkedEnvironment_InvalidNodesFile(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook)

	lggr := logger.Test(t)
	blockNumbers := map[uint64]*big.Int{
		16015286601757825753: big.NewInt(1000),
	}

	_, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load nodes")
}

func Test_LoadForkedEnvironment_OffchainClient(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)

	lggr := logger.Test(t)
	blockNumbers := map[uint64]*big.Int{
		16015286601757825753: big.NewInt(1000),
	}

	assert.Panics(t, func() {
		_, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers)
		require.NoError(t, err)
	})
}

func Test_LoadForkedEnvironment(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)

	lggr := logger.Test(t)
	blockNumbers := map[uint64]*big.Int{
		16015286601757825753: big.NewInt(1000),
	}

	forkEnv, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers, WithoutJD())
	require.NoError(t, err)
	assert.Equal(t, "fork", forkEnv.Name)
}

func Test_ApplyChangesetOutput_Timelock_NoTimeLockAddress(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)
	blockNumbers := map[uint64]*big.Int{
		16015286601757825753: big.NewInt(1000),
	}

	proposal := createMCMSTimelockProposal(t, 123, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector))

	output := cldf.ChangesetOutput{
		MCMSTimelockProposals: []mcms.TimelockProposal{*proposal},
	}

	forkEnv, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers, WithoutJD())
	require.NoError(t, err)

	_, err = forkEnv.ApplyChangesetOutput(context.Background(), output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no timelock address defined for chain selector")
}

func Test_ApplyChangesetOutput_Timelock_NoForkClient(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)
	blockNumbers := map[uint64]*big.Int{
		16015286601757825753: big.NewInt(1000),
	}

	proposal := createMCMSTimelockProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector))

	output := cldf.ChangesetOutput{
		MCMSTimelockProposals: []mcms.TimelockProposal{*proposal},
	}

	forkEnv, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers, WithoutJD())
	require.NoError(t, err)

	_, err = forkEnv.ApplyChangesetOutput(context.Background(), output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fork client defined for chain selector")
}

func Test_ApplyChangesetOutput_Timelock_FailedTx(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)
	blockNumbers := map[uint64]*big.Int{
		uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): big.NewInt(1000),
	}

	proposal := createMCMSTimelockProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector))

	output := cldf.ChangesetOutput{
		MCMSTimelockProposals: []mcms.TimelockProposal{*proposal},
	}

	forkEnv, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers, WithoutJD())
	require.NoError(t, err)

	forkEnv.ForkClients = map[uint64]ForkedOnchainClient{
		uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): MockForkedOnchainClient{returnError: true},
	}

	_, err = forkEnv.ApplyChangesetOutput(context.Background(), output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send transaction on chain")
}

func Test_ApplyChangesetOutput_Timelock(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)
	blockNumbers := map[uint64]*big.Int{
		uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): big.NewInt(1000),
	}

	proposal := createMCMSTimelockProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector))

	output := cldf.ChangesetOutput{
		MCMSTimelockProposals: []mcms.TimelockProposal{*proposal},
	}

	forkEnv, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers, WithoutJD())
	require.NoError(t, err)

	forkEnv.ForkClients = map[uint64]ForkedOnchainClient{
		uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): MockForkedOnchainClient{},
	}

	_, err = forkEnv.ApplyChangesetOutput(context.Background(), output)
	require.NoError(t, err)
}

func Test_ApplyChangesetOutput_Base_NoForkClient(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)
	blockNumbers := map[uint64]*big.Int{
		16015286601757825753: big.NewInt(1000),
	}

	proposal := createMCMSTimelockProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector))

	output := cldf.ChangesetOutput{
		MCMSTimelockProposals: []mcms.TimelockProposal{*proposal},
	}

	forkEnv, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers, WithoutJD())
	require.NoError(t, err)

	_, err = forkEnv.ApplyChangesetOutput(context.Background(), output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fork client defined for chain selector")
}

func Test_ApplyChangesetOutput_Base_FailedTx(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)
	blockNumbers := map[uint64]*big.Int{
		uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): big.NewInt(1000),
	}

	proposal := createMCMSTimelockProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector))

	output := cldf.ChangesetOutput{
		MCMSTimelockProposals: []mcms.TimelockProposal{*proposal},
	}

	forkEnv, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers, WithoutJD())
	require.NoError(t, err)

	forkEnv.ForkClients = map[uint64]ForkedOnchainClient{
		uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): MockForkedOnchainClient{returnError: true},
	}

	_, err = forkEnv.ApplyChangesetOutput(context.Background(), output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send transaction on chain")
}

func Test_ApplyChangesetOutput_Base(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)
	blockNumbers := map[uint64]*big.Int{
		uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): big.NewInt(1000),
	}

	proposal := createMCMSTimelockProposal(t, types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector), types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector))

	output := cldf.ChangesetOutput{
		MCMSTimelockProposals: []mcms.TimelockProposal{*proposal},
	}

	forkEnv, err := LoadForkedEnvironment(context.Background(), lggr, "staging", domain, blockNumbers, WithoutJD())
	require.NoError(t, err)

	forkEnv.ForkClients = map[uint64]ForkedOnchainClient{
		uint64(types.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)): MockForkedOnchainClient{},
	}

	_, err = forkEnv.ApplyChangesetOutput(context.Background(), output)
	require.NoError(t, err)
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
func createMCMSTimelockProposal(t *testing.T, timelockAddress types.ChainSelector, operationsAddress types.ChainSelector) *mcms.TimelockProposal {
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

	return proposal
}

// createBaseProposal creates a new MCMS timelock proposal for testing purposes.
func createBaseProposal(t *testing.T, metadataAddress types.ChainSelector, operationAddress types.ChainSelector) *mcms.Proposal {
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

	return proposal
}
