package cre

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
)

const defaultBinary = "cre"

// CLIRunner runs the CRE CLI via os/exec. Run executes the binary and captures stdout/stderr.
type CLIRunner struct {
	// BinaryPath is the executable to run. Empty means "cre" (resolved via PATH).
	BinaryPath string
	Stdout     io.Writer
	Stderr     io.Writer
}

// NewCLIRunner returns a CLIRunner that resolves "cre" from PATH.
func NewCLIRunner() *CLIRunner {
	return &CLIRunner{}
}

func (r *CLIRunner) binary() string {
	if r.BinaryPath != "" {
		return r.BinaryPath
	}

	return defaultBinary
}

// Run runs the binary and captures stdout and stderr. Exit code 0 returns (res, nil);
// exit code != 0 returns (res, *ExitError) so callers get both result and error.
// Runner-related failures (binary not found, context canceled) return (nil, err).
func (r *CLIRunner) Run(ctx context.Context, args ...string) (*CallResult, error) {
	//nolint:gosec // G204: This is intentional - we're running a CLI tool with user-provided arguments.
	// The binary path is controlled via configuration, and args are expected to be user-provided CLI arguments.
	cmd := exec.CommandContext(ctx, r.binary(), args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = wrapWriter(&stdout, r.Stdout)
	cmd.Stderr = wrapWriter(&stderr, r.Stderr)

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

func wrapWriter(buf *bytes.Buffer, stream io.Writer) io.Writer {
	if stream == nil {
		return buf
	}
	return io.MultiWriter(buf, stream)
}
