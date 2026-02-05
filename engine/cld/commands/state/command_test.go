package state

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
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

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Verify root command
	assert.Equal(t, "state", cmd.Use)
	assert.Equal(t, stateShort, cmd.Short)
	assert.NotEmpty(t, cmd.Long, "state command should have a Long description")

	// Verify NO persistent flags on parent (all flags are local to subcommands)
	envFlag := cmd.PersistentFlags().Lookup("environment")
	assert.Nil(t, envFlag, "environment flag should NOT be persistent")

	// Verify subcommands
	subs := cmd.Commands()
	require.Len(t, subs, 1)
	assert.Equal(t, "generate", subs[0].Use)
}

// TestNewCommand_GenerateFlags verifies the generate subcommand has correct local flags.
func TestNewCommand_GenerateFlags(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Find the generate subcommand
	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "generate" {
			found = true

			// Environment flag - local to generate (not persistent)
			e := sub.Flags().Lookup("environment")
			require.NotNil(t, e, "environment flag should be on generate")
			assert.Equal(t, "e", e.Shorthand)

			// Persist flag
			p := sub.Flags().Lookup("persist")
			require.NotNil(t, p)
			assert.Equal(t, "p", p.Shorthand)
			assert.Equal(t, "false", p.Value.String())

			// Output flag (--out/-o, with --outputPath as normalized alias)
			o := sub.Flags().Lookup("out")
			require.NotNil(t, o)
			assert.Equal(t, "o", o.Shorthand)
			assert.Empty(t, o.Value.String())

			// Previous state flag (--prev/-s, with --previousState as normalized alias)
			s := sub.Flags().Lookup("prev")
			require.NotNil(t, s)
			assert.Equal(t, "s", s.Shorthand)
			assert.Empty(t, s.Value.String())

			// Print flag (default true)
			pr := sub.Flags().Lookup("print")
			require.NotNil(t, pr)
			assert.Equal(t, "true", pr.Value.String())

			break
		}
	}
	require.True(t, found, "generate subcommand not found")
}

