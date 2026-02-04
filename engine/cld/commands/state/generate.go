package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
)

// addPreviousStateAlias adds --previousState as deprecated alias for --prev.
func addPreviousStateAlias(cmd *cobra.Command) {
	existingNormalize := cmd.Flags().GetNormalizeFunc()
	cmd.Flags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "previousState" {
			return pflag.NormalizedName("prev")
		}
		if existingNormalize != nil {
			return existingNormalize(f, name)
		}

		return pflag.NormalizedName(name)
	})
}

var (
	generateShort = "Generate latest state from the environment"

	generateLong = cli.LongDesc(`
		Generate the latest deployment state by reading on-chain data.

		This command connects to the configured RPC endpoints for the environment,
		reads contract state, and produces a JSON representation of the current
		deployment. The operation may take several minutes for large deployments.

		By default, the generated state is not saved. Use --persist to save
		to disk, or --print to output the full JSON to stdout.
	`)

	generateExample = cli.Examples(`
		# Generate and print state for staging environment
		myapp state generate -e staging

		# Generate and save state to default location (also prints)
		myapp state generate -e staging --persist

		# Generate and save to custom path without printing
		myapp state generate -e staging -p -o /path/to/state.json --print=false

		# Generate using previous state for incremental updates
		myapp state generate -e mainnet -p --prev /path/to/old-state.json
	`)
)

// newGenerateCmd creates the "generate" subcommand for generating state.
func newGenerateCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   generateShort,
		Long:    generateLong,
		Example: generateExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGenerate(cmd, cfg)
		},
	}

	// Shared flags (use GetString/GetBool to retrieve)
	flags.Environment(cmd)
	flags.Print(cmd)
	flags.Output(cmd, "")

	// Local flags specific to this command
	cmd.Flags().BoolP("persist", "p", false, "Persist state to disk")
	cmd.Flags().StringP("prev", "s", "", "Previous state file path")

	// Deprecated alias: --previousState -> --prev
	addPreviousStateAlias(cmd)

	return cmd
}

// runGenerate executes the generate command logic.
func runGenerate(cmd *cobra.Command, cfg Config) error {
	// Get flag values
	envKey, _ := cmd.Flags().GetString("environment")
	persist, _ := cmd.Flags().GetBool("persist")
	output, _ := cmd.Flags().GetString("out")
	previousState, _ := cmd.Flags().GetString("prev")
	shouldPrint, _ := cmd.Flags().GetBool("print")

	deps := cfg.deps()
	envdir := cfg.Domain.EnvDir(envKey)
	viewTimeout := 10 * time.Minute

	// --- Load all data first ---

	cmd.Printf("Generate latest state for %s in environment: %s\n", cfg.Domain, envKey)
	cmd.Printf("This command may take a while to complete, please be patient. Timeout set to %v\n", viewTimeout)

	ctx, cancel := context.WithTimeout(cmd.Context(), viewTimeout)
	defer cancel()

	env, err := deps.EnvironmentLoader(ctx, cfg.Domain, envKey, environment.WithLogger(cfg.Logger))
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	var prevState domain.JSONSerializer
	if previousState != "" {
		// Load from custom path specified by --prev flag
		data, readErr := os.ReadFile(previousState)
		if readErr != nil {
			return fmt.Errorf("failed to load previous state from %s: %w", previousState, readErr)
		}
		raw := json.RawMessage(data)
		prevState = &raw
	} else {
		// Load from default envdir location
		var loadErr error
		prevState, loadErr = deps.StateLoader(envdir)
		if loadErr != nil && !os.IsNotExist(loadErr) {
			return fmt.Errorf("failed to load previous state: %w", loadErr)
		}
	}

	state, err := cfg.ViewState(env, prevState)
	if err != nil {
		return fmt.Errorf("unable to snapshot state: %w", err)
	}

	// --- Execute logic with loaded data ---

	if persist {
		if err := deps.StateSaver(envdir, output, state); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}

		outputPath := output
		if outputPath == "" {
			outputPath = envdir.ViewStateFilePath()
		}
		cmd.Printf("State saved to: %s\n", outputPath)
	}

	if shouldPrint {
		b, err := state.MarshalJSON()
		if err != nil {
			return fmt.Errorf("unable to marshal state: %w", err)
		}
		cmd.Println(string(b))
	}

	return nil
}
