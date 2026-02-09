package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

//nolint:paralleltest
func TestNewDurablePipelineRunCmd(t *testing.T) {
	env := "testnet"
	changesetName := "0001_test_changeset"
	testDomain := domain.NewDomain(t.TempDir(), "test")

	// Create workspace structure and test input file
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	// Create test YAML file
	yamlContent := `environment: testnet
domain: test
changesets:
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
        value: 100`

	yamlFileName := "test-input.yaml"
	yamlFilePath := filepath.Join(inputsDir, yamlFileName)
	require.NoError(t, os.WriteFile(yamlFilePath, []byte(yamlContent), 0644)) //nolint:gosec

	// Set up the test to run from within the workspace
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWd))
	})

	tempLoadEnv := environment.Load
	// mock the loadEnv function to avoid loading a real environment
	loadEnv = func(ctx context.Context, domain domain.Domain, envName string, options ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
		return fdeployment.Environment{}, nil
	}
	t.Cleanup(func() {
		loadEnv = tempLoadEnv
	})

	tests := []struct {
		name                     string
		args                     []string
		applyErr                 error
		expectedErr              error
		shouldCallApplyChanges   bool
		shouldCallDecodeProposal bool
	}{
		{
			name: "successful execution",
			args: []string{
				"execute",
				"--environment", env,
				"--changeset", changesetName,
				"--input-file", yamlFileName,
				"--dry-run",
			},
			shouldCallApplyChanges:   true,
			shouldCallDecodeProposal: true,
		},
		{
			name: "error in applyChangeSet",
			args: []string{
				"execute",
				"--environment", env,
				"--changeset", changesetName,
				"--input-file", yamlFileName,
				"--dry-run",
			},
			applyErr:                 errors.New("changeset error"),
			shouldCallApplyChanges:   true,
			shouldCallDecodeProposal: false,
			expectedErr:              errors.New("changeset error"),
		},
		{
			name: "error unknown flag",
			args: []string{
				"execute",
				"--environment", env,
				"--changeset", changesetName,
				"--input-file", yamlFileName,
				"--fake-flag",
				"--dry-run",
			},
			shouldCallApplyChanges:   false,
			shouldCallDecodeProposal: false,
			expectedErr:              errors.New("unknown flag: --fake-flag"),
		},
		{
			name: "execution with proposal decoding",
			args: []string{
				"execute",
				"--environment", env,
				"--changeset", changesetName,
				"--input-file", yamlFileName,
				"--dry-run",
			},
			shouldCallApplyChanges:   true,
			shouldCallDecodeProposal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decodeCalled := false
			decodeProposalCtxProvider := func(env fdeployment.Environment) (analyzer.ProposalContext, error) {
				decodeCalled = true
				return &mockProposalContext{t: t}, nil
			}

			changesetStub := stubChangeset{
				ApplyCalled: false,
				StubError:   tt.applyErr,
			}

			sharedCommands := NewCommands(logger.Test(t))
			rootCmd := sharedCommands.NewDurablePipelineCmds(
				testDomain,
				func(envName string) (*changeset.ChangesetsRegistry, error) {
					rp := changesetsRegistryProviderStub{
						BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
						AddChangesetAction: func(registry *changeset.ChangesetsRegistry) {
							registry.Add(changesetName, changeset.Configure(&changesetStub).With(1))
						},
					}

					if err := rp.Init(); err != nil {
						return nil, fmt.Errorf("failed to init changesets %w", err)
					}

					return rp.Registry(), nil
				},
				decodeProposalCtxProvider,
				fresolvers.NewConfigResolverManager(),
			)

			require.NotNil(t, rootCmd)
			tt.args[0] = "run"
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if tt.expectedErr != nil {
				require.EqualError(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.shouldCallApplyChanges, changesetStub.ApplyCalled)
			require.Equal(t, tt.shouldCallDecodeProposal, decodeCalled)
		})
	}
}

