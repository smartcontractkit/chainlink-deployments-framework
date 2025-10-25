package changeset

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

func TestChangesets_WithConfig(t *testing.T) {
	t.Parallel()

	cfg := "my config"
	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			if config != "my config" {
				t.Errorf("Expected config \"%s\" but got \"%s\"", cfg, config)
			}

			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)
	env := deployment.Environment{Logger: logger.Test(t)}
	configured := Configure(cs).With(cfg)

	_, err := configured.Apply(env)
	require.NoError(t, err, "Failed to apply changeset with config")

	options, err := configured.Configurations()
	require.NoError(t, err, "Failed to get options")
	assert.Nil(t, options.InputChainOverrides)
}

func TestChangesets_WithJSON(t *testing.T) {
	t.Parallel()

	cfg := "my config"
	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config []string) (deployment.ChangesetOutput, error) {
			if config[0] != "my config" {
				t.Errorf("Expected config \"%s\" but got \"%s\"", cfg, config)
			}

			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config []string) error { return nil },
	)
	env := deployment.Environment{Logger: logger.Test(t)}
	configured := Configure(cs).WithJSON([]string{}, "{\"payload\":[\"my config\"]}")

	_, err := configured.Apply(env)
	require.NoError(t, err, "Failed to apply changeset with JSON config")

	options, err := configured.Configurations()
	require.NoError(t, err, "Failed to get options")
	assert.Nil(t, options.InputChainOverrides)
}

func TestChangesets_WithJSON_EmptyInput(t *testing.T) {
	t.Parallel()

	cfg := ""
	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config []string) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config []string) error { return nil },
	)
	env := deployment.Environment{Logger: logger.Test(t)}
	configured := Configure(cs).WithJSON([]string{}, cfg)

	// Apply should fail when input is empty
	_, err := configured.Apply(env)
	require.ErrorContains(t, err, "input is empty")

	// But Configurations should succeed for discovery purposes
	configs, err := configured.Configurations()
	require.NoError(t, err)
	require.Nil(t, configs.InputChainOverrides) // Chain overrides should be nil when input is missing
}

func TestChangesets_WithJSON_StrictPayloadUnmarshaling(t *testing.T) {
	t.Parallel()

	type TestConfig struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	tests := []struct {
		name        string
		inputJSON   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid payload with only known fields",
			inputJSON:   `{"payload":{"value":"test","count":42}}`,
			expectError: false,
		},
		{
			name:        "invalid payload with unknown field",
			inputJSON:   `{"payload":{"value":"test","count":42,"unknownField":"value"}}`,
			expectError: true,
			errorMsg:    "failed to unmarshal payload: json: unknown field \"unknownField\"",
		},
		{
			name:        "invalid payload with multiple unknown fields",
			inputJSON:   `{"payload":{"value":"test","count":42,"unknownField1":"value1","unknownField2":"value2"}}`,
			expectError: true,
			errorMsg:    "failed to unmarshal payload: json: unknown field \"unknownField1\"",
		},
		{
			name:        "invalid payload with nested unknown field",
			inputJSON:   `{"payload":{"value":"test","count":42,"nested":{"unknown":"field"}}}`,
			expectError: true,
			errorMsg:    "failed to unmarshal payload: json: unknown field \"nested\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cs := deployment.CreateChangeSet(
				func(e deployment.Environment, config TestConfig) (deployment.ChangesetOutput, error) {
					return deployment.ChangesetOutput{}, nil
				},
				func(e deployment.Environment, config TestConfig) error { return nil },
			)
			env := deployment.Environment{Logger: logger.Test(t)}
			configured := Configure(cs).WithJSON(TestConfig{}, tt.inputJSON)

			_, err := configured.Apply(env)

			if tt.expectError {
				require.Error(t, err, "Expected error for test case: %s", tt.name)
				require.ErrorContains(t, err, tt.errorMsg, "Error message should contain expected text")
			} else {
				require.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

func TestChangesets_WithEnvInput(t *testing.T) {
	expectedConfig := "config from env"
	t.Setenv("DURABLE_PIPELINE_INPUT", `{"payload":"`+expectedConfig+`"}`)

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			require.Equal(t, "config from env modified", config)

			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)
	env := deployment.Environment{Logger: logger.Test(t)}
	configured := Configure(cs).
		WithEnvInput(
			InputModifierFunc(func(config string) (string, error) {
				return config + " modified", nil
			}))

	_, err := configured.Apply(env)
	require.NoError(t, err)

	options, err := configured.Configurations()
	require.NoError(t, err)
	assert.Nil(t, options.InputChainOverrides)
}

func TestChangesets_WithDurablePipelineInput(t *testing.T) {
	type testConfig struct {
		Value string
	}
	expectedConfig := testConfig{Value: "config from env"}

	payload, err := json.Marshal(expectedConfig)
	require.NoError(t, err)
	inputJSON := `{"payload":` + string(payload) + `}`
	t.Setenv("DURABLE_PIPELINE_INPUT", inputJSON)

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config testConfig) (deployment.ChangesetOutput, error) {
			require.Equal(t, expectedConfig, config)

			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config testConfig) error { return nil },
	)
	env := deployment.Environment{Logger: logger.Test(t)}
	configured := Configure(cs).WithEnvInput()

	_, err = configured.Apply(env)
	require.NoError(t, err)

	options, err := configured.Configurations()
	require.NoError(t, err)
	assert.Nil(t, options.InputChainOverrides)
}

