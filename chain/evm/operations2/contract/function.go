package contract

// FunctionInput is the input structure for all reads and writes.
type FunctionInput[ARGS any] struct {
	// Args are the parameters passed to the contract call.
	Args ARGS `json:"args"`
}
