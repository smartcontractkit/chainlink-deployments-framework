package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func findCmd(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}

	return nil
}

func TestNewEnvCmds_BasicStructure(t *testing.T) {
	t.Parallel()
	var c Commands
	root := c.NewEnvCmds(domain.NewDomain("/tmp", "keystone"))
	require.NotNil(t, root, "NewEnvCmds returned nil")
	require.Equal(t, "env", root.Use)
	require.Equal(t, "Env commands", root.Short)

	f := root.PersistentFlags().Lookup("environment")
	require.NotNil(t, f, "persistent flag \"environment\" not found")

	// top‐level subcommands: load, secrets
	for _, name := range []string{"load"} {
		require.NotNil(t, findCmd(root, name), "subcommand %q not present", name)
	}

	load := findCmd(root, "load")
	require.NotNil(t, load, "load missing")
	require.Empty(t, load.Commands(), "expected load to have no sub-commands")
}

func TestEnvLoad_Command(t *testing.T) {
	t.Parallel()
	domain := domain.NewDomain("/tmp", "keystone")
	cmd := Commands{}.newEnvLoad(domain)
	require.NotNil(t, cmd, "newEnvLoad returned nil")
	require.Equal(t, "load", cmd.Use)
	require.Equal(t, "Runs load environment sanity check", cmd.Short)
	require.Equal(t, envLoadLong, cmd.Long, "LongDesc for load not wired up correctly")
	require.Equal(t, envLoadExample, cmd.Example, "Example for load not wired up correctly")
	require.False(t, cmd.Flags().HasFlags(), "expected no local flags on load command")
}
