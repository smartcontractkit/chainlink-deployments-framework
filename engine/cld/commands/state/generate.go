package state

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
)

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
		# Generate state for staging environment (no output)
		myapp state generate -e staging

		# Generate and save state to default location
		myapp state generate -e staging --persist

		# Generate, save to custom path, and print to stdout
		myapp state generate -e staging -p -o /path/to/state.json --print

		# Generate using previous state for incremental updates
		myapp state generate -e mainnet -p --prev /path/to/old-state.json
	`)
)

// generateFlags holds all flags for the generate command.
type generateFlags struct {
	environment   string
	persist       bool
	output        string
	previousState string
	print         bool
}

// newGenerateCmd creates the "generate" subcommand for generating state.
func newGenerateCmd(cfg Config) *cobra.Command {
	var f generateFlags

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   generateShort,
		Long:    generateLong,
		Example: generateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(cmd, cfg, f)
		},
	}

	// Shared flags
	flags.Environment(cmd, &f.environment)
	flags.Print(cmd, &f.print)
	flags.Output(cmd, &f.output, "")

	// Local flags specific to this command
	cmd.Flags().BoolVarP(&f.persist, "persist", "p", false, "Persist state to disk")
	cmd.Flags().StringVarP(&f.previousState, "prev", "s", "", "Previous state file path")
	cmd.Flags().StringVar(&f.previousState, "previousState", "", "Previous state file path")
	_ = cmd.Flags().MarkDeprecated("previousState", "use --prev instead")

	return cmd
}

// runGenerate executes the generate command logic.
func runGenerate(cmd *cobra.Command, cfg Config, f generateFlags) error {
	deps := cfg.deps()
	envdir := cfg.Domain.EnvDir(f.environment)
	viewTimeout := 10 * time.Minute

	// --- Load all data first ---

	cmd.Printf("Generate latest state for %s in environment: %s\n", cfg.Domain, f.environment)
	cmd.Printf("This command may take a while to complete, please be patient. Timeout set to %v\n", viewTimeout)

	ctx, cancel := context.WithTimeout(cmd.Context(), viewTimeout)
	defer cancel()

	env, err := deps.EnvironmentLoader(ctx, cfg.Domain, f.environment, environment.WithLogger(cfg.Logger))
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	prevState, err := deps.StateLoader(envdir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load previous state: %w", err)
	}

	state, err := cfg.ViewState(env, prevState)
	if err != nil {
		return fmt.Errorf("unable to snapshot state: %w", err)
	}

	// --- Execute logic with loaded data ---

	if f.persist {
		if err := deps.StateSaver(envdir, f.output, state); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}

		outputPath := f.output
		if outputPath == "" {
			outputPath = envdir.ViewStateFilePath()
		}
		cmd.Printf("State saved to: %s\n", outputPath)
	}

	if f.print {
		b, err := state.MarshalJSON()
		if err != nil {
			return fmt.Errorf("unable to marshal state: %w", err)
		}
		cmd.Println(string(b))
	}

	return nil
}
