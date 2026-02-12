package mcmsv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/samber/lo"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	mcmsevmbindings "github.com/smartcontractkit/mcms/sdk/evm/bindings"
	mocksdk "github.com/smartcontractkit/mcms/sdk/mocks"
	"github.com/smartcontractkit/mcms/types"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	evmchain "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldf_env "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	cldf_scaffold "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/scaffold"
	testenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/test/environment"
	testruntime "github.com/smartcontractkit/chainlink-deployments-framework/engine/test/runtime"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// TestMCMSv2_DelegatedModularCommands verifies that modular commands are properly
// delegated from the legacy mcmsv2 command.
func TestMCMSv2_DelegatedModularCommands(t *testing.T) {
	t.Parallel()

	lggr := logger.Nop()
	domain := cldf_domain.NewDomain(t.TempDir(), "testdomain")
	proposalCtxProvider := func(_ cldf.Environment) (analyzer.ProposalContext, error) {
		return nil, nil //nolint:nilnil
	}

	cmd := BuildMCMSv2Cmd(lggr, domain, proposalCtxProvider)
	require.NotNil(t, cmd)

	// Verify migrated commands are available as subcommands
	migratedCommands := []string{"analyze-proposal", "convert-upf", "execute-fork", "error-decode-evm"}
	subcommandNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommandNames[sub.Use] = true
	}

	for _, cmdName := range migratedCommands {
		require.True(t, subcommandNames[cmdName], "expected migrated command '%s' to be available as subcommand of mcmsv2", cmdName)
	}

	// Verify each migrated command has expected flags (proving they come from modular package)
	flagTests := []struct {
		subcommand string
		flags      []string
	}{
		{
			subcommand: "analyze-proposal",
			flags:      []string{"environment", "proposal", "proposalKind", "output", "format"},
		},
		{
			subcommand: "convert-upf",
			flags:      []string{"environment", "proposal", "proposalKind", "output"},
		},
		{
			subcommand: "execute-fork",
			flags:      []string{"environment", "proposal", "proposalKind", "selector", "test-signer"},
		},
		{
			subcommand: "error-decode-evm",
			flags:      []string{"environment", "error-file"},
		},
	}

	for _, tt := range flagTests {
		t.Run(tt.subcommand, func(t *testing.T) {
			t.Parallel()

			subCmd, _, err := cmd.Find([]string{tt.subcommand})
			require.NoError(t, err)
			require.Equal(t, tt.subcommand, subCmd.Use)

			for _, flagName := range tt.flags {
				flag := subCmd.Flags().Lookup(flagName)
				require.NotNil(t, flag, "expected flag '%s' on delegated command '%s'", flagName, tt.subcommand)
			}
		})
	}
}

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
		// NOTE: The following commands have been migrated to engine/cld/commands/mcms
		// and have their own tests there. They use local flags instead of parent's
		// persistent flags, so flag parsing tests are not applicable here:
		// - analyze-proposal
		// - convert-upf
		// - execute-fork
		// - error-decode-evm
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			proposalCtxProvider := func(environment cldf.Environment) (analyzer.ProposalContext, error) {
				return analyzer.NewDefaultProposalContext(environment)
			}

			cmd := BuildMCMSv2Cmd(lggr, cldf_domain.MustGetDomain("exemplar"), proposalCtxProvider)
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
				execCmd := BuildMCMSv2Cmd(lggr, cldf_domain.MustGetDomain("exemplar"), proposalCtxProvider)
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

			getInspectorFromChainSelector = func(cfgv2) (mcmssdk.Inspector, error) {
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
	envp, err := loader.Load(t.Context(), testenv.WithEVMSimulatedN(t, 1))
	require.NoError(t, err)
	lggr := logger.Test(t)

	// FIXME: do we need to setup this folder?
	err = os.MkdirAll("domains/exemplar", 0o700)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll("domains") })

	chain := slices.Collect(maps.Values(envp.BlockChains.EVMChains()))[0]
	timelockAddress, _, env := deployTimelockAndCallProxy(t, *envp, chain, nil, nil, nil)

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
			name: "empty options for TON",
			cfg:  &cfgv2{chainSelector: chainsel.TON_MAINNET.Selector},
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
				env:           env,
				blockchains:   env.BlockChains,
				timelockProposal: &mcms.TimelockProposal{
					TimelockAddresses: map[types.ChainSelector]string{
						types.ChainSelector(chain.Selector): timelockAddress,
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
					modifiedEnv := env
					modifiedEnv.DataStore = datastore.NewMemoryDataStore().Seal()

					return modifiedEnv
				}(),
				timelockProposal: &mcms.TimelockProposal{
					TimelockAddresses: map[types.ChainSelector]string{
						types.ChainSelector(chain.Selector): timelockAddress,
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
				env:           env,
				blockchains:   env.BlockChains,
				timelockProposal: &mcms.TimelockProposal{
					TimelockAddresses: map[types.ChainSelector]string{
						types.ChainSelector(1): timelockAddress,
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
					modifiedEnv := env
					modifiedEnv.DataStore = datastore.NewMemoryDataStore().Seal()
					modifiedEnv.ExistingAddresses = cldf.NewMemoryAddressBook() //nolint:staticcheck

					return modifiedEnv
				}(),
				timelockProposal: &mcms.TimelockProposal{
					TimelockAddresses: map[types.ChainSelector]string{
						types.ChainSelector(chain.Selector): timelockAddress,
					},
				},
			},
			assert: errorContains(fmt.Sprintf("failed to find call proxy contract for timelock %v", timelockAddress)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := timelockExecuteOptions(t.Context(), lggr, tt.cfg)
			tt.assert(t, got, err)
		})
	}
}

