package changeset

import (
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

var MyChangeSet deployment.ChangeSetV2[string] = MyChangeSetImpl{}

type MyChangeSetImpl struct{}

func (m MyChangeSetImpl) Apply(_ deployment.Environment, _ string) (deployment.ChangesetOutput, error) {
	return deployment.ChangesetOutput{}, nil
}

func (m MyChangeSetImpl) VerifyPreconditions(_ deployment.Environment, _ string) error { return nil }

func TestChangesets_PostProcess(t *testing.T) {
	t.Parallel()

	env := deployment.Environment{
		Logger: logger.Test(t),
	}
	var executed = false
	configured := Configure(MyChangeSet).
		With("MyString").
		ThenWith(func(e deployment.Environment, o deployment.ChangesetOutput) (deployment.ChangesetOutput, error) {
			executed = true
			return o, nil
		})
	if executed {
		t.Errorf("Post process function should not yet have been called.")
	}
	_, err := configured.Apply(env)
	require.NoError(t, err, "Apply should not return an error")
	require.True(t, executed, "Post process function should have been called.")

	configs, err := configured.Configurations()
	require.NoError(t, err)
	assert.Nil(t, configs.InputChainOverrides)
}

