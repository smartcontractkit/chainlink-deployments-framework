package analyzer

import (
	"context"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

type Annotation interface {
	Name() string
	Value() any
}

type Annotations []Annotation

type Annotated interface {
	AddAnnotation(annotation Annotation)
	Annotations() Annotations
}

type AnalyzedProposal interface {
	Annotated
	Operations() AnalyzedOperations
}

type AnalyzedOperation interface {
	Annotated
	Calls() AnalyzedCalls
}

type AnalyzedOperations []AnalyzedOperation

type AnalyzedCall interface {
	Annotated
	Name() string
	Inputs() AnalyzedParameters
	Outputs() AnalyzedParameters
}

type AnalyzedCalls []AnalyzedCall

type AnalyzedParameter interface {
	Annotated
	Name() string
	Type() string // reflect.Type?
	Value() any   // reflect.Value?
}

type AnalyzedParameters []AnalyzedParameter

type AnalyzerContext interface{}

type ExecutionContext interface{} // domain, environment, etc.

type BaseAnalyzer interface {
	ID() string
	Dependencies() []BaseAnalyzer
}

type ProposalAnalyzer interface {
	BaseAnalyzer
	// can we merge the contexts? can we replace the ExecutionContext with cldf's Environment?
	// should the "TimelockProposal be a "DecodeProposal" instead?
	Analyze(ctx *context.Context, actx AnalyzerContext, ectx ExecutionContext, proposal *mcms.TimelockProposal) (AnalyzedProposal, error)
}

type OperationAnalyzer interface {
	BaseAnalyzer
	Analyze(ctx *context.Context, actx AnalyzerContext, ectx ExecutionContext, operation mcmstypes.Operation) (AnalyzedOperation, error)
}

type CallAnalyzer interface {
	BaseAnalyzer
	Analyze(ctx *context.Context, actx AnalyzerContext, ectx ExecutionContext, call any /*???*/) (AnalyzedOperation, error)
}

type ParameterAnalyzer interface {
	BaseAnalyzer
	Analyze(ctx *context.Context, actx AnalyzerContext, ectx ExecutionContext, param any /*???*/) (AnalyzedParameter, error)
}

type AnalyzerEngine interface {
	Run(ctx *context.Context, actx AnalyzerContext, ectx ExecutionContext, proposal *mcms.TimelockProposal) (AnalyzedProposal, error)
	RegisterAnalyzer(analyzer BaseAnalyzer) error // do we need to add a method for each type? like RegisterProposalAnalyzer?
}
