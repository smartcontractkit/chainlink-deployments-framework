package operations

import "sync/atomic"

var disableIdempotency atomic.Bool

// SetIdempotencyDisabled configures whether execution idempotency (report reuse) is disabled.
// When true, ExecuteOperation, ExecuteOperationN, and ExecuteSequence always run
// fresh regardless of prior successful reports.
func SetIdempotencyDisabled(disabled bool) {
	disableIdempotency.Store(disabled)
}

// IdempotencyDisabled reports whether execution idempotency is disabled.
func IdempotencyDisabled() bool {
	return disableIdempotency.Load()
}

func shouldReusePreviousReport(forceExecute bool) bool {
	return !forceExecute && !IdempotencyDisabled()
}
