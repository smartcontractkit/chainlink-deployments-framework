package commands

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	proposalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
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

	cmd, err := cmds.State(dom, StateConfig{
		ViewState: func(_ fdeployment.Environment, _ json.Marshaler) (json.Marshaler, error) {
			return json.RawMessage(`{}`), nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "state", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	envFlag := cmd.PersistentFlags().Lookup("environment")
	assert.Nil(t, envFlag, "environment flag should NOT be persistent")

	subs := cmd.Commands()
	require.Len(t, subs, 1)
	assert.Equal(t, "generate", subs[0].Use)

	genEnvFlag := subs[0].Flags().Lookup("environment")
	require.NotNil(t, genEnvFlag)
	assert.Equal(t, "e", genEnvFlag.Shorthand)
}

func TestCommands_State_MissingViewState(t *testing.T) {
	t.Parallel()

	lggr := logger.Nop()
	cmds := New(lggr)
	dom := domain.NewDomain("/tmp", "testdomain")

	cmd, err := cmds.State(dom, StateConfig{
		ViewState: nil,
	})

	require.Error(t, err)
	assert.Nil(t, cmd)
	require.ErrorContains(t, err, "ViewState")
}

func TestCommands_MCMS_Success(t *testing.T) {
	t.Parallel()

	lggr := logger.Nop()
	cmds := New(lggr)
	dom := domain.NewDomain(t.TempDir(), "testdomain")
	proposalCtxProvider := func(_ fdeployment.Environment) (experimentalanalyzer.ProposalContext, error) {
		return nil, nil //nolint:nilnil
	}

	cmd, err := cmds.MCMS(dom, MCMSConfig{
		ProposalContextProvider: proposalCtxProvider,
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "mcms", cmd.Use)

	// Ensure v2 command is exposed through the commands factory path.
	subCmd, _, findErr := cmd.Find([]string{"analyze-proposal-v2"})
	require.NoError(t, findErr)
	require.NotNil(t, subCmd)
	assert.Equal(t, "analyze-proposal-v2", subCmd.Use)
}

func TestCommands_MCMS_ForwardsProposalAnalyzers(t *testing.T) {
	t.Parallel()

	lggr := logger.Nop()
	cmds := New(lggr)
	dom := domain.NewDomain(t.TempDir(), "testdomain")
	proposalCtxProvider := func(_ fdeployment.Environment) (experimentalanalyzer.ProposalContext, error) {
		return nil, nil //nolint:nilnil
	}

	cmd, err := cmds.MCMS(dom, MCMSConfig{
		ProposalContextProvider: proposalCtxProvider,
		ProposalAnalyzers:       []proposalanalyzer.BaseAnalyzer{nil},
	})

	require.Error(t, err)
	assert.Nil(t, cmd)
	require.ErrorContains(t, err, "ProposalAnalyzers[0] cannot be nil")
}

func TestCommands_MCMS_MissingProposalContextProvider(t *testing.T) {
	t.Parallel()

	lggr := logger.Nop()
	cmds := New(lggr)
	dom := domain.NewDomain(t.TempDir(), "testdomain")

	cmd, err := cmds.MCMS(dom, MCMSConfig{})

	require.Error(t, err)
	assert.Nil(t, cmd)
	require.ErrorContains(t, err, "missing required fields: ProposalContextProvider")
}
