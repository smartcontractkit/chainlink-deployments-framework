package cre

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

const defaultBinary = "cre"

// CLIRunner runs the CRE CLI via os/exec. Call executes the binary and captures stdout/stderr.
type CLIRunner struct {
	// BinaryPath is the executable to run. Empty means "cre" (resolved via PATH).
	BinaryPath string
}

func (r *CLIRunner) binary() string {
	if r.BinaryPath != "" {
		return r.BinaryPath
	}

	return defaultBinary
}

// Call runs the binary and captures stdout and stderr. Exit code 0 returns (res, nil);
// exit code != 0 returns (res, *ExitError) so callers get both result and error.
// Runner-related failures (binary not found, context canceled) return (nil, err).
func (r *CLIRunner) Call(ctx context.Context, args ...string) (*CallResult, error) {
	//nolint:gosec // G204: This is intentional - we're running a CLI tool with user-provided arguments.
	// The binary path is controlled via configuration, and args are expected to be user-provided CLI arguments.
	cmd := exec.CommandContext(ctx, r.binary(), args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := &CallResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: 0,
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			return res, &ExitError{
				ExitCode: res.ExitCode,
				Stdout:   res.Stdout,
				Stderr:   res.Stderr,
			}
		}

		return nil, err
	}

	return res, nil
}
