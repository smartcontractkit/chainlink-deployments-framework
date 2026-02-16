package mcms

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// mockProposalContextProvider creates a mock proposal context provider for tests.
func mockProposalContextProvider(_ cldf.Environment) (analyzer.ProposalContext, error) {
	// Return a nil context - this is intentional for testing as we don't need actual proposal context
	return nil, nil //nolint:nilnil
}

// newTestCommand creates a command with test configuration.
func newTestCommand(t *testing.T) (*cobra.Command, error) {
	t.Helper()

	return NewCommand(Config{
		Logger:                  logger.Nop(),
		Domain:                  domain.NewDomain(t.TempDir(), "testdomain"),
		ProposalContextProvider: mockProposalContextProvider,
	})
}

func TestNewCommand_Structure(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Verify parent command
	require.Equal(t, "mcms", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.NotEmpty(t, cmd.Long)

	// Verify subcommands exist
	subcommands := cmd.Commands()
	require.GreaterOrEqual(t, len(subcommands), 4, "expected at least 4 subcommands")

	// Collect subcommand names
	subcommandNames := make(map[string]bool)
	for _, sub := range subcommands {
		subcommandNames[sub.Use] = true
	}

	// Verify expected subcommands
	expectedSubcommands := []string{"error-decode-evm", "analyze-proposal", "convert-upf", "execute-fork"}
	for _, expected := range expectedSubcommands {
		require.True(t, subcommandNames[expected], "expected subcommand '%s' not found", expected)
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      Config
		expectedErr string
	}{
		{
			name:        "missing all required fields",
			config:      Config{},
			expectedErr: "mcms.Config: missing required fields: Logger, Domain, ProposalContextProvider",
		},
		{
			name: "missing Logger only",
			config: Config{
				Domain:                  domain.NewDomain(t.TempDir(), "test"),
				ProposalContextProvider: mockProposalContextProvider,
			},
			expectedErr: "mcms.Config: missing required fields: Logger",
		},
		{
			name: "missing ProposalContextProvider only",
			config: Config{
				Logger: logger.Nop(),
				Domain: domain.NewDomain(t.TempDir(), "test"),
			},
			expectedErr: "mcms.Config: missing required fields: ProposalContextProvider",
		},
		{
			name: "valid config",
			config: Config{
				Logger:                  logger.Nop(),
				Domain:                  domain.NewDomain(t.TempDir(), "test"),
				ProposalContextProvider: mockProposalContextProvider,
			},
			expectedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewCommand_InvalidConfigReturnsError(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{})

	require.EqualError(t, err, "mcms.Config: missing required fields: Logger, Domain, ProposalContextProvider")
	require.Nil(t, cmd)
}

func TestSubcommands_HaveRequiredFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		subcommand string
		flags      []string
	}{
		{
			subcommand: "error-decode-evm",
			flags:      []string{"environment", "error-file"},
		},
		{
			subcommand: "analyze-proposal",
			flags:      []string{"environment", "proposal", "proposalKind", "output", "format"},
		},
		{
			subcommand: "convert-upf",
			flags:      []string{"environment", "proposal", "proposalKind", "output"},
		},
		{
			subcommand: "execute-fork",
			flags:      []string{"environment", "proposal", "proposalKind", "selector", "test-signer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.subcommand, func(t *testing.T) {
			t.Parallel()

			cmd, err := newTestCommand(t)
			require.NoError(t, err)

			subCmd, _, err := cmd.Find([]string{tt.subcommand})
			require.NoError(t, err)
			require.Equal(t, tt.subcommand, subCmd.Use)

			for _, flagName := range tt.flags {
				flag := subCmd.Flags().Lookup(flagName)
				require.NotNil(t, flag, "expected flag '%s' not found in %s", flagName, tt.subcommand)
			}
		})
	}
}

func TestErrorDecode_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	cmd.SetArgs([]string{"error-decode-evm"})
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	execErr := cmd.Execute()
	require.ErrorContains(t, execErr, "required flag")
}

func TestAnalyzeProposal_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	cmd.SetArgs([]string{"analyze-proposal"})
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	execErr := cmd.Execute()
	require.ErrorContains(t, execErr, "required flag")
}

func TestConvertUpf_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	cmd.SetArgs([]string{"convert-upf"})
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	execErr := cmd.Execute()
	require.ErrorContains(t, execErr, "required flag")
}

func TestExecuteFork_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t)
	require.NoError(t, err)

	cmd.SetArgs([]string{"execute-fork"})
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	execErr := cmd.Execute()
	require.ErrorContains(t, execErr, "required flag")
}
