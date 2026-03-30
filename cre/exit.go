package cre

import "fmt"

// ExitError is returned when the CRE process ran and exited with a non-zero code.
// Use errors.As to inspect ExitCode, Stdout, and Stderr. Result is still returned
// from Run (CLIRunner.Run) so callers can log or inspect output.
type ExitError struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("cre: exited with code %d", e.ExitCode)
}
