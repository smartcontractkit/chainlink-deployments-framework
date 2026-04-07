package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// stubChangeset implements ChangeSetV2 for testing.
type stubChangeset struct {
	ApplyCalled bool
	StubError   error
}

func (s *stubChangeset) Apply(_ fdeployment.Environment, _ any) (fdeployment.ChangesetOutput, error) {
	s.ApplyCalled = true
	return fdeployment.ChangesetOutput{}, s.StubError
}

func (s *stubChangeset) VerifyPreconditions(_ fdeployment.Environment, _ any) error {
	return nil
}

var _ fdeployment.ChangeSetV2[any] = (*stubChangeset)(nil)

// stubProposalChangeset implements ChangeSetV2 that generates a proposal for testing.
type stubProposalChangeset struct {
	TimelockProposal mcms.TimelockProposal
}

func (s *stubProposalChangeset) Apply(_ fdeployment.Environment, _ any) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{MCMSTimelockProposals: []mcms.TimelockProposal{s.TimelockProposal}}, nil
}

func (s *stubProposalChangeset) VerifyPreconditions(_ fdeployment.Environment, _ any) error {
	return nil
}

var _ fdeployment.ChangeSetV2[any] = (*stubProposalChangeset)(nil)

// registryProviderStub provides a changeset registry for tests.
type registryProviderStub struct {
	*changeset.BaseRegistryProvider
	AddAction func(*changeset.ChangesetsRegistry)
}

func (p *registryProviderStub) Init() error {
	p.AddAction(p.Registry())
	return nil
}

// mockProposalContext implements analyzer.ProposalContext for testing.
type mockProposalContext struct{}

func (m *mockProposalContext) SetRenderer(analyzer.Renderer) {}
func (m *mockProposalContext) GetRenderer() analyzer.Renderer {
	return analyzer.NewMarkdownRenderer()
}

func (m *mockProposalContext) FieldsContext(uint64) *analyzer.FieldContext {
	return &analyzer.FieldContext{}
}

func (m *mockProposalContext) GetSolanaDecoderRegistry() analyzer.SolanaDecoderRegistry {
	return nil
}

func (m *mockProposalContext) GetEVMRegistry() analyzer.EVMABIRegistry {
	return nil
}

// loadProposal is a helper function to load the generated proposal from disk for assertions.
func loadProposal(t *testing.T, proposalsDir string) (*mcms.TimelockProposal, string) {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(proposalsDir, "*.json"))
	require.NoError(t, err)
	require.Len(t, files, 1)

	proposal, err := mcms.LoadProposal(mcmstypes.KindTimelockProposal, files[0])
	require.NoError(t, err)

	return proposal.(*mcms.TimelockProposal), files[0]
}

//nolint:paralleltest
func TestRunCmd_Success(t *testing.T) {
	env := "testnet"
	changesetName := "0001_test_changeset"
	testDomain := domain.NewDomain(t.TempDir(), "test")

	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	yamlContent := `environment: testnet
domain: test
changesets:
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
        value: 100`
	yamlFileName := "test-input.yaml"
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, yamlFileName), []byte(yamlContent), 0o600))

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	changesetStub := &stubChangeset{}
	loadChangesets := func(envName string) (*changeset.ChangesetsRegistry, error) {
		rp := &registryProviderStub{
			BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
			AddAction: func(reg *changeset.ChangesetsRegistry) {
				reg.Add(changesetName, changeset.Configure(changesetStub).With(1))
			},
		}
		if initErr := rp.Init(); initErr != nil {
			return nil, initErr
		}

		return rp.Registry(), nil
	}

	decodeCalled := false
	decodeProvider := func(fdeployment.Environment) (analyzer.ProposalContext, error) {
		decodeCalled = true
		return &mockProposalContext{}, nil
	}

	cfg := &Config{
		Logger:                    logger.Test(t),
		Domain:                    testDomain,
		LoadChangesets:            loadChangesets,
		DecodeProposalCtxProvider: decodeProvider,
		ConfigResolverManager:     fresolvers.NewConfigResolverManager(),
		Deps: Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, nil
			},
		},
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"run",
		"--environment", env,
		"--changeset", changesetName,
		"--input-file", yamlFileName,
		"--dry-run",
	})

	err = cmd.Execute()
	require.NoError(t, err)
	require.True(t, changesetStub.ApplyCalled)
	require.True(t, decodeCalled)
}

//nolint:paralleltest
func TestRunCmd_ApplyError(t *testing.T) {
	env := "testnet"
	changesetName := "0001_test_changeset"
	testDomain := domain.NewDomain(t.TempDir(), "test")

	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	yamlContent := `environment: testnet
domain: test
changesets:
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia`
	yamlFileName := "test-input.yaml"
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, yamlFileName), []byte(yamlContent), 0o600))

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	applyErr := errors.New("changeset apply failed")
	changesetStub := &stubChangeset{StubError: applyErr}

	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		rp := &registryProviderStub{
			BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
			AddAction: func(reg *changeset.ChangesetsRegistry) {
				reg.Add(changesetName, changeset.Configure(changesetStub).With(1))
			},
		}
		_ = rp.Init()

		return rp.Registry(), nil
	}

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                testDomain,
		LoadChangesets:        loadChangesets,
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
		Deps: Deps{
			EnvironmentLoader: func(context.Context, domain.Domain, string, ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, nil
			},
		},
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"run",
		"--environment", env,
		"--changeset", changesetName,
		"--input-file", yamlFileName,
		"--dry-run",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.EqualError(t, err, "changeset apply failed")
}