func Test_setRootCommand(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	lggr, logs := logger.TestObserved(t, zapcore.InfoLevel)

	loader := testenv.NewLoader()
	env, err := loader.Load(t.Context(), testenv.WithEVMSimulatedN(t, 1))
	require.NoError(t, err)
	err = os.MkdirAll("domains/exemplar", 0o700)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll("domains") })

	chain := slices.Collect(maps.Values(env.BlockChains.EVMChains()))[0]
	inspector := mcmsevmsdk.NewInspector(chain.Client)

	privateKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	signer := mcms.NewPrivateKeySigner(privateKey)
	signerAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	tests := []struct {
		name            string
		cfg             *cfgv2
		setup           func(t *testing.T, cfg *cfgv2) (mcmAddress string)
		skipNonceErrors bool
		assertion       func(t require.TestingT, mcmAddress string, cfg *cfgv2, err error, args ...any)
	}{
		{
			name: "success",
			cfg: &cfgv2{
				kind:          types.KindTimelockProposal,
				chainSelector: chain.Selector,
				envStr:        env.Name,
				env:           *env,
				blockchains:   env.BlockChains,
			},
			setup: func(t *testing.T, cfg *cfgv2) string {
				t.Helper()
				mcmAddress, uenv := deployMcm(t, *env, chain, signerAddress)
				cfg.env = uenv
				cfg.proposal = testMcmProposal(t, chain, mcmAddress)
				signProposal(t, &cfg.proposal, signer, chain)

				return mcmAddress
			},
			assertion: func(t require.TestingT, mcmAddress string, cfg *cfgv2, err error, args ...any) {
				require.NoError(t, err)

				root, _, err := inspector.GetRoot(ctx, mcmAddress)
				require.NoError(t, err)

				merkleTree, err := cfg.proposal.MerkleTree()
				require.NoError(t, err)
				require.Equal(t, merkleTree.Root, root)
			},
		},
		{
			name: "success on retry",
			cfg: &cfgv2{
				kind:          types.KindTimelockProposal,
				chainSelector: chain.Selector,
				envStr:        env.Name,
				env:           *env,
				blockchains:   env.BlockChains,
			},
			setup: func(t *testing.T, cfg *cfgv2) string {
				t.Helper()

				mcmAddress, uenv := deployMcm(t, *env, chain, signerAddress)
				cfg.env = uenv
				cfg.proposal = testMcmProposal(t, chain, mcmAddress)
				signProposal(t, &cfg.proposal, signer, chain)

				// call SetRoot the first time
				err := setRootCommand(ctx, lggr, cfg)
				require.NoError(t, err)

				root, _, err := inspector.GetRoot(ctx, mcmAddress)
				require.NoError(t, err)
				merkleTree, err := cfg.proposal.MerkleTree()
				require.NoError(t, err)
				require.Equal(t, merkleTree.Root, root)

				return mcmAddress
			},
			assertion: func(t require.TestingT, mcmAddress string, cfg *cfgv2, err error, args ...any) {
				require.NoError(t, err)

				root, _, err := inspector.GetRoot(ctx, mcmAddress)
				require.NoError(t, err)
				merkleTree, err := cfg.proposal.MerkleTree()
				require.NoError(t, err)
				require.Equal(t, merkleTree.Root, root)

				require.Equal(t, 1, logs.FilterMessage(fmt.Sprintf("Root %v already set in MCM contract %v", root, mcmAddress)).Len())
			},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			mcmAddress := tt.setup(t, tt.cfg)
			err := setRootCommand(ctx, lggr, tt.cfg)
			tt.assertion(t, mcmAddress, tt.cfg, err)
		})
	}
}

