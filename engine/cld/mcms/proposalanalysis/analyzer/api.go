package analyzer

import (
	"math/big"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotationstore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/format"
)

// Annotation types and helpers for analyzers.
type (
	Annotation  = annotation.Annotation
	Annotations = annotation.Annotations
	Severity    = annotation.Severity
	Risk        = annotation.Risk
	DiffValue   = annotation.DiffValue
)

const (
	AnnotationSeverityName = annotation.AnnotationSeverityName
	AnnotationSeverityType = annotation.AnnotationSeverityType
	AnnotationRiskName     = annotation.AnnotationRiskName
	AnnotationRiskType     = annotation.AnnotationRiskType
	AnnotationDiffName     = annotation.AnnotationDiffName
	AnnotationDiffType     = annotation.AnnotationDiffType

	SeverityError   = annotation.SeverityError
	SeverityWarning = annotation.SeverityWarning
	SeverityInfo    = annotation.SeverityInfo
	SeverityDebug   = annotation.SeverityDebug

	RiskHigh   = annotation.RiskHigh
	RiskMedium = annotation.RiskMedium
	RiskLow    = annotation.RiskLow
)

func NewAnnotation(name, atype string, value any) Annotation {
	return annotation.New(name, atype, value)
}

func SeverityAnnotation(level Severity) Annotation {
	return annotation.SeverityAnnotation(level)
}

func RiskAnnotation(level Risk) Annotation {
	return annotation.RiskAnnotation(level)
}

func DiffAnnotation(field string, oldVal, newVal any, valueType string) Annotation {
	return annotation.DiffAnnotation(field, oldVal, newVal, valueType)
}

// Dependency annotation store types and helpers.
type (
	DependencyAnnotationStore     = annotationstore.DependencyAnnotationStore
	DependencyAnnotationPredicate = annotationstore.DependencyAnnotationPredicate
	AnnotationLevel               = annotationstore.AnnotationLevel
)

const (
	AnnotationLevelProposal       = annotationstore.AnnotationLevelProposal
	AnnotationLevelBatchOperation = annotationstore.AnnotationLevelBatchOperation
	AnnotationLevelCall           = annotationstore.AnnotationLevelCall
	AnnotationLevelParameter      = annotationstore.AnnotationLevelParameter
)

func ByAnnotationLevel(level AnnotationLevel) DependencyAnnotationPredicate {
	return annotationstore.ByLevel(level)
}

func ByAnnotationName(name string) DependencyAnnotationPredicate {
	return annotationstore.ByName(name)
}

func ByAnnotationType(atype string) DependencyAnnotationPredicate {
	return annotationstore.ByType(atype)
}

func ByAnnotationAnalyzer(analyzerID string) DependencyAnnotationPredicate {
	return annotationstore.ByAnalyzer(analyzerID)
}

// Decoded proposal types passed to analyzers.
type (
	DecoderConfig           = decoder.Config
	ProposalDecoder         = decoder.ProposalDecoder
	DecodeInstructionFn     = decoder.DecodeInstructionFn
	DecodedTimelockProposal = decoder.DecodedTimelockProposal
	DecodedBatchOperations  = decoder.DecodedBatchOperations
	DecodedBatchOperation   = decoder.DecodedBatchOperation
	DecodedCalls            = decoder.DecodedCalls
	DecodedCall             = decoder.DecodedCall
	DecodedParameters       = decoder.DecodedParameters
	DecodedParameter        = decoder.DecodedParameter
)

// Formatting helpers commonly used by analyzers and renderers.
func CommaGroupBigInt(n *big.Int) string {
	return format.CommaGroupBigInt(n)
}

func FormatTokenAmount(amount *big.Int, decimals uint8) string {
	return format.FormatTokenAmount(amount, decimals)
}