func TestChangesets_WithDurablePipelineInput_Empty(t *testing.T) {
	t.Setenv("DURABLE_PIPELINE_INPUT", "")

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)
	env := deployment.Environment{Logger: logger.Test(t)}
	configured := Configure(cs).WithEnvInput()

	// Apply should fail when input is empty
	_, err := configured.Apply(env)
	require.ErrorContains(t, err, "input is empty")
}

func TestChangesets_ConfigProvider(t *testing.T) {
	t.Parallel()

	provider := func() (string, error) { return "value", nil }
	executed := false
	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			executed = true
			if config != "value" {
				t.Errorf("Expected \"value\" but got \"%s\"", config)
			}

			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)
	env := deployment.Environment{Logger: logger.Test(t)}
	configured := Configure(cs).WithConfigFrom(provider)

	_, err := configured.Apply(env)
	require.True(t, executed, "Changeset should have been executed")
	require.NoError(t, err, "Failed to apply changeset with config provider")

	options, err := configured.Configurations()
	require.NoError(t, err, "Failed to get options")
	assert.Nil(t, options.InputChainOverrides)
}

func TestChangesets_ConfigProviderWithError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("error in config provider")
	provider := func() (string, error) { return "value", expectedErr }
	executed := false
	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			executed = true
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)
	env := deployment.Environment{Logger: logger.Test(t)}
	fromProvider := Configure(cs).WithConfigFrom(provider)

	_, err := fromProvider.Apply(env)

	if executed {
		t.Fatalf("Changeset should not have been executed")
	}
	if err == nil {
		t.Fatalf("Failed to propagate error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Incorrect error propagated: %v", err)
	}
}

type TestConfig struct {
	Chains []uint64
}

func TestInputChainOverrides_With(t *testing.T) {
	t.Parallel()

	expected := []uint64{1, 2, 3}
	cfg := TestConfig{Chains: expected}
	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, c TestConfig) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, c TestConfig) error { return nil },
	)
	configured := Configure(cs).With(cfg)
	_, err := configured.Configurations()
	require.NoError(t, err)
}

func TestInputChainOverrides_WithConfigFrom(t *testing.T) {
	t.Parallel()

	expected := []uint64{4, 5, 6}
	provider := func() (TestConfig, error) { return TestConfig{Chains: expected}, nil }
	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, c TestConfig) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, c TestConfig) error { return nil },
	)
	configured := Configure(cs).WithConfigFrom(provider)
	_, err := configured.Configurations()
	require.NoError(t, err)
}

type Config struct {
	ChainOverrides []uint64
	Payload        TestConfig
}

func TestInputChainOverrides_WithJSON_ChainOverridesField(t *testing.T) {
	t.Parallel()

	config := Config{
		Payload:        TestConfig{Chains: []uint64{1, 2, 3}},
		ChainOverrides: []uint64{10, 11, 12}, // this field should be used for chain overrides, higher priority
	}
	input, err := json.Marshal(config)
	require.NoError(t, err, "Failed to marshal config to JSON")

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, c TestConfig) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, c TestConfig) error { return nil },
	)
	configured := Configure(cs).WithJSON(TestConfig{}, string(input))
	configs, err := configured.Configurations()
	require.NoError(t, err)
	assert.Equal(t, config.ChainOverrides, configs.InputChainOverrides)
}

func TestWithConfigResolver_Success(t *testing.T) {
	type TestConfigType struct {
		Value string
		Count int
	}

	manager := resolvers.NewConfigResolverManager()
	resolver := func(input map[string]any) (any, error) {
		return TestConfigType{
			Value: input["value"].(string),
			Count: int(input["count"].(float64)),
		}, nil
	}
	info := resolvers.ResolverInfo{
		Description: "Test resolver",
	}

	manager.Register(resolver, info)

	// Set up environment variable
	inputJSON := `{
		"payload": {
			"value": "test-value",
			"count": 42
		}
	}`
	t.Setenv("DURABLE_PIPELINE_INPUT", inputJSON)

	// Create changeset
	executed := false
	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config TestConfigType) (deployment.ChangesetOutput, error) {
			executed = true
			assert.Equal(t, "test-value", config.Value)
			assert.Equal(t, 42, config.Count)

			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config TestConfigType) error { return nil },
	)

	configured := Configure(cs).WithConfigResolver(resolver)
	env := deployment.Environment{Logger: logger.Test(t)}

	_, err := configured.Apply(env)
	require.NoError(t, err)
	assert.True(t, executed)

	// Test configurations
	configs, err := configured.Configurations()
	require.NoError(t, err)
	assert.NotNil(t, configs.ConfigResolver)
}

