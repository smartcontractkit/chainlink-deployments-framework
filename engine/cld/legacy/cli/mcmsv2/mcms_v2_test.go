package common

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	mocksdk "github.com/smartcontractkit/mcms/sdk/mocks"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
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
	var tests = []test{
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
