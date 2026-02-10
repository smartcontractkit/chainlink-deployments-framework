package types

import (
	"context"
	"encoding/json"
	"io"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// ----- annotation -----

type Annotation interface {
	Name() string
	Type() string
	Value() any
}

type Annotations []Annotation

type Annotated interface {
	AddAnnotations(annotations ...Annotation)
	Annotations() Annotations
	GetAnnotationsByName(name string) Annotations
	GetAnnotationsByType(atype string) Annotations
	GetAnnotationsByAnalyzer(analyzerID string) Annotations
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
	ContractType() string
	ContractVersion() string
}

type DecodedParameters []DecodedParameter

type DecodedParameter interface {
	Name() string
	Type() string
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
	ContractType() string
	ContractVersion() string
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

	// GetAnnotationsFrom returns annotations from a specific analyzer at the current context level.
	// For ProposalAnalyzers, this queries the proposal; for CallAnalyzers, the call; etc.
	// This is useful for accessing results from dependency analyzers.
	// Returns empty slice if the analyzer ID is not found or no annotations exist.
	GetAnnotationsFrom(analyzerID string) Annotations
}

type ExecutionContext interface {
	Domain() cldfdomain.Domain
	EnvironmentName() string
	BlockChains() chain.BlockChains
	DataStore() datastore.DataStore
	// Environment() Environment
}

// AnalyzerRequest encapsulates the analyzer and execution contexts passed to analyzer methods.
type AnalyzerRequest struct {
	AnalyzerContext  AnalyzerContext
	ExecutionContext ExecutionContext
}

// ----- analyzers -----

type BaseAnalyzer interface {
	ID() string
	Dependencies() []string // Returns IDs of dependent analyzers
}

type ProposalAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzerRequest, proposal DecodedTimelockProposal) bool
	Analyze(ctx context.Context, req AnalyzerRequest, proposal DecodedTimelockProposal) (Annotations, error)
}

type BatchOperationAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzerRequest, operation DecodedBatchOperation) bool
	Analyze(ctx context.Context, req AnalyzerRequest, operation DecodedBatchOperation) (Annotations, error)
}

type CallAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzerRequest, call DecodedCall) bool
	Analyze(ctx context.Context, req AnalyzerRequest, call DecodedCall) (Annotations, error)
}

type ParameterAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzerRequest, param DecodedParameter) bool
	Analyze(ctx context.Context, req AnalyzerRequest, param DecodedParameter) (Annotations, error)
}

// ----- formatter -----

// FormatterRequest encapsulates the context passed to formatter methods.
type FormatterRequest struct {
	Domain          string
	EnvironmentName string
}

// Formatter transforms an AnalyzedProposal into a specific output format
type Formatter interface {
	ID() string
	Format(ctx context.Context, w io.Writer, req FormatterRequest, proposal AnalyzedProposal) error
}

// ----- engine -----

type DecodeInstructionFn = experimentalanalyzer.DecodeInstructionFn

type AnalyzerEngine interface {
	Run(ctx context.Context, domain cldfdomain.Domain, environmentName string, proposal *mcms.TimelockProposal) (AnalyzedProposal, error)

	RegisterAnalyzer(analyzer BaseAnalyzer) error

	RegisterFormatter(formatter Formatter) error

	RegisterEVMABIMappings(evmABIMappings map[string]string) error

	RegisterSolanaDecoders(solanaDecoders map[string]DecodeInstructionFn) error

	Format(ctx context.Context, w io.Writer, formatterID string, proposal AnalyzedProposal) error
}
