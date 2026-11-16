package mcms

import (
	"context"
	"errors"
	"fmt"
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
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestLoadProposalConfig(t *testing.T) {
	t.Parallel()

	domainsRoot := filepath.Clean(filepath.Join(integrationModulePath, "..", "testdata", "domains"))
	domain := cldfdomain.NewDomain(domainsRoot, integrationDomainName)

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
		dom                 cldfdomain.Domain
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
					_ context.Context, _ cldfdomain.Domain, _ string, _ logger.Logger, _ ...cldfenv.LoadEnvironmentOption,
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
			dom:  domain,
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
			dom:  domain,
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
			dom:  domain,
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
				BuildParams: &cldfenv.BuildParams{
					Networks: []cfgnet.Network{{
						Type:          cfgnet.NetworkTypeTestnet,
						ChainSelector: chainsel.GETH_TESTNET.Selector,
						RPCs:          []cfgnet.RPC{{HTTPURL: fmt.Sprintf("http://rpc.cldev.sh/%d", chainsel.GETH_TESTNET.Selector)}},
					}},
				},
			},
			deps: &Deps{
				ProposalLoader:        mcms.LoadProposal,
				EnvironmentBuilder:    (&fakeBuilder{selector: []uint64{chainsel.GETH_TESTNET.Selector}}).build,
				ForkEnvironmentLoader: nopForkEnvLoader,
			},
			opts: []any{randomSalt},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, got.TimelockProposal.SaltOverride)
			},
		},
		{
			name: "failure: fork environment load error",
			flags: ProposalFlags{
				ProposalKind:  string(mcmstypes.KindTimelockProposal),
				ProposalPath:  proposalFilePath,
				Environment:   "testnet",
				ChainSelector: chainsel.GETH_TESTNET.Selector,
				Fork:          true,
				BuildParams: &cldfenv.BuildParams{
					Networks: []cfgnet.Network{{
						Type:          cfgnet.NetworkTypeTestnet,
						ChainSelector: chainsel.GETH_TESTNET.Selector,
						RPCs:          []cfgnet.RPC{{HTTPURL: fmt.Sprintf("http://rpc.cldev.sh/%d", chainsel.GETH_TESTNET.Selector)}},
					}},
				},
			},
			deps: &Deps{
				ProposalLoader:     mcms.LoadProposal,
				EnvironmentBuilder: (&fakeBuilder{selector: []uint64{chainsel.GETH_TESTNET.Selector}}).build,
				ForkEnvironmentLoader: func(_ context.Context, _ cldfdomain.Domain, env cldf.Environment, _ *cfgnet.Config,
					_ cfgenv.OnchainConfig, _ map[uint64]*big.Int, _ cldfenv.ProviderType,
					_ ...cldfenv.LoadEnvironmentOption,
				) (cldfenv.ForkedEnvironment, error) {
					return cldfenv.ForkedEnvironment{}, errors.New("fork environment load error")
				},
			},
			assert: func(t *testing.T, got *ProposalConfig, err error) {
				t.Helper()
				require.ErrorContains(t, err, "error loading forked environment: fork environment load error")
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

// ----- helpers ---------------------------------------------------------------

func nopEnvLoader(
	_ context.Context, dom cldfdomain.Domain, envKey string, _ logger.Logger, _ ...cldfenv.LoadEnvironmentOption,
) (cldf.Environment, error) {
	return cldf.Environment{Name: envKey}, nil
}

func nopForkEnvLoader(
	_ context.Context, _ cldfdomain.Domain, env cldf.Environment, _ *cfgnet.Config, _ cfgenv.OnchainConfig,
	_ map[uint64]*big.Int, _ cldfenv.ProviderType, _ ...cldfenv.LoadEnvironmentOption,
) (cldfenv.ForkedEnvironment, error) {
	return cldfenv.ForkedEnvironment{Environment: cldf.Environment{Name: env.Name}}, nil
}
