package internal

import (
	"context"
	"errors"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

type analyzerEngine struct {
	proposalAnalyzers  []analyzer.ProposalAnalyzer
	operationAnalyzers []analyzer.OperationAnalyzer
	callAnalyzers      []analyzer.CallAnalyzer
	parameterAnalyzer  []analyzer.ParameterAnalyzer
}

var _ analyzer.AnalyzerEngine = &analyzerEngine{}

func NewAnalyzerEngine() *analyzerEngine {
	return &analyzerEngine{}
}

func (*analyzerEngine) Run(
	ctx *context.Context, actx analyzer.AnalyzerContext, ectx analyzer.ExecutionContext, proposal *mcms.TimelockProposal,
) (analyzer.AnalyzedProposal, error) {
	return nil, errors.New("not implemented")
}

func (*analyzerEngine) RegisterAnalyzer(analyzer analyzer.BaseAnalyzer) error {
	return errors.New("not implemented")
}
