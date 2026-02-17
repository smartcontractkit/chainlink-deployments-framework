package analyzer

type AnalyzedProposal interface {
	Annotated
	BatchOperations() AnalyzedBatchOperations
}

type AnalyzedBatchOperation interface {
	Annotated
	ChainSelector() uint64
	Calls() AnalyzedCalls
}

type AnalyzedBatchOperations []AnalyzedBatchOperation

type AnalyzedCalls []AnalyzedCall

type AnalyzedCall interface {
	Annotated
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
	Annotated
	Name() string
	Type() string
	Value() any
}
