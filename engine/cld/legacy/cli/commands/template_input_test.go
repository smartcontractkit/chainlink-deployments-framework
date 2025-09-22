package commands

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/require"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// Test input types for template generation
type SimpleInput struct {
	Name  string `yaml:"name" json:"name"`
	Value int    `yaml:"value" json:"value"`
	Flag  bool   `yaml:"flag" json:"flag"`
}

type ComplexInput struct {
	BasicField   string         `yaml:"basic_field" json:"basic_field"`
	NumberField  uint64         `yaml:"number_field" json:"number_field"`
	Address      common.Address `yaml:"address" json:"address"`
	FloatField   float64        `yaml:"float_field" json:"float_field"`
	SliceField   []string       `yaml:"slice_field" json:"slice_field"`
	MapField     map[string]int `yaml:"map_field" json:"map_field"`
	InterfaceMap map[string]any `yaml:"interface_map" json:"interface_map"`
	NestedStruct *SimpleInput   `yaml:"nested_struct" json:"nested_struct"`
	PointerField *string        `yaml:"pointer_field" json:"pointer_field"`
}

type DeepNestedInput struct {
	Level1 struct {
		Level2 struct {
			Level3 struct {
				Value string `yaml:"value" json:"value"`
			} `yaml:"level3" json:"level3"`
		} `yaml:"level2" json:"level2"`
	} `yaml:"level1" json:"level1"`
}

// Mock resolver for testing
func MockTemplateResolver(input map[string]any) (any, error) {
	return map[string]any{"resolved": true}, nil
}

// Test changesets that implement fdeployment.ChangeSetV2 with specific input types
type SimpleInputChangeset struct{}

func (s *SimpleInputChangeset) Apply(_ fdeployment.Environment, _ SimpleInput) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}

func (s *SimpleInputChangeset) VerifyPreconditions(_ fdeployment.Environment, _ SimpleInput) error {
	return nil
}

type ComplexInputChangeset struct{}

func (c *ComplexInputChangeset) Apply(_ fdeployment.Environment, _ ComplexInput) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}

func (c *ComplexInputChangeset) VerifyPreconditions(_ fdeployment.Environment, _ ComplexInput) error {
	return nil
}

type DeepNestedInputChangeset struct{}

func (d *DeepNestedInputChangeset) Apply(_ fdeployment.Environment, _ DeepNestedInput) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}

func (d *DeepNestedInputChangeset) VerifyPreconditions(_ fdeployment.Environment, _ DeepNestedInput) error {
	return nil
}

type SliceInputChangeset struct{}

func (s *SliceInputChangeset) Apply(_ fdeployment.Environment, _ []uint64) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}

func (s *SliceInputChangeset) VerifyPreconditions(_ fdeployment.Environment, _ []uint64) error {
	return nil
}

type MapInputChangeset struct{}

func (m *MapInputChangeset) Apply(_ fdeployment.Environment, _ map[string]int) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}

func (m *MapInputChangeset) VerifyPreconditions(_ fdeployment.Environment, _ map[string]int) error {
	return nil
}

type IgnoredFieldsInput struct {
	VisibleField    string `yaml:"visible_field" json:"visible_field"`
	AnotherVisible  int    `yaml:"another_visible" json:"another_visible"`
	YamlIgnored     string `yaml:"-"`
	JsonIgnored     string `json:"-"`
	BothIgnored     string `yaml:"-" json:"-"`
	unexportedField string //nolint:unused // This should also be ignored
}

type IgnoredFieldsChangeset struct{}

func (i *IgnoredFieldsChangeset) Apply(_ fdeployment.Environment, _ IgnoredFieldsInput) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}

func (i *IgnoredFieldsChangeset) VerifyPreconditions(_ fdeployment.Environment, _ IgnoredFieldsInput) error {
	return nil
}

