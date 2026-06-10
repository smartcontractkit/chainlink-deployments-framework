package sequenceutils

import (
	"fmt"
	"reflect"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// ExecuteOnChainSequenceAndMerge executes a sequence and merges the output into the given OnChainOutput.
// On error, the accumulated agg is returned unchanged.
//
// Env metadata is a single record per deployment. If both agg and the sequence output set env metadata
// with different values, merge fails with deployment.ErrInvalidConfig rather than silently overwriting.
// Re-merging the same env (shared pointer or equal Metadata value) is allowed.
func ExecuteOnChainSequenceAndMerge[IN any, DEP any](
	b operations.Bundle,
	deps DEP,
	seq *operations.Sequence[IN, OnChainOutput, DEP],
	input IN,
	agg OnChainOutput,
) (OnChainOutput, error) {
	if seq == nil {
		return agg, fmt.Errorf("%w: sequence is required", deployment.ErrInvalidConfig)
	}
	report, err := operations.ExecuteSequence(b, seq, deps, input)
	if err != nil {
		return agg, fmt.Errorf("failed to execute %s: %w", seq.ID(), err)
	}
	if envMetadataConflicts(agg.Metadata.Env, report.Output.Metadata.Env) {
		return agg, fmt.Errorf("%w: conflicting env metadata from sequence %s",
			deployment.ErrInvalidConfig, seq.ID())
	}
	agg.BatchOps = append(agg.BatchOps, report.Output.BatchOps...)
	agg.Metadata.Addresses = append(agg.Metadata.Addresses, report.Output.Metadata.Addresses...)
	agg.Metadata.Contracts = append(agg.Metadata.Contracts, report.Output.Metadata.Contracts...)
	agg.Metadata.Chains = append(agg.Metadata.Chains, report.Output.Metadata.Chains...)
	if report.Output.Metadata.Env != nil {
		agg.Metadata.Env = report.Output.Metadata.Env
	}

	return agg, nil
}

func envMetadataConflicts(existing, incoming *datastore.EnvMetadata) bool {
	if existing == nil || incoming == nil {
		return false
	}
	if existing == incoming {
		return false
	}

	return !reflect.DeepEqual(existing.Metadata, incoming.Metadata)
}
