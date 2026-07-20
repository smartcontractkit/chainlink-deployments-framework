package gas

// EIP7825MaxTxGasLimit is the EIP-7825 per-transaction gas limit cap (2^24).
// Used as a reference value in docs and tests; YAML max_tx_gas_limit accepts any uint64.
const EIP7825MaxTxGasLimit = uint64(16_777_216)

// CapGasLimit returns gas unchanged when maxTxGas is 0; otherwise returns min(gas, maxTxGas).
func CapGasLimit(gas, maxTxGas uint64) uint64 {
	if maxTxGas == 0 || gas <= maxTxGas {
		return gas
	}

	return maxTxGas
}
