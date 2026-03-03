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
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer/internal/logger"
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
	lggr := logger.FromContext(ctx)
	analyzedProposal := &analyzedProposal{decodedProposal: decodedProposal}
	actx.proposal = analyzedProposal

	for _, proposalAnalyzer := range ae.proposalAnalyzers {
		// TODO: pre and post execution Analyze
		if !proposalAnalyzer.Matches(ctx, actx, decodedProposal) {
			continue
		}

		annotations, err := proposalAnalyzer.Analyze(ctx, actx, ectx, decodedProposal)
		if err != nil {
			lggr.Warnf("proposal analyzer %q failed: %w", proposalAnalyzer.ID(), err)
			continue
		}
		actx.proposal.AddAnnotations(annotations...)
	}

	for _, batchOp := range decodedProposal.BatchOperations() {
		analyzedBatchOperation, err := ae.analyzeBatchOperation(ctx, actx, ectx, batchOp)
		if err != nil {
			lggr.Warnf("failed to analyze batch operation: %w", err)
			continue
		}
		analyzedProposal.batchOperations = append(analyzedProposal.batchOperations, analyzedBatchOperation)
	}

	actx.proposal = nil // clear context

	return analyzedProposal, errors.New("not implemented")
}

func (ae *analyzerEngine) analyzeBatchOperation(
	ctx context.Context,
	actx *analyzerContext,
	ectx *executionContext,
	decodedBatchOperation analyzer.DecodedBatchOperation,
) (analyzer.AnalyzedBatchOperation, error) {
	lggr := logger.FromContext(ctx)
	analyzedBatchOp := &analyzedBatchOperation{decodedBatchOperation: decodedBatchOperation}
	actx.batchOperation = analyzedBatchOp

	for _, batchOperationAnalyzer := range ae.batchOperationAnalyzers {
		// TODO: pre and post execution Analyze
		if !batchOperationAnalyzer.Matches(ctx, actx, decodedBatchOperation) {
			continue
		}

		annotations, err := batchOperationAnalyzer.Analyze(ctx, actx, ectx, decodedBatchOperation)
		if err != nil {
			lggr.Warnf("batch operation analyzer %q failed: %w", batchOperationAnalyzer.ID(), err)
			continue
		}
		analyzedBatchOp.AddAnnotations(annotations...)
	}

	for _, call := range decodedBatchOperation.Calls() {
		analyzedCall, err := ae.analyzeCall(ctx, actx, ectx, call)
		if err != nil {
			lggr.Warnf("failed to analyze call: %w", err)
			continue
		}
		analyzedBatchOp.calls = append(analyzedBatchOp.calls, analyzedCall)
	}

	actx.batchOperation = nil // clear context

	return analyzedBatchOp, errors.New("not implemented")
}

func (ae *analyzerEngine) analyzeCall(
	ctx context.Context,
	actx *analyzerContext,
	ectx *executionContext,
	decodedCall analyzer.DecodedCall,
) (analyzer.AnalyzedCall, error) {
	lggr := logger.FromContext(ctx)
	analyzedCall := &analyzedCall{decodedCall: decodedCall}
	actx.call = analyzedCall

	for _, callAnalyzer := range ae.callAnalyzers {
		// TODO: pre and post execution Analyze
		if !callAnalyzer.Matches(ctx, actx, decodedCall) {
			continue
		}

		annotations, err := callAnalyzer.Analyze(ctx, actx, ectx, decodedCall)
		if err != nil {
			lggr.Warnf("call analyzer %q failed: %w", callAnalyzer.ID(), err)
			continue
		}
		analyzedCall.AddAnnotations(annotations...)
	}

	for _, input := range decodedCall.Inputs() {
		analyzedInput, err := ae.analyzeParameter(ctx, actx, ectx, input)
		if err != nil {
			lggr.Warnf("failed to analyze method input: %w", err)
			continue
		}
		analyzedCall.inputs = append(analyzedCall.inputs, analyzedInput)
	}
	for _, output := range decodedCall.Outputs() {
		analyzedOutput, err := ae.analyzeParameter(ctx, actx, ectx, output)
		if err != nil {
			lggr.Warnf("failed to analyze method output: %w", err)
			continue
		}
		analyzedCall.outputs = append(analyzedCall.outputs, analyzedOutput)
	}

	actx.call = nil // clear context

	return analyzedCall, nil
}

// TODO: analyzeParameter or (analyzeInput + analyzeOutput)?
func (ae *analyzerEngine) analyzeParameter(
	ctx context.Context,
	actx *analyzerContext,
	ectx *executionContext,
	decodedParameter analyzer.DecodedParameter,
) (analyzer.AnalyzedParameter, error) {
	lggr := logger.FromContext(ctx)
	analyzedParam := &analyzedParameter{decodedParameter: decodedParameter}

	for _, parameterAnalyzer := range ae.parameterAnalyzers {
		// TODO: pre and post execution Analyze
		if !parameterAnalyzer.Matches(ctx, actx, decodedParameter) {
			continue
		}

		annotations, err := parameterAnalyzer.Analyze(ctx, actx, ectx, decodedParameter)
		if err != nil {
			lggr.Warnf("parameter analyzer %q failed: %w", parameterAnalyzer.ID(), err)
			continue
		}
		analyzedParam.AddAnnotations(annotations...)
	}

	return analyzedParam, nil
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

var _ analyzer.AnalyzedParameter = &analyzedParameter{}

type analyzedParameter struct {
	*annotated
	decodedParameter analyzer.DecodedParameter
}

func (a analyzedParameter) Name() string {
	return a.decodedParameter.Name()
}

func (a analyzedParameter) Type() string {
	return a.decodedParameter.Type()
}

func (a analyzedParameter) Value() any {
	return a.decodedParameter.Value()
}
