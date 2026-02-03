// Package flags provides reusable flag helpers for CLI commands.
//
// This package should only contain common flags that can be used by multiple commands
// to ensure unified naming and consistent behavior across the CLI.
// Command-specific flags should be defined locally in the command file.
package flags

import "github.com/spf13/cobra"

// Environment adds the required --environment/-e flag to a command.
//
// Usage:
//
//	var env string
//	flags.Environment(cmd, &env)
func Environment(cmd *cobra.Command, dest *string) {
	cmd.Flags().StringVarP(dest, "environment", "e", "", "Deployment environment (required)")
	_ = cmd.MarkFlagRequired("environment")
}

// Print adds the --print flag for explicitly printing output to stdout.
//
// Usage:
//
//	var print bool
//	flags.Print(cmd, &print)
func Print(cmd *cobra.Command, dest *bool) {
	cmd.Flags().BoolVar(dest, "print", false, "Print output to stdout")
}

// Output adds the --out/-o flag for specifying output file path.
// Also registers --outputPath as a deprecated alias for backwards compatibility.
//
// Usage:
//
//	var out string
//	flags.Output(cmd, &out, "")
func Output(cmd *cobra.Command, dest *string, defaultValue string) {
	cmd.Flags().StringVarP(dest, "out", "o", defaultValue, "Output file path")
	cmd.Flags().StringVar(dest, "outputPath", defaultValue, "Output file path")
	_ = cmd.Flags().MarkDeprecated("outputPath", "use --out instead")
}
