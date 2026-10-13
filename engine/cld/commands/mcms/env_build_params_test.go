package mcms

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const (
	buildParamsDomain = "testdomain"
	buildParamsEnv    = "testnet"
)

var buildParamsSelector = chainsel.GETH_TESTNET.Selector

// buildParamsDoc lists the selector used by the test proposal.
const buildParamsDoc = `
domain: testdomain
environment: testnet
catalog:
  grpc: catalog.example.com:443
networks:
  - chain_selector: 3379446385462418246
`

// fakeBuilder records its calls.
type fakeBuilder struct {
	calls    int
	params   cldfenv.BuildParams
	selector []uint64
	err      error
}

func (f *fakeBuilder) build(
	_ context.Context, params cldfenv.BuildParams, _ ...cldfenv.BuildOption,
) (cldf.Environment, error) {
	f.calls++
	f.params = params
	if f.err != nil {
		return cldf.Environment{}, f.err
	}

	chains := map[uint64]chain.BlockChain{}
	for _, sel := range f.selector {
		chains[sel] = evm.Chain{Selector: sel}
	}

	return cldf.Environment{
		Name:        params.Environment,
		BlockChains: chain.NewBlockChains(chains),
		DataStore:   fdatastore.NewMemoryDataStore().Seal(),
	}, nil
}

func failingEnvLoader(t *testing.T) EnvironmentLoaderFunc {
	t.Helper()

	return func(
		context.Context, domain.Domain, string, logger.Logger, ...cldfenv.LoadEnvironmentOption,
	) (cldf.Environment, error) {
		t.Error("the file based environment loader must not be called")

		return cldf.Environment{}, errors.New("unexpected call")
	}
}

func TestBuildEnvironment(t *testing.T) {
	t.Parallel()

	params := cldfenv.BuildParams{Domain: buildParamsDomain, Environment: buildParamsEnv}
	dom := domain.NewDomain("/does/not/exist", buildParamsDomain)

	tests := []struct {
		name      string
		dom       domain.Domain
		builder   *fakeBuilder
		selectors []uint64
		wantErr   string
	}{
		{
			name:      "success",
			dom:       dom,
			builder:   &fakeBuilder{selector: []uint64{buildParamsSelector}},
			selectors: []uint64{buildParamsSelector},
		},
		{
			name:    "domain mismatch",
			dom:     domain.NewDomain("/does/not/exist", "other"),
			builder: &fakeBuilder{},
			wantErr: `build params are for domain "testdomain", but this binary is for domain "other"`,
		},
		{
			name:    "builder error",
			dom:     dom,
			builder: &fakeBuilder{err: errors.New("catalog unreachable")},
			wantErr: "failed to build environment from build params: catalog unreachable",
		},
		{
			name:      "selector missing from networks is named",
			dom:       dom,
			builder:   &fakeBuilder{selector: []uint64{buildParamsSelector}},
			selectors: []uint64{buildParamsSelector, 42},
			wantErr:   "chain selector 42 not found in the supplied build params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps := &Deps{EnvironmentBuilder: tt.builder.build}

			env, err := buildEnvironment(t.Context(), logger.Nop(), tt.dom, deps, params, tt.selectors)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, buildParamsEnv, env.Name)
			require.Equal(t, 1, tt.builder.calls)
			require.Equal(t, params, tt.builder.params)
		})
	}
}

func TestBuildEnvironment_WarnsWhenCatalogIsEmpty(t *testing.T) {
	t.Parallel()

	lggr, logs := logger.TestObserved(t, zapcore.WarnLevel)
	builder := &fakeBuilder{selector: []uint64{buildParamsSelector}}
	params := cldfenv.BuildParams{Domain: buildParamsDomain, Environment: buildParamsEnv}

	_, err := buildEnvironment(t.Context(), lggr, domain.NewDomain("/x", buildParamsDomain),
		&Deps{EnvironmentBuilder: builder.build}, params, []uint64{buildParamsSelector})

	require.NoError(t, err)
	require.Equal(t, 1, logs.Len())
	require.Contains(t, logs.All()[0].Message, "catalog returned no addresses")
}

