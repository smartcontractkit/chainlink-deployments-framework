package renderer

import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"

// Analyzed proposal graph types consumed by renderers.
type (
	AnalyzedProposal       = analyzer.AnalyzedProposal
	AnalyzedBatchOperation = analyzer.AnalyzedBatchOperation
	AnalyzedCall           = analyzer.AnalyzedCall
	AnalyzedParameter      = analyzer.AnalyzedParameter

	AnalyzedBatchOperations = analyzer.AnalyzedBatchOperations
	AnalyzedCalls           = analyzer.AnalyzedCalls
	AnalyzedParameters      = analyzer.AnalyzedParameters

	AnalyzedProposalNode       = analyzer.AnalyzedProposalNode
	AnalyzedBatchOperationNode = analyzer.AnalyzedBatchOperationNode
	AnalyzedCallNode           = analyzer.AnalyzedCallNode
	AnalyzedParameterNode      = analyzer.AnalyzedParameterNode
)

func NewAnalyzedProposalNode(batchOps AnalyzedBatchOperations) *AnalyzedProposalNode {
	return analyzer.NewAnalyzedProposalNode(batchOps)
}

func NewAnalyzedBatchOperationNode(chainSelector uint64, calls AnalyzedCalls) *AnalyzedBatchOperationNode {
	return analyzer.NewAnalyzedBatchOperationNode(chainSelector, calls)
}

func NewAnalyzedCallNode(
	to string,
	name string,
	inputs AnalyzedParameters,
	outputs AnalyzedParameters,
	data []byte,
	contractType string,
	contractVersion string,
	additional map[string]any,
) *AnalyzedCallNode {
	return analyzer.NewAnalyzedCallNode(
		to,
		name,
		inputs,
		outputs,
		data,
		contractType,
		contractVersion,
		additional,
	)
}

func NewAnalyzedParameterNode(name, atype string, value any) *AnalyzedParameterNode {
	return analyzer.NewAnalyzedParameterNode(name, atype, value)
}
