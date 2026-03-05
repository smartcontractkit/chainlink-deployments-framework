package analyzer

import (
	"context"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// ExecutionContext encapsulates the execution context passed to an analyzer.
type ExecutionContext interface {
	Domain() cldfdomain.Domain
	EnvironmentName() string
	BlockChains() chain.BlockChains
	DataStore() datastore.DataStore
}

// AnalyzeRequest encapsulates the analyzer context, execution context, and annotation store passed to analyzer.
type AnalyzeRequest[T any] struct {
	AnalyzerContext           T
	ExecutionContext          ExecutionContext
	DependencyAnnotationStore DependencyAnnotationStore
}

// ProposalAnalyzeRequest encapsulates the execution context and annotation store passed to a proposal analyzer.
type ProposalAnalyzeRequest struct {
	ExecutionContext          ExecutionContext
	DependencyAnnotationStore DependencyAnnotationStore
}

type BaseAnalyzer interface {
	ID() string
	// Dependencies returns the IDs of analyzers that must run before this analyzer.
	//
	// The returned strings MUST correspond to the ID() values of other registered analyzers.
	// Implementations MUST NOT introduce circular dependencies (directly or indirectly).
	// The engine uses this list to:
	//   - schedule analyzers in dependency order
	//   - scope AnnotationStore reads to only these dependency IDs
	Dependencies() []string
}

type ProposalAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req ProposalAnalyzeRequest, proposal DecodedTimelockProposal) bool
	Analyze(ctx context.Context, req ProposalAnalyzeRequest, proposal DecodedTimelockProposal) (Annotations, error)
}

type BatchOperationAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest[BatchOperationAnalyzerContext], operation DecodedBatchOperation) bool
	Analyze(ctx context.Context, req AnalyzeRequest[BatchOperationAnalyzerContext], operation DecodedBatchOperation) (Annotations, error)
}

type CallAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest[CallAnalyzerContext], call DecodedCall) bool
	Analyze(ctx context.Context, req AnalyzeRequest[CallAnalyzerContext], call DecodedCall) (Annotations, error)
}

type ParameterAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest[ParameterAnalyzerContext], param DecodedParameter) bool
	Analyze(ctx context.Context, req AnalyzeRequest[ParameterAnalyzerContext], param DecodedParameter) (Annotations, error)
}

type ParameterAnalyzerContext interface {
	// Proposal returns the current proposal-level context.
	Proposal() DecodedTimelockProposal
	// BatchOperation returns the current batch operation context.
	BatchOperation() DecodedBatchOperation
	// Call returns the current call-level context.
	Call() DecodedCall
}

type CallAnalyzerContext interface {
	// Proposal returns the current proposal-level context.
	Proposal() DecodedTimelockProposal
	// BatchOperation returns the current batch operation context.
	BatchOperation() DecodedBatchOperation
}

type BatchOperationAnalyzerContext interface {
	// Proposal returns the current proposal-level context.
	Proposal() DecodedTimelockProposal
}
