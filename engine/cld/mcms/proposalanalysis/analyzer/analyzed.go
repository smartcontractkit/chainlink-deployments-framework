package analyzer

type AnalyzedProposal interface {
	Annotated
	BatchOperations() AnalyzedBatchOperations
}

type AnalyzedBatchOperation interface {
	Annotated
	Calls() AnalyzedCalls
}

type AnalyzedBatchOperations []AnalyzedBatchOperation

type AnalyzedCalls []AnalyzedCall

type AnalyzedCall interface {
	Annotated
	Name() string
	Inputs() AnalyzedParameters
	Outputs() AnalyzedParameters
}

type AnalyzedParameters []AnalyzedParameter

type AnalyzedParameter interface {
	Annotated
	Name() string
	Type() string
	Value() any
}
