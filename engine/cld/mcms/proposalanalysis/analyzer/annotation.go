package analyzer

type Annotation interface {
	Name() string
	Type() string
	Value() any
	AnalyzerID() string
}

type Annotations []Annotation

// AnnotationPredicate is a function that tests whether an annotation matches
// a given condition.
type AnnotationPredicate func(Annotation) bool

// ByName returns a predicate that matches annotations with the given name.
func ByName(name string) AnnotationPredicate {
	return func(ann Annotation) bool {
		return ann.Name() == name
	}
}

// ByType returns a predicate that matches annotations with the given type.
func ByType(atype string) AnnotationPredicate {
	return func(ann Annotation) bool {
		return ann.Type() == atype
	}
}

// ByAnalyzer returns a predicate that matches annotations produced by the
// given analyzer ID.
func ByAnalyzer(analyzerID string) AnnotationPredicate {
	return func(ann Annotation) bool {
		return ann.AnalyzerID() == analyzerID
	}
}

type Annotated interface {
	// AddAnnotations mutates the underlying analyzed object by appending annotations.
	// Implementations are expected to be used by a single analysis pipeline and are
	// not required to provide internal synchronization for concurrent mutation.
	AddAnnotations(annotations ...Annotation)
	Annotations() Annotations
}
