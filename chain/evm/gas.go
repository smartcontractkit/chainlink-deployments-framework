package evm

import "math/bits"

// EIP7825MaxTxGasLimit is the EIP-7825 per-transaction gas limit cap (2^24).
const EIP7825MaxTxGasLimit = uint64(16_777_216)

type gasEstimateAdjuster interface {
	GasLimitBufferBps() uint64
	MaxTxGasLimit() uint64
}

// GasLimitBufferBpsFromClient returns the configured gas limit buffer for the client,
// or 0 when the client does not support gas limit buffering.
func GasLimitBufferBpsFromClient(client any) uint64 {
	if c, ok := client.(gasEstimateAdjuster); ok {
		return c.GasLimitBufferBps()
	}

	return 0
}

// MaxTxGasLimitFromClient returns the configured per-transaction gas limit cap for the client,
// or 0 when no cap is configured.
func MaxTxGasLimitFromClient(client any) uint64 {
	if c, ok := client.(gasEstimateAdjuster); ok {
		return c.MaxTxGasLimit()
	}

	return 0
}

// ApplyGasLimitBuffer increases an estimated or explicit gas limit by bufferBps basis points.
// bufferBps uses basis points where 2500 means +25%. A bufferBps of 0 returns estimated unchanged.
// On overflow, returns the maximum uint64 gas limit rather than wrapping.
func ApplyGasLimitBuffer(estimated, bufferBps uint64) uint64 {
	if bufferBps == 0 || estimated == 0 {
		return estimated
	}

	hi, lo := bits.Mul64(estimated, bufferBps)
	if hi != 0 {
		return ^uint64(0)
	}
	increment := lo / 10_000

	result, carry := bits.Add64(estimated, increment, 0)
	if carry != 0 {
		return ^uint64(0)
	}

	return result
}

// CapGasLimit returns gas unchanged when maxTxGas is 0; otherwise returns min(gas, maxTxGas).
func CapGasLimit(gas, maxTxGas uint64) uint64 {
	if maxTxGas == 0 || gas <= maxTxGas {
		return gas
	}

	return maxTxGas
}

// ApplyGasLimitWithBufferAndCap applies bufferBps to estimated, then caps at maxTxGas when non-zero.
func ApplyGasLimitWithBufferAndCap(estimated, bufferBps, maxTxGas uint64) uint64 {
	return CapGasLimit(ApplyGasLimitBuffer(estimated, bufferBps), maxTxGas)
}