//nolint:paralleltest
func TestNewDurablePipelineInputGenerateCmd(t *testing.T) {
	env := "testnet"
	testDomain := domain.NewDomain(t.TempDir(), "test")

	tempLoadEnv := environment.Load
	// mock the loadEnv function to avoid loading a real environment
	loadEnv = func(ctx context.Context, domain domain.Domain, envName string, options ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
		return fdeployment.Environment{}, nil
	}
	t.Cleanup(func() {
		loadEnv = tempLoadEnv
	})

	// Create workspace structure for input file
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	// Mock workspace root discovery
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceRoot, "domains"), 0755))

	// Set up the test to run from within the workspace
	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWd))
	})

	formatTests := []struct {
		name              string
		formatDescription string
		inputsFileName    string
		mockInputContent  string
	}{
		{
			name:              "object format",
			formatDescription: "legacy object format",
			inputsFileName:    "test-inputs-object.yaml",
			mockInputContent: `environment: testnet
domain: test
changesets:
  0001_test_changeset:
    payload:
      chain: optimism_sepolia
      value: 100`,
		},
		{
			name:              "array format",
			formatDescription: "new array format",
			inputsFileName:    "test-inputs-array.yaml",
			mockInputContent: `environment: testnet
domain: test
changesets:
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
        value: 100`,
		},
	}

	for _, formatTest := range formatTests {
		t.Run(formatTest.name, func(t *testing.T) {
			// Create input file for this format
			inputsFilePath := filepath.Join(inputsDir, formatTest.inputsFileName)
			require.NoError(t, os.WriteFile(inputsFilePath, []byte(formatTest.mockInputContent), 0644)) //nolint:gosec

			tests := []struct {
				name        string
				args        func(t *testing.T) []string
				resolverErr error
				expectedErr string
				checkOutput func(t *testing.T, output string)
			}{
				{
					name: "successful generation with YAML output",
					args: func(t *testing.T) []string {
						t.Helper()
						return []string{
							"input-generate",
							"--environment", env,
							"--inputs", formatTest.inputsFileName,
							"--output", filepath.Join(t.TempDir(), "output.yaml"),
						}
					},
					checkOutput: func(t *testing.T, output string) {
						t.Helper()
						require.Contains(t, output, "environment: testnet")
						require.Contains(t, output, "domain: test")
						require.Contains(t, output, "changesets:")
						require.Contains(t, output, "resolvedChain: optimism_sepolia_resolved")
					},
				},
				{
					name: "successful generation with JSON output",
					args: func(t *testing.T) []string {
						t.Helper()
						return []string{
							"input-generate",
							"--environment", env,
							"--inputs", formatTest.inputsFileName,
							"--json",
							"--output", filepath.Join(t.TempDir(), "output.json"),
						}
					},
					checkOutput: func(t *testing.T, output string) {
						t.Helper()
						require.Contains(t, output, `"environment": "testnet"`)
						require.Contains(t, output, `"domain": "test"`)
						require.Contains(t, output, `"changesets"`)
						require.Contains(t, output, `"resolvedChain": "optimism_sepolia_resolved"`)
					},
				},
				{
					name: "missing environment flag",
					args: func(t *testing.T) []string {
						t.Helper()
						return []string{
							"input-generate",
							"--inputs", formatTest.inputsFileName,
						}
					},
					expectedErr: "required flag(s) \"environment\" not set",
				},
				{
					name: "missing inputs flag",
					args: func(t *testing.T) []string {
						t.Helper()
						return []string{
							"input-generate",
							"--environment", env,
						}
					},
					expectedErr: "required flag(s) \"inputs\" not set",
				},
				{
					name: "resolver error",
					args: func(t *testing.T) []string {
						t.Helper()
						return []string{
							"input-generate",
							"--environment", env,
							"--inputs", formatTest.inputsFileName,
						}
					},
					resolverErr: errors.New("resolver failed"),
					expectedErr: "failed to resolve config for changeset \"0001_test_changeset\": resolver failed",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// Use named resolver functions
					var testResolver fresolvers.ConfigResolver
					if tt.resolverErr != nil {
						testResolver = InputGenerateErrorResolver
					} else {
						testResolver = InputGenerateResolver
					}

					// Create resolver manager
					resolverManager := fresolvers.NewConfigResolverManager()
					resolverManager.Register(testResolver, fresolvers.ResolverInfo{
						Description: "Test Resolver",
						ExampleYAML: "chain: optimism_sepolia\nvalue: 100",
					})

					sharedCommands := NewCommands(logger.Test(t))
					rootCmd := sharedCommands.NewDurablePipelineCmds(
						testDomain,
						func(envName string) (*changeset.ChangesetsRegistry, error) {
							rp := changesetsRegistryProviderStub{
								BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
								AddChangesetAction: func(registry *changeset.ChangesetsRegistry) {
									cs := &stubChangeset{resolver: testResolver}
									registry.Add("0001_test_changeset", changeset.Configure(cs).WithConfigResolver(testResolver))
								},
							}

							if err := rp.Init(); err != nil {
								return nil, fmt.Errorf("failed to init changesets %w", err)
							}

							return rp.Registry(), nil
						},
						nil, // No proposal context needed for input generation
						resolverManager,
					)

					require.NotNil(t, rootCmd)
					args := tt.args(t)
					rootCmd.SetArgs(args)
					err := rootCmd.Execute()

					if tt.expectedErr != "" {
						require.Error(t, err)
						require.Contains(t, err.Error(), tt.expectedErr)
					} else {
						require.NoError(t, err)

						if tt.checkOutput != nil {
							// Find --output flag and read the file
							for i, arg := range args {
								if arg == "--output" && i+1 < len(args) {
									outputBytes, err := os.ReadFile(args[i+1])
									require.NoError(t, err)
									tt.checkOutput(t, string(outputBytes))

									break
								}
							}
						}
					}
				})
			}
		})
	}
}

// Define named resolver functions
func MockTestResolver(input map[string]any) (any, error)   { return nil, errors.New("resolver failed") }
func MockFirstResolver(input map[string]any) (any, error)  { return nil, errors.New("resolver failed") }
func MockSecondResolver(input map[string]any) (any, error) { return nil, errors.New("resolver failed") }
func MockUnregisteredResolver(input map[string]any) (any, error) {
	return nil, errors.New("resolver failed")
}

// Add resolver functions that can be used for TestBuildInputGenerateCmd
func InputGenerateResolver(input map[string]any) (any, error) {
	// This will be the behavior we want for the successful test cases
	// Handle different chains based on input
	var chain string
	if inputChain, ok := input["chain"]; ok {
		if chainStr, ok := inputChain.(string); ok {
			chain = chainStr + "_resolved"
		}
	}

	return map[string]any{
		"resolvedChain": chain,
		"resolvedValue": 200,
	}, nil
}

// Add a resolver that returns an error
func InputGenerateErrorResolver(input map[string]any) (any, error) {
	return nil, errors.New("resolver failed")
}

