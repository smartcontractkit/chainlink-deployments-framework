package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	fcre "github.com/smartcontractkit/chainlink-deployments-framework/cre"
)

func TestNewCLIRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		binaryPath string
		apiKey     string
		wantPath   string
		wantKey    string
	}{
		{name: "empty_defaults_to_cre", binaryPath: "", apiKey: "", wantPath: defaultBinary, wantKey: ""},
		{name: "custom_path", binaryPath: "/opt/cre", apiKey: "", wantPath: "/opt/cre", wantKey: ""},
		{name: "with_api_key", binaryPath: "/bin/sh", apiKey: "k", wantPath: "/bin/sh", wantKey: "k"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewCLIRunner(tt.binaryPath, tt.apiKey)
			require.Equal(t, tt.wantPath, r.binaryPath)
			require.Equal(t, tt.wantKey, r.apiKey)
			require.Equal(t, tt.apiKey, r.rawAPIKey)
			require.Nil(t, r.apiKeysByName)
		})
	}
}

func Test_parseNamedAPIKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		ok   bool
		want namedAPIKeys
	}{
		{name: "valid_map_two_names", raw: `{"prod-1":"k1","prod-2":"k2"}`, ok: true, want: namedAPIKeys{"prod-1": "k1", "prod-2": "k2"}},
		{name: "valid_map_one_name", raw: `{"prod-1":"k1"}`, ok: true, want: namedAPIKeys{"prod-1": "k1"}},
		{name: "whitespace_around_object", raw: "  \n{\"prod-1\":\"k1\"}\n  ", ok: true, want: namedAPIKeys{"prod-1": "k1"}},
		{name: "not_json_plain_string", raw: "abcd"},
		{name: "json_string_literal", raw: `"abcd"`},
		{name: "json_array", raw: `["a","b"]`},
		{name: "json_number", raw: `42`},
		{name: "json_null", raw: `null`},
		{name: "empty_map", raw: `{}`},
		{name: "empty_value_rejected", raw: `{"prod-1":""}`},
		{name: "empty_key_rejected", raw: `{"":"k1"}`},
		{name: "empty_string", raw: ""},
		{name: "malformed_object", raw: `{"prod-1":`},
		{name: "object_with_non_string_value", raw: `{"prod-1":123}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseNamedAPIKeys(tt.raw)
			require.Equal(t, tt.ok, ok)
			if tt.ok {
				require.Equal(t, tt.want, got)
			} else {
				require.Nil(t, got)
			}
		})
	}
}

func TestNewCLIRunner_NamedKeysMode(t *testing.T) {
	t.Parallel()

	raw := `{"prod-1":"k1","prod-2":"k2"}`
	r := NewCLIRunner("/bin/sh", raw)
	require.Empty(t, r.apiKey, "apiKey must be empty until a name is selected")
	require.Equal(t, namedAPIKeys{"prod-1": "k1", "prod-2": "k2"}, r.apiKeysByName)
	require.Equal(t, raw, r.rawAPIKey, "rawAPIKey preserves the original value for debugging")
}

func TestNewCLIRunner_SingleKeyMode_PreservesUnparseableValues(t *testing.T) {
	t.Parallel()

	// Values that look JSON-ish but don't parse as an object of strings stay as
	// the literal single key value — backward compatible.
	cases := []string{
		"plain-secret",
		`{"prod-1":""}`,
		`{}`,
		`{not-json`,
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			r := NewCLIRunner("/bin/sh", raw)
			require.Equal(t, raw, r.apiKey)
			require.Equal(t, raw, r.rawAPIKey)
			require.Nil(t, r.apiKeysByName)
		})
	}
}

func TestCLIRunner_Run_NamedKeysWithoutSelection_Errors(t *testing.T) {
	t.Parallel()

	r := NewCLIRunner("/bin/sh", `{"prod-1":"k1","prod-2":"k2"}`)
	res, err := r.Run(t.Context(), nil, "-c", "echo unreachable")
	require.Nil(t, res)
	require.Error(t, err)
	require.ErrorContains(t, err, "named API keys configured but no name selected")
	// Available names are listed in sorted order.
	require.ErrorContains(t, err, "[prod-1 prod-2]")
}

func TestCLIRunner_WithNamedAPIKey_ResolvesKeyForSubprocess(t *testing.T) {
	// Cannot use t.Parallel: relies on a known parent env state via t.Setenv.
	t.Setenv(envCREAPIKey, "from-parent-must-be-overridden")

	r := NewCLIRunner("/bin/sh", `{"prod-1":"k1","prod-2":"k2"}`)
	selected, err := r.WithNamedAPIKey("prod-2")
	require.NoError(t, err)

	res, err := selected.Run(t.Context(), nil, "-c", `printf '%s' "$`+envCREAPIKey+`"`)
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.Equal(t, "k2", string(res.Stdout))
}

func TestCLIRunner_WithNamedAPIKey_UnknownNameErrors(t *testing.T) {
	t.Parallel()

	r := NewCLIRunner("/bin/sh", `{"prod-1":"k1","prod-2":"k2"}`)
	got, err := r.WithNamedAPIKey("missing")
	require.Nil(t, got)
	require.Error(t, err)
	require.ErrorContains(t, err, `API key "missing" not configured`)
	require.ErrorContains(t, err, "[prod-1 prod-2]")
}

func TestCLIRunner_WithNamedAPIKey_SingleKeyRunnerErrors(t *testing.T) {
	t.Parallel()

	r := NewCLIRunner("/bin/sh", "plain-secret")
	got, err := r.WithNamedAPIKey("prod-1")
	require.Nil(t, got)
	require.ErrorContains(t, err, "not configured with named API keys")
}

func TestCLIRunner_WithNamedAPIKey_DoesNotMutateParent(t *testing.T) {
	t.Parallel()

	parent := NewCLIRunner("/bin/sh", `{"prod-1":"k1","prod-2":"k2"}`)
	_, err := parent.WithNamedAPIKey("prod-1")
	require.NoError(t, err)

	// Parent still rejects Run because no name has been selected on it.
	res, err := parent.Run(t.Context(), nil, "-c", "echo unreachable")
	require.Nil(t, res)
	require.ErrorContains(t, err, "named API keys configured but no name selected")
}

func TestCLIRunner_WithNamedAPIKey_CanRebindOnClone(t *testing.T) {
	t.Parallel()

	parent := NewCLIRunner("/bin/sh", `{"prod-1":"k1","prod-2":"k2"}`)
	first, err := parent.WithNamedAPIKey("prod-1")
	require.NoError(t, err)
	second, err := first.WithNamedAPIKey("prod-2")
	require.NoError(t, err)

	res, err := second.Run(t.Context(), nil, "-c", `printf '%s' "$`+envCREAPIKey+`"`)
	require.NoError(t, err)
	require.Equal(t, "k2", string(res.Stdout))
}

func TestCLIRunner_APIKeyEnv(t *testing.T) {
	// Cannot use t.Parallel: subtests use t.Setenv.
	tests := []struct {
		name           string
		parentAPIKey   string
		apiKey         string
		wantSubprocess string
	}{
		{ //nolint:gosec // G101: fixed test fixture value, not a real credential
			name:           "with_api_key_sets_subprocess_env",
			parentAPIKey:   "",
			apiKey:         "test-api-key-value",
			wantSubprocess: "test-api-key-value",
		},
		{
			name:           "without_api_key_inherits_unset_parent",
			parentAPIKey:   "",
			apiKey:         "",
			wantSubprocess: "",
		},
		{
			name:           "with_api_key_overrides_parent_env",
			parentAPIKey:   "from-parent",
			apiKey:         "from-runner",
			wantSubprocess: "from-runner",
		},
	}

	shArgs := []string{"-c", `printf '%s' "$` + envCREAPIKey + `"`}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envCREAPIKey, tt.parentAPIKey)

			r := NewCLIRunner("/bin/sh", tt.apiKey)
			res, err := r.Run(t.Context(), nil, shArgs...)
			require.NoError(t, err)
			require.Equal(t, 0, res.ExitCode)
			require.Equal(t, tt.wantSubprocess, string(res.Stdout))
		})
	}
}

func Test_envForCRECLI(t *testing.T) {
	t.Setenv(envCREAPIKey, "old-api")
	t.Setenv("CUSTOM_KEY", "old-custom")

	tests := []struct {
		name           string
		apiKey         string
		extraEnv       map[string]string
		mustContain    []string
		mustNotContain []string
	}{
		{
			name:        "empty_api_key_no_extra_passes_through_parent",
			apiKey:      "",
			extraEnv:    nil,
			mustContain: []string{envCREAPIKey + "=old-api", "CUSTOM_KEY=old-custom"},
		},
		{
			name:     "non_empty_replaces_existing_and_extra_overrides",
			apiKey:   "new-api",
			extraEnv: map[string]string{"CUSTOM_KEY": "new-custom"},
			mustContain: []string{
				envCREAPIKey + "=new-api",
				"CUSTOM_KEY=new-custom",
			},
			mustNotContain: []string{
				envCREAPIKey + "=old-api",
				"CUSTOM_KEY=old-custom",
			},
		},
		{
			name:   "extra_env_cannot_override_runner_api_key",
			apiKey: "runner-api",
			extraEnv: map[string]string{
				envCREAPIKey: "override-api",
			},
			mustContain: []string{
				envCREAPIKey + "=runner-api",
			},
			mustNotContain: []string{
				envCREAPIKey + "=override-api",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := envForCRECLI(tt.apiKey, tt.extraEnv)
			for _, s := range tt.mustContain {
				require.Contains(t, got, s)
			}
			for _, s := range tt.mustNotContain {
				require.NotContains(t, got, s)
			}
		})
	}
}

func TestCLIRunner_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupCtx     func(*testing.T) context.Context
		runner       *cliRunner
		args         []string
		wantErr      bool
		wantResNil   bool
		wantExitCode int
		wantStdout   string
		wantStderr   string
		wantErrIs    error
		wantExitErr  bool
	}{
		{
			name:       "binary_not_found",
			runner:     NewCLIRunner(t.TempDir()+"/nonexistent-cre-xyz", ""),
			args:       []string{"build"},
			wantErr:    true,
			wantResNil: true,
		},
		{
			name:   "context_already_canceled",
			runner: NewCLIRunner("/bin/sh", ""),
			setupCtx: func(t *testing.T) context.Context {
				t.Helper()
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				return ctx
			},
			args:       []string{"-c", "echo unreachable"},
			wantErr:    true,
			wantResNil: true,
			wantErrIs:  context.Canceled,
		},
		{
			name:         "nonzero_exit_captures_output",
			runner:       NewCLIRunner("/bin/sh", ""),
			args:         []string{"-c", `echo "fail out"; echo "fail err" >&2; exit 41`},
			wantErr:      true,
			wantExitCode: 41,
			wantStdout:   "fail out\n",
			wantStderr:   "fail err\n",
			wantExitErr:  true,
		},
		{
			name:         "success_with_output",
			runner:       NewCLIRunner("/bin/sh", ""),
			args:         []string{"-c", `echo "hello stdout"; echo "hello stderr" >&2`},
			wantStdout:   "hello stdout\n",
			wantStderr:   "hello stderr\n",
			wantExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if tt.setupCtx != nil {
				ctx = tt.setupCtx(t)
			}

			res, err := tt.runner.Run(ctx, nil, tt.args...)

			if tt.wantResNil {
				require.Nil(t, res)
			}
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				if tt.wantExitErr {
					var exitErr *fcre.ExitError
					require.ErrorAs(t, err, &exitErr)
					require.Equal(t, tt.wantExitCode, exitErr.ExitCode)
				}
			} else {
				require.NoError(t, err)
			}

			if res != nil {
				require.Equal(t, tt.wantExitCode, res.ExitCode)
				require.Equal(t, tt.wantStdout, string(res.Stdout))
				require.Equal(t, tt.wantStderr, string(res.Stderr))
			}
		})
	}
}

func TestCLIRunner_StreamingWriters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantStdout string
		wantStderr string
	}{
		{
			name:       "stdout_streamed",
			args:       []string{"-c", `echo "hello from stdout"`},
			wantStdout: "hello from stdout\n",
			wantStderr: "",
		},
		{
			name:       "stderr_streamed",
			args:       []string{"-c", `echo "hello from stderr" >&2`},
			wantStdout: "",
			wantStderr: "hello from stderr\n",
		},
		{
			name:       "both_streamed",
			args:       []string{"-c", `echo "out"; echo "err" >&2`},
			wantStdout: "out\n",
			wantStderr: "err\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var streamOut, streamErr bytes.Buffer
			r := NewCLIRunner("/bin/sh", "")
			r.Stdout = &streamOut
			r.Stderr = &streamErr

			res, err := r.Run(t.Context(), nil, tt.args...)
			require.NoError(t, err)

			require.Equal(t, tt.wantStdout, streamOut.String(), "streamed stdout")
			require.Equal(t, tt.wantStderr, streamErr.String(), "streamed stderr")

			require.Equal(t, tt.wantStdout, string(res.Stdout), "captured stdout")
			require.Equal(t, tt.wantStderr, string(res.Stderr), "captured stderr")
		})
	}
}

func TestCLIRunner_ContextRegistries(t *testing.T) {
	t.Parallel()

	want := []fcre.ContextRegistryEntry{{ID: "a", Label: "L", Type: "off-chain"}}
	r := NewCLIRunner("/bin/sh", "", WithContextRegistries(want))
	got := r.ContextRegistries()
	require.Equal(t, want, got)
	// Returned slice is a copy; mutating it does not affect the runner.
	got[0].ID = "mutated"
	got2 := r.ContextRegistries()
	require.Equal(t, "a", got2[0].ID)
}

func TestCLIRunner_NilWriters_DefaultBehavior(t *testing.T) {
	t.Parallel()

	r := NewCLIRunner("/bin/sh", "")
	res, err := r.Run(t.Context(), nil, "-c", `echo "works"`)
	require.NoError(t, err)
	require.Equal(t, "works\n", string(res.Stdout))
}