func TestNewDurablePipelineTemplateInputCmd(t *testing.T) {
	t.Parallel()

	env := "testnet"

	tests := []struct {
		name            string
		args            []string
		setupMocks      func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error)
		expectedErr     string
		expectedYAML    string
		checkOutputFile func(t *testing.T, outputPath string)
	}{
		{
			name: "successful template generation for simple input",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "0001_simple_changeset",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				resolverManager := fresolvers.NewConfigResolverManager()

				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						cs := &SimpleInputChangeset{}
						registry.Add("0001_simple_changeset", changeset.Configure(cs).With(SimpleInput{}))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedYAML: `# Generated via template-input command
environment: testnet
domain: test
changesets:
  # Input type: commands.SimpleInput
  - 0001_simple_changeset:
      # Optional: Chain overrides (uncomment if needed)
      # chainOverrides:
      #   - 1  # Chain selector 1
      #   - 2  # Chain selector 2
      payload:
        name: # string
        value: # int
        flag: # bool
`,
		},
		{
			name: "successful template generation with config resolver",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "0002_resolver_changeset",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				resolverManager := fresolvers.NewConfigResolverManager()
				resolverManager.Register(MockTemplateResolver, fresolvers.ResolverInfo{
					Description: "Test Template Resolver",
					ExampleYAML: "test: value",
				})

				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						cs := &stubChangeset{resolver: MockTemplateResolver}
						registry.Add("0002_resolver_changeset", changeset.Configure(cs).WithConfigResolver(MockTemplateResolver))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedYAML: `# Generated via template-input command
environment: testnet
domain: test
changesets:
  # Config Resolver: github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli/commands.MockTemplateResolver
  # Input type: map[string]interface {}
  - 0002_resolver_changeset:
      # Optional: Chain overrides (uncomment if needed)
      # chainOverrides:
      #   - 1  # Chain selector 1
      #   - 2  # Chain selector 2
      payload:
        # Map[string]interface {}
        example_key: # interface {}
`,
		},
		{
			name: "template generation with complex input types",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "0003_complex_changeset",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				resolverManager := fresolvers.NewConfigResolverManager()

				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						cs := &ComplexInputChangeset{}
						registry.Add("0003_complex_changeset", changeset.Configure(cs).With(ComplexInput{}))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedYAML: `# Generated via template-input command
environment: testnet
domain: test
changesets:
  # Input type: commands.ComplexInput
  - 0003_complex_changeset:
      # Optional: Chain overrides (uncomment if needed)
      # chainOverrides:
      #   - 1  # Chain selector 1
      #   - 2  # Chain selector 2
      payload:
        basic_field: # string
        number_field: # uint64
        address: # common.Address
        float_field: # float64
        slice_field:
          - # string
        map_field:
          example_key: # int
        interface_map:
          example_key: "interface{} - provide appropriate value"
        nested_struct:
          name: # string
          value: # int
          flag: # bool
        pointer_field: # string
`,
		},
		{
			name: "template generation with multiple changesets",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "0004_changeset1,0005_changeset2",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				resolverManager := fresolvers.NewConfigResolverManager()

				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						cs1 := &SimpleInputChangeset{}
						registry.Add("0004_changeset1", changeset.Configure(cs1).With(SimpleInput{}))

						cs2 := &SimpleInputChangeset{}
						registry.Add("0005_changeset2", changeset.Configure(cs2).With(SimpleInput{}))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedYAML: `# Generated via template-input command
environment: testnet
domain: test
changesets:
  # Input type: commands.SimpleInput
  - 0004_changeset1:
      # Optional: Chain overrides (uncomment if needed)
      # chainOverrides:
      #   - 1  # Chain selector 1
      #   - 2  # Chain selector 2
      payload:
        name: # string
        value: # int
        flag: # bool

  # ----------------------------------------
  # Input type: commands.SimpleInput
  - 0005_changeset2:
      # Optional: Chain overrides (uncomment if needed)
      # chainOverrides:
      #   - 1  # Chain selector 1
      #   - 2  # Chain selector 2
      payload:
        name: # string
        value: # int
        flag: # bool
`,
		},
		{
			name: "template generation with depth limit",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "0006_deep_changeset",
				"--depth", "2",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				resolverManager := fresolvers.NewConfigResolverManager()

				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						cs := &DeepNestedInputChangeset{}
						registry.Add("0006_deep_changeset", changeset.Configure(cs).With(DeepNestedInput{}))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedYAML: `# Generated via template-input command
environment: testnet
domain: test
changesets:
  # Input type: commands.DeepNestedInput
  - 0006_deep_changeset:
      # Optional: Chain overrides (uncomment if needed)
      # chainOverrides:
      #   - 1  # Chain selector 1
      #   - 2  # Chain selector 2
      payload:
        level1:
          level2: ...
`,
		},
		{
			name: "template generation with root-level slice type",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "0008_slice_changeset",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				resolverManager := fresolvers.NewConfigResolverManager()

				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						cs := &SliceInputChangeset{}
						registry.Add("0008_slice_changeset", changeset.Configure(cs).With([]uint64{}))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedYAML: `# Generated via template-input command
environment: testnet
domain: test
changesets:
  # Input type: []uint64
  - 0008_slice_changeset:
      # Optional: Chain overrides (uncomment if needed)
      # chainOverrides:
      #   - 1  # Chain selector 1
      #   - 2  # Chain selector 2
      payload:
        # Array of uint64
        - # uint64
`,
		},
		{
			name: "template generation with root-level map type",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "0009_map_changeset",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				resolverManager := fresolvers.NewConfigResolverManager()

				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						cs := &MapInputChangeset{}
						registry.Add("0009_map_changeset", changeset.Configure(cs).With(map[string]int{}))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedYAML: `# Generated via template-input command
environment: testnet
domain: test
changesets:
  # Input type: map[string]int
  - 0009_map_changeset:
      # Optional: Chain overrides (uncomment if needed)
      # chainOverrides:
      #   - 1  # Chain selector 1
      #   - 2  # Chain selector 2
      payload:
        # Map[string]int
        example_key: # int
`,
		},
		{
			name: "template generation with ignored fields",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "0010_ignored_fields_changeset",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				resolverManager := fresolvers.NewConfigResolverManager()

				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						cs := &IgnoredFieldsChangeset{}
						registry.Add("0010_ignored_fields_changeset", changeset.Configure(cs).With(IgnoredFieldsInput{}))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedYAML: `# Generated via template-input command
environment: testnet
domain: test
changesets:
  # Input type: commands.IgnoredFieldsInput
  - 0010_ignored_fields_changeset:
      # Optional: Chain overrides (uncomment if needed)
      # chainOverrides:
      #   - 1  # Chain selector 1
      #   - 2  # Chain selector 2
      payload:
        visible_field: # string
        another_visible: # int
`,
		},
		{
			name: "missing environment flag",
			args: []string{
				"template-input",
				"--changeset", "0008_test_changeset",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				return changeset.NewChangesetsRegistry(), fresolvers.NewConfigResolverManager(), nil
			},
			expectedErr: "required flag(s) \"environment\" not set",
		},
		{
			name: "missing changeset flag",
			args: []string{
				"template-input",
				"--environment", env,
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				return changeset.NewChangesetsRegistry(), fresolvers.NewConfigResolverManager(), nil
			},
			expectedErr: "required flag(s) \"changeset\" not set",
		},
		{
			name: "nonexistent changeset",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "nonexistent_changeset",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						// Don't add the changeset
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), fresolvers.NewConfigResolverManager(), nil
			},
			expectedErr: "get configurations for changeset nonexistent_changeset:",
		},
		{
			name: "unregistered resolver",
			args: []string{
				"template-input",
				"--environment", env,
				"--changeset", "0007_unregistered_resolver_changeset",
			},
			setupMocks: func() (*changeset.ChangesetsRegistry, *fresolvers.ConfigResolverManager, error) {
				resolverManager := fresolvers.NewConfigResolverManager()
				// Don't register the resolver

				rp := migrationsRegistryProviderStub{
					BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
					AddMigrationAction: func(registry *changeset.ChangesetsRegistry) {
						cs := &stubChangeset{resolver: MockTemplateResolver}
						registry.Add("0007_unregistered_resolver_changeset", changeset.Configure(cs).WithConfigResolver(MockTemplateResolver))
					},
				}

				if err := rp.Init(); err != nil {
					return nil, nil, err
				}

				return rp.Registry(), resolverManager, nil
			},
			expectedErr: "resolver for changeset 0007_unregistered_resolver_changeset is not registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testDomain := fdomain.NewDomain(t.TempDir(), "test")
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
				nil, // No proposal context needed for template generation
				resolverManager,
			)

			require.NotNil(t, rootCmd)
			rootCmd.SetArgs(tt.args)

			// Capture output using SetOut
			var output strings.Builder
			rootCmd.SetOut(&output)

			err := rootCmd.Execute()

			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)

				outputStr := output.String()

				// Assert exact YAML match
				require.Equal(t, tt.expectedYAML, outputStr, "Generated YAML should match expected format exactly")

				// Check output file if specified
				if tt.checkOutputFile != nil {
					// Find --output flag and check the file
					for i, arg := range tt.args {
						if arg == "--output" && i+1 < len(tt.args) {
							tt.checkOutputFile(t, tt.args[i+1])
							break
						}
					}
				}
			}
		})
	}
}

func TestGenerateFieldValueWithDepthLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		inputType      reflect.Type
		expectedOutput string
		maxDepth       int
		description    string
	}{
		{
			name:           "string type",
			inputType:      reflect.TypeOf(""),
			expectedOutput: " # string",
			maxDepth:       5,
			description:    "Should generate type comment for string",
		},
		{
			name:           "int type",
			inputType:      reflect.TypeOf(0),
			expectedOutput: " # int",
			maxDepth:       5,
			description:    "Should generate type comment for int",
		},
		{
			name:           "bool type",
			inputType:      reflect.TypeOf(false),
			expectedOutput: " # bool",
			maxDepth:       5,
			description:    "Should generate type comment for bool",
		},
		{
			name:           "uint64 type",
			inputType:      reflect.TypeOf(uint64(0)),
			expectedOutput: " # uint64",
			maxDepth:       5,
			description:    "Should generate type comment for uint64",
		},
		{
			name:           "common.Address type",
			inputType:      reflect.TypeOf(common.Address{}),
			expectedOutput: " # common.Address",
			maxDepth:       5,
			description:    "Should generate type comment for common.Address",
		},
		{
			name:           "float64 type",
			inputType:      reflect.TypeOf(float64(0)),
			expectedOutput: " # float64",
			maxDepth:       5,
			description:    "Should generate type comment for float64",
		},
		{
			name:           "pointer type",
			inputType:      reflect.TypeOf((*string)(nil)),
			expectedOutput: " # string",
			maxDepth:       5,
			description:    "Should handle pointer types by dereferencing",
		},
		{
			name:           "slice type",
			inputType:      reflect.TypeOf([]string{}),
			expectedOutput: "\n  - # string",
			maxDepth:       5,
			description:    "Should generate array format for slice",
		},
		{
			name:           "map type",
			inputType:      reflect.TypeOf(map[string]int{}),
			expectedOutput: "\n  example_key: # int",
			maxDepth:       5,
			description:    "Should generate map format",
		},
		{
			name:           "interface type",
			inputType:      reflect.TypeOf((*interface{})(nil)).Elem(),
			expectedOutput: `"interface{} - provide appropriate value"`,
			maxDepth:       5,
			description:    "Should handle interface{} type",
		},
		{
			name:           "depth exceeded",
			inputType:      reflect.TypeOf(""),
			expectedOutput: " ...",
			maxDepth:       0,
			description:    "Should return ... when depth is exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := generateFieldValueWithDepthLimit(tt.inputType, "  ", 1, make(map[reflect.Type]bool), tt.maxDepth)
			require.NoError(t, err, tt.description)
			require.Equal(t, tt.expectedOutput, result, tt.description)
		})
	}
}

func TestGenerateStructYAMLWithDepthLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		inputType      reflect.Type
		expectedFields []string
		maxDepth       int
		description    string
	}{
		{
			name:      "simple struct",
			inputType: reflect.TypeOf(SimpleInput{}),
			expectedFields: []string{
				"name: # string",
				"value: # int",
				"flag: # bool",
			},
			maxDepth:    5,
			description: "Should generate YAML for simple struct",
		},
		{
			name:      "complex struct with nested types",
			inputType: reflect.TypeOf(ComplexInput{}),
			expectedFields: []string{
				"basic_field: # string",
				"slice_field:",
				"- # string",
				"map_field:",
				"example_key: # int",
			},
			maxDepth:    5,
			description: "Should generate YAML for complex struct with nested types",
		},
		{
			name:      "depth limited struct",
			inputType: reflect.TypeOf(DeepNestedInput{}),
			expectedFields: []string{
				"level1:",
			},
			maxDepth:    1,
			description: "Should respect depth limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := generateStructYAMLWithDepthLimit(tt.inputType, "  ", 0, make(map[reflect.Type]bool), tt.maxDepth)
			require.NoError(t, err, tt.description)

			for _, expectedField := range tt.expectedFields {
				require.Contains(t, result, expectedField, "Result should contain field: %q\nActual result:\n%s", expectedField, result)
			}
		})
	}
}

func TestGetFieldName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		field        reflect.StructField
		expectedName string
		description  string
	}{
		{
			name: "yaml tag present",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `yaml:"test_field" json:"testField"`,
			},
			expectedName: "test_field",
			description:  "Should use yaml tag when present",
		},
		{
			name: "json tag only",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `json:"testField"`,
			},
			expectedName: "testField",
			description:  "Should use json tag when yaml tag is not present",
		},
		{
			name: "no tags",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  "",
			},
			expectedName: "testfield",
			description:  "Should use lowercase field name when no tags present",
		},
		{
			name: "yaml tag with options",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `yaml:"test_field,omitempty"`,
			},
			expectedName: "test_field",
			description:  "Should extract field name from yaml tag ignoring options",
		},
		{
			name: "empty yaml tag",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  `yaml:"" json:"testField"`,
			},
			expectedName: "testField",
			description:  "Should fallback to json tag when yaml tag is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := getFieldName(tt.field)
			require.Equal(t, tt.expectedName, result, tt.description)
		})
	}
}
