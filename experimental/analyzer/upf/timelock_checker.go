package upf

import "strings"

// timelockBatchChecker provides chain-specific logic for detecting timelock batch functions.
type timelockBatchChecker interface {
	isTimelockBatch(functionName string) bool
}

// evmTimelockChecker handles EVM chains.
// Matches full function signatures for scheduleBatch and bypasserExecuteBatch.
type evmTimelockChecker struct{}

func (evmTimelockChecker) isTimelockBatch(functionName string) bool {
	return functionName == "function scheduleBatch((address,uint256,bytes)[] calls, bytes32 predecessor, bytes32 salt, uint256 delay) returns()" ||
		functionName == "function bypasserExecuteBatch((address,uint256,bytes)[] calls) payable returns()"
}

// solanaTimelockChecker handles Solana chain.
// Matches exact function names: ScheduleBatch, BypasserExecuteBatch.
type solanaTimelockChecker struct{}

func (solanaTimelockChecker) isTimelockBatch(functionName string) bool {
	return functionName == "ScheduleBatch" || functionName == "BypasserExecuteBatch"
}

// suiAptosTimelockChecker handles both Sui and Aptos chains.
// Sui: mcms::timelock_schedule_batch, mcms::timelock_bypasser_execute_batch
// Aptos: package::module::timelock_schedule_batch, package::module::timelock_bypasser_execute_batch
// Uses HasSuffix to prevent false positives like "::timelock_schedule_batch_helper".
type suiAptosTimelockChecker struct{}

func (suiAptosTimelockChecker) isTimelockBatch(functionName string) bool {
	return strings.HasSuffix(functionName, "::timelock_schedule_batch") ||
		strings.HasSuffix(functionName, "::timelock_bypasser_execute_batch")
}

// tonTimelockChecker handles TON chain.
// TON: ContractType::ScheduleBatch(0x...), ContractType::BypasserExecuteBatch(0x...)
// Uses Contains because the opcode suffix (0x...) varies.
type tonTimelockChecker struct{}

func (tonTimelockChecker) isTimelockBatch(functionName string) bool {
	return strings.Contains(functionName, "::ScheduleBatch(") ||
		strings.Contains(functionName, "::BypasserExecuteBatch(")
}

// timelockBatchCheckers is a list of chain-specific checkers for timelock batch functions.
var timelockBatchCheckers = []timelockBatchChecker{
	evmTimelockChecker{},
	solanaTimelockChecker{},
	suiAptosTimelockChecker{},
	tonTimelockChecker{},
}

// isTimelockBatchFunction checks if the function name corresponds to a timelock batch operation
// across different chain families (EVM, Solana, Sui, Aptos, TON).
func isTimelockBatchFunction(functionName string) bool {
	for _, checker := range timelockBatchCheckers {
		if checker.isTimelockBatch(functionName) {
			return true
		}
	}

	return false
}
