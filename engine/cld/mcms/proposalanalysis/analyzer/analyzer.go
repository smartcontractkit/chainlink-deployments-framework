package analyzer

import (
	"context"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
)

// AnalyzerContext provides access to the current stage of analysis within the proposal structure.
// Only some accessor methods are relevant depending on which analyzer type is being executed.
// Analysis proceeds from the most granular components (parameters), up through calls and batch operations, to the full proposal.
// Therefore, context accessors are applicable for the current and higher (ancestor) analysis levels.
// Accessors may return zero values when invoked at stages where corresponding context is not available.
type AnalyzerContext interface {
	// Proposal returns the current proposal-level context.
	// This is primarily meaningful for ProposalAnalyzer execution.
	Proposal() AnalyzedProposal
	// BatchOperation returns the current batch operation context.
	// This is primarily meaningful for ProposalAnalyzer and BatchOperationAnalyzer execution.
	BatchOperation() AnalyzedBatchOperation
	// Call returns the current call-level context.
	// This is primarily meaningful for ProposalAnalyzer, BatchOperationAnalyzer and CallAnalyzer execution.
	Call() AnalyzedCall

	// GetAnnotationsFrom returns annotations from a specific analyzer.
	// This is useful for accessing results from dependency analyzers.
	// Returns empty slice if the analyzer ID is not found or no annotations exist.
	GetAnnotationsFrom(analyzerID string) Annotations
}

type ExecutionContext interface {
	Domain() cldfdomain.Domain
	EnvironmentName() string
	BlockChains() chain.BlockChains
	DataStore() datastore.DataStore
}

// AnalyzeRequest encapsulates the analyzer and execution contexts passed to analyzer methods.
type AnalyzeRequest struct {
	AnalyzerContext  AnalyzerContext
	ExecutionContext ExecutionContext
}

type BaseAnalyzer interface {
	ID() string
	// Dependencies returns the IDs of analyzers that must run before this analyzer.
	//
	// The returned strings MUST correspond to the ID() values of other registered analyzers.
	// Implementations MUST NOT introduce circular dependencies (directly or indirectly).
	//
	// Analyzers may depend only on other analyzers of the same type.
	// For example, a ProposalAnalyzer may only depend on other ProposalAnalyzer instances,
	// a BatchOperationAnalyzer may only depend on other BatchOperationAnalyzer instances,
	// and a CallAnalyzer may only depend on other CallAnalyzer instances.
	// This restriction exists because analyzers of different types are already executed in a fixed dependency order:
	// parameter analyzers run before call analyzers, which run before batch operation analyzers, which in turn run before proposal analyzers.
	Dependencies() []string
}

type ProposalAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest, proposal decoder.DecodedTimelockProposal) bool
	Analyze(ctx context.Context, req AnalyzeRequest, proposal decoder.DecodedTimelockProposal) (Annotations, error)
}

type BatchOperationAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest, operation decoder.DecodedBatchOperation) bool
	Analyze(ctx context.Context, req AnalyzeRequest, operation decoder.DecodedBatchOperation) (Annotations, error)
}

type CallAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest, call decoder.DecodedCall) bool
	Analyze(ctx context.Context, req AnalyzeRequest, call decoder.DecodedCall) (Annotations, error)
}

type ParameterAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest, param decoder.DecodedParameter) bool
	Analyze(ctx context.Context, req AnalyzeRequest, param decoder.DecodedParameter) (Annotations, error)
}