func TestWithConfigResolver_EmptyInput(t *testing.T) {
	manager := resolvers.NewConfigResolverManager()
	resolver := func(input map[string]any) (any, error) {
		return "config", nil
	}
	manager.Register(resolver, resolvers.ResolverInfo{Description: "Test"})

	t.Setenv("DURABLE_PIPELINE_INPUT", "")

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)

	configured := Configure(cs).WithConfigResolver(resolver)
	env := deployment.Environment{Logger: logger.Test(t)}

	_, err := configured.Apply(env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input is empty")
}

func TestWithConfigResolver_InvalidJSON(t *testing.T) {
	manager := resolvers.NewConfigResolverManager()
	resolver := func(input map[string]any) (any, error) {
		return "config", nil
	}
	manager.Register(resolver, resolvers.ResolverInfo{Description: "Test"})

	t.Setenv("DURABLE_PIPELINE_INPUT", "invalid json")

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)

	configured := Configure(cs).WithConfigResolver(resolver)
	env := deployment.Environment{Logger: logger.Test(t)}

	_, err := configured.Apply(env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse resolver input as JSON")
}

func TestWithConfigResolver_MissingPayload(t *testing.T) {
	manager := resolvers.NewConfigResolverManager()
	resolver := func(input map[string]any) (any, error) {
		return "config", nil
	}
	manager.Register(resolver, resolvers.ResolverInfo{Description: "Test"})

	t.Setenv("DURABLE_PIPELINE_INPUT", `{"payload": null}`)

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)

	configured := Configure(cs).WithConfigResolver(resolver)
	env := deployment.Environment{Logger: logger.Test(t)}

	_, err := configured.Apply(env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'payload' field is required and cannot be empty")
}

func TestWithConfigResolver_ResolverError(t *testing.T) {
	manager := resolvers.NewConfigResolverManager()
	expectedError := errors.New("resolver failed")
	resolver := func(input map[string]any) (any, error) {
		return nil, expectedError
	}
	manager.Register(resolver, resolvers.ResolverInfo{Description: "Test"})

	inputJSON := `{"payload": {"value": "test"}}`
	t.Setenv("DURABLE_PIPELINE_INPUT", inputJSON)

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)

	configured := Configure(cs).WithConfigResolver(resolver)
	env := deployment.Environment{Logger: logger.Test(t)}

	_, err := configured.Apply(env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config resolver failed")
	assert.Contains(t, err.Error(), "resolver failed")
}

func TestWithConfigResolver_WrongReturnType(t *testing.T) {
	manager := resolvers.NewConfigResolverManager()
	resolver := func(input map[string]any) (any, error) {
		return 42, nil // Return int instead of string
	}
	manager.Register(resolver, resolvers.ResolverInfo{Description: "Test"})

	inputJSON := `{"payload": {"value": "test"}}`
	t.Setenv("DURABLE_PIPELINE_INPUT", inputJSON)

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config string) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config string) error { return nil },
	)

	configured := Configure(cs).WithConfigResolver(resolver)
	env := deployment.Environment{Logger: logger.Test(t)}

	_, err := configured.Apply(env)
	require.Error(t, err)

	fmt.Println(err.Error())

	assert.Contains(t, err.Error(), "resolver returned")
}

func TestWithConfigResolver_ChainOverrides(t *testing.T) {
	type TestConfigWithChains struct {
		Value  string
		Chains []uint64
	}

	manager := resolvers.NewConfigResolverManager()
	resolver := func(input map[string]any) (any, error) {
		return TestConfigWithChains{
			Value:  "test",
			Chains: []uint64{1, 2, 3},
		}, nil
	}
	manager.Register(resolver, resolvers.ResolverInfo{Description: "Test"})

	// Test with explicit chain overrides
	inputJSON := `{
		"payload": {"value": "test"},
		"chainOverrides": [10, 20, 30]
	}`
	t.Setenv("DURABLE_PIPELINE_INPUT", inputJSON)

	cs := deployment.CreateChangeSet(
		func(e deployment.Environment, config TestConfigWithChains) (deployment.ChangesetOutput, error) {
			return deployment.ChangesetOutput{}, nil
		},
		func(e deployment.Environment, config TestConfigWithChains) error { return nil },
	)

	configured := Configure(cs).WithConfigResolver(resolver)
	configs, err := configured.Configurations()
	require.NoError(t, err)

	// Explicit chain overrides should take precedence
	assert.Equal(t, []uint64{10, 20, 30}, configs.InputChainOverrides)
}

