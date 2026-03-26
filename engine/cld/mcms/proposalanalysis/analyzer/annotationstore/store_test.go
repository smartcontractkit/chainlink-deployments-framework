package annotationstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
)

func TestNewScopedDependencyAnnotationStore_DependencyAnnotations(t *testing.T) {
	t.Parallel()

	store := NewScopedDependencyAnnotationStore(
		[]string{"dep-a", "dep-b"},
		map[AnnotationLevel]annotation.Annotations{
			AnnotationLevelProposal: {
				annotation.NewWithAnalyzer("proposal-a", "string", "ok", "dep-a"),
				annotation.NewWithAnalyzer("proposal-self", "string", "skip", "self"),
			},
			AnnotationLevelBatchOperation: {
				annotation.NewWithAnalyzer("batch-b", "string", "ok", "dep-b"),
			},
			AnnotationLevelCall: {
				annotation.NewWithAnalyzer("call-other", "string", "skip", "other"),
			},
			AnnotationLevelParameter: {
				annotation.NewWithAnalyzer("param-a", "string", "ok", "dep-a"),
			},
		},
	)

	anns := store.DependencyAnnotations()
	require.Len(t, anns, 3)
	require.Equal(t, []string{"proposal-a", "batch-b", "param-a"}, []string{
		anns[0].Name(),
		anns[1].Name(),
		anns[2].Name(),
	})
}

func TestScopedDependencyAnnotationStore_Filter(t *testing.T) {
	t.Parallel()

	store := NewScopedDependencyAnnotationStore(
		[]string{"dep-a", "dep-b"},
		map[AnnotationLevel]annotation.Annotations{
			AnnotationLevelProposal: {
				annotation.NewWithAnalyzer("severity", "enum", "high", "dep-a"),
			},
			AnnotationLevelCall: {
				annotation.NewWithAnalyzer("severity", "enum", "medium", "dep-b"),
				annotation.NewWithAnalyzer("reason", "string", "decoded", "dep-b"),
			},
		},
	)

	result := store.Filter(ByLevel(AnnotationLevelCall), ByName("severity"), ByAnalyzer("dep-b"))
	require.Len(t, result, 1)
	require.Equal(t, "medium", result[0].Value())

	result = store.Filter(ByType("string"))
	require.Len(t, result, 1)
	require.Equal(t, "reason", result[0].Name())

	result = store.Filter()
	require.Len(t, result, 3)
}

func TestScopedDependencyAnnotationStore_DependencyAnnotationsReturnsCopy(t *testing.T) {
	t.Parallel()

	store := NewScopedDependencyAnnotationStore(
		[]string{"dep-a"},
		map[AnnotationLevel]annotation.Annotations{
			AnnotationLevelProposal: {
				annotation.NewWithAnalyzer("severity", "enum", "high", "dep-a"),
			},
		},
	)

	anns := store.DependencyAnnotations()
	require.Len(t, anns, 1)

	anns[0] = annotation.NewWithAnalyzer("tampered", "string", "x", "dep-a")

	fresh := store.DependencyAnnotations()
	require.Len(t, fresh, 1)
	require.Equal(t, "severity", fresh[0].Name())
}

func TestNewScopedDependencyAnnotationStore_NoDependencies(t *testing.T) {
	t.Parallel()

	store := NewScopedDependencyAnnotationStore(
		nil,
		map[AnnotationLevel]annotation.Annotations{
			AnnotationLevelProposal: {
				annotation.NewWithAnalyzer("severity", "enum", "high", "dep-a"),
			},
		},
	)

	require.Empty(t, store.DependencyAnnotations())
	require.Empty(t, store.Filter(ByLevel(AnnotationLevelProposal)))
}
