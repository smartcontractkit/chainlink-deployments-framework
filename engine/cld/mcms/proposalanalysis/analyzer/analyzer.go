package analyzer

import (
	"context"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
)

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
}

// AnalyzeRequest encapsulates the analyzer and execution contexts passed to analyzer methods.
type AnalyzeRequest struct {
	AnalyzerContext  AnalyzerContext
	ExecutionContext ExecutionContext
}

type BaseAnalyzer interface {
	ID() string
	Dependencies() []string // Returns IDs of dependent analyzers
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
