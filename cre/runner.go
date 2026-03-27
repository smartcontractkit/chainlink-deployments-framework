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

// WorkflowRunner is a placeholder for the future CRE v2 Go client. No methods yet—the real API will be added when the CRE Go library is integrated.
//
// TODO: Add methods (e.g. DeployWorkflow) and supporting config types once the library contract is clear.
// TODO: Revisit layout: consider moving [WorkflowRunner] and concrete clients to a dedicated file or subpackage when the surface grows.
type WorkflowRunner interface{}

// CRERunner groups CLI and Go API access to CRE (v1 subprocess + v2 client).
// The default implementation is built with [NewCRERunner]; other implementations may be used in tests or for alternate wiring.
//
// If CRERunner is nil, do not call [CRERunner.CLI] or [CRERunner.Client] (unlike a nil concrete pointer, a nil interface cannot be used as a receiver).
type CRERunner interface {
	CLI() CLIRunner
	Client() WorkflowRunner
}

// crerunner is the default [CRERunner] implementation.
type crerunner struct {
	cli      CLIRunner
	workflow WorkflowRunner
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

// WithClient sets the Go API [WorkflowRunner] (v2). The accessor is [CRERunner.Client] for a familiar call shape.
func WithClient(client WorkflowRunner) CRERunnerOption {
	return func(r *crerunner) {
		r.workflow = client
	}
}

// CLI returns the CLI runner, or nil if none was configured.
func (r *crerunner) CLI() CLIRunner {
	if r == nil {
		return nil
	}

	return r.cli
}

// Client returns the Go API client ([WorkflowRunner]), or nil if none was configured.
func (r *crerunner) Client() WorkflowRunner {
	if r == nil {
		return nil
	}

	return r.workflow
}
