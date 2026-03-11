package pipeline

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/pipeline/input"
	dprun "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/pipeline/run"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

var (
	runShort = "Run a durable pipeline changeset"

	runLong = `
		Run a durable pipeline changeset.

		This command applies a changeset against the specified environment,
		resolves any timelock proposals, and persists artifacts.
`

	runExample = `
		# Dry-run of changeset 0001_test_changeset in testnet
		chainlink-deployments durable-pipeline run \
  		--environment testnet \
  		--changeset 0001_test_changeset \
  		--input-file inputs.yaml \
  		--dry-run

		# Run changeset by name with input file
		chainlink-deployments durable-pipeline run \
  		--environment testnet \
  		--changeset 0001_test_changeset \
  		--input-file inputs.yaml

		# Run changeset by index position with array format input file.
		chainlink-deployments durable-pipeline run \
  		--environment testnet \
  		--input-file inputs.yaml \
  		--changeset-index 0
`
)

type runFlags struct {
	environment    string
	changeset      string
	dryRun         bool
	inputFile      string
	changesetIndex int
}

func newRunCmd(cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   runShort,
		Long:    runLong,
		Example: runExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := runFlags{
				environment:    flags.MustString(cmd.Flags().GetString("environment")),
				changeset:      flags.MustString(cmd.Flags().GetString("changeset")),
				dryRun:         flags.MustBool(cmd.Flags().GetBool("dry-run")),
				inputFile:      flags.MustString(cmd.Flags().GetString("input-file")),
				changesetIndex: flags.MustInt(cmd.Flags().GetInt("changeset-index")),
			}

			return runRun(cmd, cfg, f)
		},
	}

	flags.Environment(cmd)
	cmd.Flags().BoolP("dry-run", "d", false, "Use a read-only JD backend. WARNING: still uses real chain clients as of now!")
	cmd.Flags().StringP("changeset", "c", "", "changeset to apply by name")
	cmd.Flags().StringP("input-file", "i", "", "YAML input file name. Not the full path, just the name")
	cmd.Flags().IntP("changeset-index", "x", 0, "Index of changeset to run by position in array format input file")

	_ = cmd.MarkFlagRequired("input-file")
	cmd.MarkFlagsMutuallyExclusive("changeset", "changeset-index")
	cmd.MarkFlagsOneRequired("changeset", "changeset-index")

	return cmd
}

func runRun(cmd *cobra.Command, cfg *Config, f runFlags) error {
	envdir := cfg.Domain.EnvDir(f.environment)
	artdir := envdir.ArtifactsDir()

	var actualChangesetName string

	if f.changeset != "" {
		actualChangesetName = f.changeset
		if err := input.PrepareInputForRunByName(f.inputFile, f.changeset, cfg.Domain, f.environment); err != nil {
			return fmt.Errorf("failed to parse input file: %w", err)
		}
	} else {
		var err error
		actualChangesetName, err = input.PrepareInputForRunByIndex(f.inputFile, f.changesetIndex, cfg.Domain, f.environment)
		if err != nil {
			return fmt.Errorf("failed to get changeset at index %d: %w", f.changesetIndex, err)
		}
	}

	if err := artdir.SetDurablePipelines(strconv.FormatInt(time.Now().UnixNano(), 10)); err != nil {
		return err
	}

	registry, err := cfg.LoadChangesets(f.environment)
	if err != nil {
		return err
	}

	envOptions, err := dprun.ConfigureEnvironmentOptions(registry, actualChangesetName, f.dryRun, cfg.Logger)
	if err != nil {
		return err
	}

	regCfg, err := registry.GetConfigurations(actualChangesetName)
	if err != nil {
		return fmt.Errorf("failed to get configurations for %s: %w", actualChangesetName, err)
	}

	if regCfg.ConfigResolver != nil {
		if cfg.ConfigResolverManager.NameOf(regCfg.ConfigResolver) == "" {
			return fmt.Errorf("resolver for %s is not registered", actualChangesetName)
		}
	}

	reports, err := artdir.LoadOperationsReports(actualChangesetName)
	if err != nil {
		return fmt.Errorf("failed to load operations report: %w", err)
	}

	originalReportsLen := len(reports)
	cfg.Logger.Infof("Loaded %d operations reports", originalReportsLen)
	reporter := operations.NewMemoryReporter(operations.WithReports(reports))

	envOptions = append(envOptions, environment.WithReporter(reporter))
	deps := cfg.deps()
	env, err := deps.EnvironmentLoader(cmd.Context(), cfg.Domain, f.environment, envOptions...)
	if err != nil {
		return err
	}

	indexStr := ""
	if f.changeset == "" {
		indexStr = fmt.Sprintf(" (at index %d)", f.changesetIndex)
	}
	cfg.Logger.Infof("Applying %s durable pipeline for changeset %s%s for environment: %s\n",
		cfg.Domain, actualChangesetName, indexStr, f.environment,
	)

	out, err := registry.Apply(actualChangesetName, env)
	if saveErr := dprun.SaveReports(reporter, originalReportsLen, cfg.Logger, artdir, actualChangesetName); saveErr != nil {
		cfg.Logger.Errorf("failed to save reports: %v", saveErr)
	}
	if err != nil {
		return err
	}

	if len(out.DescribedTimelockProposals) == 0 && cfg.DecodeProposalCtxProvider != nil {
		out.DescribedTimelockProposals = make([]string, len(out.MCMSTimelockProposals))
		proposalContext, err := cfg.DecodeProposalCtxProvider(env)
		if err != nil {
			return err
		}
		for idx, proposal := range out.MCMSTimelockProposals {
			describedProposal, err := analyzer.DescribeTimelockProposal(cmd.Context(), proposalContext, env, &proposal)
			if err != nil {
				cfg.Logger.Errorf("failed to describe time lock proposal %d: %v", idx, err)
				continue
			}
			out.DescribedTimelockProposals[idx] = describedProposal
		}
	}

	if err := artdir.SaveChangesetOutput(actualChangesetName, out); err != nil {
		cfg.Logger.Errorf("failed to save changeset artifacts: %v", err)
		return err
	}

	return nil
}
