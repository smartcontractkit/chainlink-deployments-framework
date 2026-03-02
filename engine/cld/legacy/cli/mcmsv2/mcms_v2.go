// Package mcmsv2 provides legacy CLI commands for MCMS (Multi-Chain Management Service) proposals.
//
// Deprecated: This package is maintained for backward compatibility only.
// All active MCMS functionality has been migrated:
//   - Proposal analysis, UPF conversion, fork testing, error decoding → engine/cld/commands/mcms
//   - Proposal execution commands (check-quorum, set-root, execute-chain, etc.) → mcms-tools
//
// For new integrations, use the modular commands package:
//
//	import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands"
//
//	cmds := commands.New(lggr)
//	mcmsCmd, err := cmds.MCMS(domain, proposalCtxProvider)
//	if err != nil {
//	    return err
//	}
//	rootCmd.AddCommand(mcmsCmd)
package mcmsv2

import (
	"fmt"

	"github.com/spf13/cobra"

	mcmscmd "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/mcms"
	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

type buildOptions struct {
	proposalAnalyzers []proposalanalysis.BaseAnalyzer
}

// BuildOption configures BuildMCMSv2Cmd.
type BuildOption func(*buildOptions)

// WithProposalAnalyzers registers custom analyzers for analyze-proposal-v2.
func WithProposalAnalyzers(analyzers ...proposalanalysis.BaseAnalyzer) BuildOption {
	return func(o *buildOptions) {
		o.proposalAnalyzers = analyzers
	}
}

// BuildMCMSv2Cmd creates the mcmsv2 command with all subcommands.
// This function delegates to the modular mcms package for backward compatibility.
//
// Deprecated: Use the modular commands package for new integrations.
// Execution commands (check-quorum, set-root, execute-chain, timelock-execute-chain)
// have been moved to mcms-tools: https://github.com/smartcontractkit/mcms-tools
func BuildMCMSv2Cmd(
	lggr logger.Logger,
	domain cldf_domain.Domain,
	proposalContextProvider analyzer.ProposalContextProvider,
	opts ...BuildOption,
) *cobra.Command {
	if lggr == nil {
		panic("nil logger received")
	}
	if proposalContextProvider == nil {
		panic("nil proposal context provider received")
	}
	options := &buildOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}

	cmd := cobra.Command{
		Use:   "mcmsv2",
		Short: "Manage MCMS proposals",
		Long: `Commands for managing MCMS proposals.

NOTE: Execution commands (check-quorum, set-root, execute-chain, timelock-execute-chain,
reset-proposal, get-op-count) have been moved to mcms-tools.
Install with: brew install smartcontractkit/tap/mcms-tools

Available commands here:
  - analyze-proposal: Analyze proposal and provide human readable output
  - analyze-proposal-v2: Analyze timelock proposal using the v2 framework
  - convert-upf: Convert proposal to UPF (universal proposal format)
  - execute-fork: Execute proposal on forked environment for testing
  - error-decode-evm: Decode EVM transaction errors
`,
	}

	// Delegate to modular mcms commands
	mcmsCmd, err := mcmscmd.NewCommand(mcmscmd.Config{
		Logger:                  lggr,
		Domain:                  domain,
		ProposalContextProvider: proposalContextProvider,
		ProposalAnalyzers:       options.proposalAnalyzers,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create modular mcms commands: %v", err))
	}

	// Add all subcommands from modular package to this command
	for _, sub := range mcmsCmd.Commands() {
		cmd.AddCommand(sub)
	}

	return &cmd
}
