package sequenceutils

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	mcms_types "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

func TestExecuteOnChainSequenceAndMerge_success(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{
			BatchOps: []mcms_types.BatchOperation{sampleBatchOp()},
			Metadata: datastore.MetadataBundle{
				Addresses: []datastore.AddressRef{{
					Address:       "0xabc",
					ChainSelector: 1,
					Type:          "Timelock",
					Version:       semver.MustParse("1.0.0"),
				}},
				Chains: []datastore.ChainMetadata{{ChainSelector: 1, Metadata: "chain-a"}},
			},
		}, nil
	})

	agg, err := ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, seq, struct{}{}, OnChainOutput{})
	require.NoError(t, err)
	require.Len(t, agg.BatchOps, 1)
	require.Len(t, agg.Metadata.Addresses, 1)
	require.Len(t, agg.Metadata.Chains, 1)
}

func TestExecuteOnChainSequenceAndMerge_preservesAggOnExecuteFailure(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	seqErr := errors.New("sequence failed")

	okSeq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{
			Metadata: datastore.MetadataBundle{
				Addresses: []datastore.AddressRef{{
					Address:       "0xabc",
					ChainSelector: 1,
					Type:          "Timelock",
					Version:       semver.MustParse("1.0.0"),
				}},
			},
		}, nil
	})
	failSeq := operations.NewSequence(
		"test-seq-fail",
		semver.MustParse("1.0.0"),
		"test sequence fail",
		func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
			return OnChainOutput{}, seqErr
		},
	)

	agg, err := ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, okSeq, struct{}{}, OnChainOutput{})
	require.NoError(t, err)
	require.Len(t, agg.Metadata.Addresses, 1)

	agg, err = ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, failSeq, struct{}{}, agg)
	require.Error(t, err)
	require.ErrorContains(t, err, seqErr.Error())
	require.Len(t, agg.Metadata.Addresses, 1)
}

func TestExecuteOnChainSequenceAndMerge_appendsChainsWithoutDeduping(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{
			Metadata: datastore.MetadataBundle{
				Chains: []datastore.ChainMetadata{{ChainSelector: 1, Metadata: "a"}},
			},
		}, nil
	})

	agg := OnChainOutput{}
	agg, err := ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, seq, struct{}{}, agg)
	require.NoError(t, err)
	agg, err = ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, seq, struct{}{}, agg)
	require.NoError(t, err)
	require.Len(t, agg.Metadata.Chains, 2)
	require.Equal(t, uint64(1), agg.Metadata.Chains[0].ChainSelector)
	require.Equal(t, uint64(1), agg.Metadata.Chains[1].ChainSelector)
}

func TestExecuteOnChainSequenceAndMerge_nilSequence(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	agg := OnChainOutput{Metadata: datastore.MetadataBundle{
		Addresses: []datastore.AddressRef{{Address: "0xabc", ChainSelector: 1, Type: "Timelock", Version: semver.MustParse("1.0.0")}},
	}}

	out, err := ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, nil, struct{}{}, agg)
	require.Error(t, err)
	require.ErrorIs(t, err, deployment.ErrInvalidConfig)
	require.ErrorContains(t, err, "sequence is required")
	require.Equal(t, agg, out)
}

func TestExecuteOnChainSequenceAndMerge_sameEnvPointer(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	envMeta := &datastore.EnvMetadata{Metadata: "shared"}
	seq1 := operations.NewSequence(
		"test-seq-env-shared-1",
		semver.MustParse("1.0.0"),
		"test sequence shared env 1",
		func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
			return OnChainOutput{Metadata: datastore.MetadataBundle{Env: envMeta}}, nil
		},
	)
	seq2 := operations.NewSequence(
		"test-seq-env-shared-2",
		semver.MustParse("1.0.0"),
		"test sequence shared env 2",
		func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
			return OnChainOutput{Metadata: datastore.MetadataBundle{Env: envMeta}}, nil
		},
	)

	agg, err := ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, seq1, struct{}{}, OnChainOutput{})
	require.NoError(t, err)
	require.Same(t, envMeta, agg.Metadata.Env)

	agg, err = ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, seq2, struct{}{}, agg)
	require.NoError(t, err)
	require.Same(t, envMeta, agg.Metadata.Env)
}

func TestExecuteOnChainSequenceAndMerge_equivalentEnvValues(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	seq1 := operations.NewSequence(
		"test-seq-env-equiv-1",
		semver.MustParse("1.0.0"),
		"test sequence equivalent env 1",
		func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
			return OnChainOutput{Metadata: datastore.MetadataBundle{
				Env: &datastore.EnvMetadata{Metadata: "shared"},
			}}, nil
		},
	)
	seq2 := operations.NewSequence(
		"test-seq-env-equiv-2",
		semver.MustParse("1.0.0"),
		"test sequence equivalent env 2",
		func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
			return OnChainOutput{Metadata: datastore.MetadataBundle{
				Env: &datastore.EnvMetadata{Metadata: "shared"},
			}}, nil
		},
	)

	agg, err := ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, seq1, struct{}{}, OnChainOutput{})
	require.NoError(t, err)
	require.Equal(t, "shared", agg.Metadata.Env.Metadata)

	agg, err = ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, seq2, struct{}{}, agg)
	require.NoError(t, err)
	require.Equal(t, "shared", agg.Metadata.Env.Metadata)
}

func TestExecuteOnChainSequenceAndMerge_envConflict(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	envMeta := &datastore.EnvMetadata{Metadata: "staging"}
	seq := operations.NewSequence(
		"test-seq-env-conflict",
		semver.MustParse("1.0.0"),
		"test sequence env conflict",
		func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
			return OnChainOutput{
				BatchOps: []mcms_types.BatchOperation{sampleBatchOp()},
				Metadata: datastore.MetadataBundle{
					Env: envMeta,
					Addresses: []datastore.AddressRef{{
						Address:       "0xdef",
						ChainSelector: 2,
						Type:          "Timelock",
						Version:       semver.MustParse("1.0.0"),
					}},
				},
			}, nil
		},
	)

	agg := OnChainOutput{Metadata: datastore.MetadataBundle{Env: &datastore.EnvMetadata{Metadata: "prod"}}}
	agg, err := ExecuteOnChainSequenceAndMerge(env.OperationsBundle, struct{}{}, seq, struct{}{}, agg)
	require.Error(t, err)
	require.ErrorIs(t, err, deployment.ErrInvalidConfig)
	require.ErrorContains(t, err, "conflicting env metadata")
	require.Equal(t, "prod", agg.Metadata.Env.Metadata)
	require.Empty(t, agg.BatchOps)
	require.Empty(t, agg.Metadata.Addresses)
}
