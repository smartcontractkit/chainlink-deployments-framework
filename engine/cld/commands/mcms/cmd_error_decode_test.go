package mcms

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const (
	errorDecodeTestABI = `[{"type":"error","name":"InsufficientFunds","inputs":[{"name":"needed","type":"uint256"}]}]`
	errorDecodeTestKey = "Router 1.2.0"
)

// errorDecodeTestRevert returns the selector and the encoded argument of
// InsufficientFunds(7), as hex strings with a 0x prefix.
func errorDecodeTestRevert(t *testing.T) (string, string) {
	t.Helper()

	parsed, err := abi.JSON(strings.NewReader(errorDecodeTestABI))
	require.NoError(t, err)

	errDef := parsed.Errors["InsufficientFunds"]
	arg := common.LeftPadBytes(big.NewInt(7).Bytes(), 32)

	return "0x" + hex.EncodeToString(errDef.ID[:4]), "0x" + hex.EncodeToString(arg)
}

// errorDecodeTestProvider returns a proposal context provider that knows the test ABI.
func errorDecodeTestProvider(env cldf.Environment) (analyzer.ProposalContext, error) {
	return analyzer.NewDefaultProposalContext(env,
		analyzer.WithEVMABIMappings(map[string]string{errorDecodeTestKey: errorDecodeTestABI}))
}

// newErrorDecodeTestCommand builds the mcms command on a domain root that does not exist.
func newErrorDecodeTestCommand(t *testing.T, deps Deps) *cobra.Command {
	t.Helper()

	cmd, err := NewCommand(Config{
		Logger:                  logger.Nop(),
		Domain:                  domain.NewDomain(filepath.Join(t.TempDir(), "does-not-exist"), buildParamsDomain),
		ProposalContextProvider: errorDecodeTestProvider,
		Deps:                    deps,
	})
	require.NoError(t, err)

	return cmd
}

// emptyEnvLoader loads an environment that has an empty datastore.
func emptyEnvLoader(
	_ context.Context, _ domain.Domain, envKey string, _ logger.Logger, _ ...cldfenv.LoadEnvironmentOption,
) (cldf.Environment, error) {
	return cldf.Environment{
		Name:              envKey,
		DataStore:         fdatastore.NewMemoryDataStore().Seal(),
		ExistingAddresses: cldf.NewMemoryAddressBook(),
	}, nil
}

// errorDecodeBuilder wraps fakeBuilder and, like the real Build, sets an empty address book.
func errorDecodeBuilder(fake *fakeBuilder) EnvironmentBuilderFunc {
	return func(ctx context.Context, params cldfenv.BuildParams, opts ...cldfenv.BuildOption) (cldf.Environment, error) {
		env, err := fake.build(ctx, params, opts...)
		env.ExistingAddresses = cldf.NewMemoryAddressBook()

		return env, err
	}
}

// runErrorDecodeTest runs the command and returns what it wrote to stdout.
// Cobra writes the usage text of a failed command to stderr, which is dropped.
func runErrorDecodeTest(t *testing.T, cmd *cobra.Command, stdin string, args ...string) (string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{"error-decode-evm"}, args...))

	err := cmd.Execute()

	return stdout.String(), err
}

func writeErrorDecodeFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "tx_error.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestErrorDecode_Output(t *testing.T) {
	t.Parallel()

	selector, arg := errorDecodeTestRevert(t)
	customError := fmt.Sprintf(`{"execution_error":{"RevertReasonRaw":{"selector":%q,"data":%q}}}`, selector, arg)
	unknownError := `{"execution_error":{"RevertReasonRaw":{"selector":"0x70de1b4b"}}}`
	underlyingOnly := fmt.Sprintf(`{"execution_error":{"UnderlyingReasonRaw":%q}}`, selector+strings.TrimPrefix(arg, "0x"))
	decodedOnly := `{"execution_error":{"RevertReasonDecoded":"InsufficientSigners()"}}`

	tests := []struct {
		name string
		// input is the document. It is passed on stdin when viaStdin is set, or in a file otherwise.
		input    string
		viaStdin bool
		format   string
		// wantStdout is compared as text when format is text, and as JSON when format is json.
		wantStdout string
	}{
		{
			name:       "text from file",
			input:      customError,
			format:     "text",
			wantStdout: fmt.Sprintf("Revert Reason: %s - decoded: InsufficientFunds(7) @%s\n", selector, errorDecodeTestKey),
		},
		{
			name:     "json from file",
			input:    customError,
			format:   "json",
			viaStdin: false,
			wantStdout: fmt.Sprintf(`{"revert_reason":{"raw":%q,"decoded":"InsufficientFunds(7) @%s"}}`,
				selector+strings.TrimPrefix(arg, "0x"), errorDecodeTestKey),
		},
		{
			name:     "json from stdin",
			input:    customError,
			format:   "json",
			viaStdin: true,
			wantStdout: fmt.Sprintf(`{"revert_reason":{"raw":%q,"decoded":"InsufficientFunds(7) @%s"}}`,
				selector+strings.TrimPrefix(arg, "0x"), errorDecodeTestKey),
		},
		{
			name:       "json with a selector the registry does not know",
			input:      unknownError,
			format:     "json",
			wantStdout: `{"revert_reason":{"raw":"0x70de1b4b","decoded":"custom error 0x70de1b4b"}}`,
		},
		{
			name:   "json with an underlying reason only",
			input:  underlyingOnly,
			format: "json",
			wantStdout: fmt.Sprintf(`{"underlying_reason":{"raw":%q,"decoded":"InsufficientFunds(7) @%s"}}`,
				selector+strings.TrimPrefix(arg, "0x"), errorDecodeTestKey),
		},
		{
			name:       "json with a reason that was already decoded",
			input:      decodedOnly,
			format:     "json",
			wantStdout: `{"revert_reason":{"decoded":"InsufficientSigners()"}}`,
		},
		{
			name:       "text with a reason that was already decoded has no selector and does not panic",
			input:      decodedOnly,
			format:     "text",
			wantStdout: "Revert Reason:  - decoded: InsufficientSigners()\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := newErrorDecodeTestCommand(t, Deps{EnvironmentLoader: emptyEnvLoader})

			args := []string{"-e", "staging", "--format", tt.format}
			stdin := ""
			if tt.viaStdin {
				args = append(args, "--error-file", "-")
				stdin = tt.input
			} else {
				args = append(args, "--error-file", writeErrorDecodeFile(t, tt.input))
			}

			stdout, err := runErrorDecodeTest(t, cmd, stdin, args...)

			require.NoError(t, err)
			if tt.format == "json" {
				require.JSONEq(t, tt.wantStdout, stdout)
				require.True(t, json.Valid([]byte(stdout)), "stdout must hold only the JSON document")

				return
			}
			require.Equal(t, tt.wantStdout, stdout)
		})
	}
}

