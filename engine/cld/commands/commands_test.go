package commands

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestNew(t *testing.T) {
	t.Parallel()

	lggr := logger.Nop()
	cmds := New(lggr)

	require.NotNil(t, cmds)
	assert.Equal(t, lggr, cmds.lggr)
}

func TestCommands_State(t *testing.T) {
	t.Parallel()

	lggr := logger.Nop()
	cmds := New(lggr)
	dom := domain.NewDomain("/tmp", "testdomain")

	cmd := cmds.State(dom, StateConfig{
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return json.RawMessage(`{}`), nil
		},
	})

	require.NotNil(t, cmd)
	assert.Equal(t, "state", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long, "state command should have Long description")

	// Verify NO persistent flags on parent (all flags are local to subcommands)
	envFlag := cmd.PersistentFlags().Lookup("environment")
	assert.Nil(t, envFlag, "environment flag should NOT be persistent")

	// Verify generate subcommand exists with local flags
	subs := cmd.Commands()
	require.Len(t, subs, 1)
	assert.Equal(t, "generate", subs[0].Use)

	// Environment flag should be on generate subcommand (local)
	genEnvFlag := subs[0].Flags().Lookup("environment")
	require.NotNil(t, genEnvFlag)
	assert.Equal(t, "e", genEnvFlag.Shorthand)
}
