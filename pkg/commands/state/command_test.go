package state

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// mockState implements json.Marshaler for testing.
type mockState struct {
	Data map[string]any `json:"data"`
}

func (m *mockState) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Data)
}

func (m *mockState) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.Data)
}

// TestNewCommand_Structure verifies the command structure is correct.
func TestNewCommand_Structure(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
	})

	// Verify root command
	assert.Equal(t, "state", cmd.Use)
	assert.Equal(t, "State commands", cmd.Short)

	// Verify environment flag is persistent
	envFlag := cmd.PersistentFlags().Lookup("environment")
	require.NotNil(t, envFlag)
	assert.Equal(t, "e", envFlag.Shorthand)

	// Verify subcommands
	subs := cmd.Commands()
	require.Len(t, subs, 1)
	assert.Equal(t, "generate", subs[0].Use)
}

// TestNewCommand_GenerateFlags verifies the generate subcommand has correct flags.
func TestNewCommand_GenerateFlags(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
	})

	// Find the generate subcommand
	var generateCmd *struct{}
	for _, sub := range cmd.Commands() {
		if sub.Use == "generate" {
			// Check local flags
			p := sub.Flags().Lookup("persist")
			require.NotNil(t, p)
			assert.Equal(t, "p", p.Shorthand)
			assert.Equal(t, "false", p.Value.String())

			o := sub.Flags().Lookup("outputPath")
			require.NotNil(t, o)
			assert.Equal(t, "o", o.Shorthand)
			assert.Empty(t, o.Value.String())

			s := sub.Flags().Lookup("previousState")
			require.NotNil(t, s)
			assert.Equal(t, "s", s.Shorthand)
			assert.Empty(t, s.Value.String())

			return
		}
	}
	if generateCmd == nil {
		t.Fatal("generate subcommand not found")
	}
}

