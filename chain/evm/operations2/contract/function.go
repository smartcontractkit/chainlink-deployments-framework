package contract

// FunctionInput is the input structure for all reads and writes.
type FunctionInput[ARGS any] struct {
	// Args are the parameters passed to the contract call.
	Args ARGS `json:"args"`
	// GasLimit optionally overrides the deployer gas limit on EVM writes.
	// Normally set by RetryWriteWithGasBoost after gas-related failures; may also be set manually.
	GasLimit uint64 `json:"gasLimit,omitempty"`
	// GasPrice optionally overrides the deployer legacy gas price on EVM writes (wei).
	// When non-zero, the write uses a legacy fee transaction and clears any EIP-1559 GasFeeCap/GasTipCap.
	// Normally set by RetryWriteWithGasBoost after gas-related failures; may also be set manually.
	GasPrice uint64 `json:"gasPrice,omitempty"`
}
