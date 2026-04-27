package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/generate"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entrypoint. It uses a dedicated FlagSet so it can be
// called multiple times (e.g. in tests) without conflicting with the global
// flag.CommandLine that the testing harness parses before any test runs.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("operations-gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "operations_gen_config.yaml", "Path to config file")
	showVersion := fs.Bool("version", false, "Print version information and exit")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *showVersion {
		fmt.Fprintf(stdout, "operations-gen version=%s commit=%s date=%s\n", version, commit, date)
		return 0
	}

	if err := generate.GenerateFile(*configPath); err != nil {
		fmt.Fprintf(stderr, "Error generating operations: %v\n", err)
		return 1
	}

	return 0
}
