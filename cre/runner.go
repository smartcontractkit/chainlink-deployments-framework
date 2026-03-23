package cre

import "context"

// CallResult holds stdout, stderr, and exit code from a completed CRE call.
type CallResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner is used to invoke the CRE CLI.
type Runner interface {
	Run(ctx context.Context, args ...string) (*CallResult, error)
}
