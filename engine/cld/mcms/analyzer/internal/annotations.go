package internal

import (
	"slices"

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

func NewAnnotation(name, atype string, value any) annotation {
	return annotation{name: name, atype: atype, value: value}
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

// ----- shared global annotation -----
// consider moving to a separate "annotations" package and removing "Annotation" prefixes
const (
	AnnotationSeverityName = "cld.severity" // review: core.severity? common.severity? cld:severity?
	AnnotationSeverityType = "enum"         // string? reflect.Type?

	AnnotationRiskName = "cld.risk"
	AnnotationRiskType = "enum"
)

var (
	AnnotationValidSeverities = []string{"unknown", "debug", "info", "warning", "error"} // review: should we be more strict and implement proper enum types?
	AnnotationValidRisks      = []string{"unknown", "low", "medium", "high"}             // review: should we be more strict and implement proper enum types?
)

func SeverityAnnotation(value string) annotation {
	if !slices.Contains(AnnotationValidSeverities, value) {
		value = "unknown"
	}

	return NewAnnotation(AnnotationSeverityName, AnnotationSeverityType, value)
}

func RiskAnnotation(value string) annotation {
	if !slices.Contains(AnnotationValidRisks, value) {
		value = "unknown"
	}

	return NewAnnotation(AnnotationRiskName, AnnotationRiskType, value)
}
