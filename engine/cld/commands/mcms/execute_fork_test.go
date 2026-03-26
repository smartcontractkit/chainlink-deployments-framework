package mcms

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestExecuteFork_FlagShortcuts(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	// Find the execute-fork subcommand
	executeForkCmd, _, err := cmd.Find([]string{"execute-fork"})
	require.NoError(t, err)

	// Verify shorthand flags exist
	tests := []struct {
		longFlag  string
		shortFlag string
	}{
		{"environment", "e"},
		{"proposal", "p"},
		{"proposalKind", "k"},
		{"selector", "s"},
	}

	for _, tt := range tests {
		flag := executeForkCmd.Flags().Lookup(tt.longFlag)
		require.NotNil(t, flag, "flag %s should exist", tt.longFlag)
		require.Equal(t, tt.shortFlag, flag.Shorthand, "flag %s should have shorthand %s", tt.longFlag, tt.shortFlag)
	}
}

func TestExecuteFork_RequiredFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantErrMatch string
	}{
		{
			name:         "missing all required flags",
			args:         []string{"execute-fork"},
			wantErrMatch: "required flag",
		},
		{
			name:         "missing proposal flag",
			args:         []string{"execute-fork", "-e", "staging", "-s", "1"},
			wantErrMatch: "required flag",
		},
		{
			name:         "missing environment flag",
			args:         []string{"execute-fork", "-p", "/path/to/proposal.json", "-s", "1"},
			wantErrMatch: "required flag",
		},
		{
			name:         "missing selector flag",
			args:         []string{"execute-fork", "-e", "staging", "-p", "/path/to/proposal.json"},
			wantErrMatch: "required flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd, err := newTestCommand(t)
			require.NoError(t, err)

			cmd.SetArgs(tt.args)
			out := new(bytes.Buffer)
			cmd.SetOut(out)
			cmd.SetErr(out)

			execErr := cmd.Execute()
			require.ErrorContains(t, execErr, tt.wantErrMatch)
		})
	}
}

func TestExecuteFork_DefaultProposalKind(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	// Find the execute-fork subcommand
	executeForkCmd, _, err := cmd.Find([]string{"execute-fork"})
	require.NoError(t, err)

	// Verify default proposal kind - uses the types.KindTimelockProposal string value
	kindFlag := executeForkCmd.Flags().Lookup("proposalKind")
	require.NotNil(t, kindFlag)
	require.Equal(t, "TimelockProposal", kindFlag.DefValue, "default proposalKind should be 'TimelockProposal'")
}

func TestExecuteFork_TestSignerFlag(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	// Find the execute-fork subcommand
	executeForkCmd, _, err := cmd.Find([]string{"execute-fork"})
	require.NoError(t, err)

	// Verify test-signer flag exists and is boolean
	testSignerFlag := executeForkCmd.Flags().Lookup("test-signer")
	require.NotNil(t, testSignerFlag)
	require.Equal(t, "false", testSignerFlag.DefValue, "test-signer default should be false")
	require.Equal(t, "bool", testSignerFlag.Value.Type(), "test-signer should be boolean")
}

func TestExecuteFork_CommandDescriptions(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	// Find the execute-fork subcommand
	executeForkCmd, _, err := cmd.Find([]string{"execute-fork"})
	require.NoError(t, err)

	// Verify descriptions are set
	require.NotEmpty(t, executeForkCmd.Short, "Short description should be set")
	require.NotEmpty(t, executeForkCmd.Long, "Long description should be set")
	require.NotEmpty(t, executeForkCmd.Example, "Example should be set")
}

func TestForkConfig_Structure(t *testing.T) {
	t.Parallel()

	// Verify forkConfig has all necessary fields
	cfg := &forkConfig{}

	// These assertions verify the struct fields exist at compile time
	// and provide documentation of expected structure
	_ = cfg.kind
	_ = cfg.proposal
	_ = cfg.timelockProposal
	_ = cfg.chainSelector
	_ = cfg.blockchains
	_ = cfg.envStr
	_ = cfg.env
	_ = cfg.forkedEnv
	_ = cfg.fork
	_ = cfg.proposalCtx
}

func TestExecuteFork_Config_MissingProposalContextProvider(t *testing.T) {
	t.Parallel()

	// Create a command without ProposalContextProvider
	_, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain(t.TempDir(), "testdomain"),
		// ProposalContextProvider is missing
	})

	require.ErrorContains(t, err, "ProposalContextProvider")
}

