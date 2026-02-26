package annotated

import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"

// AnnotationPredicate is a function that tests whether an annotation matches
// a given condition.
type AnnotationPredicate func(annotation.Annotation) bool

// ByName returns a predicate that matches annotations with the given name.
func ByName(name string) AnnotationPredicate {
	return func(ann annotation.Annotation) bool {
		return ann.Name() == name
	}
}

// ByType returns a predicate that matches annotations with the given type.
func ByType(atype string) AnnotationPredicate {
	return func(ann annotation.Annotation) bool {
		return ann.Type() == atype
	}
}

// ByAnalyzer returns a predicate that matches annotations produced by the
// given analyzer ID.
func ByAnalyzer(analyzerID string) AnnotationPredicate {
	return func(ann annotation.Annotation) bool {
		return ann.AnalyzerID() == analyzerID
	}
}
