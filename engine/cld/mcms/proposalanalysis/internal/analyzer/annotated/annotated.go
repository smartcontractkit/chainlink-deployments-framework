package annotated

import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/annotation"

type Annotated interface {
	Annotations() annotation.Annotations
	Filter(preds ...AnnotationPredicate) annotation.Annotations
}

// BaseAnnotated provides a default implementation of the Annotated interface.
type BaseAnnotated struct {
	annotations annotation.Annotations
}

var _ Annotated = &BaseAnnotated{}

func (a *BaseAnnotated) AddAnnotations(annotations ...annotation.Annotation) {
	a.annotations = append(a.annotations, annotations...)
}

func (a *BaseAnnotated) Annotations() annotation.Annotations {
	return a.annotations
}

// Filter returns all annotations matching every provided predicate.
// Predicates can be composed using the ByName, ByType, and ByAnalyzer helpers.
func (a *BaseAnnotated) Filter(preds ...AnnotationPredicate) annotation.Annotations {
	var result annotation.Annotations
	for _, ann := range a.annotations {
		matches := true
		for _, pred := range preds {
			if pred == nil {
				continue
			}
			if !pred(ann) {
				matches = false
				break
			}
		}
		if matches {
			result = append(result, ann)
		}
	}

	return result
}