func Test_executeChainCommand(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	lggr, logs := logger.TestObserved(t, zapcore.InfoLevel)

	loader := testenv.NewLoader()
	env, err := loader.Load(t.Context(), testenv.WithEVMSimulatedN(t, 1))
	require.NoError(t, err)
	err = os.MkdirAll("domains/exemplar", 0o700)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll("domains") })

	chain := slices.Collect(maps.Values(env.BlockChains.EVMChains()))[0]
	inspector := mcmsevmsdk.NewInspector(chain.Client)

	privateKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	signer := mcms.NewPrivateKeySigner(privateKey)
	signerAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	tests := []struct {
		name            string
		cfg             *cfgv2
		setup           func(t *testing.T, cfg *cfgv2) (mcmAddress string)
		skipNonceErrors bool
		assertion       func(t require.TestingT, mcmAddress string, cfg *cfgv2, err error, args ...any)
	}{
		{
			name: "success",
			cfg: &cfgv2{
				kind:          types.KindTimelockProposal,
				chainSelector: chain.Selector,
				envStr:        env.Name,
				env:           *env,
				blockchains:   env.BlockChains,
			},
			setup: func(t *testing.T, cfg *cfgv2) string {
				t.Helper()
				mcmAddress, uenv := deployMcm(t, *env, chain, signerAddress)
				cfg.env = uenv
				cfg.proposal = testMcmProposal(t, chain, mcmAddress)

				signProposal(t, &cfg.proposal, signer, chain)

				err := setRootCommand(ctx, lggr, cfg)
				require.NoError(t, err)

				return mcmAddress
			},
			assertion: func(t require.TestingT, mcmAddress string, cfg *cfgv2, err error, args ...any) {
				require.NoError(t, err)

				opCount, err := inspector.GetOpCount(ctx, mcmAddress)
				require.NoError(t, err)
				require.Equal(t, uint64(1), opCount)
			},
		},
		{
			name: "success on retry",
			cfg: &cfgv2{
				kind:          types.KindTimelockProposal,
				chainSelector: chain.Selector,
				envStr:        env.Name,
				env:           *env,
				blockchains:   env.BlockChains,
			},
			setup: func(t *testing.T, cfg *cfgv2) string {
				t.Helper()

				mcmAddress, uenv := deployMcm(t, *env, chain, signerAddress)
				cfg.env = uenv
				cfg.proposal = testMcmProposal(t, chain, mcmAddress)

				signProposal(t, &cfg.proposal, signer, chain)

				err := setRootCommand(ctx, lggr, cfg)
				require.NoError(t, err)

				err = executeChainCommand(ctx, lggr, cfg, false)
				require.NoError(t, err)

				opCount, err := inspector.GetOpCount(ctx, mcmAddress)
				require.NoError(t, err)
				require.Equal(t, uint64(1), opCount)

				return mcmAddress
			},
			assertion: func(t require.TestingT, mcmAddress string, cfg *cfgv2, err error, args ...any) {
				require.NoError(t, err)

				opCount, err := inspector.GetOpCount(ctx, mcmAddress)
				require.NoError(t, err)
				require.Equal(t, uint64(1), opCount)
				require.Equal(t, logs.FilterMessage("operation already executed").All()[0].ContextMap(), map[string]any{ //nolint:testifylint
					"index":   int64(0),
					"opCount": uint64(1),
					"txNonce": uint64(0),
				})
			},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			mcmAddress := tt.setup(t, tt.cfg)
			err := setRootCommand(ctx, lggr, tt.cfg)
			require.NoError(t, err)
			err = executeChainCommand(ctx, lggr, tt.cfg, tt.skipNonceErrors)
			tt.assertion(t, mcmAddress, tt.cfg, err)
		})
	}
}

