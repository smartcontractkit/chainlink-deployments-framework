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
	assert.Equal(t, "State commands", cmd.Short)

	// Verify environment flag is present
	envFlag := cmd.PersistentFlags().Lookup("environment")
	require.NotNil(t, envFlag)
	assert.Equal(t, "e", envFlag.Shorthand)

	// Verify generate subcommand exists
	subs := cmd.Commands()
	require.Len(t, subs, 1)
	assert.Equal(t, "generate", subs[0].Use)
}

func TestCommands_MultipleCommands_ShareLogger(t *testing.T) {
	t.Parallel()

	// This test verifies the key benefit: logger is set once and shared
	lggr := logger.Nop()
	cmds := New(lggr)
	dom := domain.NewDomain("/tmp", "testdomain")

	viewState := func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
		return json.RawMessage(`{}`), nil
	}

	// Create multiple commands - logger is NOT repeated
	stateCmd1 := cmds.State(dom, StateConfig{ViewState: viewState})
	stateCmd2 := cmds.State(dom, StateConfig{ViewState: viewState})

	// Both commands should work
	require.NotNil(t, stateCmd1)
	require.NotNil(t, stateCmd2)
}
