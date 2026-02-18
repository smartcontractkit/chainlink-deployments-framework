package analyzer

const (
	AnnotationSeverityName = "cld.severity"
	AnnotationSeverityType = "enum"

	AnnotationRiskName = "cld.risk"
	AnnotationRiskType = "enum"
)

// Severity levels for use with AnnotationSeverityName.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
	SeverityDebug   = "debug"
)

// Risk levels for use with AnnotationRiskName.
const (
	RiskHigh   = "high"
	RiskMedium = "medium"
	RiskLow    = "low"
)

// Value type annotation.
// Analyzers set this to describe the semantic type of a decoded parameter value.
// The renderer uses this to decide how to format the value for display.
const (
	AnnotationValueTypeName = "cld.value_type"
	AnnotationValueTypeType = "string"
)

// SeverityAnnotation creates an annotation indicating analysis severity.
func SeverityAnnotation(level string) Annotation {
	return NewAnnotation(AnnotationSeverityName, AnnotationSeverityType, level)
}

// RiskAnnotation creates an annotation indicating risk level.
func RiskAnnotation(level string) Annotation {
	return NewAnnotation(AnnotationRiskName, AnnotationRiskType, level)
}

// ValueTypeAnnotation describes the semantic type of a parameter value.
// The analyzer knows what the raw decoded value represents (e.g., an Ethereum
// address, a token amount) and expresses that as a value type. The renderer
// reads this and decides how to format it for display.
// Examples: "ethereum.address", "ethereum.uint256", "hex", "truncate:20".
func ValueTypeAnnotation(valueType string) Annotation {
	return NewAnnotation(AnnotationValueTypeName, AnnotationValueTypeType, valueType)
}
