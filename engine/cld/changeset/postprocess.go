package changeset

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

type PostProcessor func(e deployment.Environment, config deployment.ChangesetOutput) (deployment.ChangesetOutput, error)

type PostProcessingMigration internalMigration

var _ PostProcessingMigration = PostProcessingMigrationImpl[any]{}

type PostProcessingMigrationImpl[C any] struct {
	migration     MigrationImpl[C]
	postProcessor PostProcessor
}

func (ccs PostProcessingMigrationImpl[C]) noop() {}

func (ccs PostProcessingMigrationImpl[C]) Apply(env deployment.Environment) (deployment.ChangesetOutput, error) {
	env.Logger.Debugf("Post-processing ChangesetOutput from %T", ccs.migration.migration.operation)
	output, err := ccs.migration.Apply(env)
	if err != nil {
		return output, err
	}

	return ccs.postProcessor(env, output)
}

func (ccs PostProcessingMigrationImpl[C]) Configurations() (Configurations, error) {
	return ccs.migration.Configurations()
}
