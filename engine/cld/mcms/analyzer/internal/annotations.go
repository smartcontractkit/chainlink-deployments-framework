package internal

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

// TODO: consider converting Annotation into simple type
var _ analyzer.Annotation = &annotation{}

type annotation struct {
	name  string
	atype string
	value any
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

// ---------------------------------------------------------------------

var _ analyzer.Annotated = &annotated{}

type annotated struct {
	annotations analyzer.Annotations
}

func (a *annotated) AddAnnotations(annotations ...analyzer.Annotation) {
	a.annotations = append(a.annotations, annotations...)
}

func (a annotated) Annotations() analyzer.Annotations {
	return a.annotations
}
