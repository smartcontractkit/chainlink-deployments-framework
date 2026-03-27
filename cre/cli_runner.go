package cre

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
)

const defaultBinary = "cre"

// cliRunner runs the CRE CLI via os/exec. Run executes the binary and captures stdout/stderr.
// It implements the [CLIRunner] interface.
type cliRunner struct {
	binaryPath string
	// Stdout, if set, receives a real-time copy of the process stdout while it runs.
	Stdout io.Writer
	// Stderr, if set, receives a real-time copy of the process stderr while it runs.
	Stderr io.Writer
}

var _ CLIRunner = (*cliRunner)(nil)

// NewCLIRunner returns a [cliRunner] for the given binary path. An empty path defaults to "cre"
// (resolved via PATH).
func NewCLIRunner(binaryPath string) *cliRunner {
	if binaryPath == "" {
		binaryPath = defaultBinary
	}

	return &cliRunner{binaryPath: binaryPath}
}

// Run executes the binary and captures stdout and stderr. Exit code 0 returns (res, nil);
// exit code != 0 returns (res, *ExitError) so callers get both result and error.
// CLI invocation failures (binary not found, context canceled) return (nil, err).
func (r *cliRunner) Run(ctx context.Context, args ...string) (*CallResult, error) {
	//nolint:gosec // G204: This is intentional - we're running a CLI tool with user-provided arguments.
	// The binary path is controlled via configuration, and args are expected to be user-provided CLI arguments.
	cmd := exec.CommandContext(ctx, r.binaryPath, args...)

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
