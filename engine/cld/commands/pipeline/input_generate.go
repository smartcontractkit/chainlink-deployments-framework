package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/pipeline/input"
)

var (
	inputGenerateShort = "Generate durable-pipeline input using registered config resolvers"

	inputGenerateLong = `
		Generate durable-pipeline input configurations.

		Reads an inputs file, resolves each changeset via registered config resolvers,
		and outputs the resulting config in YAML or JSON.
`

	inputGenerateExample = `
		# Generate config from inputs.yaml and print
		chainlink-deployments durable-pipeline input-generate \
  			--environment testnet \
  			--inputs inputs.yaml

		# Write JSON output to file
		chainlink-deployments durable-pipeline input-generate \
		  --environment testnet \
		  --inputs inputs.yaml \
		  --json \
		  --output config.json
	`
)

type inputGenerateFlags struct {
	environment  string
	inputs       string
	output       string
	formatAsJSON bool
}

func newInputGenerateCmd(cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "input-generate",
		Short:   inputGenerateShort,
		Long:    inputGenerateLong,
		Example: inputGenerateExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := inputGenerateFlags{
				environment:  flags.MustString(cmd.Flags().GetString("environment")),
				inputs:       flags.MustString(cmd.Flags().GetString("inputs")),
				output:       flags.MustString(cmd.Flags().GetString("output")),
				formatAsJSON: flags.MustBool(cmd.Flags().GetBool("json")),
			}

			return runInputGenerate(cmd, cfg, f)
		},
	}

	flags.Environment(cmd)
	cmd.Flags().StringP("inputs", "i", "", "Inputs file name (required)")
	cmd.Flags().BoolP("json", "j", false, "Emit JSON instead of YAML")
	cmd.Flags().StringP("output", "o", "", "Output file path (optional; prints to stdout if omitted)")

	_ = cmd.MarkFlagRequired("inputs")

	return cmd
}

func runInputGenerate(cmd *cobra.Command, cfg *Config, f inputGenerateFlags) error {
	registry, err := cfg.LoadChangesets(f.environment)
	if err != nil {
		return fmt.Errorf("load changesets registry: %w", err)
	}

	output, err := input.Generate(input.GenerateOptions{
		InputsFileName:  f.inputs,
		Domain:          cfg.Domain,
		EnvKey:          f.environment,
		Registry:        registry,
		ResolverManager: cfg.ConfigResolverManager,
		FormatAsJSON:    f.formatAsJSON,
		OutputPath:      f.output,
	})
	if err != nil {
		return err
	}

	if f.output != "" {
		format := "YAML"
		if f.formatAsJSON {
			format = "JSON"
		}
		cfg.Logger.Infof("Generated %s config written to: %s", format, f.output)
	} else {
		fmt.Fprint(cmd.OutOrStdout(), output)
	}

	return nil
}
