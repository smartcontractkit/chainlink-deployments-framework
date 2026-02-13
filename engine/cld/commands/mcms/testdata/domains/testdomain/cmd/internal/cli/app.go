// Package cli provides the CLI App for interacting with the testdomain domain.
package cli

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"

	clilib "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
)

type App struct {
	*clilib.Base
}

func NewApp() (*App, error) {
	lggr, err := clilib.NewLogger(zapcore.DebugLevel)
	if err != nil {
		return nil, err
	}

	app := &App{
		Base: clilib.NewBase(lggr, &cobra.Command{
			Use:   "testdomain",
			Short: "Manage testdomain deployments",
		}),
	}

	// Add your Domain specific commands here
	app.AddCommand(
	//...
	)

	return app, nil
}
