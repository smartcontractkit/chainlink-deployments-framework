package mcms

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/mcms/sdk/evm"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

const (
	errorDecodeFormatText = "text"
	errorDecodeFormatJSON = "json"

	// errorDecodeStdin is the --error-file value that reads the error from stdin.
	errorDecodeStdin = "-"
)

var (
	errorDecodeShort = "Decodes the provided tx error data using the domain ABI registry"

	errorDecodeLong = text.LongDesc(`
		Decodes EVM transaction error data using the domain's ABI registry.

		This command reads a JSON document containing transaction error information
		and attempts to decode the revert reason using the registered ABIs. The
		document must have an "execution_error" key. Pass "-" as --error-file to
		read the document from stdin.

		By default the command prints text. With --format json it prints one JSON
		object to stdout, for callers that parse the result.

		By default the environment is read from the domain directory on disk. To
		build it from a BuildParams YAML document instead, with no domain directory
		needed, set the MCMS_ENV_BUILD_PARAMS environment variable (preferred, as the
		document carries secrets) or pass --env-build-params. The document then owns
		the domain and environment, so -e must be omitted. Decoding needs no chain
		connection, so the document does not need to list any networks. The ABIs
		come from the domain's proposal context and the catalog addresses.
	`)

	errorDecodeExample = text.Examples(`
		# Decode an error from a JSON file
		myapp mcms error-decode-evm -e staging --error-file ./tx_error.json

		# Decode an error from stdin and print JSON
		cat ./tx_error.json | myapp mcms error-decode-evm -e staging --error-file - --format json

		# Decode using a BuildParams document instead of a domain directory
		MCMS_ENV_BUILD_PARAMS="$(cat params.yaml)" myapp mcms error-decode-evm --error-file ./tx_error.json
	`)
)

type errorDecodeFlags struct {
	environment string
	errorFile   string
	format      string
	buildParams *cldfenvironment.BuildParams
}

// errorDecodeOutput is the JSON document printed with --format json.
type errorDecodeOutput struct {
	// RevertReason is set when the error has a revert reason.
	RevertReason *errorDecodeReason `json:"revert_reason,omitempty"`
	// UnderlyingReason is set when the error has a reason from a nested call.
	UnderlyingReason *errorDecodeReason `json:"underlying_reason,omitempty"`
}

// errorDecodeReason is one reason in the JSON output.
type errorDecodeReason struct {
	// Raw is the hex encoded revert data. It is empty when the input had only a decoded reason.
	Raw string `json:"raw,omitempty"`
	// Decoded is the readable reason. It is empty when the data could not be decoded.
	Decoded string `json:"decoded,omitempty"`
}

// newErrorDecodeCmd creates the "error-decode-evm" subcommand.
func newErrorDecodeCmd(cfg Config) *cobra.Command {
	var buildParams func() *cldfenvironment.BuildParams

	cmd := &cobra.Command{
		Use:     "error-decode-evm",
		Short:   errorDecodeShort,
		Long:    errorDecodeLong,
		Example: errorDecodeExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f := errorDecodeFlags{
				environment: flags.MustString(cmd.Flags().GetString("environment")),
				errorFile:   flags.MustString(cmd.Flags().GetString("error-file")),
				format:      flags.MustString(cmd.Flags().GetString("format")),
				buildParams: buildParams(),
			}

			return runErrorDecode(cmd, cfg, f)
		},
	}

	// Flags
	flags.Environment(cmd)
	buildParams = withEnvBuildParams(cmd, cfg.deps())
	cmd.Flags().String("error-file", "",
		"Path to the JSON file containing tx error, or - to read it from stdin (required)")
	_ = cmd.MarkFlagRequired("error-file")
	cmd.Flags().String("format", errorDecodeFormatText, "Output format: text, json")

	return cmd
}

// runErrorDecode executes the error-decode-evm command logic.
func runErrorDecode(cmd *cobra.Command, cfg Config, f errorDecodeFlags) error {
	ctx := cmd.Context()
	deps := cfg.deps()

	// --- Load all data first ---

	if f.format != errorDecodeFormatText && f.format != errorDecodeFormatJSON {
		return fmt.Errorf("unknown format %q, must be %q or %q", f.format, errorDecodeFormatText, errorDecodeFormatJSON)
	}

	// Read the failed transaction data
	data, err := readErrorDecodeInput(cmd, f.errorFile)
	if err != nil {
		return err
	}

	// Parse execution error from JSON
	execErr, err := ReadExecutionErrorFromFile(data)
	if err != nil {
		return err
	}

	// Load environment to get ABI registry (no chains needed to decode)
	var env cldf.Environment
	if f.buildParams != nil {
		env, err = buildEnvironment(ctx, cfg.Logger, cfg.Domain, deps, *f.buildParams, nil)
	} else {
		env, err = deps.EnvironmentLoader(ctx, cfg.Domain, f.environment, cfg.Logger,
			cldfenvironment.OnlyLoadChainsFor([]uint64{}),
			cldfenvironment.WithoutJD())
	}
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

	if f.format == errorDecodeFormatJSON {
		return writeErrorDecodeJSON(cmd.OutOrStdout(), execErr, decoded)
	}

	// Output decoded revert reason
	if decoded.RevertReasonDecoded {
		cmd.Printf("Revert Reason: %s - decoded: %s\n", revertSelectorHex(execErr), decoded.RevertReason)
	} else {
		cmd.Println("Revert Reason: (could not decode)")
	}

	// Output decoded underlying reason if available
	if decoded.UnderlyingReasonDecoded {
		cmd.Printf("Underlying Reason: %s - decoded: %s\n", execErr.UnderlyingReasonRaw, decoded.UnderlyingReason)
	}

	return nil
}

// readErrorDecodeInput reads the error document from a file, or from stdin for "-".
func readErrorDecodeInput(cmd *cobra.Command, path string) ([]byte, error) {
	if path == errorDecodeStdin {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("error reading stdin: %w", err)
		}
		if len(data) == 0 {
			return nil, errors.New("error reading stdin: input is empty")
		}

		return data, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", path, err)
	}

	return data, nil
}

// revertSelectorHex returns the revert selector as 0x-prefixed hex, or an empty
// string when the error has no raw revert data.
func revertSelectorHex(execErr *evm.ExecutionError) string {
	if execErr.RevertReasonRaw == nil {
		return ""
	}

	return "0x" + execErr.RevertReasonRaw.HexSelector()
}

// writeErrorDecodeJSON writes the decoded error as one JSON object.
func writeErrorDecodeJSON(w io.Writer, execErr *evm.ExecutionError, decoded DecodedExecutionError) error {
	out := errorDecodeOutput{}

	if execErr.RevertReasonRaw != nil || decoded.RevertReasonDecoded {
		reason := &errorDecodeReason{}
		if combined := execErr.RevertReasonRaw.Combined(); len(combined) > 0 {
			reason.Raw = "0x" + hex.EncodeToString(combined)
		}
		if decoded.RevertReasonDecoded {
			reason.Decoded = decoded.RevertReason
		}
		out.RevertReason = reason
	}

	if execErr.UnderlyingReasonRaw != "" || decoded.UnderlyingReasonDecoded {
		reason := &errorDecodeReason{Raw: execErr.UnderlyingReasonRaw}
		if decoded.UnderlyingReasonDecoded {
			reason.Decoded = decoded.UnderlyingReason
		}
		out.UnderlyingReason = reason
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("error writing JSON output: %w", err)
	}

	return nil
}
