package analyzer

type Annotation interface {
	Name() string
	Type() string
	Value() any
}

type Annotations []Annotation

type Annotated interface {
	AddAnnotations(annotations ...Annotation)
	Annotations() Annotations
}
