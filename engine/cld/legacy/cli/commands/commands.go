// Package commands contains common commands that can be injected into each domain's CLI
// application.
package commands

import (
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
)

// Commands provides a set of common commands that can be integrated into Domain-specific CLIs.
type Commands struct {
	lggr logger.Logger
}

// NewCommands creates a new instance of Commands.
func NewCommands(lggr logger.Logger) *Commands {
	return &Commands{lggr: lggr}
}
