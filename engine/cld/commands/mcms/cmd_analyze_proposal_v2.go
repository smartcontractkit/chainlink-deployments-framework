package mcms

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis"
	proposalanalysisanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	analysisdecoder "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	analysisrenderer "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
)

var (
	analyzeProposalV2Short = "Analyze proposal using the v2 analysis framework"

	analyzeProposalV2Long = text.LongDesc(`
		Analyzes a proposal and renders a human-readable report using the new
		proposal analysis framework.

		This command only supports timelock proposals and is intended to replace
		the legacy analyze-proposal command.
	`)

	analyzeProposalV2Example = text.Examples(`
		# Analyze a proposal and print to stdout
		myapp mcms analyze-proposal-v2 -e staging -p ./proposal.json

		# Analyze and save to a file in markdown format
		myapp mcms analyze-proposal-v2 -e staging -p ./proposal.json -o analysis.md
	`)
)

type analyzeProposalV2Flags struct {
	environment   string
	proposalPath  string
	chainSelector uint64
	output        string
	format        string
}

func newAnalyzeProposalV2Cmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "analyze-proposal-v2",
		Short:   analyzeProposalV2Short,
		Long:    analyzeProposalV2Long,
		Example: analyzeProposalV2Example,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := analyzeProposalV2Flags{
				environment:   flags.MustString(cmd.Flags().GetString("environment")),
				proposalPath:  flags.MustString(cmd.Flags().GetString("proposal")),
				chainSelector: flags.MustUint64(cmd.Flags().GetUint64("selector")),
				output:        flags.MustString(cmd.Flags().GetString("output")),
				format:        flags.MustString(cmd.Flags().GetString("format")),
			}

			return runAnalyzeProposalV2(cmd, cfg, f)
		},
	}

	flags.Environment(cmd)
	flags.Proposal(cmd)
	flags.ChainSelector(cmd, false)

	cmd.Flags().StringP("output", "o", "", "Output file to write analysis result")
	cmd.Flags().String("format", analysisrenderer.IDMarkdown, "Output format: markdown")

	return cmd
}

func runAnalyzeProposalV2(cmd *cobra.Command, cfg Config, f analyzeProposalV2Flags) error {
	ctx := cmd.Context()
	deps := cfg.deps()

	proposalCfg, err := LoadProposalConfig(ctx, cfg.Logger, cfg.Domain, deps, cfg.ProposalContextProvider,
		ProposalFlags{
			ProposalPath:  f.proposalPath,
			ProposalKind:  string(types.KindTimelockProposal),
			Environment:   f.environment,
			ChainSelector: f.chainSelector,
		},
		acceptExpiredProposal,
	)
	if err != nil {
		return fmt.Errorf("error creating config: %w", err)
	}

	if proposalCfg.TimelockProposal == nil {
		return errors.New("expected proposal be a timelock proposal")
	}

	rendererID, err := normalizeRendererFormat(f.format)
	if err != nil {
		return err
	}

	markdownRenderer, err := analysisrenderer.NewMarkdownRenderer()
	if err != nil {
		return fmt.Errorf("create markdown renderer: %w", err)
	}

	engine := proposalanalysis.NewAnalyzerEngine()
	if registerErr := engine.RegisterRenderer(markdownRenderer); registerErr != nil {
		return fmt.Errorf("register renderer: %w", registerErr)
	}
	if analyzerErr := registerProposalAnalyzers(engine, cfg.ProposalAnalyzers); analyzerErr != nil {
		return analyzerErr
	}

	var evmABIMappings map[string]string
	var solanaDecoders map[string]analysisdecoder.DecodeInstructionFn
	if proposalCfg.ProposalCtx != nil {
		if proposalCfg.ProposalCtx.GetEVMRegistry() != nil {
			evmABIMappings = proposalCfg.ProposalCtx.GetEVMRegistry().GetAllABIs()
		}
		if proposalCfg.ProposalCtx.GetSolanaDecoderRegistry() != nil {
			solanaDecoders = proposalCfg.ProposalCtx.GetSolanaDecoderRegistry().Decoders()
		}
	}

	analyzedProposal, err := engine.Run(ctx, proposalanalysis.RunRequest{
		Domain:      cfg.Domain,
		Environment: &proposalCfg.Env,
		DecoderConfig: analysisdecoder.Config{
			EVMABIMappings: evmABIMappings,
			SolanaDecoders: solanaDecoders,
		},
	}, proposalCfg.TimelockProposal)
	if err != nil {
		return fmt.Errorf("run analysis engine: %w", err)
	}

	var out bytes.Buffer
	if err := engine.RenderTo(&out, rendererID, analysisrenderer.RenderRequest{
		Domain:          cfg.Domain.Key(),
		EnvironmentName: proposalCfg.EnvStr,
	}, analyzedProposal); err != nil {
		return fmt.Errorf("render analysis output: %w", err)
	}

	if f.output == "" {
		cmd.Println(out.String())
		return nil
	}

	if err := os.WriteFile(f.output, out.Bytes(), 0o600); err != nil {
		return err
	}

	return nil
}

func registerProposalAnalyzers(engine proposalanalysis.AnalyzerEngine, analyzers []proposalanalysisanalyzer.BaseAnalyzer) error {
	for _, analyzer := range analyzers {
		if err := engine.RegisterAnalyzer(analyzer); err != nil {
			return fmt.Errorf("register proposal analyzer %q: %w", analyzer.ID(), err)
		}
	}

	return nil
}

func normalizeRendererFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", analysisrenderer.IDMarkdown, "md":
		return analysisrenderer.IDMarkdown, nil
	default:
		return "", fmt.Errorf("unknown format %q: only markdown is supported for analyze-proposal-v2", format)
	}
}
