package mcms

import (
	"context"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainwrappers"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestLoadProposalConfig(t *testing.T) {
	t.Parallel()

	// Write a valid TimelockProposal to a shared temp file for tests that need real loading.
	proposalFilePath := filepath.Join(t.TempDir(), "proposal.json")
	require.NoError(t, os.WriteFile(proposalFilePath, testProposalWithoutChangesetsJSON, 0o600))
	proposal, err := mcms.LoadProposal(mcmstypes.KindTimelockProposal, proposalFilePath)
	require.NoError(t, err)
	timelockProposal := proposal.(*mcms.TimelockProposal)
	converters, err := chainwrappers.BuildConverters(timelockProposal.ChainMetadata)
	require.NoError(t, err)
	mcmProposal, _, err := timelockProposal.Convert(t.Context(), converters)
	require.NoError(t, err)

	tests := []struct {
		name                string
		dom                 domain.Domain
		deps                *Deps
		proposalCtxProvider analyzer.ProposalContextProvider
		flags               ProposalFlags
		opts                []any
		assert              func(t *testing.T, got *ProposalConfig, err error)
	}{
		{
			name:  "failure: unknown proposal kind",
			flags: ProposalFlags{ProposalKind: "UnknownKind"},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()
				require.ErrorContains(t, err, "unknown proposal kind 'UnknownKind'")
			},
		},
		{
			name: "failure: proposal loader returns error",
			flags: ProposalFlags{
				ProposalKind:  string(mcmstypes.KindTimelockProposal),
				Environment:   "testnet",
				ChainSelector: chainsel.GETH_TESTNET.Selector,
			},
			deps: &Deps{
				ProposalLoader: func(_ mcmstypes.ProposalKind, _ string) (mcms.ProposalInterface, error) {
					return nil, errors.New("failed to read file")
				},
				EnvironmentLoader: nopEnvLoader,
			},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()
				require.ErrorContains(t, err, "error loading proposal: failed to read file")
			},
		},
		{
			name: "failure: environment loader returns error",
			flags: ProposalFlags{
				ProposalKind:  string(mcmstypes.KindProposal),
				Environment:   "testnet",
				ChainSelector: chainsel.GETH_TESTNET.Selector,
			},
			deps: &Deps{
				ProposalLoader: func(_ mcmstypes.ProposalKind, _ string) (mcms.ProposalInterface, error) {
					return &mcms.Proposal{}, nil
				},
				EnvironmentLoader: func(
					_ context.Context, _ domain.Domain, _ string, _ logger.Logger, _ ...cldfenv.LoadEnvironmentOption,
				) (cldf.Environment, error) {
					return cldf.Environment{}, errors.New("environment error")
				},
			},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()
				require.ErrorContains(t, err, "error loading environment: environment error")
			},
		},
		{
			name: "failure: expired proposal without acceptExpiredProposal option",
			flags: ProposalFlags{
				ProposalKind:  string(mcmstypes.KindTimelockProposal),
				Environment:   "testnet",
				ChainSelector: chainsel.GETH_TESTNET.Selector,
			},
			deps: &Deps{
				ProposalLoader: func(_ mcmstypes.ProposalKind, _ string) (mcms.ProposalInterface, error) {
					return nil, errors.New("proposal has expired: valid_until exceeded")
				},
				EnvironmentLoader: nopEnvLoader,
			},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()
				require.ErrorContains(t, err, "error loading proposal: proposal has expired: valid_until exceeded")
			},
		},
		{
			name: "failure: proposal context provider returns error",
			flags: ProposalFlags{
				ProposalKind:  string(mcmstypes.KindProposal),
				Environment:   "testnet",
				ChainSelector: chainsel.GETH_TESTNET.Selector,
			},
			deps: &Deps{
				ProposalLoader: func(_ mcmstypes.ProposalKind, _ string) (mcms.ProposalInterface, error) {
					return &mcms.Proposal{}, nil
				},
				EnvironmentLoader: nopEnvLoader,
			},
			proposalCtxProvider: func(_ cldf.Environment) (analyzer.ProposalContext, error) {
				return nil, errors.New("failed to build proposal context")
			},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()
				require.ErrorContains(t, err, "error creating proposal context")
			},
		},
		{
			name: "success: loads proposal config",
			flags: ProposalFlags{
				ProposalPath:  proposalFilePath,
				ProposalKind:  string(mcmstypes.KindTimelockProposal),
				Environment:   "testnet",
				ChainSelector: chainsel.GETH_TESTNET.Selector,
			},
			deps: &Deps{
				ProposalLoader: func(_ mcmstypes.ProposalKind, _ string) (mcms.ProposalInterface, error) {
					return proposal, nil
				},
				EnvironmentLoader: nopEnvLoader,
			},
			proposalCtxProvider: func(_ cldf.Environment) (analyzer.ProposalContext, error) {
				return nil, nil //nolint:nilnil
			},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()

				want := &ProposalConfig{
					Kind:             mcmstypes.KindTimelockProposal,
					Proposal:         mcmProposal,
					TimelockProposal: timelockProposal,
					ChainSelector:    chainsel.GETH_TESTNET.Selector,
					EnvStr:           "testnet",
					Env:              cldf.Environment{Name: "testnet"},
				}

				require.NoError(t, err)
				require.Equal(t, want, got)
			},
		},
		{
			name: "success: loads proposal with acceptExpiredProposal option",
			flags: ProposalFlags{
				ProposalKind:  string(mcmstypes.KindTimelockProposal),
				ProposalPath:  proposalFilePath,
				Environment:   "testnet",
				ChainSelector: chainsel.GETH_TESTNET.Selector,
			},
			deps: &Deps{
				// load the real proposal but inject an artificial expiry error.
				ProposalLoader: func(kind mcmstypes.ProposalKind, _ string) (mcms.ProposalInterface, error) {
					return proposal, errors.New("proposal has expired: valid_until exceeded")
				},
				EnvironmentLoader: nopEnvLoader,
			},
			opts: []any{acceptExpiredProposal},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Equal(t, mcmstypes.KindTimelockProposal, got.Kind)
				require.Equal(t, "testnet", got.EnvStr)
				require.Equal(t, proposal, got.TimelockProposal)
			},
		},
		{
			name: "success: loads proposal with randomSalt option",
			flags: ProposalFlags{
				ProposalKind:  string(mcmstypes.KindTimelockProposal),
				ProposalPath:  proposalFilePath,
				Environment:   "testnet",
				ChainSelector: chainsel.GETH_TESTNET.Selector,
				Fork:          true,
			},
			deps: &Deps{
				ProposalLoader:        mcms.LoadProposal,
				EnvironmentLoader:     nopEnvLoader,
				ForkEnvironmentLoader: nopForkEnvLoader,
			},
			opts: []any{randomSalt},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, got.TimelockProposal.SaltOverride)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := LoadProposalConfig(t.Context(), logger.Nop(), tt.dom, tt.deps,
				tt.proposalCtxProvider, tt.flags, tt.opts...)

			tt.assert(t, got, err)
		})
	}
}