//nolint:paralleltest
func TestRunCmd_UnknownFlag(t *testing.T) {
	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return changeset.NewChangesetsRegistry(), nil },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"run",
		"--environment", "testnet",
		"--changeset", "0001_cs1",
		"--input-file", "x.yaml",
		"--fake-flag",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, "unknown flag: --fake-flag", err.Error())
}

//nolint:paralleltest
func TestRunCmd_LoadChangesetsError(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), "testnet", "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "x.yaml"), []byte(`environment: testnet
domain: test
changesets:
  - 0001_cs1:
      payload: {}`), 0o600))

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	loadErr := errors.New("load failed")
	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                testDomain,
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return nil, loadErr },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
		Deps: Deps{
			EnvironmentLoader: func(context.Context, domain.Domain, string, ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, nil
			},
		},
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"run",
		"--environment", "testnet",
		"--changeset", "0001_cs1",
		"--input-file", "x.yaml",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, "load failed", err.Error())
}

//nolint:paralleltest
func TestRunCmd_ResolverNotRegistered(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), "testnet", "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "x.yaml"), []byte(`environment: testnet
domain: test
changesets:
  - 0001_cs1:
      payload: {}`), 0o600))

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	unregisteredResolver := func(map[string]any) (any, error) { return map[string]any{}, nil }
	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		reg := changeset.NewChangesetsRegistry()
		cs := &stubChangeset{}
		reg.Add("0001_cs1", changeset.Configure(cs).WithConfigResolver(unregisteredResolver))

		return reg, nil
	}

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                testDomain,
		LoadChangesets:        loadChangesets,
		ConfigResolverManager: fresolvers.NewConfigResolverManager(), // empty, resolver not registered
		Deps: Deps{
			EnvironmentLoader: func(context.Context, domain.Domain, string, ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, nil
			},
		},
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"run",
		"--environment", "testnet",
		"--changeset", "0001_cs1",
		"--input-file", "x.yaml",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, "resolver for 0001_cs1 is not registered", err.Error())
}

//nolint:paralleltest
func TestRunCmd_ByIndex(t *testing.T) {
	env := "testnet"
	testDomain := domain.NewDomain(t.TempDir(), "test")

	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	yamlContent := `environment: testnet
domain: test
changesets:
  - 0001_cs_first:
      payload: {a: 1}
  - 0002_cs_second:
      payload: {b: 2}`
	yamlFileName := "input.yaml"
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, yamlFileName), []byte(yamlContent), 0o600))

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	changesetStub := &stubChangeset{}
	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		rp := &registryProviderStub{
			BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
			AddAction: func(reg *changeset.ChangesetsRegistry) {
				reg.Add("0001_cs_first", changeset.Configure(changesetStub).With(1))
				reg.Add("0002_cs_second", changeset.Configure(changesetStub).With(2))
			},
		}
		_ = rp.Init()

		return rp.Registry(), nil
	}

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                testDomain,
		LoadChangesets:        loadChangesets,
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
		Deps: Deps{
			EnvironmentLoader: func(context.Context, domain.Domain, string, ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, nil
			},
		},
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"run",
		"--environment", env,
		"--input-file", yamlFileName,
		"--changeset-index", "1",
		"--dry-run",
	})

	err = cmd.Execute()
	require.NoError(t, err)
	require.True(t, changesetStub.ApplyCalled)
}

//nolint:paralleltest
func TestRunCmd_InvalidInputFile(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	workspaceRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceRoot, "domains"), 0o755))

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		reg := changeset.NewChangesetsRegistry()
		reg.Add("0001_cs1", changeset.Configure(&stubChangeset{}).With(1))

		return reg, nil
	}

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                testDomain,
		LoadChangesets:        loadChangesets,
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
		Deps: Deps{
			EnvironmentLoader: func(context.Context, domain.Domain, string, ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, nil
			},
		},
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"run",
		"--environment", "testnet",
		"--changeset", "0001_cs1",
		"--input-file", "nonexistent.yaml",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.GreaterOrEqual(t, len(err.Error()), 28)
	require.Equal(t, "failed to parse input file: ", err.Error()[:28])
}

//nolint:paralleltest
func TestRunCmd_EnvironmentLoaderError(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), "testnet", "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "x.yaml"), []byte(`environment: testnet
domain: test
changesets:
  - 0001_cs1:
      payload: {}`), 0o600))

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	loadErr := errors.New("environment load failed")
	cfg := &Config{
		Logger: logger.Test(t),
		Domain: testDomain,
		LoadChangesets: func(string) (*changeset.ChangesetsRegistry, error) {
			reg := changeset.NewChangesetsRegistry()
			reg.Add("0001_cs1", changeset.Configure(&stubChangeset{}).With(1))

			return reg, nil
		},
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
		Deps: Deps{
			EnvironmentLoader: func(context.Context, domain.Domain, string, ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, loadErr
			},
		},
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"run",
		"--environment", "testnet",
		"--changeset", "0001_cs1",
		"--input-file", "x.yaml",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.EqualError(t, err, "environment load failed")
}

func TestRunCmd_SubcommandExists(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return changeset.NewChangesetsRegistry(), nil },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	subCmds := cmd.Commands()
	require.NotEmpty(t, subCmds)

	var found bool
	for _, c := range subCmds {
		if c.Name() == "run" {
			found = true
			require.Equal(t, "run", c.Use)
			require.NotNil(t, c.Flags().Lookup("environment"))
			require.NotNil(t, c.Flags().Lookup("changeset"))
			require.NotNil(t, c.Flags().Lookup("input-file"))

			break
		}
	}
	require.True(t, found)
}

func TestRunCmd_RequiresInputFile(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return changeset.NewChangesetsRegistry(), nil },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	// Execute without required --input-file
	cmd.SetArgs([]string{"run", "--environment", "testnet", "--changeset", "0001_cs1"})
	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, `required flag(s) "input-file" not set`, err.Error())
}

//nolint:paralleltest
func TestRunCmd_ReturnsProposal(t *testing.T) {
	env := "testnet"
	changesetName := "0001_test_changeset"
	workspaceRoot := t.TempDir()
	domainsRoot := filepath.Join(workspaceRoot, "domains")
	testDomain := domain.NewDomain(domainsRoot, "test")
	envRoot := filepath.Join(domainsRoot, testDomain.String(), env)
	inputsDir := filepath.Join(envRoot, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))
	proposalsDir := filepath.Join(envRoot, "proposals")

	yamlContent := `environment: testnet