//nolint:paralleltest
func TestBuildListCmd(t *testing.T) {
	env := "testnet"
	testDomain := domain.NewDomain(t.TempDir(), "test")

	tempLoadEnv := environment.Load
	// mock the loadEnv function to avoid loading a real environment
	loadEnv = func(ctx context.Context, domain domain.Domain, envName string, options ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
		return fdeployment.Environment{}, nil
	}
	t.Cleanup(func() {
		loadEnv = tempLoadEnv
	})

	tests := []struct {
		name                   string
		args                   []string
		setupMocks             func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error)
		expectedErr            string
		expectedOutputContains []string
	}{
		{
			name: "successful listing with mixed changeset types",
			args: []string{
				"list",
				"--environment", env,
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				// Create resolver manager first
				resolverManager := fresolvers.NewConfigResolverManager()
				resolverManager.Register(MockTestResolver, fresolvers.ResolverInfo{
					Description: "Test Resolver",
					ExampleYAML: "test: value",
				})

				// Create registry with mixed changeset types
				rp := changesetsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddChangesetAction: func(registry *changeset.ChangesetsRegistry) {
						// Static changeset (no resolver)
						staticChangeset := &stubChangeset{}
						registry.Add("0001_static_changeset", changeset.Configure(staticChangeset).With(1))

						// Dynamic changeset (with resolver)
						dynamicChangeset := &stubChangeset{resolver: MockTestResolver}
						registry.Add("0002_dynamic_changeset", changeset.Configure(dynamicChangeset).WithConfigResolver(MockTestResolver))

						// Error changeset (resolver not registered with manager)
						errorChangeset := &stubChangeset{resolver: MockUnregisteredResolver}
						registry.Add("0003_error_changeset", changeset.Configure(errorChangeset).WithConfigResolver(MockUnregisteredResolver))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedOutputContains: []string{
				"Durable Pipeline Info for test",
				"Legend: DYNAMIC = config resolver | STATIC = YAML input | ERROR = misconfigured",
				"STATIC", "0001_static_changeset", "YAML input file",
				"DYNAMIC", "0002_dynamic_changeset", "TestResolver",
				"ERROR", "0003_error_changeset", "Resolver not registered",
				"Available Config Resolvers:",
				"TestResolver",
			},
		},
		{
			name: "empty registry and no fresolvers",
			args: []string{
				"list",
				"--environment", env,
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				rp := changesetsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddChangesetAction: func(registry *changeset.ChangesetsRegistry) {
						// Add no changesets
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				resolverManager := fresolvers.NewConfigResolverManager()

				return rp.Registry(), resolverManager, nil
			},
			expectedOutputContains: []string{
				"Durable Pipeline Info for test",
				"Available Config Resolvers:",
				"Registered Changesets:",
			},
		},
		{
			name: "multiple fresolvers listed",
			args: []string{
				"list",
				"--environment", env,
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				// Create resolver manager with multiple fresolvers
				resolverManager := fresolvers.NewConfigResolverManager()

				resolverManager.Register(MockFirstResolver, fresolvers.ResolverInfo{
					Description: "First Resolver",
					ExampleYAML: "test: value1",
				})

				resolverManager.Register(MockSecondResolver, fresolvers.ResolverInfo{
					Description: "Second Resolver",
					ExampleYAML: "test: value2",
				})

				rp := changesetsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddChangesetAction: func(registry *changeset.ChangesetsRegistry) {
						cs := &stubChangeset{resolver: MockFirstResolver}
						registry.Add("0001_test_changeset", changeset.Configure(cs).WithConfigResolver(MockFirstResolver))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedOutputContains: []string{
				"Durable Pipeline Info for test",
				"DYNAMIC", "0001_test_changeset", "FirstResolver",
				"Available Config Resolvers:",
				"FirstResolver",
				"SecondResolver",
			},
		},
		{
			name: "missing environment flag",
			args: []string{
				"list",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				return changeset.NewChangesetsRegistry(), fresolvers.NewConfigResolverManager(), nil
			},
			expectedErr: "required flag(s) \"environment\" not set",
		},
		{
			name: "registry load error",
			args: []string{
				"list",
				"--environment", env,
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				return nil, fresolvers.NewConfigResolverManager(), errors.New("registry load failed")
			},
			expectedErr: "failed to load changesets registry: registry load failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, resolverManager, mockErr := tt.setupMocks()

			sharedCommands := NewCommands(logger.Test(t))
			rootCmd := sharedCommands.NewDurablePipelineCmds(
				testDomain,
				func(envName string) (*changeset.ChangesetsRegistry, error) {
					require.Equal(t, env, envName)
					if mockErr != nil {
						return nil, mockErr
					}

					return registry, nil
				},
				nil, // No proposal context needed for listing
				resolverManager,
			)

			require.NotNil(t, rootCmd)
			rootCmd.SetArgs(tt.args)

			// Use command's output writer
			var output strings.Builder
			rootCmd.SetOut(&output)

			err := rootCmd.Execute()

			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)

				// Check output contains expected strings
				outputStr := output.String()
				for _, expected := range tt.expectedOutputContains {
					require.Contains(t, outputStr, expected, "Output should contain: %q\nActual output:\n%s", expected, outputStr)
				}
			}
		})
	}
}

type mockProposalContext struct {
	t *testing.T
}

func (m *mockProposalContext) SetRenderer(r analyzer.Renderer) {
	// No-op for mock
}

func (m *mockProposalContext) GetRenderer() analyzer.Renderer {
	return analyzer.NewMarkdownRenderer()
}

func (m *mockProposalContext) FieldsContext(chainSelector uint64) *analyzer.FieldContext {
	return &analyzer.FieldContext{}
}
func (m *mockProposalContext) GetSolanaDecoderRegistry() analyzer.SolanaDecoderRegistry {
	// Return a mock SolanaDecoderRegistry with a dummy decoder for testing
	registry, err := analyzer.NewEnvironmentSolanaRegistry(
		fdeployment.Environment{}, // env unused by current methods
		map[string]analyzer.DecodeInstructionFn{
			"DummyProgram 1.0.0": nil, // Use nil as a stand-in decoder for testing
		},
	)
	require.NoError(m.t, err, "failed to create mock EVM registry")

	return registry
}
func (m *mockProposalContext) GetEVMRegistry() analyzer.EVMABIRegistry {
	// Return a mock EVMRegistry with a dummy ABI for testing
	abiJSON := `[
		{
			"constant": false,
			"inputs": [],
			"name": "dummyFunction",
			"outputs": [],
			"type": "function"
		}
	]`
	_, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil
	}

	registry, err := analyzer.NewEnvironmentEVMRegistry(
		fdeployment.Environment{}, // env unused by current methods
		map[string]string{
			"DummyContract 1.0.0": abiJSON,
		},
	)
	require.NoError(m.t, err, "failed to create mock EVM registry")

	return registry
}

type changesetsRegistryProviderStub struct {
	*changeset.BaseRegistryProvider
	AddChangesetAction func(registry *changeset.ChangesetsRegistry)
}

func (p *changesetsRegistryProviderStub) Init() error {
	registry := p.Registry()

	p.AddChangesetAction(registry)

	return nil
}

