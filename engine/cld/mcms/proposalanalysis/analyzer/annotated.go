package analyzer

type annotation struct {
	name       string
	atype      string
	value      any
	analyzerID string
}

var _ Annotation = &annotation{}

func (a *annotation) Name() string { return a.name }
func (a *annotation) Type() string { return a.atype }
func (a *annotation) Value() any   { return a.value }

func NewAnnotation(name, atype string, value any) Annotation {
	return &annotation{
		name:  name,
		atype: atype,
		value: value,
	}
}

// NewAnnotationWithAnalyzer creates a new annotation with analyzer ID
// This is used internally by the engine to associate annotations with the
// analyzer that produced them
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

func (a *BaseAnnotated) GetAnnotationsByName(name string) Annotations {
	var result Annotations
	for _, ann := range a.annotations {
		if ann.Name() == name {
			result = append(result, ann)
		}
	}

	return result
}

func (a *BaseAnnotated) GetAnnotationsByType(atype string) Annotations {
	var result Annotations
	for _, ann := range a.annotations {
		if ann.Type() == atype {
			result = append(result, ann)
		}
	}

	return result
}

// GetAnnotationsByAnalyzer returns all annotations produced by the given analyzer ID.
func (a *BaseAnnotated) GetAnnotationsByAnalyzer(analyzerID string) Annotations {
	var result Annotations
	for _, ann := range a.annotations {
		if tracked, ok := ann.(*annotation); ok && tracked.analyzerID == analyzerID {
			result = append(result, ann)
		}
	}

	return result
}
