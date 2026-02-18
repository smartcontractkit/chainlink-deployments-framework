package analyzer

type annotation struct {
	name       string
	atype      string
	value      any
	analyzerID string
}

var _ Annotation = &annotation{}

func (a *annotation) Name() string       { return a.name }
func (a *annotation) Type() string       { return a.atype }
func (a *annotation) Value() any         { return a.value }
func (a *annotation) AnalyzerID() string { return a.analyzerID }

// NewAnnotation creates a new annotation with the given name, type, and value.
func NewAnnotation(name, atype string, value any) Annotation {
	return &annotation{
		name:  name,
		atype: atype,
		value: value,
	}
}

// NewAnnotationWithAnalyzer creates a new annotation with analyzer ID.
// This is used internally by the engine to associate annotations with the
// analyzer that produced them.
func NewAnnotationWithAnalyzer(name, atype string, value any, analyzerID string) Annotation {
	return &annotation{
		name:       name,
		atype:      atype,
		value:      value,
		analyzerID: analyzerID,
	}
}

// BaseAnnotated provides a default implementation of the Annotated interface.
type BaseAnnotated struct {
	annotations Annotations
}

var _ Annotated = &BaseAnnotated{}

func (a *BaseAnnotated) AddAnnotations(annotations ...Annotation) {
	a.annotations = append(a.annotations, annotations...)
}

func (a *BaseAnnotated) Annotations() Annotations {
	return a.annotations
}

// Filter returns all annotations matching the given predicate.
// Predicates can be composed using the ByName, ByType, and ByAnalyzer helpers.
func (a *BaseAnnotated) Filter(pred AnnotationPredicate) Annotations {
	var result Annotations
	for _, ann := range a.annotations {
		if pred(ann) {
			result = append(result, ann)
		}
	}

	return result
}

// GetAnnotationsByName returns all annotations matching the given name.
func (a *BaseAnnotated) GetAnnotationsByName(name string) Annotations {
	return a.Filter(ByName(name))
}

// GetAnnotationsByType returns all annotations matching the given type.
func (a *BaseAnnotated) GetAnnotationsByType(atype string) Annotations {
	return a.Filter(ByType(atype))
}

// GetAnnotationsByAnalyzer returns all annotations produced by the given analyzer ID.
func (a *BaseAnnotated) GetAnnotationsByAnalyzer(analyzerID string) Annotations {
	return a.Filter(ByAnalyzer(analyzerID))
}
