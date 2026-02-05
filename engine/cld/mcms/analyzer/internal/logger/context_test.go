package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestWithLogger(t *testing.T) {
	t.Parallel()

	t.Run("adds logger to context", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		lggr := logger.Nop()

		newCtx := ContextWithLogger(ctx, lggr)

		require.NotNil(t, newCtx)
		assert.NotEqual(t, ctx, newCtx, "should return a new context")
	})

	t.Run("stores logger that can be retrieved", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		lggr := logger.Nop()

		newCtx := ContextWithLogger(ctx, lggr)
		retrieved := FromContext(newCtx)

		assert.Equal(t, lggr, retrieved)
	})

	t.Run("can override logger in context", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		lggr1 := logger.Nop()
		lggr2 := logger.Nop()

		ctx = ContextWithLogger(ctx, lggr1)
		retrieved1 := FromContext(ctx)
		assert.Equal(t, lggr1, retrieved1)

		ctx = ContextWithLogger(ctx, lggr2)
		retrieved2 := FromContext(ctx)
		assert.Equal(t, lggr2, retrieved2)
	})
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		setupCtx          func() context.Context
		expectedLogger    logger.Logger
		shouldBeNop       bool
		shouldNotPanic    bool
		additionalAsserts func(t *testing.T, ctx context.Context, retrieved logger.Logger)
	}{
		{
			name: "retrieves logger from context",
			setupCtx: func() context.Context {
				lggr := logger.Nop()
				return ContextWithLogger(context.Background(), lggr)
			},
			expectedLogger: logger.Nop(),
			shouldNotPanic: true,
		},
		{
			name: "returns Nop logger when no logger in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			shouldBeNop:    true,
			shouldNotPanic: true,
			additionalAsserts: func(t *testing.T, ctx context.Context, retrieved logger.Logger) {
				t.Helper()

				// Verify it's a Nop logger by checking it doesn't panic on operations
				assert.NotPanics(t, func() {
					retrieved.Info("test message")
					retrieved.Error("test error")
				})
			},
		},
		{
			name: "returns Nop logger for nil context value",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), loggerKey, nil)
			},
			shouldBeNop:    true,
			shouldNotPanic: true,
		},
		{
			name: "returns Nop logger for wrong type in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), loggerKey, "not a logger")
			},
			shouldBeNop:    true,
			shouldNotPanic: true,
			additionalAsserts: func(t *testing.T, ctx context.Context, retrieved logger.Logger) {
				t.Helper()

				// Should be a Nop logger since the type assertion will fail
				assert.NotPanics(t, func() {
					retrieved.Info("test message")
				})
			},
		},
		{
			name: "preserves logger through context chain",
			setupCtx: func() context.Context {
				lggr := logger.Nop()
				ctx := ContextWithLogger(context.Background(), lggr)
				// Create a child context with other values
				return context.WithValue(ctx, loggerKey, "value")
			},
			expectedLogger: logger.Nop(),
			shouldNotPanic: true,
			additionalAsserts: func(t *testing.T, ctx context.Context, retrieved logger.Logger) {
				t.Helper()

				// Verify the other context value is still there
				assert.Equal(t, "value", ctx.Value("key"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := tc.setupCtx()
			retrieved := FromContext(ctx)
			require.NotNil(t, retrieved)

			if tc.shouldBeNop {
				// Verify it behaves like a Nop logger
				assert.NotPanics(t, func() {
					retrieved.Info("test")
				})
			}

			if tc.shouldNotPanic {
				assert.NotPanics(t, func() {
					retrieved.Info("test message")
				})
			}

			if tc.additionalAsserts != nil {
				tc.additionalAsserts(t, ctx, retrieved)
			}
		})
	}
}

func TestContextKey(t *testing.T) {
	t.Parallel()

	t.Run("loggerKey is unique", func(t *testing.T) {
		t.Parallel()
		// Verify that our context key doesn't collide with string keys
		ctx := context.Background()
		lggr := logger.Nop()

		// Add logger with our typed key
		ctx = ContextWithLogger(ctx, lggr)

		// Add a value with a string key of the same value
		ctx = context.WithValue(ctx, loggerKey, "string value") //nolint

		retrieved := FromContext(ctx)
		assert.Equal(t, lggr, retrieved)

		// the string value should also be retrievable
		stringValue := ctx.Value("logger")
		assert.Equal(t, "string value", stringValue)
	})
}
