package pipeline

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/pipeline/template"
)

var (
	templateInputShort = "Generate YAML input templates from Changesets"

	templateInputLong = `
		Generate YAML input templates from Changeset input Go struct types.

		This command helps create YAML input files by analyzing Go struct types
		from changesets and generating properly formatted YAML templates with
		example values and comments.
	`

	templateInputExample = `
		# Generate YAML template for a single changeset
		chainlink-deployments durable-pipeline template-input \
		  --environment testnet \
		  --changeset test_changeset_dynamic_inputs

		# Generate YAML template for multiple changesets
		chainlink-deployments durable-pipeline template-input \
		  --environment testnet \
		  --changeset changeset1,changeset2,changeset3

		# Configure depth limit for nested structures
		chainlink-deployments durable-pipeline template-input \
		  --environment testnet \
		  --changeset test_changeset_dynamic_inputs \
		  --depth 3

		# Save output to file
		chainlink-deployments durable-pipeline template-input \
		  --environment testnet \
		  --changeset test_changeset_dynamic_inputs > example.yaml
	`
)

type templateInputFlags struct {
	environment string
	changeset   string
	depthLimit  int
}

func newTemplateInputCmd(cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template-input",
		Short:   templateInputShort,
		Long:    templateInputLong,
		Example: templateInputExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := templateInputFlags{
				environment: flags.MustString(cmd.Flags().GetString("environment")),
				changeset:   flags.MustString(cmd.Flags().GetString("changeset")),
				depthLimit:  flags.MustInt(cmd.Flags().GetInt("depth")),
			}

			return runTemplateInput(cmd, cfg, f)
		},
	}

	flags.Environment(cmd)
	cmd.Flags().StringP("changeset", "c", "", "Changeset name(s) to generate YAML template for - comma-separated for multiple (required)")
	cmd.Flags().IntP("depth", "d", 5, "Maximum recursion depth generation for nested struct, configure this based on your struct complexity")

	_ = cmd.MarkFlagRequired("changeset")

	return cmd
}

func runTemplateInput(cmd *cobra.Command, cfg *Config, f templateInputFlags) error {
	registry, err := cfg.LoadChangesets(f.environment)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	changesetNames := strings.Split(strings.TrimSpace(f.changeset), ",")
	for i, name := range changesetNames {
		changesetNames[i] = strings.TrimSpace(name)
	}

	yamlTemplate, err := template.GenerateMultiChangesetYAML(
		cfg.Domain.String(),
		f.environment,
		changesetNames,
		registry,
		cfg.ConfigResolverManager,
		f.depthLimit,
	)
	if err != nil {
		return fmt.Errorf("generate YAML template: %w", err)
	}

	fmt.Fprint(cmd.OutOrStdout(), yamlTemplate)

	return nil
}