// TestGenerate_MissingEnvironmentFlagFails verifies required flag validation.
func TestGenerate_MissingEnvironmentFlagFails(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
	})

	require.NoError(t, err)

	// Execute without --environment flag
	cmd.SetArgs([]string{"generate"})
	execErr := cmd.Execute()

	require.ErrorContains(t, execErr, `required flag(s) "environment" not set`)
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

	var environmentLoaderCalled bool
	var stateLoaderCalled bool
	var viewStateCalled bool

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(env fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			viewStateCalled = true
			assert.Equal(t, "staging", env.Name)

			return expectedState, nil
		},
		Deps: Deps{
			EnvironmentLoader: func(_ context.Context, _ domain.Domain, envKey string, _ ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				environmentLoaderCalled = true
				assert.Equal(t, "staging", envKey)

				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(_ domain.EnvDir) (domain.JSONSerializer, error) {
				stateLoaderCalled = true

				return nil, os.ErrNotExist
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.True(t, environmentLoaderCalled, "environment loader should be called")
	assert.True(t, stateLoaderCalled, "state loader should be called")
	assert.True(t, viewStateCalled, "view state should be called")

	// Default --print=true, so state should be in output
	output := out.String()
	assert.Contains(t, output, "router")
}

// TestGenerate_WithPrintFalse verifies state is NOT printed when --print=false.
func TestGenerate_WithPrintFalse(t *testing.T) {
	t.Parallel()

	expectedState := &mockState{
		Data: map[string]any{"key": "value"},
	}

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return expectedState, nil
		},
		Deps: Deps{
			EnvironmentLoader: func(_ context.Context, _ domain.Domain, envKey string, _ ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(_ domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "--print=false"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	// With --print=false, state should NOT be in output
	output := out.String()
	assert.NotContains(t, output, "key")
	assert.NotContains(t, output, "value")
}

// TestGenerate_WithPersist verifies state is saved and path is printed.
func TestGenerate_WithPersist(t *testing.T) {
	t.Parallel()

	expectedState := &mockState{
		Data: map[string]any{"key": "value"},
	}

	var savedState json.Marshaler
	var savedOutputPath string

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return expectedState, nil
		},
		Deps: Deps{
			EnvironmentLoader: func(_ context.Context, _ domain.Domain, envKey string, _ ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(_ domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
			StateSaver: func(_ domain.EnvDir, outputPath string, state json.Marshaler) error {
				savedState = state
				savedOutputPath = outputPath

				return nil
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "-p", "--print=false"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.Equal(t, expectedState, savedState, "state should be saved")
	assert.Empty(t, savedOutputPath, "default output path should be empty")

	// Should print path, not state content (--print=false)
	output := out.String()
	assert.Contains(t, output, "State saved to:")
	assert.NotContains(t, output, `"key"`) // JSON should not be printed
}

// TestGenerate_WithCustomOutputPath verifies custom output path is used.
func TestGenerate_WithCustomOutputPath(t *testing.T) {
	t.Parallel()

	expectedOutputPath := "/custom/path/state.json"
	var savedOutputPath string

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
		Deps: Deps{
			EnvironmentLoader: func(_ context.Context, _ domain.Domain, envKey string, _ ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(_ domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
			StateSaver: func(_ domain.EnvDir, outputPath string, _ json.Marshaler) error {
				savedOutputPath = outputPath

				return nil
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "-p", "-o", expectedOutputPath})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.Equal(t, expectedOutputPath, savedOutputPath, "custom output path should be used")

	// Should print custom path
	output := out.String()
	assert.Contains(t, output, expectedOutputPath)
}

// TestGenerate_EnvironmentLoadError verifies error handling.
func TestGenerate_EnvironmentLoadError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("failed to connect to RPC")

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			t.Fatal("ViewState should not be called on environment load error")

			return json.RawMessage(`{}`), nil
		},
		Deps: Deps{
			EnvironmentLoader: func(_ context.Context, _ domain.Domain, _ string, _ ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{}, expectedError
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "failed to load environment")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestGenerate_ViewStateError verifies error handling.
func TestGenerate_ViewStateError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("contract read failed")

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return nil, expectedError
		},
		Deps: Deps{
			EnvironmentLoader: func(_ context.Context, _ domain.Domain, envKey string, _ ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(_ domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "unable to snapshot state")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestGenerate_StateSaveError verifies error handling.
func TestGenerate_StateSaveError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("permission denied")

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
		Deps: Deps{
			EnvironmentLoader: func(_ context.Context, _ domain.Domain, envKey string, _ ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
				return fdeployment.Environment{Name: envKey}, nil
			},
			StateLoader: func(_ domain.EnvDir) (domain.JSONSerializer, error) {
				return nil, os.ErrNotExist
			},
			StateSaver: func(_ domain.EnvDir, _ string, _ json.Marshaler) error {
				return expectedError
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "-p"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "failed to save state")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestConfig_Validate verifies validation catches missing required fields.
func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("missing all required fields", func(t *testing.T) {
		t.Parallel()

		cfg := Config{}
		err := cfg.Validate()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Logger")
		assert.Contains(t, err.Error(), "Domain")
		assert.Contains(t, err.Error(), "ViewState")
	})

	t.Run("missing ViewState only", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			Logger: logger.Nop(),
			Domain: domain.NewDomain("/tmp", "test"),
		}
		err := cfg.Validate()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ViewState")
		assert.NotContains(t, err.Error(), "Logger")
		assert.NotContains(t, err.Error(), "Domain")
	})

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			Logger: logger.Nop(),
			Domain: domain.NewDomain("/tmp", "test"),
			ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
				return json.RawMessage(`{}`), nil
			},
		}
		err := cfg.Validate()

		require.NoError(t, err)
	})
}

// TestNewCommand_InvalidConfigReturnsError verifies NewCommand returns error for invalid config.
func TestNewCommand_InvalidConfigReturnsError(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger:    logger.Nop(),
		Domain:    domain.NewDomain("/tmp", "testdomain"),
		ViewState: nil, // Missing required field
	})

	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "ViewState")
}

// TestNewCommand_MissingLogger verifies NewCommand returns error for missing logger.
func TestNewCommand_MissingLogger(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: nil, // Missing required field
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return json.RawMessage(`{}`), nil
		},
	})

	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "Logger")
}
