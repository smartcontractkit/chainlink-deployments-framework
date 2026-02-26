package changeset

import (
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

type PostProcessor func(e fdeployment.Environment, config fdeployment.ChangesetOutput) (fdeployment.ChangesetOutput, error)

type PostProcessingChangeSet interface {
	internalChangeSet
	WithPreHooks(hooks ...PreHook) PostProcessingChangeSet
	WithPostHooks(hooks ...PostHook) PostProcessingChangeSet
}

var _ PostProcessingChangeSet = PostProcessingChangeSetImpl[any]{}

type PostProcessingChangeSetImpl[C any] struct {
	changeset     ChangeSetImpl[C]
	postProcessor PostProcessor
	preHooks      []PreHook
	postHooks     []PostHook
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

// WithPreHooks appends pre-hooks to this changeset. Multiple calls are additive.
func (ccs PostProcessingChangeSetImpl[C]) WithPreHooks(hooks ...PreHook) PostProcessingChangeSet {
	ccs.preHooks = append(ccs.preHooks, hooks...)
	return ccs
}

// WithPostHooks appends post-hooks to this changeset. Multiple calls are additive.
func (ccs PostProcessingChangeSetImpl[C]) WithPostHooks(hooks ...PostHook) PostProcessingChangeSet {
	ccs.postHooks = append(ccs.postHooks, hooks...)
	return ccs
}

func (ccs PostProcessingChangeSetImpl[C]) getPreHooks() []PreHook   { return ccs.preHooks }
func (ccs PostProcessingChangeSetImpl[C]) getPostHooks() []PostHook { return ccs.postHooks }
