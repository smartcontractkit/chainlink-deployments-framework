package timelockdelay

import (
	"context"
	"testing"
)

func testContext(t *testing.T) context.Context {
	t.Helper()

	return t.Context()
}

func cancelledTestContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	return ctx
}