func Test_fetchPipelinePRData(t *testing.T) {
	t.Parallel()

	proposalCtxProvider := func(env cldf.Environment) (analyzer.ProposalContext, error) {
		return analyzer.NewDefaultProposalContext(env)
	}
	defaultProposal := &mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Metadata: map[string]any{
				"pipelinePullRequest": map[string]any{
					"prNumber": 123,
					"branch":   "test-branch",
				},
			},
		},
	}

	tests := []struct {
		name          string
		setupEnv      func(t *testing.T, tempDir string) (cldf_domain.Domain, *cfgv2)
		setupMocks    func(t *testing.T) commandRunnerI
		expectedError bool
		assert        func(t *testing.T, cfg *cfgv2, err error, logs *observer.ObservedLogs)
	}{
		{
			name: "success: address book update",
			setupEnv: func(t *testing.T, tempDir string) (cldf_domain.Domain, *cfgv2) {
				t.Helper()
				env, domain := createTestEnv(t, tempDir)
				cfg := &cfgv2{timelockProposal: defaultProposal, env: env}

				return domain, cfg
			},
			setupMocks: func(t *testing.T) commandRunnerI {
				t.Helper()
				commandRunner := newMockcommandRunnerI(t)

				commandRunner.EXPECT().
					Run(mock.Anything, "gh", []string{"pr", "view", "123", "--json", "files", "--jq", ".files[].path"}).
					Return([]byte("domains/testdomain/testnet/addresses.json\ndomains/testdomain/testnet/nodes.json\n"), nil).
					Once()

				commandRunner.EXPECT().
					Run(mock.Anything, "gh", []string{"api", "-H", "Accept: application/vnd.github.v3.raw+json", "/repos/smartcontractkit/chainlink-deployments/contents/domains%2Ftestdomain%2Ftestnet%2Faddresses.json?ref=test-branch"}).
					Return([]byte(`{"3379446385462418246":{"0xc74182Dbb1f1d7f0DBB092Ce1aD0d321a6911B3b":{"Type":"LinkToken","Version":"1.0.0"}}}`), nil).
					Once()

				return commandRunner
			},
			expectedError: false,
			assert: func(t *testing.T, cfg *cfgv2, err error, logs *observer.ObservedLogs) {
				t.Helper()
				addresses, aerr := cfg.env.ExistingAddresses.AddressesForChain(chainsel.GETH_TESTNET.Selector) //nolint:staticcheck
				require.NoError(t, aerr)
				require.Contains(t, addresses, "0xc74182Dbb1f1d7f0DBB092Ce1aD0d321a6911B3b")
			},
		},
		{
			name: "success: datastore update with file datastore",
			setupEnv: func(t *testing.T, tempDir string) (cldf_domain.Domain, *cfgv2) {
				t.Helper()
				env, domain := createTestEnv(t, tempDir)
				cfg := &cfgv2{timelockProposal: defaultProposal, env: env}

				return domain, cfg
			},
			setupMocks: func(t *testing.T) commandRunnerI {
				t.Helper()
				commandRunner := newMockcommandRunnerI(t)

				commandRunner.EXPECT().
					Run(mock.Anything, "gh", []string{"pr", "view", "123", "--json", "files", "--jq", ".files[].path"}).
					Return([]byte("domains/testdomain/testnet/datastore/address_refs.json\ndomains/testdomain/testnet/nodes.json\n"), nil).
					Once()

				commandRunner.EXPECT().
					Run(mock.Anything, "gh", []string{"api", "-H", "Accept: application/vnd.github.v3.raw+json", "/repos/smartcontractkit/chainlink-deployments/contents/domains%2Ftestdomain%2Ftestnet%2Fdatastore%2Faddress_refs.json?ref=test-branch"}).
					Return([]byte(`[{"address":"0x18557992f55e7E53118af1ee8Ef1134C478f3426","chainSelector":3379446385462418246,"labels":[],"type":"LinkToken","version":"1.0.0"}]`), nil).
					Once()

				return commandRunner
			},
			assert: func(t *testing.T, cfg *cfgv2, err error, logs *observer.ObservedLogs) {
				t.Helper()
				refs := cfg.env.DataStore.Addresses().Filter(
					datastore.AddressRefByChainSelector(chainsel.GETH_TESTNET.Selector),
					datastore.AddressRefByAddress("0x18557992f55e7E53118af1ee8Ef1134C478f3426"),
				)
				require.Len(t, refs, 1)
			},
		},
		{
			name: "failure: no metadata in proposal",
			setupEnv: func(t *testing.T, tempDir string) (cldf_domain.Domain, *cfgv2) {
				t.Helper()
				env, domain := createTestEnv(t, tempDir)
				cfg := &cfgv2{
					timelockProposal: &mcms.TimelockProposal{BaseProposal: mcms.BaseProposal{Metadata: nil}},
					env:              env,
				}

				return domain, cfg
			},
			setupMocks: func(t *testing.T) commandRunnerI { return nil }, //nolint:thelper
			assert: func(t *testing.T, cfg *cfgv2, err error, logs *observer.ObservedLogs) {
				t.Helper()
				require.Equal(t, 1, logs.FilterMessage("proposal does not contain metadata; skipping pipeline PR data fetch").Len())
			},
		},
		{
			name: "failure: no pipeline PR metadata",
			setupEnv: func(t *testing.T, tempDir string) (cldf_domain.Domain, *cfgv2) {
				t.Helper()
				env, domain := createTestEnv(t, tempDir)
				cfg := &cfgv2{
					timelockProposal: &mcms.TimelockProposal{
						BaseProposal: mcms.BaseProposal{Metadata: map[string]any{"someOtherKey": "value"}},
					},
					env: env,
				}

				return domain, cfg
			},
			setupMocks: func(t *testing.T) commandRunnerI { return nil }, //nolint:thelper
			assert: func(t *testing.T, cfg *cfgv2, err error, logs *observer.ObservedLogs) {
				t.Helper()
				require.Equal(t, 1, logs.FilterMessage("pipeline PR metadata not found in proposal; skipping pipeline PR data fetch").Len())
			},
		},
		{
			name: "failure: invalid pipeline PR metadata format",
			setupEnv: func(t *testing.T, tempDir string) (cldf_domain.Domain, *cfgv2) {
				t.Helper()
				env, domain := createTestEnv(t, tempDir)
				cfg := &cfgv2{
					timelockProposal: &mcms.TimelockProposal{
						BaseProposal: mcms.BaseProposal{Metadata: map[string]any{"pipelinePullRequest": "invalid-format"}},
					},
					env: env,
				}

				return domain, cfg
			},
			setupMocks: func(t *testing.T) commandRunnerI { return nil }, //nolint:thelper
			assert: func(t *testing.T, cfg *cfgv2, err error, logs *observer.ObservedLogs) {
				t.Helper()
				require.ErrorContains(t, err, "error decoding pipeline PR metadata:")
			},
		},
		{
			name: "failure: invalid JSON in address book response",
			setupEnv: func(t *testing.T, tempDir string) (cldf_domain.Domain, *cfgv2) {
				t.Helper()
				env, domain := createTestEnv(t, tempDir)
				cfg := &cfgv2{timelockProposal: defaultProposal, env: env}

				return domain, cfg
			},
			setupMocks: func(t *testing.T) commandRunnerI {
				t.Helper()
				commandRunner := newMockcommandRunnerI(t)

				commandRunner.EXPECT().
					Run(mock.Anything, "gh", []string{"pr", "view", "123", "--json", "files", "--jq", ".files[].path"}).
					Return([]byte("domains/testdomain/testnet/addresses.json\ndomains/testdomain/testnet/nodes.json\n"), nil).
					Once()

				commandRunner.EXPECT().
					Run(mock.Anything, "gh", []string{"api", "-H", "Accept: application/vnd.github.v3.raw+json", "/repos/smartcontractkit/chainlink-deployments/contents/domains%2Ftestdomain%2Ftestnet%2Faddresses.json?ref=test-branch"}).
					Return([]byte(`invalid`), nil).
					Once()

				return commandRunner
			},
			assert: func(t *testing.T, cfg *cfgv2, err error, logs *observer.ObservedLogs) {
				t.Helper()
				require.Equal(t, 1, logs.FilterMessageSnippet("failed to update environment: failed to unmarshal address book JSON: ").Len())
			},
		},
		{
			name: "failure: invalid JSON in address refs response",
			setupEnv: func(t *testing.T, tempDir string) (cldf_domain.Domain, *cfgv2) {
				t.Helper()
				env, domain := createTestEnv(t, tempDir)
				cfg := &cfgv2{timelockProposal: defaultProposal, env: env}

				return domain, cfg
			},
			setupMocks: func(t *testing.T) commandRunnerI {
				t.Helper()
				commandRunner := newMockcommandRunnerI(t)

				commandRunner.EXPECT().
					Run(mock.Anything, "gh", []string{"pr", "view", "123", "--json", "files", "--jq", ".files[].path"}).
					Return([]byte("domains/testdomain/testnet/datastore/address_refs.json\ndomains/testdomain/testnet/nodes.json\n"), nil).
					Once()

				commandRunner.EXPECT().
					Run(mock.Anything, "gh", []string{"api", "-H", "Accept: application/vnd.github.v3.raw+json", "/repos/smartcontractkit/chainlink-deployments/contents/domains%2Ftestdomain%2Ftestnet%2Fdatastore%2Faddress_refs.json?ref=test-branch"}).
					Return([]byte(`invalid`), nil).
					Once()

				return commandRunner
			},
			assert: func(t *testing.T, cfg *cfgv2, err error, logs *observer.ObservedLogs) {
				t.Helper()
				require.Equal(t, 1, logs.FilterMessageSnippet("failed to update environment: failed to unmarshal address refs JSON").Len())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			domainsDir := filepath.Join(t.TempDir(), "domains")
			err := os.MkdirAll(domainsDir, 0o700)
			require.NoError(t, err)

			lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)
			domain, cfg := tt.setupEnv(t, domainsDir)
			commandRunner := tt.setupMocks(t)

			err = fetchPipelinePRData(t.Context(), lggr, domain, cfg, proposalCtxProvider, commandRunner)

			tt.assert(t, cfg, err, logs)
		})
	}
}

