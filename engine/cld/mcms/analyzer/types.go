package analyzer

import (
	"context"
	"encoding/json"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	expanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

type Annotation interface {
	Type() string
	Name() string
	Value() any
	AnalyzerID() string
}

type Annotations []Annotation

type DecodedTimelockProposal struct {
	BatchOperations []DecodedBatchOperation
}

type DecodedBatchOperation struct {
	ChainSelector uint64
	ChainName     string
	Calls         []DecodedCall
}

type DecodedParameter struct {
	Name         string // parameter name (e.g. "chainsToAdd")
	Value        any    // raw decoded Go value (e.g. *big.Int, uint64, common.Address)
	DisplayValue any    // optional display-oriented value from the decoder (e.g., FieldValue for EVM, YamlField for Solana); nil when unavailable
}

type DecodedCall interface {
	ContractType() string
	Name() string               // method name
	To() string                 // target address
	Inputs() []DecodedParameter // ordered input parameters
	Data() []byte
	AdditionalFields() json.RawMessage
}

type AnalyzedProposal struct {
	*annotated
	Decoded         DecodedTimelockProposal
	BatchOperations []*AnalyzedBatchOperation
}

type AnalyzedBatchOperation struct {
	*annotated
	ChainSelector uint64
	ChainName     string
	Calls         []*AnalyzedCall
}

type AnalyzedCall struct {
	*annotated
	DecodedCall
	AnalyzedInputs []*AnalyzedParameter
}

type AnalyzedParameter struct {
	*annotated
	DecodedParameter
}

// AnalyzerContext provides the current position in the proposal tree so that
// analyzers can inspect their parent context
type AnalyzerContext struct {
	Proposal       *AnalyzedProposal
	BatchOperation *AnalyzedBatchOperation
	Call           *AnalyzedCall
	Parameter      *AnalyzedParameter
}

type ExecutionContext struct {
	Domain          cldfdomain.Domain
	EnvironmentName string
	Env             deployment.Environment
	proposalCtx     expanalyzer.ProposalContext
}

type AnalyzerRequest struct {
	Context   *AnalyzerContext
	Execution *ExecutionContext
}

type BaseAnalyzer interface {
	ID() string
	Dependencies() []string
}

type ProposalAnalyzer interface {
	BaseAnalyzer
	Matches(ctx context.Context, req AnalyzerRequest, proposal DecodedTimelockProposal) bool
	Analyze(ctx context.Context, req AnalyzerRequest, proposal DecodedTimelockProposal) (Annotations, error)
}

type BatchOperationAnalyzer interface {
	BaseAnalyzer
	Matches(ctx context.Context, req AnalyzerRequest, operation DecodedBatchOperation) bool
	Analyze(ctx context.Context, req AnalyzerRequest, operation DecodedBatchOperation) (Annotations, error)
}

type CallAnalyzer interface {
	BaseAnalyzer
	Matches(ctx context.Context, req AnalyzerRequest, call DecodedCall) bool
	Analyze(ctx context.Context, req AnalyzerRequest, call DecodedCall) (Annotations, error)
}

type ParameterAnalyzer interface {
	BaseAnalyzer
	Matches(ctx context.Context, req AnalyzerRequest, param DecodedParameter) bool
	Analyze(ctx context.Context, req AnalyzerRequest, param DecodedParameter) (Annotations, error)
}

type AnalyzerEngine interface {
	Run(ctx context.Context, domain cldfdomain.Domain, environmentName string, proposal *mcms.TimelockProposal) (*AnalyzedProposal, error)
	AnalyzeWithProposalContext(ctx context.Context, env deployment.Environment, proposalCtx expanalyzer.ProposalContext, proposal *mcms.TimelockProposal) (*AnalyzedProposal, error)
	Analyze(ctx context.Context, ectx *ExecutionContext, proposal *mcms.TimelockProposal) (*AnalyzedProposal, error)
	AnalyzeDecoded(ctx context.Context, ectx *ExecutionContext, decoded DecodedTimelockProposal) (*AnalyzedProposal, error)
	RegisterAnalyzer(analyzer BaseAnalyzer) error
}

func NewAnalyzerEngine() AnalyzerEngine {
	return &analyzerEngine{}
}

func RenderText(proposal *AnalyzedProposal, description string) string {
	return renderText(proposal, description)
}
