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

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
	})

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

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
	})

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

			// Output flag (new: --out, deprecated alias: --outputPath)
			o := sub.Flags().Lookup("out")
			require.NotNil(t, o)
			assert.Equal(t, "o", o.Shorthand)
			assert.Empty(t, o.Value.String())

			// Deprecated alias should also exist
			oOld := sub.Flags().Lookup("outputPath")
			require.NotNil(t, oOld, "deprecated --outputPath alias should exist")

			// Previous state flag (new: --prev, deprecated alias: --previousState)
			s := sub.Flags().Lookup("prev")
			require.NotNil(t, s)
			assert.Equal(t, "s", s.Shorthand)
			assert.Empty(t, s.Value.String())

			// Deprecated alias should also exist
			sOld := sub.Flags().Lookup("previousState")
			require.NotNil(t, sOld, "deprecated --previousState alias should exist")

			// Print flag (new)
			pr := sub.Flags().Lookup("print")
			require.NotNil(t, pr)
			assert.Equal(t, "false", pr.Value.String())

			break
		}
	}
	require.True(t, found, "generate subcommand not found")
}

// TestGenerate_MissingEnvironmentFlagFails verifies required flag validation.
func TestGenerate_MissingEnvironmentFlagFails(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return &mockState{Data: map[string]any{}}, nil
		},
	})

	// Execute without --environment flag
	cmd.SetArgs([]string{"generate"})
	err := cmd.Execute()

	require.ErrorContains(t, err, `required flag(s) "environment" not set`)
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

	cmd := NewCommand(Config{
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

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.True(t, environmentLoaderCalled, "environment loader should be called")
	assert.True(t, stateLoaderCalled, "state loader should be called")
	assert.True(t, viewStateCalled, "view state should be called")

	// Without --print flag, state should NOT be in output
	output := out.String()
	assert.NotContains(t, output, "router")
}

// TestGenerate_WithPrint verifies state is printed when --print flag is set.
func TestGenerate_WithPrint(t *testing.T) {
	t.Parallel()

	expectedState := &mockState{
		Data: map[string]any{"key": "value"},
	}

	cmd := NewCommand(Config{
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

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "--print"})

	err := cmd.Execute()

	require.NoError(t, err)
	// With --print flag, state should be in output
	output := out.String()
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
}

// TestGenerate_WithPersist verifies state is saved and path is printed.
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

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "-p"})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, expectedState, savedState, "state should be saved")
	assert.Empty(t, savedOutputPath, "default output path should be empty")

	// Should print path, not state content
	output := out.String()
	assert.Contains(t, output, "State saved to:")
	assert.NotContains(t, output, `"key"`) // JSON should not be printed
}

// TestGenerate_WithCustomOutputPath verifies custom output path is used.
func TestGenerate_WithCustomOutputPath(t *testing.T) {
	t.Parallel()

	expectedOutputPath := "/custom/path/state.json"
	var savedOutputPath string

	cmd := NewCommand(Config{
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

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "-p", "-o", expectedOutputPath})

	err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, expectedOutputPath, savedOutputPath, "custom output path should be used")

	// Should print custom path
	output := out.String()
	assert.Contains(t, output, expectedOutputPath)
}

// TestGenerate_EnvironmentLoadError verifies error handling.
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
		Deps: Deps{
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

// TestGenerate_ViewStateError verifies error handling.
func TestGenerate_ViewStateError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("contract read failed")

	cmd := NewCommand(Config{
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

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to snapshot state")
	assert.Contains(t, err.Error(), expectedError.Error())
}

// TestGenerate_StateSaveError verifies error handling.
func TestGenerate_StateSaveError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("permission denied")

	cmd := NewCommand(Config{
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

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"generate", "-e", "staging", "-p"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save state")
	assert.Contains(t, err.Error(), expectedError.Error())
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

// TestNewCommand_InvalidConfigReturnsError verifies command returns error for invalid config.
func TestNewCommand_InvalidConfigReturnsError(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(Config{
		Logger:    logger.Nop(),
		Domain:    domain.NewDomain("/tmp", "testdomain"),
		ViewState: nil, // Missing required field
	})

	// Command should be created but return error when executed
	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ViewState")
}
