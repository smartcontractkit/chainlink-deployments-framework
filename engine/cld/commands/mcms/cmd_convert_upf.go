package mcms

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/upf"
)

var (
	convertUpfShort = "Convert proposal to UPF (universal proposal format)"

	convertUpfLong = text.LongDesc(`
		Converts a proposal to UPF (Universal Proposal Format).

		This is useful for sharing proposals in a standardized format that can
		be imported into other tools and systems.
	`)

	convertUpfExample = text.Examples(`
		# Convert a proposal to UPF and print to stdout
		myapp mcms convert-upf -e staging -p ./proposal.json

		# Convert and save to a file
		myapp mcms convert-upf -e staging -p ./proposal.json -o proposal.upf.json
	`)
)

type convertUpfFlags struct {
	environment   string
	proposalPath  string
	proposalKind  string
	chainSelector uint64
	output        string
}

// newConvertUpfCmd creates the "convert-upf" subcommand.
func newConvertUpfCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "convert-upf",
		Short:   convertUpfShort,
		Long:    convertUpfLong,
		Example: convertUpfExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := convertUpfFlags{
				environment:   flags.MustString(cmd.Flags().GetString("environment")),
				proposalPath:  flags.MustString(cmd.Flags().GetString("proposal")),
				proposalKind:  flags.MustString(cmd.Flags().GetString("proposalKind")),
				chainSelector: flags.MustUint64(cmd.Flags().GetUint64("selector")),
				output:        flags.MustString(cmd.Flags().GetString("output")),
			}

			return runConvertUpf(cmd, cfg, f)
		},
	}

	// Shared flags
	flags.Environment(cmd)
	flags.Proposal(cmd)
	flags.ProposalKind(cmd, string(types.KindTimelockProposal))
	flags.ChainSelector(cmd, false) // optional

	// Output flags
	cmd.Flags().StringP("output", "o", "", "File path where the converted file will be saved")

	return cmd
}

// runConvertUpf executes the convert-upf command logic.
func runConvertUpf(cmd *cobra.Command, cfg Config, f convertUpfFlags) error {
	ctx := cmd.Context()
	deps := cfg.deps()

	// --- Load all data first ---

	proposalCfg, err := LoadProposalConfig(ctx, cfg.Logger, cfg.Domain, deps, cfg.ProposalContextProvider,
		ProposalFlags{
			ProposalPath:  f.proposalPath,
			ProposalKind:  f.proposalKind,
			Environment:   f.environment,
			ChainSelector: f.chainSelector,
		},
		acceptExpiredProposal,
	)
	if err != nil {
		return fmt.Errorf("error creating config: %w", err)
	}

	if proposalCfg.TimelockProposal == nil {
		return errors.New("expected proposal to be a TimelockProposal")
	}

	// Get signers for the proposal
	signers, err := getProposalSigners(ctx, proposalCfg, &proposalCfg.Proposal, deps)
	if err != nil {
		return fmt.Errorf("failed to get proposal signers: %w", err)
	}

	// --- Execute logic with loaded data ---

	var convertedProposal string
	if proposalCfg.TimelockProposal != nil {
		convertedProposal, err = upf.UpfConvertTimelockProposal(ctx, proposalCfg.ProposalCtx, proposalCfg.Env, proposalCfg.TimelockProposal, &proposalCfg.Proposal, signers)
	} else {
		convertedProposal, err = upf.UpfConvertProposal(ctx, proposalCfg.ProposalCtx, proposalCfg.Env, &proposalCfg.Proposal, signers)
	}
	if err != nil {
		return fmt.Errorf("failed to convert proposal to UPF format: %w", err)
	}

	// Output result
	if f.output == "" {
		cmd.Println(convertedProposal)
	} else {
		if err := os.WriteFile(f.output, []byte(convertedProposal), 0o600); err != nil {
			return err
		}
	}

	return nil
}

// getProposalSigners retrieves the signers for each chain in the proposal.
func getProposalSigners(
	ctx context.Context,
	cfg *ProposalConfig,
	proposal mcms.ProposalInterface,
	_ *Deps,
) (map[types.ChainSelector][]common.Address, error) {
	chainMeta := proposal.ChainMetadatas()
	addresses := make(map[types.ChainSelector][]common.Address, len(chainMeta))

	wrappedChains := adapters.Wrap(cfg.Env.BlockChains)
	inspectors, err := chainwrappers.BuildInspectors(&wrappedChains, cfg.TimelockProposal.ChainMetadata, cfg.TimelockProposal.Action)
	if err != nil {
		return nil, fmt.Errorf("building inspectors: %w", err)
	}

	for chainSelector, metadata := range chainMeta {
		inspector, ok := inspectors[chainSelector]
		if !ok {
			return nil, fmt.Errorf("no inspector found for chain selector %d", chainSelector)
		}

		config, err := inspector.GetConfig(ctx, metadata.MCMAddress)
		if err != nil {
			return nil, fmt.Errorf("get config for selector %d: %w", chainSelector, err)
		}

		addresses[chainSelector] = config.GetAllSigners()
	}

	return addresses, nil
}
