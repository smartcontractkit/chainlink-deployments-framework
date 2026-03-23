package cre

import (
	"fmt"
	"strings"
)

// maxExitErrorStreamBytes is the max bytes per stream included in [ExitError.Error] (UTF-8 safe enough for logs).
const maxExitErrorStreamBytes = 4096

// ExitError is returned when the CRE process ran and exited with a non-zero code.
// Use errors.As to inspect ExitCode, Stdout, and Stderr. Result is still returned
// from Call so callers can log or inspect output.
type ExitError struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
}

func (e *ExitError) Error() string {
	if e == nil {
		return "cre: exit error: <nil>"
	}
	msg := fmt.Sprintf("cre: exited with code %d", e.ExitCode)
	out := formatExitErrorStreams(e.Stdout, e.Stderr)
	if out != "" {
		return msg + ": " + out
	}
	return msg
}

func formatExitErrorStreams(stdout, stderr []byte) string {
	var b strings.Builder
	appendStream(&b, "stderr", stderr)
	appendStream(&b, "stdout", stdout)
	return strings.TrimSpace(b.String())
}

func appendStream(b *strings.Builder, label string, data []byte) {
	s := strings.TrimSpace(string(data))
	if s == "" {
		return
	}
	if len(s) > maxExitErrorStreamBytes {
		s = s[:maxExitErrorStreamBytes] + "... [truncated]"
	}
	if b.Len() > 0 {
		b.WriteString("; ")
	}
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(s)
}
