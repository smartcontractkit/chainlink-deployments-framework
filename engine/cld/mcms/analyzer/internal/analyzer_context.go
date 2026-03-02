package internal

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

// TODO: consider converting into simple struct type
var _ analyzer.AnalyzerContext = &analyzerContext{}

type analyzerContext struct {
	proposal       analyzer.AnalyzedProposal
	batchOperation analyzer.AnalyzedBatchOperation
	call           analyzer.AnalyzedCall
}

func (ac *analyzerContext) Proposal() analyzer.AnalyzedProposal {
	return ac.proposal
}

func (ac *analyzerContext) BatchOperation() analyzer.AnalyzedBatchOperation {
	return ac.batchOperation
}

func (ac *analyzerContext) Call() analyzer.AnalyzedCall {
	return ac.call
}
