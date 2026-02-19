package annotationstore

import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/annotation"

// DependencyAnnotationStore exposes dependency-scoped annotations for an analyzer.
//
// Implementations MUST enforce that reads are limited to the current analyzer's
// declared Dependencies().
type DependencyAnnotationStore interface {
	// DependencyAnnotations returns all annotations available to the current entity.
	DependencyAnnotations() annotation.Annotations

	// Filter returns dependency annotations matching every provided predicate.
	// Predicates can be composed using:
	//   - ByLevel
	//   - ByName
	//   - ByType
	//   - ByAnalyzer
	Filter(preds ...DependencyAnnotationPredicate) annotation.Annotations
}
