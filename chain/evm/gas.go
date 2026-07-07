package evm

import "math/bits"

// DefaultGasLimitBufferBps is the default proactive gas limit buffer applied to
// eth_estimateGas results and explicit gas limit overrides (+25%).
const DefaultGasLimitBufferBps = uint64(2500)

type gasLimitBufferer interface {
	GasLimitBufferBps() uint64
}

// GasLimitBufferBpsFromClient returns the configured gas limit buffer for the client,
// or 0 when the client does not support gas limit buffering.
func GasLimitBufferBpsFromClient(client any) uint64 {
	if c, ok := client.(gasLimitBufferer); ok {
		return c.GasLimitBufferBps()
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
