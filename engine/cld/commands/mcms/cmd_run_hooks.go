package mcms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/go-viper/mapstructure/v2"
	"github.com/samber/lo"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"

	cldfchangeset "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
)

var (
	runHooksShort = "Run post proposal execution hooks"

	runHooksLong = text.LongDesc(`
		Run post proposal execution hooks
	`)

	runHooksExample = text.Examples(`
		# Run post proposal execution hooks
		myapp mcms hooks -e staging -p ./proposal.json
	`)
)

type runHooksFlags struct {
	environment   string
	proposalPath  string
	proposalKind  string
	chainSelector uint64
	reports       []cldfchangeset.MCMSTimelockExecuteReport
}

type proposalMetadata struct {
	Changesets         []changesetMetadata     `json:"changesets" mapstructure:"changesets"`
	PostExecutionHooks []proposalHooksMetadata `json:"postExecutionHooks" mapstructure:"postExecutionHooks"`
}

type proposalHooksMetadata struct {
	Name  string `json:"name" mapstructure:"name"`
	Input any    `json:"input" mapstructure:"input"`
}

type changesetMetadata struct {
	Name       string   `json:"name" mapstructure:"name"`
	Operations []string `json:"operations" mapstructure:"operations"`
	Input      any      `json:"input" mapstructure:"input"`
}

func newRunProposalHooksCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hooks",
		Short:   runHooksShort,
		Long:    runHooksLong,
		Example: runHooksExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cfg.LoadChangesets == nil {
				return errors.New("load changesets function is required to run hooks")
			}

			reports, err := loadReport(flags.MustString(cmd.Flags().GetString("report")))
			if err != nil {
				return fmt.Errorf("failed to load report file: %w", err)
			}

			f := runHooksFlags{
				environment:   flags.MustString(cmd.Flags().GetString("environment")),
				proposalPath:  flags.MustString(cmd.Flags().GetString("proposal")),
				proposalKind:  flags.MustString(cmd.Flags().GetString("proposalKind")),
				chainSelector: flags.MustUint64(cmd.Flags().GetUint64("selector")),
				reports:       reports,
			}

			return runHooks(cmd.Context(), cfg, f)
		},
	}

	flags.Environment(cmd)
	flags.Proposal(cmd)
	flags.ProposalKind(cmd, string(mcmstypes.KindTimelockProposal))
	flags.ChainSelector(cmd, true)
	cmd.Flags().String("report", "", "File with timelock execution report.")
	_ = cmd.MarkFlagRequired("report")

	return cmd
}

func runHooks(ctx context.Context, cfg Config, flags runHooksFlags) error {
	if cfg.LoadChangesets == nil {
		return errors.New("LoadChangesets function is not provided, skipping proposal hooks")
	}

	deps := cfg.deps()

	proposalCfg, err := LoadProposalConfig(ctx, cfg.Logger, cfg.Domain, deps, cfg.ProposalContextProvider,
		ProposalFlags{
			ProposalPath:  flags.proposalPath,
			ProposalKind:  flags.proposalKind,
			Environment:   flags.environment,
			ChainSelector: flags.chainSelector,
		},
		acceptExpiredProposal,
	)
	if err != nil {
		return fmt.Errorf("failed to create proposal config: %w", err)
	}

	if proposalCfg.TimelockProposal == nil {
		return errors.New("expected proposal to be a TimelockProposal")
	}

	var metadata proposalMetadata
	err = mapstructure.Decode(proposalCfg.TimelockProposal.Metadata, &metadata)
	if err != nil {
		return fmt.Errorf("failed to unmarshal hooks metadata: %w", err)
	}

	changesetRegistry, err := cfg.LoadChangesets(proposalCfg.EnvStr)
	if err != nil {
		return fmt.Errorf("failed to load changesets: %w", err)
	}

	for _, changeset := range metadata.Changesets {
		reports := lo.Filter(flags.reports, func(r cldfchangeset.MCMSTimelockExecuteReport, _ int) bool {
			return slices.Contains(changeset.Operations, r.Input.OperationID.Hex())
		})

		err = changesetRegistry.RunProposalHooks(changeset.Name, proposalCfg.Env, proposalCfg.TimelockProposal,
			changeset.Input, reports)
		if err != nil {
			cfg.Logger.Errorw("proposal hook failed", "changeset", changeset.Name, "error", err)
			err = errors.Join(err, fmt.Errorf("proposal hook for changeset %q failed: %w", changeset.Name, err))
		}
	}

	return err
}

func loadReport(path string) ([]cldfchangeset.MCMSTimelockExecuteReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open report file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.UseNumber()

	var report []cldfchangeset.MCMSTimelockExecuteReport
	err = decoder.Decode(&report)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal report file: %w", err)
	}

	return report, nil
}