func TestLoadProposalConfig_BuildParams(t *testing.T) {
	t.Parallel()

	proposalPath := filepath.Join(t.TempDir(), "proposal.json")
	require.NoError(t, os.WriteFile(proposalPath, testProposalWithoutChangesetsJSON, 0o600))

	params := &cldfenv.BuildParams{Domain: buildParamsDomain, Environment: buildParamsEnv}
	dom := domain.NewDomain("/does/not/exist", buildParamsDomain)

	t.Run("uses the builder and not the file loader", func(t *testing.T) {
		t.Parallel()

		builder := &fakeBuilder{selector: []uint64{buildParamsSelector}}
		deps := &Deps{
			EnvironmentLoader:  failingEnvLoader(t),
			EnvironmentBuilder: builder.build,
			ProposalLoader:     mcms.LoadProposal,
		}

		got, err := LoadProposalConfig(t.Context(), logger.Nop(), dom, deps, nil, ProposalFlags{
			ProposalPath: proposalPath,
			ProposalKind: string(mcmstypes.KindTimelockProposal),
			Environment:  buildParamsEnv,
			BuildParams:  params,
		})

		require.NoError(t, err)
		require.Equal(t, 1, builder.calls)
		require.Equal(t, *params, builder.params)
		require.Equal(t, buildParamsEnv, got.Env.Name)
	})

	t.Run("a proposal chain missing from networks fails naming the selector", func(t *testing.T) {
		t.Parallel()

		builder := &fakeBuilder{}
		deps := &Deps{
			EnvironmentLoader:  failingEnvLoader(t),
			EnvironmentBuilder: builder.build,
			ProposalLoader:     mcms.LoadProposal,
		}

		_, err := LoadProposalConfig(t.Context(), logger.Nop(), dom, deps, nil, ProposalFlags{
			ProposalPath: proposalPath,
			ProposalKind: string(mcmstypes.KindTimelockProposal),
			Environment:  buildParamsEnv,
			BuildParams:  params,
		})

		require.ErrorContains(t, err, "chain selector 3379446385462418246 not found in the supplied build params")
	})

	t.Run("build params cannot be combined with fork", func(t *testing.T) {
		t.Parallel()

		deps := &Deps{ProposalLoader: mcms.LoadProposal, ForkEnvironmentLoader: nopForkEnvLoader}

		_, err := LoadProposalConfig(t.Context(), logger.Nop(), dom, deps, nil, ProposalFlags{
			ProposalPath: proposalPath,
			ProposalKind: string(mcmstypes.KindTimelockProposal),
			Environment:  buildParamsEnv,
			Fork:         true,
			BuildParams:  params,
		})

		require.ErrorContains(t, err, "not supported when running on a fork")
	})
}

// newBuildParamsCommand uses a domain root that does not exist, to prove no
// domain directory is read.
func newBuildParamsCommand(t *testing.T, deps Deps) *cobra.Command {
	t.Helper()

	cmd, err := NewCommand(Config{
		Logger:                  logger.Nop(),
		Domain:                  domain.NewDomain(filepath.Join(t.TempDir(), "does-not-exist"), buildParamsDomain),
		ProposalContextProvider: mockProposalContextProvider,
		Deps:                    deps,
	})
	require.NoError(t, err)

	return cmd
}

func writeTestProposal(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "proposal.json")
	require.NoError(t, os.WriteFile(path, testProposalWithoutChangesetsJSON, 0o600))

	return path
}

func getenvFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func executeCommand(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	return cmd.Execute()
}

