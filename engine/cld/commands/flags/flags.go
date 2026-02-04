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
