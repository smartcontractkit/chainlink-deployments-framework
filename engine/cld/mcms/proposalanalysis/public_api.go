package proposalanalysis

import (
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/decoder"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/renderer"
)

// Analyzer interface aliases exposed at the package root for simpler imports.
type (
	BaseAnalyzer           = analyzer.BaseAnalyzer
	AnalyzeRequest[T any]  = analyzer.AnalyzeRequest[T]
	ProposalAnalyzeRequest = analyzer.ProposalAnalyzeRequest
	AnalyzedProposal       = analyzer.AnalyzedProposal

	ProposalAnalyzer       = analyzer.ProposalAnalyzer
	BatchOperationAnalyzer = analyzer.BatchOperationAnalyzer
	CallAnalyzer           = analyzer.CallAnalyzer
	ParameterAnalyzer      = analyzer.ParameterAnalyzer

	BatchOperationAnalyzerContext = analyzer.BatchOperationAnalyzerContext
	CallAnalyzerContext           = analyzer.CallAnalyzerContext
	ParameterAnalyzerContext      = analyzer.ParameterAnalyzerContext
)

// Decoder model aliases exposed at the package root for analyzer signatures.
type (
	DecoderConfig       = decoder.Config
	DecodeInstructionFn = decoder.DecodeInstructionFn

	DecodedTimelockProposal = decoder.DecodedTimelockProposal
	DecodedBatchOperations  = decoder.DecodedBatchOperations
	DecodedBatchOperation   = decoder.DecodedBatchOperation
	DecodedCalls            = decoder.DecodedCalls
	DecodedCall             = decoder.DecodedCall
	DecodedParameters       = decoder.DecodedParameters
	DecodedParameter        = decoder.DecodedParameter
)

// Annotation aliases exposed at the package root for analyzer outputs.
type (
	Annotation  = annotation.Annotation
	Annotations = annotation.Annotations
)

// NewAnnotation creates a new analyzer annotation.
func NewAnnotation(name, atype string, value any) Annotation {
	return annotation.New(name, atype, value)
}

// Renderer aliases exposed at the package root for simpler imports.
type (
	Renderer       = renderer.Renderer
	RenderRequest  = renderer.RenderRequest
	RendererOption = renderer.Option
)

// RendererIDMarkdown is the markdown renderer identifier.
const RendererIDMarkdown = renderer.IDMarkdown

// NewMarkdownRenderer creates a markdown renderer.
func NewMarkdownRenderer(opts ...RendererOption) (Renderer, error) {
	return renderer.NewMarkdownRenderer(opts...)
}

// WithRendererTemplateDir configures template loading from a filesystem directory.
func WithRendererTemplateDir(dir string) RendererOption {
	return renderer.WithTemplateDir(dir)
}

// WithRendererTemplates configures in-memory template overrides.
func WithRendererTemplates(templates map[string]string) RendererOption {
	return renderer.WithTemplates(templates)
}

// WithRendererTemplateFuncs configures extra template functions.
func WithRendererTemplateFuncs(funcs template.FuncMap) RendererOption {
	return renderer.WithTemplateFuncs(funcs)
}
