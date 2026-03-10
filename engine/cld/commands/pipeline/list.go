package pipeline

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
)

var (
	listShort = "List durable pipeline info"

	listLong = `
		List durable pipeline info.

		Displays registered changesets (static vs dynamic) and available resolvers
		for the given environment.
	`

	listExample = `
		# List durable pipeline info for testnet
		chainlink-deployments durable-pipeline list --environment testnet
	`
)

type listFlags struct {
	environment string
}

func newListCmd(cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   listShort,
		Long:    listLong,
		Example: listExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := listFlags{
				environment: flags.MustString(cmd.Flags().GetString("environment")),
			}

			return runList(cmd, cfg, f)
		},
	}

	flags.Environment(cmd)

	return cmd
}

func runList(cmd *cobra.Command, cfg *Config, f listFlags) error {
	registry, err := cfg.LoadChangesets(f.environment)
	if err != nil {
		return fmt.Errorf("failed to load changesets registry: %w", err)
	}

	changesets := registry.ListKeys()
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "\n=== Durable Pipeline Info for %s ===\n", cfg.Domain.String())
	fmt.Fprintf(out, "\nLegend: DYNAMIC = config resolver | STATIC = YAML input | ERROR = misconfigured\n")

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "\nRegistered Changesets:\n")
	fmt.Fprintf(w, "TYPE\tNAME\tCONFIG SOURCE\n")
	fmt.Fprintf(w, "----\t----\t-------------\n")

	for _, changeset := range changesets {
		regCfg, err := registry.GetConfigurations(changeset)
		if err != nil {
			return fmt.Errorf("get configurations for %s: %w", changeset, err)
		}
		res := regCfg.ConfigResolver

		if res == nil {
			fmt.Fprintf(w, "STATIC\t%s\tYAML input file\n", changeset)
		} else {
			resolverName := cfg.ConfigResolverManager.NameOf(res)
			if resolverName == "" {
				fmt.Fprintf(w, "ERROR\t%s\tResolver not registered\n", changeset)
			} else {
				parts := strings.Split(resolverName, ".")
				shortName := parts[len(parts)-1]
				fmt.Fprintf(w, "DYNAMIC\t%s\t%s\n", changeset, shortName)
			}
		}
	}

	w.Flush()

	allResolvers := cfg.ConfigResolverManager.ListResolvers()
	fmt.Fprintf(out, "\nAvailable Config Resolvers:\n")
	for _, resolver := range allResolvers {
		parts := strings.Split(resolver, ".")
		shortName := parts[len(parts)-1]
		fmt.Fprintf(out, "  • %s\n", shortName)
	}

	return nil
}