var _ fdeployment.ChangeSetV2[any] = &stubChangeset{}

type stubChangeset struct {
	ApplyCalled bool
	StubError   error
	resolver    fresolvers.ConfigResolver
}

func (s *stubChangeset) Apply(_ fdeployment.Environment, _ any) (fdeployment.ChangesetOutput, error) {
	s.ApplyCalled = true
	return fdeployment.ChangesetOutput{}, s.StubError
}

func (s *stubChangeset) VerifyPreconditions(_ fdeployment.Environment, _ any) error {
	return nil
}

//nolint:paralleltest
func TestNewDurablePipelineInputGenerateCmd_WithDuplicates(t *testing.T) {
	env := "testnet"
	testDomain := domain.NewDomain(t.TempDir(), "test")
	inputsFileName := "test-inputs-array-duplicates.yaml"

	tempLoadEnv := environment.Load
	// mock the loadEnv function to avoid loading a real environment
	loadEnv = func(ctx context.Context, domain domain.Domain, envName string, options ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
		return fdeployment.Environment{}, nil
	}
	t.Cleanup(func() {
		loadEnv = tempLoadEnv
	})

	// Create workspace structure for input file
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	// Create mock input file with ARRAY format including duplicate changeset names
	inputsFilePath := filepath.Join(inputsDir, inputsFileName)
	mockInputContent := `environment: testnet
domain: test
changesets:
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
  - 0001_test_changeset:
      payload:
        chain: base_sepolia
  - 0002_test_changeset:
      payload:
        chain: arbitrum_sepolia
`
	require.NoError(t, os.WriteFile(inputsFilePath, []byte(mockInputContent), 0644)) //nolint:gosec

	// Mock workspace root discovery
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceRoot, "domains"), 0755))

	// Set up the test to run from within the workspace
	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWd))
	})

	// Use named resolver functions
	testResolver := InputGenerateResolver

	// Create resolver manager
	resolverManager := fresolvers.NewConfigResolverManager()
	resolverManager.Register(testResolver, fresolvers.ResolverInfo{
		Description: "Test Resolver",
		ExampleYAML: "chain: optimism_sepolia\nvalue: 100",
	})

	sharedCommands := NewCommands(logger.Test(t))
	rootcmd := sharedCommands.NewDurablePipelineCmds(
		testDomain,
		func(envName string) (*changeset.ChangesetsRegistry, error) {
			rp := changesetsRegistryProviderStub{
				BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
				AddChangesetAction: func(registry *changeset.ChangesetsRegistry) {
					cs := &stubChangeset{resolver: testResolver}
					registry.Add("0001_test_changeset", changeset.Configure(cs).WithConfigResolver(testResolver))
					registry.Add("0002_test_changeset", changeset.Configure(cs).WithConfigResolver(testResolver))
				},
			}

			if err := rp.Init(); err != nil {
				return nil, fmt.Errorf("failed to init changesets %w", err)
			}

			return rp.Registry(), nil
		},
		nil,
		resolverManager,
	)

	outputFile := filepath.Join(t.TempDir(), "output.yaml")
	args := []string{
		"input-generate",
		"--environment", env,
		"--inputs", inputsFileName,
		"--output", outputFile,
	}

	rootcmd.SetArgs(args)
	err := rootcmd.Execute()
	require.NoError(t, err)

	// Read and verify output maintains array format and preserves duplicates
	outputBytes, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	output := string(outputBytes)

	require.Contains(t, output, "environment: testnet")
	require.Contains(t, output, "domain: test")
	require.Contains(t, output, "changesets:")

	// Check that we have two instances of 0001_test_changeset with different resolved values
	require.Contains(t, output, "- 0001_test_changeset:")
	require.Contains(t, output, "- 0002_test_changeset:")
	require.Contains(t, output, "resolvedChain: optimism_sepolia_resolved")
	require.Contains(t, output, "resolvedChain: base_sepolia_resolved")
	require.Contains(t, output, "resolvedChain: arbitrum_sepolia_resolved")

	// Verify we have exactly 2 instances of 0001_test_changeset
	count := strings.Count(output, "0001_test_changeset:")
	require.Equal(t, 2, count, "Should have exactly 2 instances of 0001_test_changeset")
}

//nolint:paralleltest // due to folder manipulation
func TestSetDurablePipelineInputFromYAML_WithPathResolution(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	env := "testnet"
	changesetName := "test_changeset"

	// Create workspace structure
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	// Create a test YAML file in the inputs directory
	yamlContent := `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        value: 123
        message: "hello world"
        bigInt: 2000000000000000000000`

	yamlFileName := "test-pipeline.yaml"
	yamlFilePath := filepath.Join(inputsDir, yamlFileName)
	require.NoError(t, os.WriteFile(yamlFilePath, []byte(yamlContent), 0644)) //nolint:gosec

	// Mock workspace root discovery
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceRoot, "domains"), 0755))

	// Set up the test to run from within the workspace
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWd))
	})

	tests := []struct {
		name         string
		yamlFilePath string
		expectError  bool
		description  string
	}{
		{
			name:         "filename only - should resolve to inputs directory",
			yamlFilePath: yamlFileName,
			expectError:  false,
			description:  "Should resolve filename and successfully parse YAML",
		},
		{
			name:         "full path - should return error",
			yamlFilePath: yamlFilePath,
			expectError:  true,
			description:  "Should return error for full paths",
		},
		{
			name:         "non-existent filename",
			yamlFilePath: "non-existent.yaml",
			expectError:  true,
			description:  "Should fail when file doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:paralleltest // cannot be parallel due to os.Chdir() usage
			// Clear any previous DURABLE_PIPELINE_INPUT
			os.Unsetenv("DURABLE_PIPELINE_INPUT")

			err := setDurablePipelineInputFromYAML(tt.yamlFilePath, changesetName, testDomain, env)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)

				// Verify that DURABLE_PIPELINE_INPUT was set
				durablePipelineInput := os.Getenv("DURABLE_PIPELINE_INPUT")
				require.NotEmpty(t, durablePipelineInput, "DURABLE_PIPELINE_INPUT should be set")

				// Verify the JSON structure
				require.Contains(t, durablePipelineInput, `"value":123`, "Should contain the expected payload")
				require.Contains(t, durablePipelineInput, `"message":"hello world"`, "Should contain the expected payload")
				require.Contains(t, durablePipelineInput, `"bigInt":2000000000000000000000`, "Should contain the expected payload")
			}
		})
	}
}

