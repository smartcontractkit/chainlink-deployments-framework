package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/pipeline/input"
	dprun "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/pipeline/run"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

var (
	runShort = "Run a pipeline changeset"

	runLong = `
		Run a pipeline changeset.

		This command applies a changeset against the specified environment,
		resolves any timelock proposals, and persists artifacts.
`

	runExample = `
		# Dry-run of changeset 0001_test_changeset in testnet
		chainlink-deployments pipeline run \
  		--environment testnet \
  		--changeset 0001_test_changeset \
  		--input-file inputs.yaml \
  		--dry-run

		# Run changeset by name with input file
		chainlink-deployments pipeline run \
  		--environment testnet \
  		--changeset 0001_test_changeset \
  		--input-file inputs.yaml

		# Run changeset by index position with array format input file.
		chainlink-deployments pipeline run \
  		--environment testnet \
  		--input-file inputs.yaml \
  		--changeset-index 0

		# Run all changesets sequentially defined in the input file
		chainlink-deployments pipeline run \
  		--environment testnet \
  		--input-file inputs.yaml \
  		--all
`
)

type runFlags struct {
	environment    string
	changeset      string
	dryRun         bool
	inputFile      string
	changesetIndex int
	all            bool
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
				all:            flags.MustBool(cmd.Flags().GetBool("all")),
			}

			return runRun(cmd, cfg, f)
		},
	}

	flags.Environment(cmd)
	cmd.Flags().BoolP("dry-run", "d", false, "Use a read-only JD backend. WARNING: still uses real chain clients as of now!")
	cmd.Flags().StringP("changeset", "c", "", "changeset to apply by name")
	cmd.Flags().StringP("input-file", "i", "", "YAML input file name. Not the full path, just the name")
	cmd.Flags().IntP("changeset-index", "x", 0, "Index of changeset to run by position in array format input file")
	cmd.Flags().BoolP("all", "a", false, "Run all changesets defined in the input file in order")

	_ = cmd.MarkFlagRequired("input-file")
	cmd.MarkFlagsMutuallyExclusive("changeset", "changeset-index")
	cmd.MarkFlagsMutuallyExclusive("changeset", "all")
	cmd.MarkFlagsMutuallyExclusive("changeset-index", "all")
	cmd.MarkFlagsOneRequired("changeset", "changeset-index", "all")

	return cmd
}

func runRun(cmd *cobra.Command, cfg *Config, f runFlags) error {
	envdir := cfg.Domain.EnvDir(f.environment)
	artdir := envdir.ArtifactsDir()
	deps := cfg.deps()

	if f.all {
		return runAllChangesets(cmd, cfg, f, artdir, deps)
	}

	return runSingleChangeset(cmd, cfg, f, artdir, deps)
}

// runSingleChangeset runs a single changeset specified by name or index. It sets the
// DURABLE_PIPELINE_INPUT env var for the changeset so that it can be accessed by the registry
// and resolvers, then loads the registry and applies the changeset.
func runSingleChangeset(
	cmd *cobra.Command,
	cfg *Config,
	f runFlags,
	artdir *domain.ArtifactsDir,
	deps *Deps,
) error {
	if err := artdir.SetDurablePipelines(strconv.FormatInt(time.Now().UnixNano(), 10)); err != nil {
		return err
	}

	var changesetName string

	if f.changeset != "" {
		changesetName = f.changeset
		if err := input.PrepareInputForRunByName(f.inputFile, f.changeset, cfg.Domain, f.environment); err != nil {
			return fmt.Errorf("failed to parse input file: %w", err)
		}
	} else {
		var err error
		changesetName, err = input.PrepareInputForRunByIndex(f.inputFile, f.changesetIndex, cfg.Domain, f.environment)
		if err != nil {
			return fmt.Errorf("failed to get changeset at index %d: %w", f.changesetIndex, err)
		}
	}

	registry, err := cfg.LoadChangesets(f.environment)
	if err != nil {
		return err
	}

	indexStr := ""
	if f.changeset == "" {
		indexStr = fmt.Sprintf(" (at index %d)", f.changesetIndex)
	}
	cfg.Logger.Infof("Applying %s pipeline for changeset %s%s for environment: %s",
		cfg.Domain, changesetName, indexStr, f.environment,
	)

	return applyChangeset(cmd, cfg, f.dryRun, f.environment, changesetName, registry, artdir, deps)
}

