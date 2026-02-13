package main

import (
	"os"

	"github.com/smartcontractkit/chainlink-deployments/domains/testdomain/cmd/internal/cli"
)

func main() {
	app, err := cli.NewApp()
	if err != nil {
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
