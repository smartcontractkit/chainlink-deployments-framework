package mcmsv2

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"os"
	"slices"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	mcmsevmbindings "github.com/smartcontractkit/mcms/sdk/evm/bindings"
	mocksdk "github.com/smartcontractkit/mcms/sdk/mocks"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	datastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	testenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/test/environment"
	testruntime "github.com/smartcontractkit/chainlink-deployments-framework/engine/test/runtime"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	fpointer "github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
)

// nolint:paralleltest // uses and modifies files
func TestMCMSv2CommandFlagParsing(t *testing.T) {
	lggr := logger.Test(t)
	require.NoError(t, os.MkdirAll("domains/exemplar", 0o755))

	t.Cleanup(func() {
		_ = os.RemoveAll("domains")
	})

	type test struct {
		name         string
		args         []string
		expected     commonFlagsv2
		wantExecErr  string
		wantParseErr string
	}
	tests := []test{
		{
			name: "check-quorum",
			args: []string{"check-quorum", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
			wantExecErr: "quorum not met",
		},
		{
			name: "execute-chain",
			args: []string{"execute-chain", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
		},
		{
			name: "execute-operation",
			args: []string{"execute-operation", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
		},
		{
			name: "set-root",
			args: []string{"set-root", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
		},
		{
			name: "is-timelock-ready",
			args: []string{"is-timelock-ready", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
		},
		{
			name: "is-timelock-done",
			args: []string{"is-timelock-done", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
		},
		{
			name: "is-timelock-operation-done",
			args: []string{"is-timelock-operation-done", "--index", "1", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
		},
		{
			name: "timelock-execute-chain",
			args: []string{"timelock-execute-chain", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
		},
		{
			name: "timelock-execute-operation",
			args: []string{"timelock-execute-operation", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
		},
		{
			name: "set-root missing proposal path",
			args: []string{"set-root", "-e", "staging", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "", // error only happens during execution
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
			wantExecErr: "required flag(s)",
		},
		{
			name: "get-op-count",
			args: []string{"get-op-count", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
		},
		{
			name: "set-root invalid proposal kind",
			args: []string{"set-root", "-e", "staging", "-p", "testdata/proposal.json", "-k", "InvalidProposal", "-s", "16015286601757825753"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "InvalidProposal",
				environmentStr:  "staging",
				chainSelector:   16015286601757825753,
			},
			wantParseErr: "unknown proposal kind 'InvalidProposal'",
		},
		{
			name: "analyze-proposal",
			args: []string{"analyze-proposal", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   0,
			},
		},
		{
			name: "convert-upf",
			args: []string{"convert-upf", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   0,
			},
		},
		{
			name: "execute-fork",
			args: []string{"execute-fork", "-e", "staging", "-p", "testdata/proposal.json", "-k", "TimelockProposal", "--fork"},
			expected: commonFlagsv2{
				proposalPath:    "testdata/proposal.json",
				proposalKindStr: "TimelockProposal",
				environmentStr:  "staging",
				chainSelector:   0,
				fork:            true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			proposalCtxProvider := func(environment cldf.Environment) (analyzer.ProposalContext, error) {
				return analyzer.NewDefaultProposalContext(environment)
			}

			cmd := BuildMCMSv2Cmd(lggr, domain.MustGetDomain("exemplar"), proposalCtxProvider)
			cmd.SilenceUsage = true
			subcmd, args, err := cmd.Traverse(test.args)
			require.NoError(t, err)
			err = subcmd.ParseFlags(args)
			require.NoError(t, err)

			f, err := parseCommonFlagsv2(cmd.Flags())
			if test.wantParseErr != "" {
				require.ErrorContains(t, err, test.wantParseErr)
				return
			}

			require.NoError(t, err)

			require.Equal(t, test.expected.proposalPath, f.proposalPath)
			require.Equal(t, test.expected.proposalKindStr, f.proposalKindStr)
			require.Equal(t, test.expected.environmentStr, f.environmentStr)
			require.Equal(t, test.expected.chainSelector, f.chainSelector)
			require.Equal(t, test.expected.fork, f.fork)

			t.Run("execute", func(t *testing.T) {
				t.Parallel()

				// // TODO RE-3333: remove this once we have a way to load secrets in the test environment
				t.Skipf("RE-3333: skipping execution of %s because it requires loading secrets", test.name)
				execCmd := BuildMCMSv2Cmd(lggr, domain.MustGetDomain("exemplar"), proposalCtxProvider)
				execCmd.SilenceUsage = true
				execCmd.SetArgs(test.args)
				if !test.expected.fork { // skip running the command if it's not a fork test
					return
				}
				// try actually running the command using the parsed flags
				err := execCmd.Execute()
				require.Equal(t, test.wantExecErr == "", err == nil)
				if test.wantExecErr != "" {
					require.ErrorContains(t, err, test.wantExecErr)
				}
			})
		})
	}
}

//nolint:paralleltest // global override is not safe for t.Parallel()
func TestGetProposalSigners(t *testing.T) {
	ctx := context.Background()

	chainSelector := types.ChainSelector(999)
	mcmAddress := "0xabc"

	type args struct {
		mockGetConfig func(*mocksdk.Inspector)
	}
	tests := []struct {
		name            string
		args            args
		expectError     bool
		expectedSigners []common.Address
	}{
		{
			name: "returns expected signers",
			args: args{
				mockGetConfig: func(inspector *mocksdk.Inspector) {
					cfg := &types.Config{}
					signers := []common.Address{
						common.HexToAddress("0x1111111111111111111111111111111111111111"),
						common.HexToAddress("0x2222222222222222222222222222222222222222"),
					}
					cfg.Signers = signers
					inspector.On("GetConfig", ctx, mcmAddress).Return(cfg, nil)
				},
			},
			expectedSigners: []common.Address{
				common.HexToAddress("0x1111111111111111111111111111111111111111"),
				common.HexToAddress("0x2222222222222222222222222222222222222222"),
			},
			expectError: false,
		},
		{
			name: "returns error when GetConfig fails",
			args: args{
				mockGetConfig: func(inspector *mocksdk.Inspector) {
					inspector.On("GetConfig", ctx, mcmAddress).Return(nil, errors.New("config error"))
				},
			},
			expectError: true,
		},
	}

	original := getInspectorFromChainSelector
	defer func() { getInspectorFromChainSelector = original }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := &mcms.Proposal{}
			proposal.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{
				chainSelector: {MCMAddress: mcmAddress},
			}
			cfg := &cfgv2{chainSelector: uint64(chainSelector)}
			mockInspector := mocksdk.NewInspector(t)
			tt.args.mockGetConfig(mockInspector)

			getInspectorFromChainSelector = func(cfgv2) (sdk.Inspector, error) {
				return mockInspector, nil
			}

			result, err := getProposalSigners(*cfg, ctx, proposal)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, map[types.ChainSelector][]common.Address{
					chainSelector: tt.expectedSigners,
				}, result)
			}
		})
	}
}

//nolint:paralleltest
func Test_timelockExecuteOptions(t *testing.T) {
	loader := testenv.NewLoader()
	env, err := loader.Load(t.Context(), testenv.WithEVMSimulatedN(t, 1))
	require.NoError(t, err)

	err = os.MkdirAll("domains/exemplar", 0o700)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll("domains") })

	lggr := logger.Test(t)
	exemplarDomain := domain.MustGetDomain("exemplar")
	chain := slices.Collect(maps.Values(env.BlockChains.EVMChains()))[0]
	callProxyAddress := common.Address{}
	timelockAddress := common.Address{}

	changeset := cldf.CreateChangeSet(
		func(e cldf.Environment, config struct{}) (cldf.ChangesetOutput, error) {
			ds := datastore.NewMemoryDataStore()
			ab := cldf.NewMemoryAddressBook()
			var tx *ethtypes.Transaction

			// deploy call proxy
			callProxyAddress, tx, _, err = mcmsevmbindings.DeployCallProxy(chain.DeployerKey, chain.Client, common.Address{})
			require.NoError(t, err)
			err = ds.Addresses().Add(datastore.AddressRef{
				Address:       callProxyAddress.Hex(),
				ChainSelector: chain.Selector,
				Type:          "CallProxy",
				Version:       semver.MustParse("1.0.0"),
			})
			require.NoError(t, err)
			err = ab.Save(chain.Selector, callProxyAddress.Hex(), cldf.MustTypeAndVersionFromString("CallProxy 1.0.0"))
			require.NoError(t, err)
			_, err = chain.Confirm(tx)
			require.NoError(t, err)

			// deploy timelock
			timelockAddress, tx, _, err = mcmsevmbindings.DeployRBACTimelock(chain.DeployerKey, chain.Client, big.NewInt(0),
				chain.DeployerKey.From,
				nil,                                // proposers
				[]common.Address{callProxyAddress}, // executors
				nil,                                // bypassers
				nil,                                // cancellers
			)
			require.NoError(t, err)
			err = ds.Addresses().Add(datastore.AddressRef{
				Address:       timelockAddress.Hex(),
				ChainSelector: chain.Selector,
				Type:          "RBACTimelock",
				Version:       semver.MustParse("1.0.0"),
			})
			require.NoError(t, err)
			err = ab.Save(chain.Selector, timelockAddress.Hex(), cldf.MustTypeAndVersionFromString("RBACTimelock 1.0.0"))
			require.NoError(t, err)
			_, err = chain.Confirm(tx)
			require.NoError(t, err)

			return cldf.ChangesetOutput{AddressBook: ab, DataStore: ds}, nil
		},
		func(e cldf.Environment, config struct{}) error { return nil }, // verify,
	)
	task := testruntime.ChangesetTask(changeset, struct{}{})
	runtime := testruntime.NewFromEnvironment(*env)
	err = runtime.Exec(task)
	require.NoError(t, err)
	env = fpointer.To(runtime.Environment())

	errorContains := func(msg string) func(t *testing.T, opts []mcms.Option, err error) {
		return func(t *testing.T, opts []mcms.Option, err error) {
			t.Helper()
			require.ErrorContains(t, err, msg)
		}
	}

	tests := []struct {
		name   string
		cfg    *cfgv2
		assert func(t *testing.T, opts []mcms.Option, err error)
		// want    []mcms.Option
		// wantErr string
	}{
		{
			name: "empty options for Solana",
			cfg:  &cfgv2{chainSelector: chainsel.SOLANA_MAINNET.Selector},
			assert: func(t *testing.T, opts []mcms.Option, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Empty(t, opts)
			},
		},
		{
			name: "empty options for Aptos",
			cfg:  &cfgv2{chainSelector: chainsel.APTOS_MAINNET.Selector},
			assert: func(t *testing.T, opts []mcms.Option, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Empty(t, opts)
			},
		},
		{
			name: "empty options for Sui",
			cfg:  &cfgv2{chainSelector: chainsel.SUI_MAINNET.Selector},
			assert: func(t *testing.T, opts []mcms.Option, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Empty(t, opts)
			},
		},
		{
			name: "CallProxy option added for EVM when addresses is in DataStore",
			cfg: &cfgv2{
				chainSelector: chain.Selector,
				env:           *env,
				blockchains:   env.BlockChains,
				timelockProposal: &mcms.TimelockProposal{
					TimelockAddresses: map[types.ChainSelector]string{
						types.ChainSelector(chain.Selector): timelockAddress.Hex(),
					},
				},
			},
			assert: func(t *testing.T, opts []mcms.Option, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, opts, 1)
			},
		},
		{
			name: "CallProxy option added when addresses is in AddressBook",
			cfg: &cfgv2{
				chainSelector: chain.Selector,
				blockchains:   env.BlockChains,
				env: func() cldf.Environment {
					modifiedEnv := *env
					modifiedEnv.DataStore = datastore.NewMemoryDataStore().Seal()

					return modifiedEnv
				}(),
				timelockProposal: &mcms.TimelockProposal{
					TimelockAddresses: map[types.ChainSelector]string{
						types.ChainSelector(chain.Selector): timelockAddress.Hex(),
					},
				},
			},
			assert: func(t *testing.T, opts []mcms.Option, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Len(t, opts, 1)
			},
		},
		{
			name: "failure: no timelock addresses for chain",
			cfg: &cfgv2{
				chainSelector: chain.Selector,
				env:           *env,
				blockchains:   env.BlockChains,
				timelockProposal: &mcms.TimelockProposal{
					TimelockAddresses: map[types.ChainSelector]string{
						types.ChainSelector(1): timelockAddress.Hex(),
					},
				},
			},
			assert: errorContains(fmt.Sprintf("failed to find timelock address for chain selector %d", chain.Selector)),
		},
		{
			name: "failure: address not found in DataStore or AddressBook",
			cfg: &cfgv2{
				chainSelector: chain.Selector,
				blockchains:   env.BlockChains,
				env: func() cldf.Environment {
					modifiedEnv := *env
					modifiedEnv.DataStore = datastore.NewMemoryDataStore().Seal()
					modifiedEnv.ExistingAddresses = cldf.NewMemoryAddressBook() //nolint:staticcheck

					return modifiedEnv
				}(),
				timelockProposal: &mcms.TimelockProposal{
					TimelockAddresses: map[types.ChainSelector]string{
						types.ChainSelector(chain.Selector): timelockAddress.Hex(),
					},
				},
			},
			assert: errorContains(fmt.Sprintf("failed to find call proxy contract for timelock %v", timelockAddress.Hex())),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := timelockExecuteOptions(t.Context(), lggr, exemplarDomain, tt.cfg)
			tt.assert(t, got, err)
		})
	}
}
