package analyzer

// ProposalReport is a format-neutral structure describing a decoded proposal
// ready to be rendered in multiple output formats.
//
// Exactly one of Operations or Batches is populated depending on the proposal type.
type ProposalReport struct {
	// Action is the timelock action being signed (schedule, cancel, or bypass).
	// It is only populated for timelock proposals; empty for plain proposals.
	Action     string
	Operations []OperationReport
	Batches    []BatchReport
}

// OperationReport captures one proposal operation on a specific chain.
type OperationReport struct {
	ChainSelector uint64
	ChainName     string
	Family        string
	Calls         []*DecodedCall
}

// BatchReport groups multiple operations for timelock-style proposals.
type BatchReport struct {
	ChainSelector uint64
	ChainName     string
	Family        string
	Operations    []OperationReport
}
