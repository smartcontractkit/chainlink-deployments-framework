package cre

import (
	"context"
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLIRunner_binary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		binaryPath string
		want       string
	}{
		{"default_empty", "", defaultBinary},
		{"custom_path", "/opt/cre", "/opt/cre"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &CLIRunner{BinaryPath: tt.binaryPath}
			require.Equal(t, tt.want, r.binary())
		})
	}
}

func TestCLIRunner_Call(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		skip           string
		setupCtx       func(*testing.T) context.Context
		runner         *CLIRunner
		args           []string
		wantErr        bool
		wantResNil     bool
		wantExitCode   int
		wantErrIs      error
		checkExitError bool
	}{
		{
			name:    "binary_not_found",
			runner:  &CLIRunner{BinaryPath: filepath.Join(t.TempDir(), "nonexistent-cre-xyz")},
			args:    []string{"build"},
			wantErr: true, wantResNil: true,
		},
		{
			name:   "context_already_canceled",
			runner: &CLIRunner{BinaryPath: "cre"},
			setupCtx: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				return ctx
			},
			args:       []string{"build"},
			wantErr:    true,
			wantResNil: true,
			wantErrIs:  context.Canceled,
		},
		{
			name:           "nonzero_exit_returns_error",
			skip:           "windows",
			runner:         &CLIRunner{BinaryPath: "/bin/sh"},
			args:           []string{"-c", "exit 41"},
			wantErr:        true,
			wantResNil:     false,
			wantExitCode:   41,
			checkExitError: true,
		},
		{
			name:         "success",
			skip:         "windows",
			runner:       &CLIRunner{BinaryPath: "true"},
			args:         nil,
			wantErr:      false,
			wantResNil:   false,
			wantExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip == "windows" && runtime.GOOS == "windows" {
				t.Skip("skipped on windows")
			}
			if tt.name == "success" {
				if _, err := exec.LookPath("true"); err != nil {
					t.Skip("true not in PATH")
				}
			}
			t.Parallel()

			ctx := t.Context()
			if tt.setupCtx != nil {
				ctx = tt.setupCtx(t)
			}
			res, err := tt.runner.Call(ctx, tt.args...)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantResNil {
					require.Nil(t, res)
				} else {
					require.NotNil(t, res)
					require.Equal(t, tt.wantExitCode, res.ExitCode)
				}
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				if tt.checkExitError {
					var exitErr *ExitError
					require.ErrorAs(t, err, &exitErr)
					require.Equal(t, tt.wantExitCode, exitErr.ExitCode)
				}
				if tt.name == "binary_not_found" {
					require.True(t,
						errors.Is(err, fs.ErrNotExist) || errors.Is(err, exec.ErrNotFound) || errors.Is(err, exec.ErrDot),
						"expected not found style error, got %v", err)
				}

				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, tt.wantExitCode, res.ExitCode)
		})
	}
}
