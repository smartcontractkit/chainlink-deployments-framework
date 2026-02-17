package analyzer

type Annotation interface {
	Name() string
	Type() string
	Value() any
}

type Annotations []Annotation

type Annotated interface {
	// AddAnnotations mutates the underlying analyzed object by appending annotations.
	// Implementations are expected to be used by a single analysis pipeline and are
	// not required to provide internal synchronization for concurrent mutation.
	AddAnnotations(annotations ...Annotation)
	Annotations() Annotations
}