// TestGenerate_MissingEnvironmentFlagFails verifies required flag validation.
func TestGenerate_MissingEnvironmentFlagFails(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
	})

	// Execute without --environment flag
	cmd.SetArgs([]string{"generate"})
	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "environment" not set`)
}

// TestGenerate_Success verifies successful state generation with mocks.
func TestGenerate_Success(t *testing.T) {
	t.Parallel()

	expectedState := &mockState{
		Data: map[string]any{
			"contracts": map[string]any{
				"router": "0x1234567890abcdef",
			},
			"version": "1.0.0",
		},
	}

	// Track what was called
	var environmentLoaderCalled bool
	var stateLoaderCalled bool
	var viewStateCalled bool

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			viewStateCalled = true
			assert.Equal(t, "staging", env.Name)

			return expectedState, nil
		},
		Deps: &Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				environmentLoaderCalled = true
				assert.Equal(t, "staging", envKey)

				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(envdir domain.EnvDir) (domain.JSONSerializer, error) {
				stateLoaderCalled = true
				// Return "file not found" to simulate no previous state
				return nil, os.ErrNotExist
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.True(t, environmentLoaderCalled, "environment loader should be called")
	assert.True(t, stateLoaderCalled, "state loader should be called")
	assert.True(t, viewStateCalled, "view state should be called")

	// Verify output contains expected state data
	output := out.String()
	assert.Contains(t, output, "router")
	assert.Contains(t, output, "0x1234567890abcdef")
}

// TestGenerate_WithPreviousState verifies state generation with existing previous state.
func TestGenerate_WithPreviousState(t *testing.T) {
	t.Parallel()

	prevState := &mockState{
		Data: map[string]any{"existing": "data"},
	}
	newState := &mockState{
		Data: map[string]any{"existing": "data", "new": "value"},
	}

	var receivedPrevState json.Marshaler

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			receivedPrevState = prev
			return newState, nil
		},
		Deps: &Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(envdir domain.EnvDir) (domain.JSONSerializer, error) {
				return prevState, nil
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "mainnet"})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.NotNil(t, receivedPrevState, "previous state should be passed to ViewState")
}

// TestGenerate_WithPersist verifies state is saved when --persist flag is set.
func TestGenerate_WithPersist(t *testing.T) {
	t.Parallel()

	expectedState := &mockState{
		Data: map[string]any{"key": "value"},
	}

	var savedState json.Marshaler
	var savedOutputPath string

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			return expectedState, nil
		},
		Deps: &Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(envdir domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
			StateSaver: func(envdir domain.EnvDir, outputPath string, state json.Marshaler) error {
				savedState = state
				savedOutputPath = outputPath

				return nil
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "--persist"})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, expectedState, savedState, "state should be saved")
	assert.Empty(t, savedOutputPath, "default output path should be empty")
}

// TestGenerate_WithCustomOutputPath verifies custom output path is used.
func TestGenerate_WithCustomOutputPath(t *testing.T) {
	t.Parallel()

	expectedOutputPath := "/custom/path/state.json"
	var savedOutputPath string

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
		Deps: &Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(envdir domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
			StateSaver: func(envdir domain.EnvDir, outputPath string, state json.Marshaler) error {
				savedOutputPath = outputPath
				return nil
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "-p", "-o", expectedOutputPath})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, expectedOutputPath, savedOutputPath, "custom output path should be used")
}

// TestGenerate_EnvironmentLoadError verifies error handling for environment load failures.
func TestGenerate_EnvironmentLoadError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("failed to connect to RPC")

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			t.Fatal("ViewState should not be called on environment load error")

			return json.RawMessage(`{}`), nil
		},
		Deps: &Deps{
			EnvironmentLoader: func(_ context.Context, _ domain.Domain, _ string, _ ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, expectedError
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load environment")
	assert.Contains(t, err.Error(), expectedError.Error())
}

// TestGenerate_ViewStateError verifies error handling for ViewState failures.
func TestGenerate_ViewStateError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("contract read failed")

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			return nil, expectedError
		},
		Deps: &Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(envdir domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to snapshot state")
	assert.Contains(t, err.Error(), expectedError.Error())
}

// TestGenerate_StateSaveError verifies error handling for state save failures.
func TestGenerate_StateSaveError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("permission denied")

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
		Deps: &Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(envdir domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
			StateSaver: func(envdir domain.EnvDir, outputPath string, state json.Marshaler) error {
				return expectedError
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "--persist"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save state")
	assert.Contains(t, err.Error(), expectedError.Error())
}

// TestGenerate_NilViewStateFails verifies error when ViewState is not provided.
func TestGenerate_NilViewStateFails(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(Config{
		Logger:    logger.Nop(),
		Domain:    domain.NewDomain("/tmp", "testdomain"),
		ViewState: nil, // Not provided
		Deps: &Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(envdir domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ViewState function is required but not provided")
}

// TestGenerate_StateLoadErrorNonNotExist verifies error handling for non-NotExist state load errors.
func TestGenerate_StateLoadErrorNonNotExist(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("corrupted state file")

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			t.Fatal("ViewState should not be called on state load error")

			return json.RawMessage(`{}`), nil
		},
		Deps: &Deps{
			EnvironmentLoader: func(_ context.Context, _ domain.Domain, envKey string, _ ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(_ domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, expectedError // Non-NotExist error
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load previous state")
	assert.Contains(t, err.Error(), expectedError.Error())
}

// TestGenerate_OutputFormat verifies the output format is valid JSON.
func TestGenerate_OutputFormat(t *testing.T) {
	t.Parallel()

	expectedState := &mockState{
		Data: map[string]any{
			"chain":    "ethereum",
			"blockNum": 12345,
		},
	}

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, prev json.Marshaler) (json.Marshaler, error) {
			return expectedState, nil
		},
		Deps: &Deps{
			EnvironmentLoader: func(ctx context.Context, dom domain.Domain, envKey string, opts ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(envdir domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
		},
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Extract just the JSON part from output (skip informational messages)
	output := out.String()

	// The state JSON should be parseable
	var parsed map[string]any
	// Find the JSON in the output (starts with {)
	jsonStart := strings.Index(output, "{")
	require.GreaterOrEqual(t, jsonStart, 0, "output should contain JSON")

	jsonStr := output[jsonStart:]
	// Find end of JSON
	jsonEnd := strings.LastIndex(jsonStr, "}")
	require.GreaterOrEqual(t, jsonEnd, 0)
	jsonStr = jsonStr[:jsonEnd+1]

	err = json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, "ethereum", parsed["chain"])
	// Use type assertion for numeric comparison
	blockNum, ok := parsed["blockNum"].(float64)
	require.True(t, ok, "blockNum should be a number")
	assert.Equal(t, 12345, int(blockNum))
}
