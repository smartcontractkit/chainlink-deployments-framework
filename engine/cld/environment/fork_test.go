package environment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	chainsel "github.com/smartcontractkit/chain-selectors"
	ctf "github.com/smartcontractkit/chainlink-testing-framework/framework"
	ctfblockchain "github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	fconfig "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

var (
	gethTestnetChainID    = chainsel.GETH_TESTNET.EvmChainID
	gethTestnetSelector   = chainsel.GETH_TESTNET.Selector
	zksyncSepoliaSelector = chainsel.ETHEREUM_TESTNET_SEPOLIA_ZKSYNC_1.Selector
)

func Test_LoadForkedEnvironment(t *testing.T) {
	t.Parallel()

	localChainHostRpcURL, localChainContainerRpcURL := startLocalChain(t, gethTestnetChainID)

	type test struct {
		name          string
		domain        fdomain.Domain
		env           fdeployment.Environment
		envcfg        func(*test) *fconfig.Config
		blockNumbers  map[uint64]*big.Int
		providerType  ProviderType
		options       []LoadEnvironmentOption
		wantNumChains int
		wantErr       string
	}
	tests := []test{
		{
			name:   "failure: empty block numbers",
			domain: setupTest(t, setupTestConfig, setupAddressbook),
			env:    fdeployment.Environment{Name: "staging", Logger: logger.Nop()},
			envcfg: func(tt *test) *fconfig.Config {
				envcfg, err := fconfig.Load(tt.domain, tt.env.Name, tt.env.Logger)
				require.NoError(t, err)
				return envcfg //nolint:nlreturn
			},
			blockNumbers: map[uint64]*big.Int{},
			wantErr:      "failed to create anvil chains: no evm networks found for forking in the provided blockNumbers map",
		},
		{
			name:   "failure: bad network metadata",
			domain: setupTest(t, setupTestConfig, setupAddressbook),
			env:    fdeployment.Environment{Name: "staging", Logger: logger.Nop()},
			envcfg: func(tt *test) *fconfig.Config {
				envcfg, err := fconfig.Load(tt.domain, tt.env.Name, tt.env.Logger)
				require.NoError(t, err)
				envcfg.Networks = cfgnet.NewConfig([]cfgnet.Network{{
					ChainSelector: gethTestnetSelector,
					Metadata:      "invalid-network-metadata",
				}})
				return envcfg //nolint:nlreturn
			},
			wantErr: "failed to create anvil chains: failed to decode network metadata for chain selector 337944638546241824",
		},
		{
			name:         "failure: bad provider type",
			domain:       setupTest(t, setupTestConfig, setupAddressbook),
			env:          fdeployment.Environment{Name: "staging", Logger: logger.Nop()},
			providerType: ProviderType(255),
			options:      []LoadEnvironmentOption{WithAnvilKeyAsDeployer(), OnlyLoadChainsFor([]uint64{gethTestnetSelector})},
			envcfg: func(tt *test) *fconfig.Config {
				envcfg, err := fconfig.Load(tt.domain, tt.env.Name, tt.env.Logger)
				require.NoError(t, err)
				envcfg.Networks = cfgnet.NewConfig([]cfgnet.Network{{
					ChainSelector: gethTestnetSelector,
					RPCs:          []cfgnet.RPC{{HTTPURL: localChainHostRpcURL}},
				}})
				return envcfg //nolint:nlreturn
			},
			wantErr: "failed to create anvil chains: unknown provider type: 255",
		},
		{
			name:   "skip: unknown chain selector",
			domain: setupTest(t, setupTestConfig, setupAddressbook),
			env:    fdeployment.Environment{Name: "staging", Logger: logger.Nop()},
			envcfg: func(tt *test) *fconfig.Config {
				envcfg, err := fconfig.Load(tt.domain, tt.env.Name, tt.env.Logger)
				require.NoError(t, err)
				envcfg.Networks = cfgnet.NewConfig([]cfgnet.Network{{ChainSelector: 123456789}})
				return envcfg //nolint:nlreturn
			},
			wantNumChains: 0,
		},
		{
			name:   "skip: selected zksync chain",
			domain: setupTest(t, setupTestConfig, setupAddressbook, setupDataStore, setupNodes),
			env:    fdeployment.Environment{Name: "staging", Logger: logger.Nop()},
			envcfg: func(tt *test) *fconfig.Config {
				envcfg, err := fconfig.Load(tt.domain, tt.env.Name, tt.env.Logger)
				require.NoError(t, err)
				return envcfg //nolint:nlreturn
			},
			options:       []LoadEnvironmentOption{WithAnvilKeyAsDeployer(), OnlyLoadChainsFor([]uint64{zksyncSepoliaSelector})},
			wantNumChains: 0,
		},
		{
			name:         "success",
			domain:       setupTest(t, setupTestConfig, setupAddressbook),
			env:          fdeployment.Environment{Name: "staging", Logger: logger.Nop()},
			providerType: ProviderTypeTestContainer,
			options:      []LoadEnvironmentOption{WithAnvilKeyAsDeployer(), OnlyLoadChainsFor([]uint64{gethTestnetSelector})},
			envcfg: func(tt *test) *fconfig.Config {
				envcfg, err := fconfig.Load(tt.domain, tt.env.Name, tt.env.Logger)
				require.NoError(t, err)
				envcfg.Networks = cfgnet.NewConfig([]cfgnet.Network{{
					ChainSelector: gethTestnetSelector,
					RPCs:          []cfgnet.RPC{{HTTPURL: localChainContainerRpcURL}},
				}})
				return envcfg //nolint:nlreturn
			},
			wantNumChains: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			envcfg := tt.envcfg(&tt)

			forkEnv, err := LoadForkFromEnv(t.Context(), tt.domain, tt.env, envcfg.Networks, envcfg.Env.Onchain,
				tt.blockNumbers, tt.providerType, tt.options...)
			t.Cleanup(func() {
				for _, container := range forkEnv.Containers {
					_ = container.Terminate(context.Background())
				}
			})

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Len(t, forkEnv.BlockChains.EVMChains(), tt.wantNumChains)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
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

			domain := setupTest(t, setupTestConfig, setupAddressbook, setupDataStore, setupNodes)
			environment := fdeployment.Environment{Name: "staging", Logger: logger.Nop()}
			envcfg, err := fconfig.Load(domain, environment.Name, environment.Logger)
			require.NoError(t, err)

			forkEnv, err := LoadForkFromEnv(t.Context(), domain, environment, envcfg.Networks,
				envcfg.Env.Onchain, tt.blockNumbers, ProviderTypeTestContainer, WithoutJD())
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

func startLocalChain(t *testing.T, chainID uint64) (string, string) {
	t.Helper()

	// create CTF network so container can talk to each other; ignore errors as it might already exist
	_, _ = testcontainers.GenericNetwork(t.Context(), testcontainers.GenericNetworkRequest{ //nolint:staticcheck
		Driver: "bridge",
		Name:   ctf.DefaultNetworkName,
		Labels: testcontainers.GenericLabels(),
	})

	ports, err := freeport.Take(1)
	require.NoError(t, err)

	input := &ctfblockchain.Input{
		ContainerName: fmt.Sprintf("anvil-local-chain-%d", time.Now().UnixNano()),
		Type:          ctfblockchain.TypeAnvil,
		ChainID:       strconv.FormatUint(chainID, 10),
		Port:          strconv.Itoa(ports[0]),
	}

	output, err := ctfblockchain.NewWithContext(t.Context(), input)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, output.Container)

	hostRpcURL := output.Nodes[0].ExternalHTTPUrl
	require.NotEmpty(t, hostRpcURL)
	containerRpcURL := output.Nodes[0].InternalHTTPUrl
	require.NotEmpty(t, containerRpcURL)

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		client, err := ethclient.DialContext(t.Context(), hostRpcURL)
		require.NoError(ct, err)
		_, err = client.BlockNumber(t.Context())
		require.NoError(ct, err)
	}, time.Second*5, time.Millisecond*100)

	return hostRpcURL, containerRpcURL
}
