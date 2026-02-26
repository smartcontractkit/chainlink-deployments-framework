package annotation

const (
	AnnotationSeverityName = "cld.severity"
	AnnotationSeverityType = "enum"

	AnnotationRiskName = "cld.risk"
	AnnotationRiskType = "enum"
)

// Severity represents the severity level of an analysis finding.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
	SeverityDebug   Severity = "debug"
)

// Risk represents the risk level of an analyzed operation.
type Risk string

const (
	RiskHigh   Risk = "high"
	RiskMedium Risk = "medium"
	RiskLow    Risk = "low"
)

// Value type annotation.
// Analyzers set this to describe the semantic type of a decoded parameter value.
// The renderer uses this to decide how to format the value for display.
const (
	AnnotationValueTypeName = "cld.value_type"
	AnnotationValueTypeType = "string"
)

// SeverityAnnotation creates an annotation indicating analysis severity.
func SeverityAnnotation(level Severity) Annotation {
	return New(AnnotationSeverityName, AnnotationSeverityType, string(level))
}

// RiskAnnotation creates an annotation indicating risk level.
func RiskAnnotation(level Risk) Annotation {
	return New(AnnotationRiskName, AnnotationRiskType, string(level))
}

// ValueTypeAnnotation describes the semantic type of a parameter value.
// The analyzer knows what the raw decoded value represents (e.g., an Ethereum
// address, a token amount) and expresses that as a value type. The renderer
// reads this and decides how to format it for display.
// Examples: "ethereum.address", "ethereum.uint256", "hex", "truncate:20".
func ValueTypeAnnotation(valueType string) Annotation {
	return New(AnnotationValueTypeName, AnnotationValueTypeType, valueType)
}
