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
	require.Equal(t, "State commands", root.Short)

	// one subcommand: generate
	subs := root.Commands()
	require.Len(t, subs, 1)
	require.Equal(t, "generate", subs[0].Use)

	// persistent 'environment' flag on root
	f := root.PersistentFlags().Lookup("environment")
	require.NotNil(t, f)
	require.Equal(t, "e", f.Shorthand)
}

func TestNewStateGenerateCmd_Metadata(t *testing.T) {
	t.Parallel()

	c := NewCommands(logger.Nop())
	dom := domain.NewDomain("/tmp", "foo")
	var cfg StateConfig
	root := c.NewStateCmds(dom, cfg)

	// Find generate subcommand
	var cmd *struct{}
	for _, sub := range root.Commands() {
		if sub.Use == "generate" {
			require.Contains(t, sub.Short, "Generate latest state")

			// local flags
			p := sub.Flags().Lookup("persist")
			require.NotNil(t, p)
			require.Equal(t, "p", p.Shorthand)
			require.Equal(t, "false", p.Value.String())

			o := sub.Flags().Lookup("outputPath")
			require.NotNil(t, o)
			require.Equal(t, "o", o.Shorthand)
			require.Empty(t, o.Value.String())

			s := sub.Flags().Lookup("previousState")
			require.NotNil(t, s)
			require.Equal(t, "s", s.Shorthand)
			require.Empty(t, s.Value.String())

			return
		}
	}
	if cmd == nil {
		t.Fatal("generate subcommand not found")
	}
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

	require.Error(t, err)
	require.Contains(t, err.Error(), `required flag(s) "environment" not set`)
}