func TestErrorDecode_InputErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		stdin   string
		args    []string
		wantErr string
	}{
		{
			name:    "unknown format",
			args:    []string{"--error-file", "-", "--format", "yaml"},
			stdin:   `{"execution_error":{}}`,
			wantErr: `unknown format "yaml"`,
		},
		{
			name:    "empty stdin",
			args:    []string{"--error-file", "-"},
			stdin:   "",
			wantErr: "input is empty",
		},
		{
			name:    "stdin without execution_error",
			args:    []string{"--error-file", "-"},
			stdin:   `{"other":{}}`,
			wantErr: "must contain an 'execution_error' key",
		},
		{
			name:    "missing file",
			args:    []string{"--error-file", "/does/not/exist.json"},
			wantErr: "error reading file /does/not/exist.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := newErrorDecodeTestCommand(t, Deps{EnvironmentLoader: emptyEnvLoader})

			_, err := runErrorDecodeTest(t, cmd, tt.stdin, append([]string{"-e", "staging"}, tt.args...)...)

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestErrorDecode_BuildParams(t *testing.T) {
	t.Parallel()

	selector, arg := errorDecodeTestRevert(t)
	input := fmt.Sprintf(`{"execution_error":{"RevertReasonRaw":{"selector":%q,"data":%q}}}`, selector, arg)

	tests := []struct {
		name string
		// envDoc is the value of MCMS_ENV_BUILD_PARAMS. Empty means the variable is not set.
		envDoc string
		// viaStdin passes the input on stdin with --error-file -, and in a file otherwise.
		viaStdin bool
		// args are added to the command line.
		args      []string
		wantErr   string
		wantCalls int
	}{
		{
			name:      "document from variable builds the environment and decodes",
			envDoc:    buildParamsDoc,
			wantCalls: 1,
		},
		{
			name:      "document from flag works with stdin",
			viaStdin:  true,
			args:      []string{"--env-build-params", buildParamsDoc},
			wantCalls: 1,
		},
		{
			name:    "document cannot be combined with -e",
			envDoc:  buildParamsDoc,
			args:    []string{"-e", "staging"},
			wantErr: "cannot be combined with build params",
		},
		{
			name:    "document for another domain is rejected",
			envDoc:  strings.Replace(buildParamsDoc, "domain: testdomain", "domain: other", 1),
			wantErr: `build params are for domain "other"`,
		},
		{
			name:    "without a document -e stays required",
			wantErr: `required flag(s) "environment" not set`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := &fakeBuilder{}
			cmd := newErrorDecodeTestCommand(t, Deps{
				EnvironmentLoader:  failingEnvLoader(t),
				EnvironmentBuilder: errorDecodeBuilder(builder),
				Getenv:             getenvFrom(map[string]string{"MCMS_ENV_BUILD_PARAMS": tt.envDoc}),
			})

			args := append([]string{"--format", "json"}, tt.args...)
			stdin := ""
			if tt.viaStdin {
				args = append(args, "--error-file", "-")
				stdin = input
			} else {
				args = append(args, "--error-file", writeErrorDecodeFile(t, input))
			}

			stdout, err := runErrorDecodeTest(t, cmd, stdin, args...)

			require.Equal(t, tt.wantCalls, builder.calls)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, buildParamsEnv, builder.params.Environment)
			require.Contains(t, stdout, "InsufficientFunds(7)")
		})
	}
}