func TestWithConfigResolver_StrictPayloadUnmarshaling(t *testing.T) {
	type TestInput struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	type TestOutput struct {
		Result string
	}

	resolver := func(input TestInput) (TestOutput, error) {
		return TestOutput{Result: fmt.Sprintf("%s_%d", input.Value, input.Count)}, nil
	}

	tests := []struct {
		name        string
		inputJSON   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid payload with only known fields",
			inputJSON:   `{"payload":{"value":"test","count":42}}`,
			expectError: false,
		},
		{
			name:        "invalid payload with unknown field",
			inputJSON:   `{"payload":{"value":"test","count":42,"unknownField":"value"}}`,
			expectError: true,
			errorMsg:    "config resolver failed: unmarshal payload into changeset.TestInput: json: unknown field \"unknownField\"",
		},
		{
			name:        "invalid payload with multiple unknown fields",
			inputJSON:   `{"payload":{"value":"test","count":42,"unknownField1":"value1","unknownField2":"value2"}}`,
			expectError: true,
			errorMsg:    "config resolver failed: unmarshal payload into changeset.TestInput: json: unknown field \"unknownField1\"",
		},
		{
			name:        "invalid payload with nested unknown field",
			inputJSON:   `{"payload":{"value":"test","count":42,"nested":{"unknown":"field"}}}`,
			expectError: true,
			errorMsg:    "config resolver failed: unmarshal payload into changeset.TestInput: json: unknown field \"nested\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DURABLE_PIPELINE_INPUT", tt.inputJSON)

			cs := deployment.CreateChangeSet(
				func(e deployment.Environment, config TestOutput) (deployment.ChangesetOutput, error) {
					return deployment.ChangesetOutput{}, nil
				},
				func(e deployment.Environment, config TestOutput) error { return nil },
			)
			env := deployment.Environment{Logger: logger.Test(t)}
			configured := Configure(cs).WithConfigResolver(resolver)

			_, err := configured.Apply(env)

			if tt.expectError {
				require.Error(t, err, "Expected error for test case: %s", tt.name)
				require.ErrorContains(t, err, tt.errorMsg, "Error message should contain expected text")
			} else {
				require.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

func TestWithEnvInput_StrictPayloadUnmarshaling(t *testing.T) {
	type TestConfig struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	tests := []struct {
		name        string
		inputJSON   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid payload with only known fields",
			inputJSON:   `{"payload":{"value":"test","count":42}}`,
			expectError: false,
		},
		{
			name:        "invalid payload with unknown field",
			inputJSON:   `{"payload":{"value":"test","count":42,"unknownField":"value"}}`,
			expectError: true,
			errorMsg:    "failed to unmarshal payload: json: unknown field \"unknownField\"",
		},
		{
			name:        "invalid payload with multiple unknown fields",
			inputJSON:   `{"payload":{"value":"test","count":42,"unknownField1":"value1","unknownField2":"value2"}}`,
			expectError: true,
			errorMsg:    "failed to unmarshal payload: json: unknown field \"unknownField1\"",
		},
		{
			name:        "invalid payload with nested unknown field",
			inputJSON:   `{"payload":{"value":"test","count":42,"nested":{"unknown":"field"}}}`,
			expectError: true,
			errorMsg:    "failed to unmarshal payload: json: unknown field \"nested\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DURABLE_PIPELINE_INPUT", tt.inputJSON)

			cs := deployment.CreateChangeSet(
				func(e deployment.Environment, config TestConfig) (deployment.ChangesetOutput, error) {
					return deployment.ChangesetOutput{}, nil
				},
				func(e deployment.Environment, config TestConfig) error { return nil },
			)
			env := deployment.Environment{Logger: logger.Test(t)}
			configured := Configure(cs).WithEnvInput()

			_, err := configured.Apply(env)

			if tt.expectError {
				require.Error(t, err, "Expected error for test case: %s", tt.name)
				require.ErrorContains(t, err, tt.errorMsg, "Error message should contain expected text")
			} else {
				require.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

func TestConfigurations_ConfigResolverInfo(t *testing.T) {
	t.Parallel()

	resolver := func(input map[string]any) (any, error) {
		return "config", nil
	}

	configs := Configurations{
		ConfigResolver: resolver,
	}

	assert.Equal(t, reflect.ValueOf(resolver).Pointer(), reflect.ValueOf(configs.ConfigResolver).Pointer())
}
