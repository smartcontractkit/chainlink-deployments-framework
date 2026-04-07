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
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
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
		myapp mcms hooks                                      \
		    --environment staging                             \
		    --report ./path/to/timelock-execution-report.json \
		    --proposal ./path/to/proposal.json                \
		    --selector 12345678901234567890
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
	Name         string   `json:"name" mapstructure:"name"`
	OperationIDs []string `json:"operationIDs" mapstructure:"operationIDs"`
	Input        any      `json:"input" mapstructure:"input"`
}

func newRunProposalHooksCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "hooks",
		Short:        runHooksShort,
		Long:         runHooksLong,
		Example:      runHooksExample,
		SilenceUsage: true,
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

func runHooks(ctx context.Context, cfg Config, hFlags runHooksFlags) error {
	deps := cfg.deps()

	proposalCfg, err := LoadProposalConfig(ctx, cfg.Logger, cfg.Domain, deps, cfg.ProposalContextProvider,
		ProposalFlags{
			ProposalPath:  hFlags.proposalPath,
			ProposalKind:  hFlags.proposalKind,
			Environment:   hFlags.environment,
			ChainSelector: hFlags.chainSelector,
		},
		acceptExpiredProposal,
	)
	if err != nil {
		return fmt.Errorf("failed to create proposal config: %w", err)
	}

	if proposalCfg.TimelockProposal == nil {
		return errors.New("expected proposal to be a TimelockProposal")
	}

	return runHooksInternal(cfg, proposalCfg.Env, proposalCfg.TimelockProposal, hFlags.reports)
}

func runHooksInternal(
	cfg Config,
	env cldf.Environment,
	timelockProposal *mcms.TimelockProposal,
	reports []cldfchangeset.MCMSTimelockExecuteReport,
) error {
	if cfg.LoadChangesets == nil {
		return errors.New("LoadChangesets function is required for proposal hook execution")
	}

	var metadata proposalMetadata
	err := mapstructure.Decode(timelockProposal.Metadata, &metadata)
	if err != nil {
		return fmt.Errorf("failed to unmarshal hooks metadata: %w", err)
	}

	changesetRegistry, err := cfg.LoadChangesets(env.Name)
	if err != nil {
		return fmt.Errorf("failed to load changesets: %w", err)
	}

	for _, changeset := range metadata.Changesets {
		changesetReports := lo.Filter(reports, func(r cldfchangeset.MCMSTimelockExecuteReport, _ int) bool {
			return slices.Contains(changeset.OperationIDs, r.Input.OperationID.Hex())
		})

		herr := changesetRegistry.RunProposalHooks(changeset.Name, env, timelockProposal, changeset.Input, changesetReports)
		if herr != nil {
			cfg.Logger.Errorw("proposal hook failed", "changeset", changeset.Name, "error", herr)
			err = errors.Join(err, fmt.Errorf("proposal hook for changeset %q failed: %w", changeset.Name, herr))
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
