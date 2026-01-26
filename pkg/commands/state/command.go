package state

import (
	"github.com/spf13/cobra"
)

// NewCommand creates a new state command with all subcommands.
// The command requires an environment flag (-e) which is used by all subcommands.
//
// Usage:
//
//	rootCmd.AddCommand(state.NewCommand(state.Config{
//	    Logger:    lggr,
//	    Domain:    myDomain,
//	    ViewState: myViewStateFunc,
//	}))
func NewCommand(cfg Config) *cobra.Command {
	// Apply defaults for optional dependencies
	cfg.deps()

	cmd := &cobra.Command{
		Use:   "state",
		Short: "State commands",
	}

	// Add subcommands
	cmd.AddCommand(newGenerateCmd(cfg))

	// The environment flag is persistent because all subcommands require it.
	// Currently there's only "generate", but this pattern supports future subcommands
	// that also need the environment context.
	cmd.PersistentFlags().
		StringP("environment", "e", "", "Deployment environment (required)")
	_ = cmd.MarkPersistentFlagRequired("environment")

	return cmd
}
