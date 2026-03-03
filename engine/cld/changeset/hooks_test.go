package changeset

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestFailurePolicy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		policy FailurePolicy
		want   string
	}{
		{name: "Abort", policy: Abort, want: "Abort"},
		{name: "Warn", policy: Warn, want: "Warn"},
		{name: "unknown value", policy: FailurePolicy(42), want: "FailurePolicy(42)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.policy.String())
		})
	}
}

func TestExecuteHook_PassingHook_ReturnsNil(t *testing.T) {
	t.Parallel()

	env := hookTestEnv(t)
	def := HookDefinition{Name: "passing-hook", FailurePolicy: Abort}

	err := ExecuteHook(env, def, func(_ context.Context) error {
		return nil
	})

	require.NoError(t, err)
}

func TestExecuteHook_FailingHook_Abort_ReturnsError(t *testing.T) {
	t.Parallel()

	env := hookTestEnv(t)
	def := HookDefinition{Name: "abort-hook", FailurePolicy: Abort}
	hookErr := errors.New("something broke")

	err := ExecuteHook(env, def, func(_ context.Context) error {
		return hookErr
	})

	require.Error(t, err)
	assert.Equal(t, hookErr, err)
}

func TestExecuteHook_FailingHook_Warn_ReturnsNil(t *testing.T) {
	t.Parallel()

	env := hookTestEnv(t)
	def := HookDefinition{Name: "warn-hook", FailurePolicy: Warn}

	err := ExecuteHook(env, def, func(_ context.Context) error {
		return errors.New("non-critical failure")
	})

	require.NoError(t, err, "Warn policy should swallow the error")
}

func TestExecuteHook_ZeroTimeout_AppliesDefault(t *testing.T) {
	t.Parallel()

	env := hookTestEnv(t)
	def := HookDefinition{Name: "default-timeout", FailurePolicy: Abort, Timeout: 0}

	var receivedDeadline time.Time
	var hasDeadline bool

	err := ExecuteHook(env, def, func(ctx context.Context) error {
		receivedDeadline, hasDeadline = ctx.Deadline()
		return nil
	})

	require.NoError(t, err)
	require.True(t, hasDeadline, "context should have a deadline when Timeout is 0")

	expectedDeadline := time.Now().Add(DefaultHookTimeout)
	assert.WithinDuration(t, expectedDeadline, receivedDeadline, 2*time.Second,
		"deadline should be ~30s from now (DefaultHookTimeout)")
}

func TestExecuteHook_CustomTimeout_Applied(t *testing.T) {
	t.Parallel()

	customTimeout := 5 * time.Second
	env := hookTestEnv(t)
	def := HookDefinition{Name: "custom-timeout", FailurePolicy: Abort, Timeout: customTimeout}

	var receivedDeadline time.Time
	var hasDeadline bool

	err := ExecuteHook(env, def, func(ctx context.Context) error {
		receivedDeadline, hasDeadline = ctx.Deadline()
		return nil
	})

	require.NoError(t, err)
	require.True(t, hasDeadline, "context should have a deadline")

	expectedDeadline := time.Now().Add(customTimeout)
	assert.WithinDuration(t, expectedDeadline, receivedDeadline, 2*time.Second,
		"deadline should be ~5s from now (custom timeout)")
}

func TestExecuteHook_ExceedsTimeout_Abort_ReturnsDeadlineExceeded(t *testing.T) {
	t.Parallel()

	env := hookTestEnv(t)
	def := HookDefinition{
		Name:          "slow-hook-abort",
		FailurePolicy: Abort,
		Timeout:       50 * time.Millisecond,
	}

	err := ExecuteHook(env, def, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestExecuteHook_ExceedsTimeout_Warn_SwallowsError(t *testing.T) {
	t.Parallel()

	env := hookTestEnv(t)
	def := HookDefinition{
		Name:          "slow-hook-warn",
		FailurePolicy: Warn,
		Timeout:       50 * time.Millisecond,
	}

	err := ExecuteHook(env, def, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	require.NoError(t, err, "Warn policy should swallow deadline exceeded error")
}

func TestExecuteHook_ParentContextDeadlineRespected(t *testing.T) {
	t.Parallel()

	parentCtx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	env := fdeployment.Environment{
		Name:       "test-env",
		Logger:     logger.Test(t),
		GetContext: func() context.Context { return parentCtx },
	}
	def := HookDefinition{
		Name:          "parent-deadline",
		FailurePolicy: Abort,
		Timeout:       10 * time.Second,
	}

	err := ExecuteHook(env, def, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded,
		"parent context's shorter deadline should take effect")
}