func TestFindChangesetInData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		changesets    any
		changesetName string
		expectError   bool
		expectedData  any
		errorContains string
		description   string
	}{
		{
			name: "array format - changeset found",
			changesets: []any{
				map[string]any{
					"test_changeset": map[string]any{
						"payload": map[string]any{"value": 123},
					},
				},
				map[string]any{
					"other_changeset": map[string]any{
						"payload": map[string]any{"value": 456},
					},
				},
			},
			changesetName: "test_changeset",
			expectError:   false,
			expectedData: map[string]any{
				"payload": map[string]any{"value": 123},
			},
			description: "Should find changeset in array format",
		},
		{
			name: "array format - changeset not found",
			changesets: []any{
				map[string]any{
					"other_changeset": map[string]any{
						"payload": map[string]any{"value": 123},
					},
				},
			},
			changesetName: "test_changeset",
			expectError:   true,
			description:   "Should return error when changeset not found in array format",
		},
		{
			name:          "array format - empty",
			changesets:    []any{},
			changesetName: "test_changeset",
			expectError:   true,
			description:   "Should return error for empty array",
		},
		{
			name: "object format - should be rejected",
			changesets: map[string]any{
				"test_changeset": map[string]any{
					"payload": map[string]any{"value": 123},
				},
			},
			changesetName: "test_changeset",
			expectError:   true,
			errorContains: "expected array format",
			description:   "Should return error when object format is provided (no longer supported)",
		},
		{
			name:          "invalid format - string",
			changesets:    "invalid",
			changesetName: "test_changeset",
			expectError:   true,
			description:   "Should return error for invalid format",
		},
		{
			name:          "invalid format - nil",
			changesets:    nil,
			changesetName: "test_changeset",
			expectError:   true,
			description:   "Should return error for nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := findChangesetInData(tt.changesets, tt.changesetName, "test-file.yaml")

			if tt.expectError {
				require.Error(t, err, tt.description)
				if tt.errorContains != "" {
					require.ErrorContains(t, err, tt.errorContains, tt.description)
				}
			} else {
				require.NoError(t, err, tt.description)
				require.Equal(t, tt.expectedData, result, tt.description)
			}
		})
	}
}

//nolint:paralleltest
func TestSetDurablePipelineInputFromYAML_ArrayFormat(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	env := "testnet"

	// Create workspace structure
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	// Create a test YAML file with array format
	yamlContent := `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        value: 123
        message: "hello world"
      chainOverrides: [1, 2, 3]
  - other_changeset:
      payload:
        value: 456
        message: "goodbye world"`

	yamlFileName := "test-pipeline-array.yaml"
	yamlFilePath := filepath.Join(inputsDir, yamlFileName)
	require.NoError(t, os.WriteFile(yamlFilePath, []byte(yamlContent), 0644)) //nolint:gosec

	// Mock workspace root discovery
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceRoot, "domains"), 0755))

	// Set up the test to run from within the workspace
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWd))
	})

	tests := []struct {
		name          string
		changesetName string
		expectError   bool
		description   string
	}{
		{
			name:          "array format - first changeset found",
			changesetName: "test_changeset",
			expectError:   false,
			description:   "Should find and parse first changeset in array format",
		},
		{
			name:          "array format - second changeset found",
			changesetName: "other_changeset",
			expectError:   false,
			description:   "Should find and parse second changeset in array format",
		},
		{
			name:          "array format - changeset not found",
			changesetName: "nonexistent_changeset",
			expectError:   true,
			description:   "Should return error when changeset not found in array format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any previous DURABLE_PIPELINE_INPUT
			os.Unsetenv("DURABLE_PIPELINE_INPUT")

			err := setDurablePipelineInputFromYAML(yamlFileName, tt.changesetName, testDomain, env)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)

				// Verify that DURABLE_PIPELINE_INPUT was set
				durablePipelineInput := os.Getenv("DURABLE_PIPELINE_INPUT")
				require.NotEmpty(t, durablePipelineInput, "DURABLE_PIPELINE_INPUT should be set")

				// Verify the JSON structure based on changeset
				switch tt.changesetName {
				case "test_changeset":
					require.Contains(t, durablePipelineInput, `"value":123`, "Should contain the expected payload")
					require.Contains(t, durablePipelineInput, `"message":"hello world"`, "Should contain the expected payload")
					require.Contains(t, durablePipelineInput, `"chainOverrides":[1,2,3]`, "Should contain the expected chain overrides")
				case "other_changeset":
					require.Contains(t, durablePipelineInput, `"value":456`, "Should contain the expected payload")
					require.Contains(t, durablePipelineInput, `"message":"goodbye world"`, "Should contain the expected payload")
				}
			}
		})
	}
}

