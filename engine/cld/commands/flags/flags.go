// Package flags provides reusable flag helpers for CLI commands.
//
// This package should only contain common flags that can be used by multiple commands
// to ensure unified naming and consistent behavior across the CLI.
// Command-specific flags should be defined locally in the command file.
package flags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// MustString returns the string value, ignoring the error.
// Safe to use with registered flags where GetString cannot fail.
func MustString(s string, _ error) string { return s }

// MustBool returns the bool value, ignoring the error.
// Safe to use with registered flags where GetBool cannot fail.
func MustBool(b bool, _ error) bool { return b }

// MustUint64 returns the uint64 value, ignoring the error.
// Safe to use with registered flags where GetUint64 cannot fail.
func MustUint64(u uint64, _ error) uint64 { return u }

// Environment adds the required --environment/-e flag to a command.
// Retrieve the value with cmd.Flags().GetString("environment").
//
// Usage:
//
//	flags.Environment(cmd)
//	// later in RunE:
//	env, _ := cmd.Flags().GetString("environment")
func Environment(cmd *cobra.Command) {
	cmd.Flags().StringP("environment", "e", "", "Deployment environment (required)")
	_ = cmd.MarkFlagRequired("environment")
}

// Print adds the --print flag for printing output to stdout (default: true).
// Retrieve the value with cmd.Flags().GetBool("print").
//
// Usage:
//
//	flags.Print(cmd)
//	// later in RunE:
//	shouldPrint, _ := cmd.Flags().GetBool("print")
func Print(cmd *cobra.Command) {
	cmd.Flags().Bool("print", true, "Print output to stdout")
}

// Output adds the --out/-o flag for specifying output file path.
// Also supports deprecated --outputPath alias for backwards compatibility.
// Retrieve the value with cmd.Flags().GetString("out").
//
// Usage:
//
//	flags.Output(cmd, "")
//	// later in RunE:
//	outPath, _ := cmd.Flags().GetString("out")
func Output(cmd *cobra.Command, defaultValue string) {
	cmd.Flags().StringP("out", "o", defaultValue, "Output file path")

	// Normalize --outputPath to --out for backward compatibility (silent)
	existingNormalize := cmd.Flags().GetNormalizeFunc()
	cmd.Flags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "outputPath" {
			return pflag.NormalizedName("out")
		}
		if existingNormalize != nil {
			return existingNormalize(f, name)
		}

		return pflag.NormalizedName(name)
	})
}

// --- MCMS shared flags ---

// Proposal adds the required --proposal/-p flag for specifying proposal file path.
// Retrieve the value with cmd.Flags().GetString("proposal").
//
// Usage:
//
//	flags.Proposal(cmd)
//	// later in RunE:
//	proposalPath, _ := cmd.Flags().GetString("proposal")
func Proposal(cmd *cobra.Command) {
	cmd.Flags().StringP("proposal", "p", "", "Absolute file path containing the proposal (required)")
	_ = cmd.MarkFlagRequired("proposal")
}

// ProposalKind adds the --proposalKind/-k flag for specifying proposal type.
// The defaultKind parameter should be a valid proposal kind string (e.g., "timelock").
// Retrieve the value with cmd.Flags().GetString("proposalKind").
//
// Usage:
//
//	flags.ProposalKind(cmd, "timelock")
//	// later in RunE:
//	kind, _ := cmd.Flags().GetString("proposalKind")
func ProposalKind(cmd *cobra.Command, defaultKind string) {
	cmd.Flags().StringP("proposalKind", "k", defaultKind, "The type of proposal being ingested")
}

// ChainSelector adds the --selector/-s flag for specifying chain selector.
// If required is true, the flag is marked as required.
// Retrieve the value with cmd.Flags().GetUint64("selector").
//
// Usage:
//
//	flags.ChainSelector(cmd, false) // optional
//	flags.ChainSelector(cmd, true)  // required
//	// later in RunE:
//	selector, _ := cmd.Flags().GetUint64("selector")
func ChainSelector(cmd *cobra.Command, required bool) {
	cmd.Flags().Uint64P("selector", "s", 0, "Chain selector")
	if required {
		_ = cmd.MarkFlagRequired("selector")
	}
}
