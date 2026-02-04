package internal

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/samber/lo"
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

type analyzerEngine struct {
	proposalAnalyzers       []analyzer.ProposalAnalyzer
	batchOperationAnalyzers []analyzer.BatchOperationAnalyzer
	callAnalyzers           []analyzer.CallAnalyzer
	parameterAnalyzers      []analyzer.ParameterAnalyzer
}

var _ analyzer.AnalyzerEngine = &analyzerEngine{}

func NewAnalyzerEngine() *analyzerEngine {
	return &analyzerEngine{}
}

func (ae *analyzerEngine) Run(
	ctx context.Context,
	domain cldfdomain.Domain,
	environmentName string,
	proposal *mcms.TimelockProposal,
) (analyzer.AnalyzedProposal, error) {
	// TODO: instantiate and embed logger in ctx (if not embedded already)

	// load environment,
	mcmsChainSelectors := slices.Sorted(maps.Keys(proposal.ChainMetadata))
	chainSelectors := lo.Map(mcmsChainSelectors, func(s mcmstypes.ChainSelector, _ int) uint64 { return uint64(s) })
	env, err := cldfenvironment.Load(ctx, domain, environmentName,
		cldfenvironment.OnlyLoadChainsFor(chainSelectors),
		// cldfenvironment.WithLogger(lggr),
		cldfenvironment.WithoutJD())
	if err != nil {
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}

	decodedProposal, err := ae.decodeProposal(ctx, proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to decode timelock proposal: %w", err)
	}

	actx := &analyzerContext{}
	ectx := &executionContext{
		domain:          domain,
		environmentName: environmentName,
		blockChains:     env.BlockChains,
		dataStore:       env.DataStore,
	}

	analyzedProposal, err := ae.analyzeProposal(ctx, actx, ectx, decodedProposal)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze timelock proposal: %w", err)
	}

	return analyzedProposal, errors.New("not implemented")
}

func (ae *analyzerEngine) RegisterAnalyzer(baseAnalyzer analyzer.BaseAnalyzer) error {
	switch a := baseAnalyzer.(type) {
	case analyzer.ProposalAnalyzer:
		ae.proposalAnalyzers = append(ae.proposalAnalyzers, a)
	case analyzer.BatchOperationAnalyzer:
		ae.batchOperationAnalyzers = append(ae.batchOperationAnalyzers, a)
	case analyzer.CallAnalyzer:
		ae.callAnalyzers = append(ae.callAnalyzers, a)
	case analyzer.ParameterAnalyzer:
		ae.parameterAnalyzers = append(ae.parameterAnalyzers, a)
	default:
		return errors.New("unknown analyzer type")
	}

	return nil
}

func (ae *analyzerEngine) RegisterFormatter( /* tbd */ ) error {
	return errors.New("not implemented")
}

func (ae *analyzerEngine) decodeProposal(ctx context.Context, proposal *mcms.TimelockProposal) (analyzer.DecodedTimelockProposal, error) {
	// TODO: delegate to decoder component; try to reuse implementation from experimental/analyzer
	return nil, errors.New("not implemented")
}

func (ae *analyzerEngine) analyzeProposal(
	ctx context.Context,
	actx *analyzerContext,
	ectx *executionContext,
	decodedProposal analyzer.DecodedTimelockProposal,
) (analyzer.AnalyzedProposal, error) {
	actx.proposal = &analyzedProposal{decodedProposal: decodedProposal}

	for _, proposalAnalyzer := range ae.proposalAnalyzers {
		// TODO: pre and post execution Analyze calls

		annotations, err := proposalAnalyzer.Analyze(ctx, actx, ectx, decodedProposal)
		if err != nil {
			// log error
			continue
		}
		actx.proposal.AddAnnotations(annotations...)
	}

	for _, batchOp := range decodedProposal.BatchOperations() {
		/*analyzedBatchOperation*/ _, err := ae.analyzeBatchOperation(ctx, actx, ectx, batchOp)
		if err != nil {
			// log error
			continue
		}
		// TODO: add analyzed batch operation to analyzed proposal
	}

	proposal := actx.proposal
	actx.proposal = nil

	return proposal, errors.New("not implemented")
}

func (ae *analyzerEngine) analyzeBatchOperation(
	ctx context.Context,
	actx *analyzerContext,
	ectx *executionContext,
	decodedBatchOperation analyzer.DecodedBatchOperation,
) (analyzer.AnalyzedBatchOperation, error) {
	actx.batchOperation = &analyzedBatchOperation{decodedBatchOperation: decodedBatchOperation}

	for _, batchOperationAnalyzer := range ae.batchOperationAnalyzers {
		// TODO: pre and post execution Analyze calls

		annotations, err := batchOperationAnalyzer.Analyze(ctx, actx, ectx, decodedBatchOperation)
		if err != nil {
			// log error
			continue
		}
		actx.batchOperation.AddAnnotations(annotations...)
	}

	for _, call := range decodedBatchOperation.Calls() {
		/*call*/ _, err := ae.analyzeCall(ctx, actx, ectx, call)
		if err != nil {
			// log error
			continue
		}
		// TODO: add analyzed call to analyzed batch operation
	}

	batchOperation := actx.batchOperation
	actx.batchOperation = nil

	return batchOperation, errors.New("not implemented")
}

func (ae *analyzerEngine) analyzeCall(
	ctx context.Context,
	actx *analyzerContext,
	ectx *executionContext,
	decodedCall analyzer.DecodedCall,
) (analyzer.AnalyzedCall, error) {
	// TODO
	return nil, errors.New("not implemented")
}

// TODO: analyzeParameter or (analyzeInput + analyzeOutput)?
func (ae *analyzerEngine) analyzeParameter(
	ctx context.Context,
	actx *analyzerContext,
	ectx *executionContext,
	decodedParameter analyzer.DecodedParameter,
) (analyzer.AnalyzedParameter, error) {
	// TODO
	return nil, errors.New("not implemented")
}

// ---------------------------------------------------------------------

var _ analyzer.AnalyzedProposal = &analyzedProposal{}

type analyzedProposal struct {
	*annotated
	decodedProposal analyzer.DecodedTimelockProposal
	batchOperations analyzer.AnalyzedBatchOperations
}

func (a analyzedProposal) BatchOperations() analyzer.AnalyzedBatchOperations {
	return a.batchOperations
}

// ---------------------------------------------------------------------

var _ analyzer.AnalyzedBatchOperation = &analyzedBatchOperation{}

type analyzedBatchOperation struct {
	*annotated
	decodedBatchOperation analyzer.DecodedBatchOperation
	calls                 analyzer.AnalyzedCalls
}

func (a analyzedBatchOperation) Calls() analyzer.AnalyzedCalls {
	return a.calls
}

// ---------------------------------------------------------------------

var _ analyzer.AnalyzedCall = &analyzedCall{}

type analyzedCall struct {
	*annotated
	decodedCall analyzer.DecodedCall
	inputs      analyzer.AnalyzedParameters
	outputs     analyzer.AnalyzedParameters
}

func (a analyzedCall) Name() string {
	return a.decodedCall.Name()
}

func (a analyzedCall) Inputs() analyzer.AnalyzedParameters {
	return a.inputs
}

func (a analyzedCall) Outputs() analyzer.AnalyzedParameters {
	return a.outputs
}

// ---------------------------------------------------------------------
// TODO: analyzedParameter
