// Package cli provides a base struct for creating CLI applications using Cobra. It contains common
// functionality for creating CLI applications, such as providing a logger, adding commands and
// running the root command.
package cli

import (
	"os"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Base is a base struct for creating CLI applications using Cobra. This should be embedded into
// a struct that contains the specific commands for the CLI application.
type Base struct {
	Log logger.Logger

	rootCmd *cobra.Command
}

// NewBase creates a new CLIBase instance.
func NewBase(log logger.Logger, rootCmd *cobra.Command) *Base {
	return &Base{
		Log:     log,
		rootCmd: rootCmd,
	}
}

// AddCommand adds one or more commands to the root command of the CLI application.
func (base *Base) AddCommand(cmds ...*cobra.Command) {
	base.rootCmd.AddCommand(cmds...)
}

// Run executes the root command of the CLI application.
func (base *Base) Run() error {
	return base.rootCmd.Execute()
}

// RootCmd returns the root command of the CLI application.
func (base *Base) RootCmd() *cobra.Command {
	return base.rootCmd
}

// NewLogger creates a new logger instance from chainlink-common. This is a helper function to
// initialize a logger to provide to `NewBase`.
func NewLogger(level zapcore.Level) (logger.Logger, error) {
	c := logger.Config{Level: level}
	if os.Getenv("LOG_FORMAT") == "console" || os.Getenv("LOG_FORMAT") == "human" {
		return logger.NewWith(func(config *zap.Config) {
			config.Level.SetLevel(level)
			config.Development = true
			config.DisableStacktrace = true
			config.Encoding = "console"
			config.EncoderConfig = zap.NewDevelopmentEncoderConfig()
			config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		})
	}

	return c.New()
}
