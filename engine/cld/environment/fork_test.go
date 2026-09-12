package environment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
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

// forkSkipTestSel is the fixture network's zkSync Era testnet selector. Fork
// tests pin it when they need LoadFork to succeed without Docker: the
// chain's local fixture RPC passes the health check and the chain is then skipped
// by the deliberate (non-fatal) zkSync-VM exclusion, yielding a fork
// environment with zero chains. Pinning any other fixture chain would fail
// loudly — a chain the caller explicitly requests is no longer silently
// dropped when it has no usable public RPC.
const forkSkipTestSel = uint64(6898391096552792247)

// setupForkTestConfig preserves the shared domain configuration but replaces
// fork RPCs with local, deterministic responses. None of these loader/error
// tests should depend on public archive health or credentials.
func setupForkTestConfig(t *testing.T, domain fdomain.Domain) {
	t.Helper()
	setupTestConfig(t, domain)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)

			return
		}
		var request struct {
			ID json.RawMessage `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x123"}`, request.ID)
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	networks := fmt.Sprintf(`networks:
  - type: testnet
    chain_selector: %d
    metadata:
      is_zksync: true
    rpcs:
      - rpc_name: local-healthy
        preferred_url_scheme: http
        http_url: %s
  - type: testnet
    chain_selector: 16015286601757825753
    rpcs:
      - rpc_name: local-unhealthy
        preferred_url_scheme: http
        http_url: %s/unhealthy
`, forkSkipTestSel, server.URL, server.URL)
	root, err := os.OpenRoot(domain.DirPath())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	require.NoError(t, root.WriteFile(".config/networks/networks-testnet.yaml", []byte(networks), 0600))
}

func Test_LoadForkedEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		domain        fdomain.Domain
		env           string
		blockNumbers  map[uint64]*big.Int
		options       []LoadEnvironmentOption
		wantName      string
		wantNumChains int
		wantErr       string
	}{
		{
			name:   "Invalid Environment",
			domain: fdomain.NewDomain("dummy", "test"),
			env:    "non_existent_env",
			blockNumbers: map[uint64]*big.Int{
				1: big.NewInt(1000),
			},
			wantErr: "failed to load config",
		},
		{
			name:   "AddressBook Failure",
			domain: setupTest(t, setupForkTestConfig, setupNodes),
			env:    "staging",
			blockNumbers: map[uint64]*big.Int{
				forkSkipTestSel: big.NewInt(1000),
			},
			options: []LoadEnvironmentOption{WithoutJD()},
			wantErr: "failed to load addressbook for domain test and environment staging:",
		},
		{
			name:   "DataStore Failure",
			domain: setupTest(t, setupForkTestConfig, setupNodes, setupAddressbook),
			env:    "staging",
			blockNumbers: map[uint64]*big.Int{
				forkSkipTestSel: big.NewInt(1000),
			},
			options: []LoadEnvironmentOption{WithoutJD()},
			wantErr: "failed to load datastore for domain test and environment staging:",
		},
		{
			name:         "Empty Block Numbers",
			domain:       setupTest(t, setupForkTestConfig, setupAddressbook),
			env:          "staging",
			blockNumbers: map[uint64]*big.Int{},
			wantErr:      "failed to create anvil chains",
		},
		{
			name:   "Invalid Nodes File",
			domain: setupTest(t, setupForkTestConfig, setupAddressbook),
			env:    "staging",
			blockNumbers: map[uint64]*big.Int{
				forkSkipTestSel: big.NewInt(1000),
			},
			wantErr: "failed to load nodes",
		},
		{
			name:   "OffchainClient Failure",
			domain: setupTest(t, setupForkTestConfig, setupAddressbook, setupNodes),
			env:    "staging",
			blockNumbers: map[uint64]*big.Int{
				forkSkipTestSel: big.NewInt(1000),
			},
			wantErr: "failed to load offchain client",
		},
		{
			name:          "Skip ZK Sync chain",
			domain:        setupTest(t, setupForkTestConfig, setupAddressbook, setupDataStore, setupNodes),
			env:           "staging",
			blockNumbers:  map[uint64]*big.Int{forkSkipTestSel: big.NewInt(1000)},
			options:       []LoadEnvironmentOption{WithoutJD(), WithAnvilKeyAsDeployer()},
			wantName:      "fork",
			wantNumChains: 0,
		},
		{
			// The local Sepolia fixture returns HTTP 503, so it can never
			// pass the health check: a chain the caller
			// pinned must fail loudly with the exclusion cause instead of
			// being silently dropped (which used to surface later as a
			// misleading "failed to get forked env's chain config" error).
			name:   "Requested Chain With No Public RPC",
			domain: setupTest(t, setupForkTestConfig, setupAddressbook, setupDataStore, setupNodes),
			env:    "staging",
			blockNumbers: map[uint64]*big.Int{
				16015286601757825753: big.NewInt(1000),
			},
			options: []LoadEnvironmentOption{WithoutJD(), WithAnvilKeyAsDeployer()},
			wantErr: "no forkable public RPC for requested chain selector 16015286601757825753",
		},
		// FIXME: CI can't reach https://optimism-sepolia.drpc.org anymore, so we'll
		// skip the test for now
		// {
		// 	name:          "No Error",
		// 	domain:        setupTest(t, setupForkTestConfig, setupAddressbook, setupDataStore, setupNodes),
		// 	env:           "staging",
		// 	blockNumbers:  map[uint64]*big.Int{5224473277236331295: big.NewInt(1000)},
		// 	options:       []LoadEnvironmentOption{WithoutJD(), WithAnvilKeyAsDeployer()},
		// 	wantName:      "fork",
		// 	wantNumChains: 1,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			forkEnv, err := LoadFork(t.Context(), tt.domain, tt.env, tt.blockNumbers, tt.options...)
			t.Cleanup(func() {
				for _, container := range forkEnv.Containers {
					_ = container.Terminate(context.Background())
				}
			})

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.wantName, forkEnv.Name)
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
				forkSkipTestSel: big.NewInt(1000),
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
				forkSkipTestSel: big.NewInt(1000),
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
				forkSkipTestSel: big.NewInt(1000),
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
				forkSkipTestSel: big.NewInt(1000),
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
				forkSkipTestSel: big.NewInt(1000),
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
				forkSkipTestSel: big.NewInt(1000),
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
				forkSkipTestSel: big.NewInt(1000),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			domain := setupTest(t, setupForkTestConfig, setupAddressbook, setupDataStore, setupNodes)

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
