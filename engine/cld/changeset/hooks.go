package changeset

import (
	"context"
	"fmt"
	"time"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// DefaultHookTimeout is applied when a HookDefinition has a zero Timeout.
const DefaultHookTimeout = 30 * time.Second

// FailurePolicy determines how a hook error affects the pipeline.
type FailurePolicy int

const (
	// Abort causes a hook error to fail the pipeline.
	Abort FailurePolicy = iota
	// Warn causes a hook error to be logged while the pipeline continues.
	Warn
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

// HookEnv is the restricted environment surface exposed to hooks.
// Additional fields may be added in future versions as needs arise.
type HookEnv struct {
	Name   string
	Logger logger.Logger
}

// PreHookParams is passed to pre-hooks.
// All fields must be treated as read-only.
type PreHookParams struct {
	Env          HookEnv
	ChangesetKey string
	Config       any
}

// PostHookParams is passed to post-hooks.
// All fields must be treated as read-only.
type PostHookParams struct {
	Env          HookEnv
	ChangesetKey string
	Config       any
	Output       fdeployment.ChangesetOutput
	Err          error
}

// PreHookFunc is the signature for functions that run before changeset Apply.
// The context is derived from env.GetContext() with the hook's timeout applied.
type PreHookFunc func(ctx context.Context, params PreHookParams) error

// PostHookFunc is the signature for functions that run after changeset Apply.
// The context is derived from env.GetContext() with the hook's timeout applied.
type PostHookFunc func(ctx context.Context, params PostHookParams) error

// HookDefinition holds the metadata common to all hooks.
type HookDefinition struct {
	Name          string
	FailurePolicy FailurePolicy
	Timeout       time.Duration // zero means DefaultHookTimeout (30s)
}

// PreHook pairs a HookDefinition with a PreHookFunc.
type PreHook struct {
	HookDefinition
	Func PreHookFunc
}

// PostHook pairs a HookDefinition with a PostHookFunc.
type PostHook struct {
	HookDefinition
	Func PostHookFunc
}

// ExecuteHook runs a hook function with the configured timeout and failure
// policy. The parent context is derived from env.GetContext(); each hook
// receives a child context with its timeout applied.
//
// Returns nil when the hook succeeds or when the hook fails but the
// FailurePolicy is Warn. Returns the hook error only when the policy is Abort.
func ExecuteHook(
	env fdeployment.Environment,
	def HookDefinition,
	fn func(ctx context.Context) error,
) error {
	timeout := def.Timeout
	if timeout == 0 {
		timeout = DefaultHookTimeout
	}

	ctx, cancel := context.WithTimeout(env.GetContext(), timeout)
	defer cancel()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	if err != nil {
		env.Logger.Warnw("hook failed",
			"hook", def.Name,
			"duration", duration,
			"policy", def.FailurePolicy.String(),
			"error", err,
		)
	} else {
		env.Logger.Infow("hook completed",
			"hook", def.Name,
			"duration", duration,
		)
	}

	if err != nil && def.FailurePolicy == Warn {
		return nil
	}

	return err
}