func TestExecuteFork_OverrideForkChainDeployerKeyWithTestSigner(t *testing.T) {
	t.Parallel()

	const selector = uint64(4286062357653186312)

	tests := []struct {
		name     string
		chainID  string
		setupCfg func(t *testing.T) (*forkConfig, *bind.TransactOpts)
		assert   func(t *testing.T, cfg *forkConfig, prevTxOpts *bind.TransactOpts, err error)
	}{
		{
			name:    "success",
			chainID: "998",
			setupCfg: func(t *testing.T) (*forkConfig, *bind.TransactOpts) {
				t.Helper()
				prevPrivKey, err := crypto.GenerateKey()
				require.NoError(t, err)

				prevTxOpts, err := bind.NewKeyedTransactorWithChainID(prevPrivKey, big.NewInt(998))
				require.NoError(t, err)
				prevTxOpts.GasLimit = 10_000_000
				prevTxOpts.GasPrice = big.NewInt(100)
				prevTxOpts.GasTipCap = big.NewInt(2)
				prevTxOpts.GasFeeCap = big.NewInt(200)
				prevTxOpts.NoSend = true
				prevTxOpts.Context = context.Background()

				cfg := &forkConfig{
					chainSelector: selector,
					blockchains: chain.NewBlockChains(map[uint64]chain.BlockChain{
						selector: cldf_evm.Chain{
							Selector:    selector,
							DeployerKey: prevTxOpts,
						},
					}),
				}

				return cfg, prevTxOpts
			},
			assert: func(t *testing.T, cfg *forkConfig, prevTxOpts *bind.TransactOpts, err error) {
				t.Helper()
				require.NoError(t, err)
				updatedChain := cfg.blockchains.EVMChains()[selector]
				require.NotNil(t, updatedChain.DeployerKey)
				require.Equal(t, blockchain.DefaultAnvilPublicKey, updatedChain.DeployerKey.From.Hex())
				require.NotEqual(t, prevTxOpts.From, updatedChain.DeployerKey.From)
				require.Equal(t, prevTxOpts.GasLimit, updatedChain.DeployerKey.GasLimit)
				require.Equal(t, 0, prevTxOpts.GasPrice.Cmp(updatedChain.DeployerKey.GasPrice))
				require.Equal(t, 0, prevTxOpts.GasTipCap.Cmp(updatedChain.DeployerKey.GasTipCap))
				require.Equal(t, 0, prevTxOpts.GasFeeCap.Cmp(updatedChain.DeployerKey.GasFeeCap))
				require.Equal(t, prevTxOpts.NoSend, updatedChain.DeployerKey.NoSend)
				require.Equal(t, prevTxOpts.Context, updatedChain.DeployerKey.Context)
			},
		},
		{
			name:    "invalid chain id",
			chainID: "not-a-number",
			setupCfg: func(t *testing.T) (*forkConfig, *bind.TransactOpts) {
				t.Helper()
				cfg := &forkConfig{
					chainSelector: selector,
					blockchains: chain.NewBlockChains(map[uint64]chain.BlockChain{
						selector: cldf_evm.Chain{Selector: selector},
					}),
				}

				return cfg, nil
			},
			assert: func(t *testing.T, _ *forkConfig, _ *bind.TransactOpts, err error) {
				t.Helper()
				require.ErrorContains(t, err, "invalid chain id")
			},
		},
		{
			name:    "non evm chain",
			chainID: "998",
			setupCfg: func(t *testing.T) (*forkConfig, *bind.TransactOpts) {
				t.Helper()
				cfg := &forkConfig{
					chainSelector: selector,
					blockchains: chain.NewBlockChains(map[uint64]chain.BlockChain{
						selector: solana.Chain{Selector: selector},
					}),
				}

				return cfg, nil
			},
			assert: func(t *testing.T, _ *forkConfig, _ *bind.TransactOpts, err error) {
				t.Helper()
				require.ErrorContains(t, err, "is not an evm chain")
			},
		},
		{
			name:    "chain not found nil map",
			chainID: "998",
			setupCfg: func(t *testing.T) (*forkConfig, *bind.TransactOpts) {
				t.Helper()
				cfg := &forkConfig{
					chainSelector: selector,
					blockchains:   chain.NewBlockChains(nil),
				}

				return cfg, nil
			},
			assert: func(t *testing.T, _ *forkConfig, _ *bind.TransactOpts, err error) {
				t.Helper()
				require.ErrorContains(t, err, "chain selector")
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name:    "chain not found empty map",
			chainID: "998",
			setupCfg: func(t *testing.T) (*forkConfig, *bind.TransactOpts) {
				t.Helper()
				cfg := &forkConfig{
					chainSelector: selector,
					blockchains:   chain.NewBlockChains(map[uint64]chain.BlockChain{}),
				}

				return cfg, nil
			},
			assert: func(t *testing.T, _ *forkConfig, _ *bind.TransactOpts, err error) {
				t.Helper()
				require.ErrorContains(t, err, "chain selector")
				require.ErrorContains(t, err, "not found")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, prevTxOpts := tc.setupCfg(t)
			err := overrideForkChainDeployerKeyWithTestSigner(cfg, tc.chainID)
			tc.assert(t, cfg, prevTxOpts, err)
		})
	}
}

func TestExecuteFork_AddDecodedRevertReason(t *testing.T) {
	t.Parallel()

	mcmABI := `[{"inputs":[{"type":"bytes"}],"name":"CallReverted","type":"error"}]`
	newRegistry := func(t *testing.T) *analyzer.MockEVMABIRegistry {
		t.Helper()
		r := analyzer.NewMockEVMABIRegistry(t)
		r.EXPECT().GetAllABIs().Return(map[string]string{"MCM 1.0.0": mcmABI})

		return r
	}

	tests := []struct {
		name   string
		setup  func(t *testing.T) (analyzer.ProposalContext, error)
		assert func(t *testing.T, input error, result error)
	}{
		{
			name: "nil error returns nil",
			setup: func(_ *testing.T) (analyzer.ProposalContext, error) {
				return nil, nil //nolint:nilnil // intentional: testing nil-input path
			},
			assert: func(t *testing.T, _ error, result error) {
				t.Helper()
				assert.NoError(t, result)
			},
		},
		{
			name: "nil proposalCtx returns original error",
			setup: func(_ *testing.T) (analyzer.ProposalContext, error) {
				return nil, errors.New("some error")
			},
			assert: func(t *testing.T, input error, result error) {
				t.Helper()
				assert.Equal(t, input, result)
			},
		},
		{
			name: "nil registry returns original error",
			setup: func(t *testing.T) (analyzer.ProposalContext, error) {
				t.Helper()
				mock := analyzer.NewMockProposalContext(t)
				mock.EXPECT().GetEVMRegistry().Return(nil)

				return mock, errors.New("some error")
			},
			assert: func(t *testing.T, input error, result error) {
				t.Helper()
				assert.Equal(t, input, result)
			},
		},
		{
			name: "plain error without hex data returns original",
			setup: func(t *testing.T) (analyzer.ProposalContext, error) {
				t.Helper()
				mock := analyzer.NewMockProposalContext(t)
				mock.EXPECT().GetEVMRegistry().Return(newRegistry(t))

				return mock, errors.New("plain error without any hex data")
			},
			assert: func(t *testing.T, input error, result error) {
				t.Helper()
				assert.Equal(t, input, result)
			},
		},
		{
			name: "strategy 1: decodes rpc.DataError from error chain",
			setup: func(t *testing.T) (analyzer.ProposalContext, error) {
				t.Helper()
				mock := analyzer.NewMockProposalContext(t)
				mock.EXPECT().GetEVMRegistry().Return(newRegistry(t))
				innerRevert := packErrorString(t, "access denied")
				outerHex := packCallReverted(t, mcmABI, innerRevert)

				return mock, &mockDataError{msg: "execution reverted", data: "0x" + outerHex}
			},
			assert: func(t *testing.T, input error, result error) {
				t.Helper()
				require.Error(t, result)
				assert.Contains(t, result.Error(), "decoded:")
				assert.Contains(t, result.Error(), "access denied")
				assert.ErrorIs(t, result, input)
			},
		},
		{
			name: "strategy 2: decodes hex from error string after DecodeErr consumed DataError",
			setup: func(t *testing.T) (analyzer.ProposalContext, error) {
				t.Helper()
				mock := analyzer.NewMockProposalContext(t)
				mock.EXPECT().GetEVMRegistry().Return(newRegistry(t))
				innerRevert := packErrorString(t, "not authorized")
				outerHex := packCallReverted(t, mcmABI, innerRevert)

				return mock, fmt.Errorf("error executing chain op 0: contract error: error -`CallReverted` args [0x%s]", outerHex)
			},
			assert: func(t *testing.T, input error, result error) {
				t.Helper()
				require.Error(t, result)
				assert.Contains(t, result.Error(), "decoded:")
				assert.Contains(t, result.Error(), "not authorized")
				assert.ErrorIs(t, result, input)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pCtx, input := tc.setup(t)
			result := addDecodedRevertReason(logger.Nop(), input, pCtx)
			tc.assert(t, input, result)
		})
	}
}

func TestExecuteFork_TryDecodeHexFromErrorString(t *testing.T) {
	t.Parallel()

	mcmABI := `[{"inputs":[{"type":"bytes"}],"name":"CallReverted","type":"error"}]`
	registry := analyzer.NewMockEVMABIRegistry(t)
	registry.EXPECT().GetAllABIs().Return(map[string]string{"MCM 1.0.0": mcmABI})
	dec, err := NewErrDecoder(registry)
	require.NoError(t, err)

	tests := []struct {
		name         string
		buildErrStr  func(t *testing.T) string
		wantEmpty    bool
		wantContains []string
	}{
		{
			name:        "no hex in string",
			buildErrStr: func(*testing.T) string { return "just a plain error" },
			wantEmpty:   true,
		},
		{
			name:        "hex too short to be a selector",
			buildErrStr: func(*testing.T) string { return "failed with 0xdead" },
			wantEmpty:   true,
		},
		{
			name:        "hex with unknown selector",
			buildErrStr: func(*testing.T) string { return "error 0xdeadbeefcafebabe" },
			wantEmpty:   true,
		},
		{
			name: "decodes Error(string) hex",
			buildErrStr: func(t *testing.T) string {
				t.Helper()

				hexPayload := packErrorString(t, "test revert")

				return fmt.Sprintf("some error with 0x%s embedded", hexPayload)
			},
			wantContains: []string{"test revert"},
		},
		{
			name: "decodes CallReverted wrapping Error(string)",
			buildErrStr: func(t *testing.T) string {
				t.Helper()

				innerRevert := packErrorString(t, "forbidden")
				outerHex := packCallReverted(t, mcmABI, innerRevert)

				return fmt.Sprintf("contract error: error -`CallReverted` args [0x%s]", outerHex)
			},
			wantContains: []string{"CallReverted", "forbidden"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tryDecodeHexFromErrorString(tc.buildErrStr(t), dec)
			if tc.wantEmpty {
				assert.Empty(t, result)
			} else {
				for _, want := range tc.wantContains {
					assert.Contains(t, result, want)
				}
			}
		})
	}
}

// --- test helpers ---

// mockDataError implements ethrpc.DataError for testing strategy 1.
type mockDataError struct {
	msg  string
	data interface{}
}

func (e *mockDataError) Error() string          { return e.msg }
func (e *mockDataError) ErrorData() interface{} { return e.data }

// packErrorString ABI-encodes a standard Error(string) revert and returns
// the raw hex (no 0x prefix).
func packErrorString(t *testing.T, msg string) string {
	t.Helper()
	// Error(string) selector: 0x08c379a0
	parsedABI, err := abi.JSON(strings.NewReader(`[{"inputs":[{"type":"string"}],"name":"Error","type":"error"}]`))
	require.NoError(t, err)
	errDef := parsedABI.Errors["Error"]
	packed, err := errDef.Inputs.Pack(msg)
	require.NoError(t, err)

	return hex.EncodeToString(append(errDef.ID[:4], packed...))
}

// packCallReverted wraps innerHex (no 0x prefix) into a CallReverted(bytes)
// payload and returns the full hex (no 0x prefix).
func packCallReverted(t *testing.T, mcmABIJSON string, innerHex string) string {
	t.Helper()
	innerBytes, err := hex.DecodeString(innerHex)
	require.NoError(t, err)

	parsed, err := abi.JSON(strings.NewReader(mcmABIJSON))
	require.NoError(t, err)
	crErr := parsed.Errors["CallReverted"]
	packed, err := crErr.Inputs.Pack(innerBytes)
	require.NoError(t, err)

	return hex.EncodeToString(append(crErr.ID[:4], packed...))
}
