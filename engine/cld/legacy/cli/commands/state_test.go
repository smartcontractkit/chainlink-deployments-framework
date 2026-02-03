package commands

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestNewStateCmds_Structure(t *testing.T) {
	t.Parallel()

	c := NewCommands(logger.Nop())
	dom := domain.NewDomain("/tmp", "foo")
	var cfg StateConfig
	root := c.NewStateCmds(dom, cfg)

	// root command
	require.Equal(t, "state", root.Use)
	require.NotEmpty(t, root.Short)
	require.NotEmpty(t, root.Long, "state command should have Long description")

	// one subcommand: generate
	subs := root.Commands()
	require.Len(t, subs, 1)
	require.Equal(t, "generate", subs[0].Use)

	// NO persistent flags on root (all flags are local to subcommands)
	f := root.PersistentFlags().Lookup("environment")
	require.Nil(t, f, "environment flag should NOT be persistent")

	// environment flag is local to generate subcommand
	genEnvFlag := subs[0].Flags().Lookup("environment")
	require.NotNil(t, genEnvFlag)
	require.Equal(t, "e", genEnvFlag.Shorthand)
}

func TestNewStateGenerateCmd_Metadata(t *testing.T) {
	t.Parallel()

	c := NewCommands(logger.Nop())
	dom := domain.NewDomain("/tmp", "foo")
	var cfg StateConfig
	root := c.NewStateCmds(dom, cfg)

	// Find generate subcommand
	var found bool
	for _, sub := range root.Commands() {
		if sub.Use == "generate" {
			found = true
			require.NotEmpty(t, sub.Short)
			require.NotEmpty(t, sub.Long, "generate should have Long description")
			require.NotEmpty(t, sub.Example, "generate should have Example")

			// local flags
			p := sub.Flags().Lookup("persist")
			require.NotNil(t, p)
			require.Equal(t, "p", p.Shorthand)
			require.Equal(t, "false", p.Value.String())

			// New flag names
			o := sub.Flags().Lookup("out")
			require.NotNil(t, o)
			require.Equal(t, "o", o.Shorthand)
			require.Empty(t, o.Value.String())

			s := sub.Flags().Lookup("prev")
			require.NotNil(t, s)
			require.Equal(t, "s", s.Shorthand)
			require.Empty(t, s.Value.String())

			// Deprecated aliases for backwards compatibility
			oOld := sub.Flags().Lookup("outputPath")
			require.NotNil(t, oOld, "deprecated --outputPath alias should exist")

			sOld := sub.Flags().Lookup("previousState")
			require.NotNil(t, sOld, "deprecated --previousState alias should exist")

			// Print flag
			pr := sub.Flags().Lookup("print")
			require.NotNil(t, pr)
			require.Equal(t, "false", pr.Value.String())

			break
		}
	}
	require.True(t, found, "generate subcommand not found")
}

func TestStateGenerate_MissingEnvFails(t *testing.T) {
	t.Parallel()

	c := NewCommands(logger.Nop())
	dom := domain.NewDomain("/tmp", "foo")
	var cfg StateConfig
	root := c.NewStateCmds(dom, cfg)

	// invoke without the required --environment flag
	root.SetArgs([]string{"generate"})
	err := root.Execute()

	require.ErrorContains(t, err, `required flag(s) "environment" not set`)
}
