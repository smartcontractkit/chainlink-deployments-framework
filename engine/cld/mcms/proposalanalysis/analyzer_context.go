package proposalanalysis

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

var _ types.AnalyzerContext = &analyzerContext{}

type analyzerContext struct {
	proposal       types.AnalyzedProposal
	batchOperation types.AnalyzedBatchOperation
	call           types.AnalyzedCall
	evmRegistry    experimentalanalyzer.EVMABIRegistry
	solanaRegistry experimentalanalyzer.SolanaDecoderRegistry
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

func (ac *analyzerContext) GetEVMRegistry() experimentalanalyzer.EVMABIRegistry {
	return ac.evmRegistry
}

func (ac *analyzerContext) GetSolanaRegistry() experimentalanalyzer.SolanaDecoderRegistry {
	return ac.solanaRegistry
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
