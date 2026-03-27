package cre

import "context"

// CallResult holds stdout, stderr, and exit code from a completed CRE call.
type CallResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// CLIRunner is the interface for running the CRE binary as a subprocess (v1 / CLI access).
// The default implementation is created by [NewCLIRunner].
type CLIRunner interface {
	Run(ctx context.Context, args ...string) (*CallResult, error)
}

// Client is a placeholder for the future CRE v2 Go client. No methods yet—the real API will be added when the CRE Go library is integrated.
//
// TODO: Add methods (e.g. DeployWorkflow) and supporting config types once the library contract is clear.
// TODO: Revisit layout: consider moving [Client] and concrete clients to a dedicated file or subpackage when the surface grows.
type Client interface{}

// Runner groups CLI and Go API access to CRE (v1 subprocess + v2 client).
// The default implementation is built with [NewRunner]; other implementations may be used in tests or for alternate wiring.
//
// If the environment field CRERunner is nil, do not call [Runner.CLI] or [Runner.Client] (unlike a nil concrete pointer, a nil interface cannot be used as a receiver).
type Runner interface {
	CLI() CLIRunner
	Client() Client
}

// crerunner is the default [Runner] implementation.
type crerunner struct {
	cli    CLIRunner
	client Client
}

var _ Runner = (*crerunner)(nil)

// RunnerOption configures a [crerunner] instance for [NewRunner].
type RunnerOption func(*crerunner)

// NewRunner returns a [Runner] with the given options applied.
func NewRunner(opts ...RunnerOption) Runner {
	c := &crerunner{}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithCLI sets the CLI [CLIRunner] (subprocess).
func WithCLI(cli CLIRunner) RunnerOption {
	return func(r *crerunner) {
		r.cli = cli
	}
}

// WithClient sets the Go API [Client] (v2).
func WithClient(client Client) RunnerOption {
	return func(r *crerunner) {
		r.client = client
	}
}

// CLI returns the CLI runner, or nil if none was configured.
func (r *crerunner) CLI() CLIRunner {
	if r == nil {
		return nil
	}

	return r.cli
}

// Client returns the Go API client ([Client]), or nil if none was configured.
func (r *crerunner) Client() Client {
	if r == nil {
		return nil
	}

	return r.client
}
