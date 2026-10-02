package simulation

import (
	"context"
	"encoding/json"
)

// ViewProvider produces a scoped state view of one chain for the
// execute-fork simulation (the pre-view before the proposal runs and the
// post-view after it). Implementations read only: they dial rpcURL — the
// fork's RPC endpoint — and return the raw, unformatted stategen document
// for the addresses in scope (the statediff differ's input shape, including
// its `_errors`/`_skipped` sections). Implementations must not mutate chain
// state.
//
// The provider is deliberately environment-agnostic: everything chain-
// specific (which addresses, which RPC) arrives through the arguments, so
// the same provider serves every environment and the fork URL swap is a
// parameter, not configuration.
type ViewProvider interface {
	ScopedView(ctx context.Context, scope Scope, rpcURL string) (json.RawMessage, error)
}
