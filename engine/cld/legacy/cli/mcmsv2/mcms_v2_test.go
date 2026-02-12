package mcmsv2

import (
	"testing"

	"github.com/stretchr/testify/require"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// TestMCMSv2_DelegatedModularCommands verifies that modular commands are properly
// delegated from the legacy mcmsv2 command.
func TestMCMSv2_DelegatedModularCommands(t *testing.T) {
	t.Parallel()

	lggr := logger.Nop()
	domain := cldf_domain.NewDomain(t.TempDir(), "testdomain")
	proposalCtxProvider := func(_ cldf.Environment) (analyzer.ProposalContext, error) {
		return nil, nil //nolint:nilnil
	}

	cmd := BuildMCMSv2Cmd(lggr, domain, proposalCtxProvider)
	require.NotNil(t, cmd)

	// Verify migrated commands are available as subcommands
	migratedCommands := []string{"analyze-proposal", "convert-upf", "execute-fork", "error-decode-evm"}
	subcommandNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommandNames[sub.Use] = true
	}

	for _, cmdName := range migratedCommands {
		require.True(t, subcommandNames[cmdName], "expected migrated command '%s' to be available as subcommand of mcmsv2", cmdName)
	}

	// Verify each migrated command has expected flags (proving they come from modular package)
	flagTests := []struct {
		subcommand string
		flags      []string
	}{
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
		{
			subcommand: "error-decode-evm",
			flags:      []string{"environment", "error-file"},
		},
	}

	for _, tt := range flagTests {
		t.Run(tt.subcommand, func(t *testing.T) {
			t.Parallel()

			subCmd, _, err := cmd.Find([]string{tt.subcommand})
			require.NoError(t, err)
			require.Equal(t, tt.subcommand, subCmd.Use)

			for _, flagName := range tt.flags {
				flag := subCmd.Flags().Lookup(flagName)
				require.NotNil(t, flag, "expected flag '%s' on delegated command '%s'", flagName, tt.subcommand)
			}
		})
	}
}

func TestBuildMCMSv2Cmd_PanicsOnNilLogger(t *testing.T) {
	t.Parallel()

	domain := cldf_domain.NewDomain(t.TempDir(), "testdomain")
	proposalCtxProvider := func(_ cldf.Environment) (analyzer.ProposalContext, error) {
		return nil, nil //nolint:nilnil
	}

	require.Panics(t, func() {
		BuildMCMSv2Cmd(nil, domain, proposalCtxProvider)
	})
}

func TestBuildMCMSv2Cmd_PanicsOnNilProposalCtxProvider(t *testing.T) {
	t.Parallel()

	lggr := logger.Nop()
	domain := cldf_domain.NewDomain(t.TempDir(), "testdomain")

	require.Panics(t, func() {
		BuildMCMSv2Cmd(lggr, domain, nil)
	})
}
