package changeset

import (
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

type PostProcessor func(e fdeployment.Environment, config fdeployment.ChangesetOutput) (fdeployment.ChangesetOutput, error)

type PostProcessingChangeSet internalChangeSet

var _ PostProcessingChangeSet = PostProcessingChangeSetImpl[any]{}

type PostProcessingChangeSetImpl[C any] struct {
	changeset     ChangeSetImpl[C]
	postProcessor PostProcessor
}

func (ccs PostProcessingChangeSetImpl[C]) noop() {}

func (ccs PostProcessingChangeSetImpl[C]) Apply(env fdeployment.Environment) (fdeployment.ChangesetOutput, error) {
	env.Logger.Debugf("Post-processing ChangesetOutput from %T", ccs.changeset.changeset.operation)
	output, err := ccs.changeset.Apply(env)
	if err != nil {
		return output, err
	}

	return ccs.postProcessor(env, output)
}

func (ccs PostProcessingChangeSetImpl[C]) Configurations() (Configurations, error) {
	return ccs.changeset.Configurations()
}
