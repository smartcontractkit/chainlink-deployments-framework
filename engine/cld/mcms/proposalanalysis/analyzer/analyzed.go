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
