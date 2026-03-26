package cre

import "context"

// CallResult holds stdout, stderr, and exit code from a completed CRE call.
type CallResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// CLIInvoker runs the CRE binary as a subprocess (v1 / CLI access).
type CLIInvoker interface {
	Run(ctx context.Context, args ...string) (*CallResult, error)
}

// WorkflowAPI is the typed Go API for CRE workflow operations (v2).
// Concrete implementations may be added later (e.g. gRPC/HTTP).
//
// TODO: Flesh out [DeployWorkflowConfig] and the real transport once the CRE Go client/library API is defined.
// TODO: Add PauseWorkflow / DeleteWorkflow (and config types) to this interface and wire implementations when those APIs exist.
// TODO: Revisit layout: consider moving [WorkflowAPI], config types, and concrete clients to a dedicated file (e.g. workflow_api.go) or subpackage (e.g. cre/workflow).
type WorkflowAPI interface {
	DeployWorkflow(ctx context.Context, cfg DeployWorkflowConfig) error
}

// DeployWorkflowConfig holds parameters for DeployWorkflow. Fields are TBD.
// TODO: Align fields with the CRE Go library once available.
type DeployWorkflowConfig struct{}

// Runners groups CLI and Go API access to CRE. Use [NewRunners] with [WithCLI] and/or [WithWorkflowAPI].
type Runners struct {
	cli      CLIInvoker
	workflow WorkflowAPI
}

// RunnersOption configures a [Runners] instance.
type RunnersOption func(*Runners)

// NewRunners returns a [Runners] with the given options applied.
func NewRunners(opts ...RunnersOption) *Runners {
	r := &Runners{}
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithCLI sets the CLI [CLIInvoker] (subprocess).
func WithCLI(cli CLIInvoker) RunnersOption {
	return func(r *Runners) {
		r.cli = cli
	}
}

// WithWorkflowAPI sets the workflow [WorkflowAPI].
func WithWorkflowAPI(workflow WorkflowAPI) RunnersOption {
	return func(r *Runners) {
		r.workflow = workflow
	}
}

// CLI returns the CLI invoker, or nil if none was configured.
func (r *Runners) CLI() CLIInvoker {
	if r == nil {
		return nil
	}

	return r.cli
}

// Workflow returns the workflow API, or nil if none was configured.
func (r *Runners) Workflow() WorkflowAPI {
	if r == nil {
		return nil
	}

	return r.workflow
}
