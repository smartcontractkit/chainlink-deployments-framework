package mcms

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

var (
	analyzeProposalShort = "Analyze proposal and provide human readable output"

	analyzeProposalLong = text.LongDesc(`
		Analyzes a proposal and provides a human-readable description of its contents.

		The output includes details about the proposal's operations, targets, and
		any decoded call data. Supports markdown and text output formats.
	`)

	analyzeProposalExample = text.Examples(`
		# Analyze a proposal and print to stdout
		myapp mcms analyze-proposal -e staging -p ./proposal.json

		# Analyze and save to a file in markdown format
		myapp mcms analyze-proposal -e staging -p ./proposal.json -o analysis.md

		# Analyze and output as plain text
		myapp mcms analyze-proposal -e staging -p ./proposal.json --format text
	`)
)

type analyzeProposalFlags struct {
	environment   string
	proposalPath  string
	proposalKind  string
	chainSelector uint64
	output        string
	format        string
}

// newAnalyzeProposalCmd creates the "analyze-proposal" subcommand.
func newAnalyzeProposalCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "analyze-proposal",
		Short:   analyzeProposalShort,
		Long:    analyzeProposalLong,
		Example: analyzeProposalExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := analyzeProposalFlags{
				environment:   flags.MustString(cmd.Flags().GetString("environment")),
				proposalPath:  flags.MustString(cmd.Flags().GetString("proposal")),
				proposalKind:  flags.MustString(cmd.Flags().GetString("proposalKind")),
				chainSelector: flags.MustUint64(cmd.Flags().GetUint64("selector")),
				output:        flags.MustString(cmd.Flags().GetString("output")),
				format:        flags.MustString(cmd.Flags().GetString("format")),
			}

			return runAnalyzeProposal(cmd, cfg, f)
		},
	}

	// Shared flags
	flags.Environment(cmd)
	flags.Proposal(cmd)
	flags.ProposalKind(cmd, string(types.KindTimelockProposal))
	flags.ChainSelector(cmd, false) // optional for analyze

	// Output flags
	cmd.Flags().StringP("output", "o", "", "Output file to write analysis result")
	cmd.Flags().String("format", "markdown", "Output format: markdown (default), text")

	return cmd
}

// runAnalyzeProposal executes the analyze-proposal command logic.
func runAnalyzeProposal(cmd *cobra.Command, cfg Config, f analyzeProposalFlags) error {
	ctx := cmd.Context()
	deps := cfg.deps()

	// --- Load all data first ---

	if cfg.ProposalContextProvider == nil {
		return errors.New("proposalCtxProvider is required, please provide one in the domain cli constructor")
	}

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
		return errors.New("expected proposal to have non-nil *TimelockProposal")
	}

	// Set renderer based on format flag
	renderer, err := createRendererFromFormat(f.format)
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}
	proposalCfg.ProposalCtx.SetRenderer(renderer)

	// --- Execute logic with loaded data ---

	var analyzedProposal string
	if proposalCfg.TimelockProposal != nil {
		analyzedProposal, err = analyzer.DescribeTimelockProposal(ctx, proposalCfg.ProposalCtx, proposalCfg.Env, proposalCfg.TimelockProposal)
	} else {
		analyzedProposal, err = analyzer.DescribeProposal(ctx, proposalCfg.ProposalCtx, proposalCfg.Env, &proposalCfg.Proposal)
	}
	if err != nil {
		return fmt.Errorf("failed to describe proposal: %w", err)
	}

	// Output result
	if f.output == "" {
		cmd.Println(analyzedProposal)
	} else {
		if err := os.WriteFile(f.output, []byte(analyzedProposal), 0o600); err != nil {
			return err
		}
	}

	return nil
}

// createRendererFromFormat creates a renderer based on the format string.
func createRendererFromFormat(format string) (analyzer.Renderer, error) {
	switch format {
	case "text", "txt":
		return analyzer.NewTextRenderer(), nil
	case "markdown", "md", "":
		return analyzer.NewMarkdownRenderer(), nil
	default:
		return nil, fmt.Errorf("unknown format '%s'", format)
	}
}
