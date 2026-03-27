package cre

import "context"

// CallResult holds stdout, stderr, and exit code from a completed CRE call.
type CallResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// CLIRunner is the interface for running the CRE binary as a subprocess (v1 / CLI access).
// The default implementation is [cliRunner].
type CLIRunner interface {
	Run(ctx context.Context, args ...string) (*CallResult, error)
}

// ClientRunner is a placeholder for the future CRE v2 Go client. No methods yet—the real API will be added when the CRE Go library is integrated.
//
// TODO: Add methods (e.g. DeployWorkflow) and supporting config types once the library contract is clear.
// TODO: Revisit layout: consider moving [ClientRunner] and concrete clients to a dedicated file or subpackage when the surface grows.
type ClientRunner interface{}

// CRERunner groups CLI and Go API access to CRE (v1 subprocess + v2 client).
// The default implementation is built with [NewCRERunner]; other implementations may be used in tests or for alternate wiring.
//
// If CRERunner is nil, do not call [CRERunner.CLI] or [CRERunner.Client] (unlike a nil concrete pointer, a nil interface cannot be used as a receiver).
type CRERunner interface {
	CLI() CLIRunner
	Client() ClientRunner
}

// crerunner is the default [CRERunner] implementation.
type crerunner struct {
	cli    CLIRunner
	client ClientRunner
}

var _ CRERunner = (*crerunner)(nil)

// CRERunnerOption configures a [crerunner] instance for [NewCRERunner].
type CRERunnerOption func(*crerunner)

// NewCRERunner returns a [CRERunner] with the given options applied.
func NewCRERunner(opts ...CRERunnerOption) CRERunner {
	c := &crerunner{}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithCLI sets the CLI [CLIRunner] (subprocess).
func WithCLI(cli CLIRunner) CRERunnerOption {
	return func(r *crerunner) {
		r.cli = cli
	}
}

// WithClient sets the Go API [ClientRunner] (v2).
func WithClient(client ClientRunner) CRERunnerOption {
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

// Client returns the Go API client ([ClientRunner]), or nil if none was configured.
func (r *crerunner) Client() ClientRunner {
	if r == nil {
		return nil
	}

	return r.client
}
