package commands

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func mockViewState(_ deployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
	return json.RawMessage(`{}`), nil
}

func TestNewStateCmds_Structure(t *testing.T) {
	t.Parallel()

	c := NewCommands(logger.Nop())
	dom := domain.NewDomain(t.TempDir(), "foo")
	cfg := StateConfig{ViewState: mockViewState}
	root := c.NewStateCmds(dom, cfg)

	require.Equal(t, "state", root.Use)
	require.NotEmpty(t, root.Short)
	require.NotEmpty(t, root.Long)

	subs := root.Commands()
	require.Len(t, subs, 1)
	require.Equal(t, "generate", subs[0].Use)

	f := root.PersistentFlags().Lookup("environment")
	require.Nil(t, f, "environment flag should NOT be persistent")

	genEnvFlag := subs[0].Flags().Lookup("environment")
	require.NotNil(t, genEnvFlag)
	require.Equal(t, "e", genEnvFlag.Shorthand)
}

func TestNewStateGenerateCmd_Metadata(t *testing.T) {
	t.Parallel()

	c := NewCommands(logger.Nop())
	dom := domain.NewDomain(t.TempDir(), "foo")
	cfg := StateConfig{ViewState: mockViewState}
	root := c.NewStateCmds(dom, cfg)

	var found bool
	for _, sub := range root.Commands() {
		if sub.Use == "generate" {
			found = true
			require.NotEmpty(t, sub.Short)
			require.NotEmpty(t, sub.Long)
			require.NotEmpty(t, sub.Example)

			p := sub.Flags().Lookup("persist")
			require.NotNil(t, p)
			require.Equal(t, "p", p.Shorthand)

			o := sub.Flags().Lookup("out")
			require.NotNil(t, o)
			require.Equal(t, "o", o.Shorthand)

			s := sub.Flags().Lookup("prev")
			require.NotNil(t, s)
			require.Equal(t, "s", s.Shorthand)

			pr := sub.Flags().Lookup("print")
			require.NotNil(t, pr)

			break
		}
	}
	require.True(t, found, "generate subcommand not found")
}

func TestStateGenerate_MissingEnvFails(t *testing.T) {
	t.Parallel()

	c := NewCommands(logger.Nop())
	dom := domain.NewDomain(t.TempDir(), "foo")
	cfg := StateConfig{ViewState: mockViewState}
	root := c.NewStateCmds(dom, cfg)

	root.SetArgs([]string{"generate"})
	err := root.Execute()

	require.ErrorContains(t, err, `required flag(s) "environment" not set`)
}

func TestNewStateCmds_InvalidConfigPanics(t *testing.T) {
	t.Parallel()

	c := NewCommands(logger.Nop())
	dom := domain.NewDomain(t.TempDir(), "foo")
	cfg := StateConfig{ViewState: nil} // Missing required field

	// Configuration errors (nil ViewState) cause a panic
	require.Panics(t, func() {
		c.NewStateCmds(dom, cfg)
	})
}
