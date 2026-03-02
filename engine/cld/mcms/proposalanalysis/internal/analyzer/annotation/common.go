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

// SeverityAnnotation creates an annotation indicating analysis severity.
func SeverityAnnotation(level Severity) Annotation {
	return New(AnnotationSeverityName, AnnotationSeverityType, string(level))
}

// RiskAnnotation creates an annotation indicating risk level.
func RiskAnnotation(level Risk) Annotation {
	return New(AnnotationRiskName, AnnotationRiskType, string(level))
}
