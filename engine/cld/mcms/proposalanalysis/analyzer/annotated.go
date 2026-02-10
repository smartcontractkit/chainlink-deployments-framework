package analyzer

import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"

var _ types.Annotation = &annotation{}

type annotation struct {
	name       string
	atype      string
	value      any
	analyzerID string
}

func (a annotation) Name() string {
	return a.name
}

func (a annotation) Type() string {
	return a.atype
}

func (a annotation) Value() any {
	return a.value
}

// NewAnnotation creates a new annotation with the given name, type, and value
func NewAnnotation(name, atype string, value any) types.Annotation {
	return &annotation{
		name:  name,
		atype: atype,
		value: value,
	}
}

// NewAnnotationWithAnalyzer creates a new annotation with analyzer ID tracking
func NewAnnotationWithAnalyzer(name, atype string, value any, analyzerID string) types.Annotation {
	return &annotation{
		name:       name,
		atype:      atype,
		value:      value,
		analyzerID: analyzerID,
	}
}

// ---------------------------------------------------------------------

var _ types.Annotated = &annotated{}

type annotated struct {
	annotations types.Annotations
}

func (a *annotated) AddAnnotations(annotations ...types.Annotation) {
	a.annotations = append(a.annotations, annotations...)
}

func (a annotated) Annotations() types.Annotations {
	return a.annotations
}

// GetAnnotationsByName returns all annotations with the given name
func (a annotated) GetAnnotationsByName(name string) types.Annotations {
	var result types.Annotations
	for _, ann := range a.annotations {
		if ann.Name() == name {
			result = append(result, ann)
		}
	}
	return result
}

// GetAnnotationsByType returns all annotations with the given type
func (a annotated) GetAnnotationsByType(atype string) types.Annotations {
	var result types.Annotations
	for _, ann := range a.annotations {
		if ann.Type() == atype {
			result = append(result, ann)
		}
	}
	return result
}

// GetAnnotationsByAnalyzer returns all annotations created by the given analyzer ID
func (a annotated) GetAnnotationsByAnalyzer(analyzerID string) types.Annotations {
	var result types.Annotations
	for _, ann := range a.annotations {
		// Try to cast to our internal annotation type to access analyzerID
		if internalAnn, ok := ann.(*annotation); ok {
			if internalAnn.analyzerID == analyzerID {
				result = append(result, ann)
			}
		}
	}
	return result
}
