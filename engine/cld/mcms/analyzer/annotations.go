package analyzer

type annotated struct {
	annotations Annotations
}

func (a *annotated) AddAnnotations(annotations ...Annotation) {
	a.annotations = append(a.annotations, annotations...)
}

func (a annotated) Annotations() Annotations {
	return a.annotations
}

func (a annotated) GetAnnotationsByName(name string) Annotations {
	var result Annotations
	for _, ann := range a.annotations {
		if ann.Name() == name {
			result = append(result, ann)
		}
	}

	return result
}

func (a annotated) GetAnnotationsByType(atype string) Annotations {
	var result Annotations
	for _, ann := range a.annotations {
		if ann.Type() == atype {
			result = append(result, ann)
		}
	}

	return result
}

func (a annotated) GetAnnotationsByAnalyzer(analyzerID string) Annotations {
	var result Annotations
	for _, ann := range a.annotations {
		if ann.AnalyzerID() == analyzerID {
			result = append(result, ann)
		}
	}

	return result
}

func NewAnnotation(annotationType, name string, value any) Annotation {
	return &simpleAnnotation{annotationType: annotationType, name: name, value: value}
}

func NewAnnotationWithAnalyzer(annotationType, name string, value any, analyzerID string) Annotation {
	return &simpleAnnotation{
		annotationType: annotationType,
		name:           name,
		value:          value,
		analyzerID:     analyzerID,
	}
}

type simpleAnnotation struct {
	annotationType string
	name           string
	value          any
	analyzerID     string
}

func (a *simpleAnnotation) Type() string       { return a.annotationType }
func (a *simpleAnnotation) Name() string       { return a.name }
func (a *simpleAnnotation) Value() any         { return a.value }
func (a *simpleAnnotation) AnalyzerID() string { return a.analyzerID }