domain: test
changesets:
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
        value: 100`
	yamlFileName := "test-input.yaml"
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, yamlFileName), []byte(yamlContent), 0o600))

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	changesetStub := &stubProposalChangeset{TimelockProposal: *testProposal}
	loadChangesets := func(envName string) (*changeset.ChangesetsRegistry, error) {
		rp := &registryProviderStub{
			BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
			AddAction: func(reg *changeset.ChangesetsRegistry) {
				reg.Add(changesetName, changeset.Configure(changesetStub).With(1))
			},
		}
		if initErr := rp.Init(); initErr != nil {
			return nil, initErr
		}

		return rp.Registry(), nil
	}
	decodeProvider := func(fdeployment.Environment) (analyzer.ProposalContext, error) {
		return &mockProposalContext{}, nil
	}

	cfg := &Config{
		Logger:                    logger.Test(t),
		Domain:                    testDomain,
		LoadChangesets:            loadChangesets,
		DecodeProposalCtxProvider: decodeProvider,
		ConfigResolverManager:     fresolvers.NewConfigResolverManager(),
		Deps: Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, nil
			},
		},
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"run",
		"--environment", env,
		"--changeset", changesetName,
		"--input-file", yamlFileName,
		"--dry-run",
	})

	err = cmd.Execute()
	require.NoError(t, err)

	proposal, _ := loadProposal(t, proposalsDir)
	require.Equal(t, "v1", proposal.Version)
	require.NotNil(t, proposal.Metadata)
	require.Equal(t, map[string]any{
		"changesets": []any{
			map[string]any{
				"name": changesetName,
				"input": map[string]any{
					"payload": map[string]any{
						"chain": "optimism_sepolia",
						"value": json.Number("100"),
					},
				},
				"operationIDs": []any{"0x0206ff10261710c45650d4912ffc6d6808fd1248fb510e47d0e9482058d9c048"},
			},
		},
	}, proposal.Metadata)
}

// ----- shared test data -----

var testProposal = lo.Must(mcms.NewTimelockProposalBuilder().
	SetVersion("v1").
	SetValidUntil(2082758399).
	SetDescription("test timelock proposal").
	SetOverridePreviousRoot(true).
	SetAction(mcmstypes.TimelockActionSchedule).
	AddTimelockAddress(mcmstypes.ChainSelector(chainsel.GETH_TESTNET.Selector), "0xTimelockAddress").
	AddChainMetadata(mcmstypes.ChainSelector(chainsel.GETH_TESTNET.Selector), mcmstypes.ChainMetadata{MCMAddress: "0xMCMAddress"}).
	AddOperation(mcmstypes.BatchOperation{
		ChainSelector: mcmstypes.ChainSelector(chainsel.GETH_TESTNET.Selector),
		Transactions: []mcmstypes.Transaction{{
			To:               "0xToAddress",
			Data:             []byte("0x"),
			AdditionalFields: json.RawMessage(`{"value": 0}`),
		}},
	}).
	Build())
