package annotation

// Diff annotation.
// Analyzers use this to express a value change (old -> new).
const (
	AnnotationDiffName = "cld.diff"
	AnnotationDiffType = "diff"
)

type DiffValue struct {
	Field     string
	Old       any
	New       any
	ValueType string
}

// DiffAnnotation creates a structured diff annotation.
func DiffAnnotation(field string, oldVal, newVal any, valueType string) Annotation {
	return New(AnnotationDiffName, AnnotationDiffType, DiffValue{
		Field:     field,
		Old:       oldVal,
		New:       newVal,
		ValueType: valueType,
	})
}
