package changeset

import (
	"context"
	"fmt"
	"time"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

const defaultHookTimeout = 30 * time.Second

// FailurePolicy determines how a hook error affects the pipeline.
type FailurePolicy int

const (
	// Abort causes a hook error to fail the pipeline.
	Abort FailurePolicy = 0
	// Warn causes a hook error to be logged while the pipeline continues.
	Warn FailurePolicy = 1
)

// String returns the string representation of a FailurePolicy.
func (fp FailurePolicy) String() string {
	switch fp {
	case Abort:
		return "Abort"
	case Warn:
		return "Warn"
	default:
		return fmt.Sprintf("FailurePolicy(%d)", int(fp))
	}
}

// HookContext is the read-only context passed to every hook function.
// For pre-hooks, Output and Err are nil. For post-hooks, they reflect
// the result of Apply.
type HookContext struct {
	Env          fdeployment.Environment
	ChangesetKey string
	Config       any
	Output       *fdeployment.ChangesetOutput // nil for pre-hooks
	Err          error                        // nil for pre-hooks; nil on success
	Timestamp    time.Time
}

// PreHookFunc is the signature for functions that run before changeset Apply.
type PreHookFunc func(ctx context.Context, hctx HookContext) error

// PostHookFunc is the signature for functions that run after changeset Apply.
type PostHookFunc func(ctx context.Context, hctx HookContext) error

// HookDefinition holds the metadata common to all hooks.
type HookDefinition struct {
	Name          string
	FailurePolicy FailurePolicy
	Timeout       time.Duration // zero means defaultHookTimeout (30s)
}

// PreHook pairs a HookDefinition with a PreHookFunc.
type PreHook struct {
	HookDefinition
	Fn PreHookFunc
}

// PostHook pairs a HookDefinition with a PostHookFunc.
type PostHook struct {
	HookDefinition
	Fn PostHookFunc
}

// executeHook runs a single hook function with the configured timeout and
// failure policy. It logs the outcome via hctx.Env.Logger.
//
// Returns nil when the hook succeeds or when the hook fails but the
// FailurePolicy is Warn. Returns the hook error only when the policy is Abort.
func executeHook(
	parentCtx context.Context,
	def HookDefinition,
	fn func(context.Context, HookContext) error,
	hctx HookContext,
) error {
	timeout := def.Timeout
	if timeout == 0 {
		timeout = defaultHookTimeout
	}

	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	start := time.Now()
	err := fn(ctx, hctx)
	duration := time.Since(start)

	if err != nil {
		hctx.Env.Logger.Warnw("hook failed",
			"hook", def.Name,
			"duration", duration,
			"policy", def.FailurePolicy.String(),
			"error", err,
		)

		if def.FailurePolicy == Warn {
			return nil
		}

		return err
	}

	hctx.Env.Logger.Infow("hook completed",
		"hook", def.Name,
		"duration", duration,
	)

	return nil
}