// runAllChangesets runs all changesets defined in the input file sequentially. It sets the
// DURABLE_PIPELINE_INPUT env var for each changeset before applying,
// then loads the registry and applies each changeset in order.
func runAllChangesets(
	cmd *cobra.Command,
	cfg *Config,
	f runFlags,
	artdir *domain.ArtifactsDir,
	deps *Deps,
) error {
	dpYAML, err := input.ParseDurablePipelineYAML(f.inputFile, cfg.Domain, f.environment)
	if err != nil {
		return fmt.Errorf("failed to parse input file: %w", err)
	}

	changesets, err := input.GetAllChangesetsInOrder(dpYAML.Changesets)
	if err != nil {
		return fmt.Errorf("failed to read changesets from input file: %w", err)
	}
	if len(changesets) == 0 {
		return errors.New("no changesets found in input file")
	}

	cfg.Logger.Infof("Applying %s pipeline for all %d changesets for environment: %s",
		cfg.Domain, len(changesets), f.environment,
	)

	for i, cs := range changesets {
		cfg.Logger.Infof("[%d/%d] Applying changeset %s", i+1, len(changesets), cs.Name)

		if err := artdir.SetDurablePipelines(strconv.FormatInt(time.Now().UnixNano(), 10)); err != nil {
			return err
		}

		if err := input.SetChangesetEnvironmentVariable(cs.Name, cs.Data); err != nil {
			return fmt.Errorf("changeset %s: failed to set input: %w", cs.Name, err)
		}

		registry, err := cfg.LoadChangesets(f.environment)
		if err != nil {
			return fmt.Errorf("[%d/%d] changeset %s: failed to load changesets: %w", i+1, len(changesets), cs.Name, err)
		}

		if err := applyChangeset(cmd, cfg, f.dryRun, f.environment, cs.Name, registry, artdir, deps); err != nil {
			return fmt.Errorf("[%d/%d] changeset %s: %w", i+1, len(changesets), cs.Name, err)
		}
	}

	cfg.Logger.Infof("Successfully applied all %d changesets for environment: %s", len(changesets), f.environment)

	return nil
}

func applyChangeset(
	cmd *cobra.Command,
	cfg *Config,
	dryRun bool,
	envName string,
	changesetName string,
	registry *changeset.ChangesetsRegistry,
	artdir *domain.ArtifactsDir,
	deps *Deps,
) error {
	envOptions, err := dprun.ConfigureEnvironmentOptions(registry, changesetName, dryRun, cfg.Logger)
	if err != nil {
		return err
	}

	regCfg, err := registry.GetConfigurations(changesetName)
	if err != nil {
		return fmt.Errorf("failed to get configurations for %s: %w", changesetName, err)
	}

	if regCfg.ConfigResolver != nil {
		if cfg.ConfigResolverManager.NameOf(regCfg.ConfigResolver) == "" {
			return fmt.Errorf("resolver for %s is not registered", changesetName)
		}
	}

	reports, err := artdir.LoadOperationsReports(changesetName)
	if err != nil {
		return fmt.Errorf("failed to load operations report: %w", err)
	}

	originalReportsLen := len(reports)
	cfg.Logger.Infof("Loaded %d operations reports", originalReportsLen)
	reporter := operations.NewMemoryReporter(operations.WithReports(reports))

	envOptions = append(envOptions, environment.WithReporter(reporter))
	env, err := deps.EnvironmentLoader(cmd.Context(), cfg.Domain, envName, envOptions...)
	if err != nil {
		return err
	}

	out, err := registry.Apply(changesetName, env)
	var saveErr error
	if saveErr = dprun.SaveReports(reporter, originalReportsLen, cfg.Logger, artdir, changesetName); saveErr != nil {
		cfg.Logger.Errorf("failed to save reports: %v", saveErr)
	}
	if err != nil {
		return err
	}
	if saveErr != nil {
		return saveErr
	}

	err = saveChangesetProposalMetadata(registry, changesetName, out)
	if err != nil {
		return fmt.Errorf("failed to save changeset proposal metadata: %w", err)
	}

	// TODO: proposal decoding is handled by the CLD GH workflows; this should be removed.
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

	if err := artdir.SaveChangesetOutput(changesetName, out); err != nil {
		cfg.Logger.Errorf("failed to save changeset artifacts: %v", err)
		return err
	}

	return nil
}

func saveChangesetProposalMetadata(
	registry *changeset.ChangesetsRegistry, changesetName string, out fdeployment.ChangesetOutput,
) error {
	if len(out.MCMSTimelockProposals) == 0 {
		return nil
	}

	changesetInputJSON := os.Getenv("DURABLE_PIPELINE_INPUT")
	if len(changesetInputJSON) == 0 {
		return errors.New("durable pipeline input is empty or not set")
	}

	changesetConfig, err := registry.GetResolvedInput(changesetName, changesetInputJSON)
	if err != nil {
		return fmt.Errorf("failed to get changeset configuration: %w", err)
	}

	id := uuid.NewString()

	for i := range out.MCMSTimelockProposals {
		proposal := &out.MCMSTimelockProposals[i]
		if proposal.Metadata == nil {
			proposal.Metadata = map[string]any{}
		}

		proposal.Metadata["changesets"] = []struct {
			ID     string          `json:"id"`
			Name   string          `json:"name"`
			Input  json.RawMessage `json:"input"`
			Config any             `json:"config"`
		}{{
			ID:     id,
			Name:   changesetName,
			Input:  json.RawMessage(changesetInputJSON),
			Config: changesetConfig,
		}}

		for j := range proposal.Operations {
			batchOp := &proposal.Operations[j]
			for k := range batchOp.Transactions {
				transaction := &batchOp.Transactions[k]
				if transaction.Tags == nil {
					transaction.Tags = []string{}
				}
				transaction.Tags = append(transaction.Tags, "changeset:"+id)
			}
		}
	}

	return nil
}
