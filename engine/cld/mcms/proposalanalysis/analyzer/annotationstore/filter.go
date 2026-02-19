package annotationstore

import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"

// AnnotationLevel identifies which proposal-analysis level to read annotations from.
type AnnotationLevel string

const (
	// AnnotationLevelProposal reads proposal-level annotations.
	AnnotationLevelProposal AnnotationLevel = "proposal"
	// AnnotationLevelBatchOperation reads batch-operation-level annotations.
	AnnotationLevelBatchOperation AnnotationLevel = "batch_operation"
	// AnnotationLevelCall reads call-level annotations.
	AnnotationLevelCall AnnotationLevel = "call"
	// AnnotationLevelParameter reads parameter-level annotations.
	AnnotationLevelParameter AnnotationLevel = "parameter"
)

// DependencyAnnotationPredicate is a function that tests whether a dependency
// annotation matches a condition in the context of its analysis level.
type DependencyAnnotationPredicate func(level AnnotationLevel, ann annotation.Annotation) bool

// ByLevel returns a predicate that matches annotations at the given level.
func ByLevel(level AnnotationLevel) DependencyAnnotationPredicate {
	return func(currentLevel AnnotationLevel, _ annotation.Annotation) bool {
		return currentLevel == level
	}
}

// ByName returns a predicate that matches annotations with the given name.
func ByName(name string) DependencyAnnotationPredicate {
	return func(_ AnnotationLevel, ann annotation.Annotation) bool {
		return ann.Name() == name
	}
}

// ByType returns a predicate that matches annotations with the given type.
func ByType(atype string) DependencyAnnotationPredicate {
	return func(_ AnnotationLevel, ann annotation.Annotation) bool {
		return ann.Type() == atype
	}
}

// ByAnalyzer returns a predicate that matches annotations produced by
// the given analyzer ID.
func ByAnalyzer(analyzerID string) DependencyAnnotationPredicate {
	return func(_ AnnotationLevel, ann annotation.Annotation) bool {
		return ann.AnalyzerID() == analyzerID
	}
}
