package analyzer

import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotated"

type AnalyzedProposal interface {
	annotated.Annotated
	BatchOperations() AnalyzedBatchOperations
}

type AnalyzedBatchOperation interface {
	annotated.Annotated
	ChainSelector() uint64
	Calls() AnalyzedCalls
}

type AnalyzedBatchOperations []AnalyzedBatchOperation

type AnalyzedCalls []AnalyzedCall

type AnalyzedCall interface {
	annotated.Annotated
	To() string
	Name() string
	Inputs() AnalyzedParameters
	Outputs() AnalyzedParameters
	Data() []byte
	ContractType() string
	ContractVersion() string
	AdditionalFields() map[string]any
}

type AnalyzedParameters []AnalyzedParameter

type AnalyzedParameter interface {
	annotated.Annotated
	Name() string
	Type() string
	Value() any
}

var (
	_ AnalyzedProposal       = &AnalyzedProposalNode{}
	_ AnalyzedBatchOperation = &AnalyzedBatchOperationNode{}
	_ AnalyzedCall           = &AnalyzedCallNode{}
	_ AnalyzedParameter      = &AnalyzedParameterNode{}
)

type AnalyzedProposalNode struct {
	annotated.BaseAnnotated
	batchOps AnalyzedBatchOperations
}

func NewAnalyzedProposalNode(batchOps AnalyzedBatchOperations) *AnalyzedProposalNode {
	return &AnalyzedProposalNode{batchOps: batchOps}
}

func (a *AnalyzedProposalNode) BatchOperations() AnalyzedBatchOperations {
	return a.batchOps
}

type AnalyzedBatchOperationNode struct {
	annotated.BaseAnnotated
	chainSelector uint64
	calls         AnalyzedCalls
}

func NewAnalyzedBatchOperationNode(chainSelector uint64, calls AnalyzedCalls) *AnalyzedBatchOperationNode {
	return &AnalyzedBatchOperationNode{
		chainSelector: chainSelector,
		calls:         calls,
	}
}

func (a *AnalyzedBatchOperationNode) ChainSelector() uint64 {
	return a.chainSelector
}

func (a *AnalyzedBatchOperationNode) Calls() AnalyzedCalls {
	return a.calls
}

type AnalyzedCallNode struct {
	annotated.BaseAnnotated
	to              string
	name            string
	inputs          AnalyzedParameters
	outputs         AnalyzedParameters
	data            []byte
	contractType    string
	contractVersion string
	additional      map[string]any
}

func NewAnalyzedCallNode(
	to,
	name string,
	inputs,
	outputs AnalyzedParameters,
	data []byte,
	contractType,
	contractVersion string,
	additional map[string]any,
) *AnalyzedCallNode {
	return &AnalyzedCallNode{
		to:              to,
		name:            name,
		inputs:          inputs,
		outputs:         outputs,
		data:            data,
		contractType:    contractType,
		contractVersion: contractVersion,
		additional:      additional,
	}
}

func (a *AnalyzedCallNode) To() string {
	return a.to
}

func (a *AnalyzedCallNode) Name() string {
	return a.name
}

func (a *AnalyzedCallNode) Inputs() AnalyzedParameters {
	return a.inputs
}

func (a *AnalyzedCallNode) Outputs() AnalyzedParameters {
	return a.outputs
}

func (a *AnalyzedCallNode) Data() []byte {
	return a.data
}

func (a *AnalyzedCallNode) ContractType() string {
	return a.contractType
}

func (a *AnalyzedCallNode) ContractVersion() string {
	return a.contractVersion
}

// AdditionalFields returns a copy of the call's additional metadata.
// It returns an empty map when no additional metadata is set.
func (a *AnalyzedCallNode) AdditionalFields() map[string]any {
	if len(a.additional) == 0 {
		return map[string]any{}
	}

	out := make(map[string]any, len(a.additional))
	for k, v := range a.additional {
		out[k] = v
	}

	return out
}

type AnalyzedParameterNode struct {
	annotated.BaseAnnotated
	name  string
	atype string
	value any
}

func NewAnalyzedParameterNode(name, atype string, value any) *AnalyzedParameterNode {
	return &AnalyzedParameterNode{name: name, atype: atype, value: value}
}

func (a *AnalyzedParameterNode) Name() string {
	return a.name
}

func (a *AnalyzedParameterNode) Type() string {
	return a.atype
}

func (a *AnalyzedParameterNode) Value() any {
	return a.value
}