// ----- helpers and fixtures -----

func deployMcm(
	t *testing.T, env cldf.Environment, chain evmchain.Chain, signerAddress common.Address,
) (string, cldf.Environment) {
	t.Helper()

	mcmAddress := common.Address{}
	changeset := cldf.CreateChangeSet(
		func(e cldf.Environment, config struct{}) (cldf.ChangesetOutput, error) {
			ds := datastore.NewMemoryDataStore()
			var tx *ethtypes.Transaction

			// deploy mcm
			var mcmContract *mcmsevmbindings.ManyChainMultiSig
			var err error
			mcmAddress, tx, mcmContract, err = mcmsevmbindings.DeployManyChainMultiSig(chain.DeployerKey, chain.Client)
			require.NoError(t, err)
			_, err = chain.Confirm(tx)
			require.NoError(t, err)
			err = ds.Addresses().Add(datastore.AddressRef{
				Address:       mcmAddress.Hex(),
				ChainSelector: chain.Selector,
				Type:          "ManyChainMultiSig",
				Version:       semver.MustParse("1.0.0"),
			})
			require.NoError(t, err)

			// set config
			tx, err = mcmContract.SetConfig(chain.DeployerKey,
				[]common.Address{signerAddress}, // signerAddresses
				[]uint8{0},                      // signerGroups
				[32]uint8{1},                    // groupQuorums
				[32]uint8{0},                    // groupParents
				true,
			)
			require.NoError(t, err)
			_, err = chain.Confirm(tx)
			require.NoError(t, err)

			return cldf.ChangesetOutput{DataStore: ds}, nil
		},
		func(e cldf.Environment, config struct{}) error { return nil }, // verify,
	)

	task := testruntime.ChangesetTask(changeset, struct{}{})
	runtime := testruntime.NewFromEnvironment(env)
	err := runtime.Exec(task)
	require.NoError(t, err)

	return mcmAddress.Hex(), env
}