func TestBuildParamsCommands(t *testing.T) {
	t.Parallel()

	subcommands := []string{"analyze-proposal-v2", "convert-upf", "analyze-proposal"}

	for _, sub := range subcommands {
		// The fake builder returns no chains, so runs stop at the selector check.
		// Reaching it shows the environment came from the document.
		t.Run(sub+"/document from variable builds the environment", func(t *testing.T) {
			t.Parallel()

			builder := &fakeBuilder{}
			cmd := newBuildParamsCommand(t, Deps{
				EnvironmentLoader:  failingEnvLoader(t),
				EnvironmentBuilder: builder.build,
				Getenv:             getenvFrom(map[string]string{"MCMS_ENV_BUILD_PARAMS": buildParamsDoc}),
			})

			err := executeCommand(cmd, sub, "-p", writeTestProposal(t))

			require.ErrorContains(t, err, "chain selector 3379446385462418246 not found in the supplied build params")
			require.Equal(t, 1, builder.calls)
			require.Equal(t, buildParamsDomain, builder.params.Domain)
			require.Equal(t, buildParamsEnv, builder.params.Environment)
			require.Equal(t, "catalog.example.com:443", builder.params.Catalog.GRPC)
		})

		t.Run(sub+"/document from flag", func(t *testing.T) {
			t.Parallel()

			builder := &fakeBuilder{}
			cmd := newBuildParamsCommand(t, Deps{
				EnvironmentLoader:  failingEnvLoader(t),
				EnvironmentBuilder: builder.build,
				Getenv:             getenvFrom(nil),
			})

			err := executeCommand(cmd, sub, "-p", writeTestProposal(t), "--env-build-params", buildParamsDoc)

			require.ErrorContains(t, err, "chain selector 3379446385462418246 not found")
			require.Equal(t, 1, builder.calls)
		})

		t.Run(sub+"/environment flag conflicts with document", func(t *testing.T) {
			t.Parallel()

			builder := &fakeBuilder{}
			cmd := newBuildParamsCommand(t, Deps{
				EnvironmentLoader:  failingEnvLoader(t),
				EnvironmentBuilder: builder.build,
				Getenv:             getenvFrom(map[string]string{"MCMS_ENV_BUILD_PARAMS": buildParamsDoc}),
			})

			err := executeCommand(cmd, sub, "-p", writeTestProposal(t), "-e", buildParamsEnv)

			require.ErrorContains(t, err, "--environment cannot be combined with build params")
			require.Zero(t, builder.calls)
		})

		t.Run(sub+"/empty flag does not fall back to the variable", func(t *testing.T) {
			t.Parallel()

			builder := &fakeBuilder{}
			cmd := newBuildParamsCommand(t, Deps{
				EnvironmentLoader:  failingEnvLoader(t),
				EnvironmentBuilder: builder.build,
				Getenv:             getenvFrom(map[string]string{"MCMS_ENV_BUILD_PARAMS": buildParamsDoc}),
			})

			err := executeCommand(cmd, sub, "-p", writeTestProposal(t), "--env-build-params", "")

			require.ErrorContains(t, err, "document is empty")
			require.Zero(t, builder.calls)
		})

		t.Run(sub+"/without a document the file loader is used and -e stays required", func(t *testing.T) {
			t.Parallel()

			var loaded []string
			builder := &fakeBuilder{}
			deps := Deps{
				EnvironmentLoader: func(
					_ context.Context, _ domain.Domain, envKey string, _ logger.Logger, _ ...cldfenv.LoadEnvironmentOption,
				) (cldf.Environment, error) {
					loaded = append(loaded, envKey)

					return cldf.Environment{}, errors.New("stop after recording the call")
				},
				EnvironmentBuilder: builder.build,
				Getenv:             getenvFrom(nil),
			}

			// -e is still required without a document.
			err := executeCommand(newBuildParamsCommand(t, deps), sub, "-p", writeTestProposal(t))
			require.ErrorContains(t, err, `required flag(s) "environment" not set`)
			require.Empty(t, loaded)

			// With -e, the disk path runs exactly as before.
			err = executeCommand(newBuildParamsCommand(t, deps), sub, "-p", writeTestProposal(t), "-e", "staging")
			require.ErrorContains(t, err, "stop after recording the call")
			require.Equal(t, []string{"staging"}, loaded)
			require.Zero(t, builder.calls)
		})
	}
}

func TestBuildParamsFlag_OnlyOnSupportedCommands(t *testing.T) {
	t.Parallel()

	cmd := newBuildParamsCommand(t, Deps{})

	supported := map[string]bool{
		"analyze-proposal-v2": true,
		"convert-upf":         true,
		"analyze-proposal":    true,
		"execute-fork":        false,
		"error-decode-evm":    false,
		"hooks":               false,
	}

	for name, want := range supported {
		sub, _, err := cmd.Find([]string{name})
		require.NoError(t, err, name)
		require.Equal(t, name, sub.Name())
		require.Equal(t, want, sub.Flags().Lookup("env-build-params") != nil, name)
	}
}

func TestBuildParamsFlag_EnvironmentUsageMentionsDocument(t *testing.T) {
	t.Parallel()

	cmd := newBuildParamsCommand(t, Deps{})
	sub, _, err := cmd.Find([]string{"analyze-proposal-v2"})
	require.NoError(t, err)

	require.Contains(t, sub.Flags().Lookup("environment").Usage, "env-build-params")
}
