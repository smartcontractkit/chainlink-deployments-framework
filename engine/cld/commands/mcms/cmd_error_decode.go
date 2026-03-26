package mcms

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

var (
	errorDecodeShort = "Decodes the provided tx error data using the domain ABI registry"

	errorDecodeLong = text.LongDesc(`
		Decodes EVM transaction error data using the domain's ABI registry.

		This command reads a JSON file containing transaction error information
		and attempts to decode the revert reason using the registered ABIs.
	`)

	errorDecodeExample = text.Examples(`
		# Decode an error from a JSON file
		myapp mcms error-decode-evm -e staging --error-file ./tx_error.json
	`)
)

type errorDecodeFlags struct {
	environment string
	errorFile   string
}

// newErrorDecodeCmd creates the "error-decode-evm" subcommand.
func newErrorDecodeCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "error-decode-evm",
		Short:   errorDecodeShort,
		Long:    errorDecodeLong,
		Example: errorDecodeExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := errorDecodeFlags{
				environment: flags.MustString(cmd.Flags().GetString("environment")),
				errorFile:   flags.MustString(cmd.Flags().GetString("error-file")),
			}

			return runErrorDecode(cmd, cfg, f)
		},
	}

	// Flags
	flags.Environment(cmd)
	cmd.Flags().String("error-file", "", "Path to the JSON file containing tx error (required)")
	_ = cmd.MarkFlagRequired("error-file")

	return cmd
}

// runErrorDecode executes the error-decode-evm command logic.
func runErrorDecode(cmd *cobra.Command, cfg Config, f errorDecodeFlags) error {
	ctx := cmd.Context()

	// --- Load all data first ---

	// Read the failed transaction data file
	data, err := os.ReadFile(f.errorFile)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", f.errorFile, err)
	}

	// Parse execution error from JSON
	execErr, err := ReadExecutionErrorFromFile(data)
	if err != nil {
		return err
	}

	// Load environment to get ABI registry (no chains needed to decode)
	env, err := cldfenvironment.Load(ctx, cfg.Domain, f.environment,
		cldfenvironment.OnlyLoadChainsFor([]uint64{}),
		cldfenvironment.WithLogger(cfg.Logger),
		cldfenvironment.WithoutJD())
	if err != nil {
		return fmt.Errorf("error loading environment: %w", err)
	}

	// Create ProposalContext to get EVM registry
	var proposalCtx analyzer.ProposalContext
	proposalCtx, err = analyzer.NewDefaultProposalContext(env)
	if err != nil {
		cfg.Logger.Warnf("Failed to create default proposal context: %v. Proceeding without ABI registry.", err)
	}
	if cfg.ProposalContextProvider != nil {
		proposalCtx, err = cfg.ProposalContextProvider(env)
		if err != nil {
			return fmt.Errorf("failed to create proposal context: %w", err)
		}
	}

	// --- Execute logic with loaded data ---

	// Create error decoder from EVM registry
	var errDec *ErrDecoder
	if proposalCtx != nil && proposalCtx.GetEVMRegistry() != nil {
		errDec, err = NewErrDecoder(proposalCtx.GetEVMRegistry())
		if err != nil {
			return fmt.Errorf("error creating error decoder: %w", err)
		}
	}

	// Decode the error
	decoded := tryDecodeExecutionError(execErr, errDec)

	// Output decoded revert reason
	if decoded.RevertReasonDecoded {
		cmd.Printf("Revert Reason: %s - decoded: %s\n", execErr.RevertReasonRaw.Selector, decoded.RevertReason)
	} else {
		cmd.Println("Revert Reason: (could not decode)")
	}

	// Output decoded underlying reason if available
	if decoded.UnderlyingReasonDecoded {
		cmd.Printf("Underlying Reason: %s - decoded: %s\n", execErr.UnderlyingReasonRaw, decoded.UnderlyingReason)
	}

	return nil
}