func deployTimelockAndCallProxy(
	t *testing.T, env cldf.Environment, chain evmchain.Chain, proposers []string, bypassers []string, cancellers []string,
) (string, string, cldf.Environment) {
	t.Helper()

	callProxyAddress := common.Address{}
	timelockAddress := common.Address{}
	changeset := cldf.CreateChangeSet(
		func(e cldf.Environment, config struct{}) (cldf.ChangesetOutput, error) {
			ds := datastore.NewMemoryDataStore()
			ab := cldf.NewMemoryAddressBook()
			var tx *ethtypes.Transaction
			var err error

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
				lo.Map(proposers, func(p string, _ int) common.Address { return common.HexToAddress(p) }),
				[]common.Address{callProxyAddress},
				lo.Map(bypassers, func(p string, _ int) common.Address { return common.HexToAddress(p) }),
				lo.Map(cancellers, func(p string, _ int) common.Address { return common.HexToAddress(p) }),
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
	runtime := testruntime.NewFromEnvironment(env)
	err := runtime.Exec(task)
	require.NoError(t, err)

	return timelockAddress.Hex(), callProxyAddress.Hex(), env
}

func testMcmProposal(
	t *testing.T,
	chain evmchain.Chain,
	mcmAddress string,
) mcms.Proposal {
	t.Helper()

	proposal, err := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(2082758399).
		SetDescription("test proposal").
		SetOverridePreviousRoot(true).
		AddChainMetadata(
			types.ChainSelector(chain.Selector),
			types.ChainMetadata{MCMAddress: mcmAddress},
		).
		AddOperation(types.Operation{
			ChainSelector: types.ChainSelector(chain.Selector),
			Transaction: types.Transaction{
				To:               chain.DeployerKey.From.Hex(),
				Data:             []byte("0x"),
				AdditionalFields: json.RawMessage(`{"value": 0}`),
			},
		}).Build()
	require.NoError(t, err)

	return *proposal
}

func testTimelockProposal(
	t *testing.T,
	chain evmchain.Chain,
	timelockAddress string,
	mcmAddress string,
) (mcms.TimelockProposal, mcms.Proposal) {
	t.Helper()

	timelockProposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(2082758399).
		SetDescription("test timelock proposal").
		SetOverridePreviousRoot(true).
		SetAction(types.TimelockActionSchedule).
		AddTimelockAddress(types.ChainSelector(chain.Selector), timelockAddress).
		AddChainMetadata(types.ChainSelector(chain.Selector), types.ChainMetadata{MCMAddress: mcmAddress}).
		AddOperation(types.BatchOperation{
			ChainSelector: types.ChainSelector(chain.Selector),
			Transactions: []types.Transaction{{
				To:               chain.DeployerKey.From.Hex(),
				Data:             []byte("0x"),
				AdditionalFields: json.RawMessage(`{"value": 0}`),
			}},
		}).Build()
	require.NoError(t, err)

	mcmProposal, _, err := timelockProposal.Convert(t.Context(), map[types.ChainSelector]mcmssdk.TimelockConverter{
		types.ChainSelector(chain.Selector): &mcmsevmsdk.TimelockConverter{},
	})
	require.NoError(t, err)

	return *timelockProposal, mcmProposal
}

func signProposal(
	t *testing.T, proposal *mcms.Proposal, signer *mcms.PrivateKeySigner, chain evmchain.Chain,
) {
	t.Helper()

	inspector := mcmsevmsdk.NewInspector(chain.Client)

	signable, err := mcms.NewSignable(proposal, map[types.ChainSelector]mcmssdk.Inspector{
		types.ChainSelector(chain.Selector): inspector,
	})
	require.NoError(t, err)

	_, err = signable.SignAndAppend(signer)
	require.NoError(t, err)
}

func createTestEnv(t *testing.T, tempDir string) (cldf.Environment, cldf_domain.Domain) {
	t.Helper()

	domain := cldf_domain.NewDomain(tempDir, "testdomain")
	err := cldf_scaffold.ScaffoldDomain(domain)
	require.NoError(t, err)

	err = cldf_scaffold.ScaffoldEnvDir(domain.EnvDir("testnet"))
	require.NoError(t, err)

	env, err := cldf_env.Load(t.Context(), domain, "testnet")
	require.NoError(t, err)

	return env, domain
}