//nolint:paralleltest
func TestSetDurablePipelineInputFromYAML_ChainOverrideTypes(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	env := "testnet"

	// Create workspace structure
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	// Mock workspace root discovery
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceRoot, "domains"), 0755))

	// Set up the test to run from within the workspace
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWd))
	})

	tests := []struct {
		name          string
		yamlContent   string
		changesetName string
		expectError   bool
		expectedError string
		expectedJSON  string
	}{
		{
			name: "small integers (int type)",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        message: "test"
      chainOverrides: [1, 2, 3]`,
			changesetName: "test_changeset",
			expectError:   false,
			expectedJSON:  `"chainOverrides":[1,2,3]`,
		},
		{
			name: "large integers (should be handled as int64/uint64)",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        message: "test"
      chainOverrides: [1, 5224473277236331295, 18446744073709551615]`,
			changesetName: "test_changeset",
			expectError:   false,
			expectedJSON:  `"chainOverrides":[1,5224473277236331295,18446744073709551615]`,
		},
		{
			name: "negative number should fail",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        message: "test"
      chainOverrides: [1, -2, 3]`,
			changesetName: "test_changeset",
			expectError:   true,
			expectedError: "chain override value must be non-negative, got: -2",
		},
		{
			name: "string value should fail",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        message: "test"
      chainOverrides: [1, "invalid", 3]`,
			changesetName: "test_changeset",
			expectError:   true,
			expectedError: "chain override value must be an integer, got type string",
		},
		{
			name: "floating point should fail",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        message: "test"
      chainOverrides: [1, 2.5, 3]`,
			changesetName: "test_changeset",
			expectError:   true,
			expectedError: "chain override value must be an integer, got type float64",
		},
		{
			name: "empty chainOverrides should work",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        message: "test"
      chainOverrides: []`,
			changesetName: "test_changeset",
			expectError:   false,
			expectedJSON:  `{"chainOverrides":[],"payload":{"message":"test"}}`,
		},
		{
			name: "missing chainOverrides should work",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        message: "test"`,
			changesetName: "test_changeset",
			expectError:   false,
			expectedJSON:  `"message":"test"`, // chainOverrides should be omitted when missing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create unique YAML file for this test
			safeName := strings.ReplaceAll(strings.ReplaceAll(tt.name, " ", "-"), "/", "-")
			yamlFileName := fmt.Sprintf("test-%s.yaml", safeName)
			yamlFilePath := filepath.Join(inputsDir, yamlFileName)
			require.NoError(t, os.WriteFile(yamlFilePath, []byte(tt.yamlContent), 0644)) //nolint:gosec

			// Clear any previous DURABLE_PIPELINE_INPUT
			os.Unsetenv("DURABLE_PIPELINE_INPUT")

			err := setDurablePipelineInputFromYAML(yamlFileName, tt.changesetName, testDomain, env)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)

				// Verify that DURABLE_PIPELINE_INPUT was set
				durablePipelineInput := os.Getenv("DURABLE_PIPELINE_INPUT")
				require.NotEmpty(t, durablePipelineInput, "DURABLE_PIPELINE_INPUT should be set")

				// Verify the expected JSON content
				require.Contains(t, durablePipelineInput, tt.expectedJSON)
			}
		})
	}
}

//nolint:paralleltest
func TestSetDurablePipelineInputFromYAML_ChainSelectorKeys(t *testing.T) {
	// Test the specific bug where YAML with numeric keys (like chain selectors)
	// creates map[interface{}]interface{} types that can't be marshaled to JSON

	env := "testnet"
	testDomain := domain.NewDomain(t.TempDir(), "test")

	// Create workspace structure
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	// Mock workspace root discovery
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceRoot, "domains"), 0755))

	// Change to the workspace root directory to test relative path resolution
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWd))
	})

	tests := []struct {
		name          string
		yamlContent   string
		changesetName string
		expectError   bool
		expectedError string
		expectedJSON  string
		description   string
	}{
		{
			name: "array format with multiple chain selectors",
			yamlContent: `environment: testnet
domain: test
changesets:
  - deploy_timelock:
      payload:
        16015286601757825753: "ethereum-sepolia"
        13264668187771770619: "bsc-testnet"
        5009297550715157269: "optimism-sepolia"`,
			changesetName: "deploy_timelock",
			expectError:   false,
			expectedJSON:  `{"payload":{"13264668187771770619":"bsc-testnet","16015286601757825753":"ethereum-sepolia","5009297550715157269":"optimism-sepolia"}}`,
			description:   "Should handle multiple chain selectors as keys in array format YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create unique YAML file for this test
			safeName := strings.ReplaceAll(strings.ReplaceAll(tt.name, " ", "-"), "/", "-")
			yamlFileName := fmt.Sprintf("test-numeric-keys-%s.yaml", safeName)
			yamlFilePath := filepath.Join(inputsDir, yamlFileName)
			require.NoError(t, os.WriteFile(yamlFilePath, []byte(tt.yamlContent), 0644)) //nolint:gosec

			// Clear any previous DURABLE_PIPELINE_INPUT
			os.Unsetenv("DURABLE_PIPELINE_INPUT")

			err := setDurablePipelineInputFromYAML(yamlFileName, tt.changesetName, testDomain, env)

			if tt.expectError {
				require.Error(t, err, tt.description)
				require.Contains(t, err.Error(), tt.expectedError, tt.description)
			} else {
				require.NoError(t, err, tt.description)

				// Verify that DURABLE_PIPELINE_INPUT was set
				durablePipelineInput := os.Getenv("DURABLE_PIPELINE_INPUT")
				require.NotEmpty(t, durablePipelineInput, "DURABLE_PIPELINE_INPUT should be set")

				// The main test: ensure it's valid JSON
				var actualJSON map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(durablePipelineInput), &actualJSON), "Generated JSON should be valid")

				// Parse expected JSON for comparison
				var expectedJSON map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(tt.expectedJSON), &expectedJSON), "Expected JSON should be valid")

				// Compare the complete JSON structures
				require.Equal(t, expectedJSON, actualJSON, "Generated JSON should match expected structure exactly")
			}
		})
	}
}

//nolint:paralleltest
func TestSetDurablePipelineInputFromYAMLByIndex(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	env := "testnet"

	// Create workspace structure
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	tests := []struct {
		name          string
		yamlContent   string
		index         int
		expectedName  string
		expectedJSON  string
		expectError   bool
		errorContains string
	}{
		{
			name: "array format - select first changeset",
			yamlContent: `environment: testnet
domain: test
changesets:
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
        value: 120
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
        value: 100`,
			index:        0,
			expectedName: "0001_test_changeset",
			expectedJSON: `{"payload":{"chain":"optimism_sepolia","value":120}}`,
		},
		{
			name: "array format - select second changeset",
			yamlContent: `environment: testnet
domain: test
changesets:
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
        value: 120
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
        value: 100`,
			index:        1,
			expectedName: "0001_test_changeset",
			expectedJSON: `{"payload":{"chain":"optimism_sepolia","value":100}}`,
		},
		{
			name: "object format - should return error",
			yamlContent: `environment: testnet
domain: test
changesets:
  first_changeset:
    payload:
      value: 42
  second_changeset:
    payload:
      value: 84`,
			index:         0,
			expectError:   true,
			errorContains: "--changeset-index can only be used with array format YAML files",
		},
		{
			name: "array format with chainOverrides",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        value: 123
      chainOverrides: [1, 2, 3]`,
			index:        0,
			expectedName: "test_changeset",
			expectedJSON: `{"payload":{"value":123},"chainOverrides":[1,2,3]}`,
		},
		{
			name: "index out of range - too high",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        value: 123`,
			index:         5,
			expectError:   true,
			errorContains: "changeset index 5 is out of range",
		},
		{
			name: "index out of range - negative",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        value: 123`,
			index:         -1,
			expectError:   true,
			errorContains: "changeset index -1 is out of range",
		},
		{
			name: "empty changesets array",
			yamlContent: `environment: testnet
domain: test
changesets: []`,
			index:         0,
			expectError:   true,
			errorContains: "changeset index 0 is out of range (found 0 changesets",
		},
		{
			name: "missing payload field",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      notPayload: 123`,
			index:         0,
			expectError:   true,
			errorContains: "is missing required 'payload' field",
		},
		{
			name: "null payload field - should be valid",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload: null`,
			index:        0,
			expectedName: "test_changeset",
			expectedJSON: `{"payload":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the test to run from within the workspace first
			originalWd, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(workspaceRoot))
			t.Cleanup(func() {
				require.NoError(t, os.Chdir(originalWd))
			})

			// Create unique YAML file for this test
			yamlFileName := fmt.Sprintf("test-pipeline-%s.yaml", strings.ReplaceAll(tt.name, " ", "-"))
			yamlFilePath := filepath.Join(inputsDir, yamlFileName)
			require.NoError(t, os.WriteFile(yamlFilePath, []byte(tt.yamlContent), 0644)) //nolint:gosec
			t.Cleanup(func() {
				os.Remove(yamlFilePath)
			})

			// Clear environment variable before test
			os.Unsetenv("DURABLE_PIPELINE_INPUT")

			// Test the function
			actualName, err := setDurablePipelineInputFromYAMLByIndex(yamlFileName, tt.index, testDomain, env)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedName, actualName)

			// Verify environment variable was set correctly
			envValue := os.Getenv("DURABLE_PIPELINE_INPUT")
			require.NotEmpty(t, envValue, "DURABLE_PIPELINE_INPUT should be set")

			// Parse and compare JSON
			require.JSONEq(t, tt.expectedJSON, envValue)

			// Clean up
			os.Unsetenv("DURABLE_PIPELINE_INPUT")
		})
	}
}

func TestGetAllChangesetsInOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		changesets    any
		expectedNames []string
		expectError   bool
		errorContains string
	}{
		{
			name: "object format - should return error",
			changesets: map[string]any{
				"first":  map[string]any{"payload": map[string]any{"value": 1}},
				"second": map[string]any{"payload": map[string]any{"value": 2}},
			},
			expectError:   true,
			errorContains: "expected array format",
		},
		{
			name: "array format",
			changesets: []any{
				map[string]any{
					"first": map[string]any{"payload": map[string]any{"value": 1}},
				},
				map[string]any{
					"second": map[string]any{"payload": map[string]any{"value": 2}},
				},
			},
			expectedNames: []string{"first", "second"},
		},
		{
			name:          "invalid format - string",
			changesets:    "invalid",
			expectError:   true,
			errorContains: "has invalid 'changesets' format",
		},
		{
			name:          "invalid format - number",
			changesets:    123,
			expectError:   true,
			errorContains: "has invalid 'changesets' format",
		},
		{
			name: "array with invalid item",
			changesets: []any{
				"not a map",
			},
			expectedNames: []string{}, // Should skip invalid items
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := getAllChangesetsInOrder(tt.changesets, "test-file.yaml")

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)

				return
			}

			require.NoError(t, err)

			// Extract names from result
			var actualNames []string
			for _, changeset := range result {
				actualNames = append(actualNames, changeset.name)
			}

			// Check if the expected names match
			if len(tt.expectedNames) > 0 {
				require.Equal(t, tt.expectedNames, actualNames)
			} else {
				require.Empty(t, actualNames)
			}
		})
	}
}

//nolint:paralleltest // This test uses os.Chdir which changes global state
func TestParseDurablePipelineYAML(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	env := "testnet"

	// Create workspace structure
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	tests := []struct {
		name          string
		yamlContent   string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid YAML",
			yamlContent: `environment: testnet
domain: test
changesets:
  - test_changeset:
      payload:
        value: 123`,
		},
		{
			name: "missing environment",
			yamlContent: `domain: test
changesets:
  - test_changeset:
      payload:
        value: 123`,
			expectError:   true,
			errorContains: "missing required 'environment' field",
		},
		{
			name: "missing domain",
			yamlContent: `environment: testnet
changesets:
  - test_changeset:
      payload:
        value: 123`,
			expectError:   true,
			errorContains: "missing required 'domain' field",
		},
		{
			name: "missing changesets",
			yamlContent: `environment: testnet
domain: test`,
			expectError:   true,
			errorContains: "missing required 'changesets' field",
		},
		{
			name: "invalid YAML",
			yamlContent: `environment: testnet
domain: test
changesets: [
  invalid yaml structure`,
			expectError:   true,
			errorContains: "failed to parse input file",
		},
	}

	//nolint:paralleltest // This test uses os.Chdir which changes global state
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the test to run from within the workspace first
			originalWd, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(workspaceRoot))
			t.Cleanup(func() {
				require.NoError(t, os.Chdir(originalWd))
			})

			// Create unique YAML file for this test
			yamlFileName := fmt.Sprintf("test-parse-%s.yaml", strings.ReplaceAll(tt.name, " ", "-"))
			yamlFilePath := filepath.Join(inputsDir, yamlFileName)
			require.NoError(t, os.WriteFile(yamlFilePath, []byte(tt.yamlContent), 0644)) //nolint:gosec
			t.Cleanup(func() {
				os.Remove(yamlFilePath)
			})

			// Test the function
			result, err := parseDurablePipelineYAML(yamlFileName, testDomain, env)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
				require.Nil(t, result)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "testnet", result.Environment)
			require.Equal(t, "test", result.Domain)
			require.NotNil(t, result.Changesets)
		})
	}
}

//nolint:paralleltest
func TestSetDurablePipelineInputFromYAML_NullPayload(t *testing.T) {
	testDomain := domain.NewDomain(t.TempDir(), "test")
	env := "testnet"

	// Create workspace structure
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	// Mock workspace root discovery
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceRoot, "domains"), 0755))

	// Set up the test to run from within the workspace
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWd))
	})

	tests := []struct {
		name          string
		yamlContent   string
		changesetName string
		expectError   bool
		expectedJSON  string
		description   string
	}{
		{
			name: "array format with null payload - should be valid",
			yamlContent: `environment: testnet
domain: test
changesets:
  - deploy_link_token:
      payload: null`,
			changesetName: "deploy_link_token",
			expectError:   false,
			expectedJSON:  `{"payload":null}`,
			description:   "Should allow explicit null payload in array format",
		},
		{
			name: "array format with missing payload - should error",
			yamlContent: `environment: testnet
domain: test
changesets:
  - deploy_link_token:
      notPayload: 123`,
			changesetName: "deploy_link_token",
			expectError:   true,
			description:   "Should error when payload field is completely missing",
		},
		{
			name: "array format with empty payload object - should be valid",
			yamlContent: `environment: testnet
domain: test
changesets:
  - deploy_link_token:
      payload: {}`,
			changesetName: "deploy_link_token",
			expectError:   false,
			expectedJSON:  `{"payload":{}}`,
			description:   "Should allow empty object as payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create unique YAML file for this test
			safeName := strings.ReplaceAll(strings.ReplaceAll(tt.name, " ", "-"), "/", "-")
			yamlFileName := fmt.Sprintf("test-null-payload-%s.yaml", safeName)
			yamlFilePath := filepath.Join(inputsDir, yamlFileName)
			require.NoError(t, os.WriteFile(yamlFilePath, []byte(tt.yamlContent), 0644)) //nolint:gosec

			// Clear any previous DURABLE_PIPELINE_INPUT
			os.Unsetenv("DURABLE_PIPELINE_INPUT")

			err := setDurablePipelineInputFromYAML(yamlFileName, tt.changesetName, testDomain, env)

			if tt.expectError {
				require.Error(t, err, tt.description)
				require.Contains(t, err.Error(), "is missing required 'payload' field", tt.description)
			} else {
				require.NoError(t, err, tt.description)

				// Verify that DURABLE_PIPELINE_INPUT was set
				durablePipelineInput := os.Getenv("DURABLE_PIPELINE_INPUT")
				require.NotEmpty(t, durablePipelineInput, "DURABLE_PIPELINE_INPUT should be set")

				// Verify the JSON structure
				require.JSONEq(t, tt.expectedJSON, durablePipelineInput, tt.description)
			}
		})
	}
}

//nolint:paralleltest
func TestDurablePipelineRunWithObjectFormatError(t *testing.T) {
	env := "testnet"
	testDomain := domain.NewDomain(t.TempDir(), "test")

	// Create workspace structure
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0755))

	// Create test YAML file with OBJECT format (should fail with --changeset-index)
	yamlContent := `environment: testnet
domain: test
changesets:
  0001_test_changeset:
    payload:
      chain: optimism_sepolia
      value: 120
  0002_test_changeset:
    payload:
      chain: optimism_sepolia
      value: 100`

	yamlFileName := "test-object-format.yaml"
	yamlFilePath := filepath.Join(inputsDir, yamlFileName)
	require.NoError(t, os.WriteFile(yamlFilePath, []byte(yamlContent), 0644)) //nolint:gosec

	// Set up the test to run from within the workspace
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWd))
	})

	tempLoadEnv := environment.Load
	// mock the loadEnv function to avoid loading a real environment
	loadEnv = func(ctx context.Context, domain domain.Domain, envName string, options ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
		return fdeployment.Environment{}, nil
	}
	t.Cleanup(func() {
		loadEnv = tempLoadEnv
	})

	// Create a changeset stub
	changesetStub := stubChangeset{
		ApplyCalled: false,
		StubError:   nil,
	}

	// Create a registry with the changeset
	sharedCommands := NewCommands(logger.Test(t))
	rootCmd := sharedCommands.NewDurablePipelineCmds(
		testDomain,
		func(envName string) (*changeset.ChangesetsRegistry, error) {
			rp := changesetsRegistryProviderStub{
				BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
				AddChangesetAction: func(registry *changeset.ChangesetsRegistry) {
					registry.Add("0001_test_changeset", changeset.Configure(&changesetStub).With(1))
				},
			}

			if initErr := rp.Init(); initErr != nil {
				return nil, fmt.Errorf("failed to init changesets %w", initErr)
			}

			return rp.Registry(), nil
		},
		func(env fdeployment.Environment) (analyzer.ProposalContext, error) {
			return &mockProposalContext{t: t}, nil
		},
		fresolvers.NewConfigResolverManager(),
	)

	// Test using --changeset-index with object format (should fail)
	args := []string{
		"run",
		"--environment", env,
		"--input-file", yamlFileName,
		"--changeset-index", "0",
	}

	rootCmd.SetArgs(args)
	err = rootCmd.Execute()

	// Should get an error about object format not being supported
	require.Error(t, err)
	require.Contains(t, err.Error(), "--changeset-index can only be used with array format YAML files")
}
