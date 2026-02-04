package analyzer

import (
	"context"
	"encoding/json"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// ----- annotation -----

type Annotation interface {
	Name() string
	Type() string // TODO: replace with enum
	Value() any
}

type Annotations []Annotation

type Annotated interface {
	AddAnnotations(annotations ...Annotation)
	Annotations() Annotations
}

// ----- decoded -----

type DecodedTimelockProposal interface {
	BatchOperations() DecodedBatchOperations
}

type DecodedBatchOperations []DecodedBatchOperation

type DecodedBatchOperation interface {
	ChainSelector() uint64
	Calls() DecodedCalls
}

type DecodedCalls []DecodedCall

type DecodedCall interface { // DecodedCall or DecodedTransaction?
	To() string   // review: current analyzer uses "Address"
	Name() string // review: current analyzer uses "Method"
	Inputs() DecodedParameters
	Outputs() DecodedParameters
	Data() []byte
	AdditionalFields() json.RawMessage
}

type DecodedParameters []DecodedParameter

type DecodedParameter interface {
	Name() string
	Value() any
}

// ----- analyzed -----

type AnalyzedProposal interface {
	Annotated
	BatchOperations() AnalyzedBatchOperations
}

type AnalyzedBatchOperation interface {
	Annotated
	Calls() AnalyzedCalls
}

type AnalyzedBatchOperations []AnalyzedBatchOperation

type AnalyzedCalls []AnalyzedCall

type AnalyzedCall interface {
	Annotated
	Name() string
	Inputs() AnalyzedParameters
	Outputs() AnalyzedParameters
}

type AnalyzedParameters []AnalyzedParameter

type AnalyzedParameter interface {
	Annotated
	Name() string
	Type() string // reflect.Type?
	Value() any   // reflect.Value?
}

// ----- contexts -----

type AnalyzerContext interface {
	Proposal() AnalyzedProposal
	BatchOperation() AnalyzedBatchOperation
	Call() AnalyzedCall
}

type ExecutionContext interface {
	Domain() cldfdomain.Domain
	EnvironmentName() string
	BlockChains() chain.BlockChains
	DataStore() datastore.DataStore
	// Environment() Environment
}

// ----- analyzers -----

type BaseAnalyzer interface {
	ID() string
	Dependencies() []BaseAnalyzer
}

type ProposalAnalyzer interface {
	BaseAnalyzer
	Analyze(ctx context.Context, actx AnalyzerContext, ectx ExecutionContext, proposal DecodedTimelockProposal) (Annotations, error)
}

type BatchOperationAnalyzer interface {
	BaseAnalyzer
	Analyze(ctx context.Context, actx AnalyzerContext, ectx ExecutionContext, operation DecodedBatchOperation) (Annotations, error)
}

type CallAnalyzer interface {
	BaseAnalyzer
	Analyze(ctx context.Context, actx AnalyzerContext, ectx ExecutionContext, call DecodedCall) (Annotations, error)
}

type ParameterAnalyzer interface {
	BaseAnalyzer
	Analyze(ctx context.Context, actx AnalyzerContext, ectx ExecutionContext, param DecodedParameter) (Annotations, error)
}

// ----- engine -----

type AnalyzerEngine interface {
	Run(ctx context.Context, domain cldfdomain.Domain, environmentName string, proposal *mcms.TimelockProposal) (AnalyzedProposal, error)

	RegisterAnalyzer(analyzer BaseAnalyzer) error // do we need to add a method for each type? like RegisterProposalAnalyzer?

	RegisterFormatter( /* tbd */ ) error
}
