package analyzer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"slices"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	expanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

type analyzerEngine struct {
	proposalAnalyzers       []ProposalAnalyzer
	batchOperationAnalyzers []BatchOperationAnalyzer
	callAnalyzers           []CallAnalyzer
	parameterAnalyzers      []ParameterAnalyzer
}

func (ae *analyzerEngine) RegisterAnalyzer(baseAnalyzer BaseAnalyzer) error {
	switch a := baseAnalyzer.(type) {
	case ProposalAnalyzer:
		ae.proposalAnalyzers = append(ae.proposalAnalyzers, a)
	case BatchOperationAnalyzer:
		ae.batchOperationAnalyzers = append(ae.batchOperationAnalyzers, a)
	case CallAnalyzer:
		ae.callAnalyzers = append(ae.callAnalyzers, a)
	case ParameterAnalyzer:
		ae.parameterAnalyzers = append(ae.parameterAnalyzers, a)
	default:
		return errors.New("unknown analyzer type")
	}

	return nil
}

func (ae *analyzerEngine) Run(
	ctx context.Context,
	domain cldfdomain.Domain,
	environmentName string,
	proposal *mcms.TimelockProposal,
) (*AnalyzedProposal, error) {
	mcmsChainSelectors := slices.Sorted(maps.Keys(proposal.ChainMetadata))
	chainSelectors := make([]uint64, len(mcmsChainSelectors))

	for i, s := range mcmsChainSelectors {
		chainSelectors[i] = uint64(s)
	}

	env, err := cldfenvironment.Load(ctx, domain, environmentName,
		cldfenvironment.OnlyLoadChainsFor(chainSelectors),
		cldfenvironment.WithoutJD())
	if err != nil {
		return nil, fmt.Errorf("load environment: %w", err)
	}

	proposalCtx, err := expanalyzer.NewDefaultProposalContext(env)
	if err != nil {
		return nil, fmt.Errorf("create proposal context: %w", err)
	}

	ectx := &ExecutionContext{
		Domain:          domain,
		EnvironmentName: environmentName,
		Env:             env,
		proposalCtx:     proposalCtx,
	}

	return ae.Analyze(ctx, ectx, proposal)
}

func (ae *analyzerEngine) Analyze(
	ctx context.Context,
	ectx *ExecutionContext,
	proposal *mcms.TimelockProposal,
) (*AnalyzedProposal, error) {
	decoded, err := decodeProposal(ctx, ectx, proposal)
	if err != nil {
		return nil, fmt.Errorf("decode proposal: %w", err)
	}

	return ae.AnalyzeDecoded(ctx, ectx, *decoded)
}

func (ae *analyzerEngine) AnalyzeWithProposalContext(
	ctx context.Context,
	env deployment.Environment,
	proposalCtx expanalyzer.ProposalContext,
	proposal *mcms.TimelockProposal,
) (*AnalyzedProposal, error) {
	ectx := &ExecutionContext{
		Env:         env,
		proposalCtx: proposalCtx,
	}

	return ae.Analyze(ctx, ectx, proposal)
}

func (ae *analyzerEngine) AnalyzeDecoded(
	ctx context.Context,
	ectx *ExecutionContext,
	decoded DecodedTimelockProposal,
) (*AnalyzedProposal, error) {
	actx := &AnalyzerContext{}
	req := AnalyzerRequest{Context: actx, Execution: ectx}

	result := &AnalyzedProposal{annotated: &annotated{}, Decoded: decoded}
	actx.Proposal = result

	for _, pa := range ae.proposalAnalyzers {
		if !pa.Matches(ctx, req, decoded) {
			continue
		}

		annotations, err := pa.Analyze(ctx, req, decoded)
		if err != nil {
			log.Printf("WARN: proposal analyzer %q failed: %v", pa.ID(), err)

			continue
		}

		result.AddAnnotations(annotations...)
	}

	for _, batch := range decoded.BatchOperations {
		result.BatchOperations = append(result.BatchOperations, ae.analyzeBatch(ctx, actx, req, batch))
	}

	actx.Proposal = nil

	return result, nil
}

func (ae *analyzerEngine) analyzeBatch(
	ctx context.Context,
	actx *AnalyzerContext,
	req AnalyzerRequest,
	batch DecodedBatchOperation,
) *AnalyzedBatchOperation {
	result := &AnalyzedBatchOperation{
		annotated:     &annotated{},
		ChainSelector: batch.ChainSelector,
		ChainName:     batch.ChainName,
	}
	actx.BatchOperation = result

	for _, ba := range ae.batchOperationAnalyzers {
		if !ba.Matches(ctx, req, batch) {
			continue
		}

		annotations, err := ba.Analyze(ctx, req, batch)
		if err != nil {
			log.Printf("WARN: batch operation analyzer %q failed: %v", ba.ID(), err)

			continue
		}

		result.AddAnnotations(annotations...)
	}

	for _, call := range batch.Calls {
		result.Calls = append(result.Calls, ae.analyzeCall(ctx, actx, req, call))
	}

	actx.BatchOperation = nil

	return result
}

func (ae *analyzerEngine) analyzeCall(
	ctx context.Context,
	actx *AnalyzerContext,
	req AnalyzerRequest,
	dc DecodedCall,
) *AnalyzedCall {
	result := &AnalyzedCall{annotated: &annotated{}, DecodedCall: dc}
	actx.Call = result

	for _, ca := range ae.callAnalyzers {
		if !ca.Matches(ctx, req, dc) {
			continue
		}

		annotations, err := ca.Analyze(ctx, req, dc)
		if err != nil {
			log.Printf("WARN: call analyzer %q failed: %v", ca.ID(), err)

			continue
		}

		result.AddAnnotations(annotations...)
	}

	for _, input := range dc.Inputs() {
		result.AnalyzedInputs = append(result.AnalyzedInputs, ae.analyzeParameter(ctx, actx, req, input))
	}

	actx.Call = nil

	return result
}

func (ae *analyzerEngine) analyzeParameter(
	ctx context.Context,
	actx *AnalyzerContext,
	req AnalyzerRequest,
	dp DecodedParameter,
) *AnalyzedParameter {
	result := &AnalyzedParameter{annotated: &annotated{}, DecodedParameter: dp}
	actx.Parameter = result

	for _, pa := range ae.parameterAnalyzers {
		if !pa.Matches(ctx, req, dp) {
			continue
		}

		annotations, err := pa.Analyze(ctx, req, dp)
		if err != nil {
			log.Printf("WARN: parameter analyzer %q failed: %v", pa.ID(), err)

			continue
		}

		result.AddAnnotations(annotations...)
	}

	actx.Parameter = nil

	return result
}
