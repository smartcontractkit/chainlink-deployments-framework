package proposalanalysis

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
)

var _ types.AnalyzerContext = &analyzerContext{}

type analyzerContext struct {
	proposal       types.AnalyzedProposal
	batchOperation types.AnalyzedBatchOperation
	call           types.AnalyzedCall
}

func (ac *analyzerContext) Proposal() types.AnalyzedProposal {
	return ac.proposal
}

func (ac *analyzerContext) BatchOperation() types.AnalyzedBatchOperation {
	return ac.batchOperation
}

func (ac *analyzerContext) Call() types.AnalyzedCall {
	return ac.call
}

// GetAnnotationsFrom returns annotations from a specific analyzer
func (ac *analyzerContext) GetAnnotationsFrom(analyzerID string) types.Annotations {
	var annotations types.Annotations
	if ac.call != nil {
		annotations = append(annotations, ac.call.GetAnnotationsByAnalyzer(analyzerID)...)
	}
	if ac.batchOperation != nil {
		annotations = append(annotations, ac.batchOperation.GetAnnotationsByAnalyzer(analyzerID)...)
	}
	if ac.proposal != nil {
		annotations = append(annotations, ac.proposal.GetAnnotationsByAnalyzer(analyzerID)...)
	}
	return annotations
}
