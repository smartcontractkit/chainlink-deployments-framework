package state

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

// newGenerateCmd creates the "generate" subcommand for generating state.
func newGenerateCmd(cfg Config) *cobra.Command {
	var (
		persist       bool
		outputPath    string
		prevStatePath string // NOTE: This flag is defined but not currently used in the original code.
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate latest state from the environment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(cmd, cfg, persist, outputPath, prevStatePath)
		},
	}

	// These flags are local to the generate command only.
	cmd.Flags().BoolVarP(&persist, "persist", "p", false, "Persist state to disk")
	cmd.Flags().StringVarP(&outputPath, "outputPath", "o", "", "Output path. Default is <product>/<environment>/state.json")
	cmd.Flags().StringVarP(&prevStatePath, "previousState", "s", "", "Previous state's path. Default is <product>/<environment>/state.json")

	return cmd
}

// runGenerate executes the generate command logic.
// This is separated from the RunE closure to improve testability.
// Note: prevStatePath is currently unused but kept for future implementation.
func runGenerate(cmd *cobra.Command, cfg Config, persist bool, outputPath, _ string) error {
	// Fail fast if ViewState is not provided
	if cfg.ViewState == nil {
		return errors.New("ViewState function is required but not provided")
	}

	deps := cfg.deps()

	// Get environment flag from parent command (persistent flag)
	envKey, _ := cmd.Flags().GetString("environment")
	envdir := cfg.Domain.EnvDir(envKey)

	// Set a timeout for the view operation as it may take a while
	viewTimeout := 10 * time.Minute

	cmd.Printf("Generate latest state for %s in environment: %s\n", cfg.Domain, envKey)
	cmd.Printf("This command may take a while to complete, please be patient. Timeout set to %v\n", viewTimeout)

	ctx, cancel := context.WithTimeout(cmd.Context(), viewTimeout)
	defer cancel()

	// Load the environment using the injected loader
	env, err := deps.EnvironmentLoader(ctx, cfg.Domain, envKey, environment.WithLogger(cfg.Logger))
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Load the previous state using the injected loader
	prevState, err := deps.StateLoader(envdir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load previous state: %w", err)
	}

	// Generate the new state using the provided ViewState function
	state, err := cfg.ViewState(env, prevState)
	if err != nil {
		return fmt.Errorf("unable to snapshot state: %w", err)
	}

	// Marshal state for output
	b, err := state.MarshalJSON()
	if err != nil {
		return fmt.Errorf("unable to marshal state: %w", err)
	}

	// Persist state if requested
	if persist {
		if err := deps.StateSaver(envdir, outputPath, state); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}
	}

	// Output the state to stdout
	cmd.Println(string(b))

	return nil
}
