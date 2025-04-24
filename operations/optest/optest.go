// Package optest provides utilities for operations testing.
package optest

import (
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// NewBundle creates a new operations bundle for testing with a no-op logger
// and a memory reporter.
func NewBundle(t *testing.T) operations.Bundle {
	t.Helper()

	return operations.NewBundle(
		t.Context, logger.Nop(), operations.NewMemoryReporter(),
	)
}
