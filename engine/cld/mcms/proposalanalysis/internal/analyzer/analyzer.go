package analyzer

import (
	"context"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/annotationstore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/decoder"
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
	DependencyAnnotationStore annotationstore.DependencyAnnotationStore
}

// ProposalAnalyzeRequest encapsulates the execution context and annotation store passed to a proposal analyzer.
type ProposalAnalyzeRequest struct {
	ExecutionContext          ExecutionContext
	DependencyAnnotationStore annotationstore.DependencyAnnotationStore
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
	CanAnalyze(ctx context.Context, req ProposalAnalyzeRequest, proposal decoder.DecodedTimelockProposal) bool
	Analyze(ctx context.Context, req ProposalAnalyzeRequest, proposal decoder.DecodedTimelockProposal) (annotation.Annotations, error)
}

type BatchOperationAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest[BatchOperationAnalyzerContext], operation decoder.DecodedBatchOperation) bool
	Analyze(ctx context.Context, req AnalyzeRequest[BatchOperationAnalyzerContext], operation decoder.DecodedBatchOperation) (annotation.Annotations, error)
}

type CallAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest[CallAnalyzerContext], call decoder.DecodedCall) bool
	Analyze(ctx context.Context, req AnalyzeRequest[CallAnalyzerContext], call decoder.DecodedCall) (annotation.Annotations, error)
}

type ParameterAnalyzer interface {
	BaseAnalyzer
	CanAnalyze(ctx context.Context, req AnalyzeRequest[ParameterAnalyzerContext], param decoder.DecodedParameter) bool
	Analyze(ctx context.Context, req AnalyzeRequest[ParameterAnalyzerContext], param decoder.DecodedParameter) (annotation.Annotations, error)
}

type ParameterAnalyzerContext interface {
	// Proposal returns the current proposal-level context.
	Proposal() decoder.DecodedTimelockProposal
	// BatchOperation returns the current batch operation context.
	BatchOperation() decoder.DecodedBatchOperation
	// Call returns the current call-level context.
	Call() decoder.DecodedCall
}

type CallAnalyzerContext interface {
	// Proposal returns the current proposal-level context.
	Proposal() decoder.DecodedTimelockProposal
	// BatchOperation returns the current batch operation context.
	BatchOperation() decoder.DecodedBatchOperation
}

type BatchOperationAnalyzerContext interface {
	// Proposal returns the current proposal-level context.
	Proposal() decoder.DecodedTimelockProposal
}