func TestLoadProposalConfig_ForkBlockNumber(t *testing.T) {
	t.Parallel()

	proposalFilePath := filepath.Join(t.TempDir(), "proposal.json")
	require.NoError(t, os.WriteFile(proposalFilePath, testProposalWithoutChangesetsJSON, 0o600))

	var gotBlocks map[uint64]*big.Int
	deps := &Deps{
		ProposalLoader: mcms.LoadProposal,
		ForkEnvironmentLoader: func(_ context.Context, _ domain.Domain, _ string, blockNumbers map[uint64]*big.Int, _ ...cldfenv.LoadEnvironmentOption) (cldfenv.ForkedEnvironment, error) {
			gotBlocks = blockNumbers
			return cldfenv.ForkedEnvironment{Environment: cldf.Environment{Name: "testnet"}}, nil
		},
	}
	flags := ProposalFlags{
		ProposalKind:  string(mcmstypes.KindTimelockProposal),
		ProposalPath:  proposalFilePath,
		Environment:   "testnet",
		ChainSelector: chainsel.GETH_TESTNET.Selector,
		Fork:          true,
	}

	_, err := LoadProposalConfig(t.Context(), logger.Nop(), domain.Domain{}, deps, nil, flags,
		ForkBlockNumberOption(12345))
	require.NoError(t, err)
	require.Equal(t, map[uint64]*big.Int{flags.ChainSelector: big.NewInt(12345)}, gotBlocks)
}

// ----- helpers -----

func nopEnvLoader(
	_ context.Context, _ domain.Domain, envKey string, _ logger.Logger, _ ...cldfenv.LoadEnvironmentOption,
) (cldf.Environment, error) {
	return cldf.Environment{Name: envKey}, nil
}

func nopForkEnvLoader(
	_ context.Context, _ domain.Domain, envKey string, _ map[uint64]*big.Int, _ ...cldfenv.LoadEnvironmentOption,
) (cldfenv.ForkedEnvironment, error) {
	return cldfenv.ForkedEnvironment{Environment: cldf.Environment{Name: envKey}}, nil
}
