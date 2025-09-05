package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
)

// NewEnvCmds creates a new set of commands for managing environment.
func (c Commands) NewEnvCmds(
	domain domain.Domain,
) *cobra.Command {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Env commands",
	}

	envCmd.AddCommand(
		c.newEnvLoad(domain),
	)

	envCmd.PersistentFlags().
		StringP("environment", "e", "", "Deployment environment (required)")
	_ = envCmd.MarkPersistentFlagRequired("environment")

	return envCmd
}

var (
	envLoadLong = cli.LongDesc(`
		Runs a sanity check by loading the environment configuration and verifying connectivity.
`)

	envLoadExample = cli.Examples(`
  		# Verify that the staging environment loads correctly
  		exemplar env load --environment staging
`)
)

// newEnvLoad creates the "load" subcommand for environment checks.
func (c Commands) newEnvLoad(domain domain.Domain) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "load",
		Short:   "Runs load environment sanity check",
		Long:    envLoadLong,
		Example: envLoadExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			_, err := environment.Load(
				cmd.Context,
				c.lggr,
				envKey,
				domain,
				true,
			)
			if err != nil {
				return fmt.Errorf("LoadEnvironment failed: %w", err)
			}
			cmd.Println("✅ Environment loaded successfully.")

			return nil
		},
	}

	return cmd
}
