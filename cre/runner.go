package cre

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Client is a placeholder for the future CRE v2 Go client. No methods yet—the real API will be added when the CRE Go library is integrated.
// TODO: Add methods (e.g. DeployWorkflow) and supporting config types once the library contract is clear.
// TODO: Revisit layout: consider moving [Client] and concrete clients to a dedicated file or subpackage when the surface grows.
type Client interface{}

// Runner groups CLI and Go API access to CRE (v1 subprocess + v2 client).
type Runner interface {
	CLI() CLIRunner
	Client() Client
}

// runner is the default [Runner] implementation.
type runner struct {
	cli    CLIRunner
	client Client
}

var _ Runner = (*runner)(nil)

const (
	RegistryTypeOnChain  = "on-chain"
	RegistryTypeOffChain = "off-chain"
)

var validRegistryTypes = []string{
	RegistryTypeOnChain,
	RegistryTypeOffChain,
}

// RunnerOption configures a [runner] instance for [NewRunner].
type RunnerOption func(*runner)

// NewRunner returns a [Runner] with the given options applied.
func NewRunner(opts ...RunnerOption) Runner {
	c := &runner{}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithCLI sets the CLI [CLIRunner].
func WithCLI(cli CLIRunner) RunnerOption {
	return func(r *runner) {
		r.cli = cli
	}
}

// WithClient sets the Go API [Client].
func WithClient(client Client) RunnerOption {
	return func(r *runner) {
		r.client = client
	}
}

// CLI returns the CLI runner, or nil if none was configured.
func (r *runner) CLI() CLIRunner {
	if r == nil {
		return nil
	}

	return r.cli
}

// Client returns the Go API client ([Client]), or nil if none was configured.
func (r *runner) Client() Client {
	if r == nil {
		return nil
	}

	return r.client
}

// ContextRegistryEntry is one registry in context.yaml.
type ContextRegistryEntry struct {
	ID               string   `json:"id" mapstructure:"id" yaml:"id"`
	Label            string   `json:"label" mapstructure:"label" yaml:"label"`
	Type             string   `json:"type" mapstructure:"type" yaml:"type"` // "on-chain" or "off-chain"
	Address          string   `json:"address,omitempty" mapstructure:"address,omitempty" yaml:"address,omitempty"`
	ChainName        string   `json:"chainName,omitempty" mapstructure:"chain_name,omitempty" yaml:"chain_name,omitempty"`
	SecretsAuthFlows []string `json:"secretsAuthFlows,omitempty" mapstructure:"secrets_auth_flows,omitempty" yaml:"secrets_auth_flows,omitempty"`
}

// Validate checks that required fields (id, label, type) are non-empty.
func (r ContextRegistryEntry) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("registry id is required")
	}
	if strings.TrimSpace(r.Label) == "" {
		return fmt.Errorf("registry %q: label is required", r.ID)
	}

	registryType := strings.TrimSpace(r.Type)
	if registryType == "" {
		return fmt.Errorf("registry %q: type is required", r.ID)
	}
	if !isValidRegistryType(registryType) {
		return fmt.Errorf("registry %q: invalid type %q (allowed: %s)", r.ID, r.Type, strings.Join(validRegistryTypes, ", "))
	}

	return nil
}

// CLIRunner is the interface for running the CRE binary as a subprocess (v1 / CLI access).
type CLIRunner interface {
	// Run executes the CLI with optional per-invocation env vars.
	// Sensitive values should be passed via env and never written to disk.
	Run(ctx context.Context, env map[string]string, args ...string) (*CallResult, error)
	// ContextRegistries returns workflow registries defined from domain.yaml.
	ContextRegistries() []ContextRegistryEntry
}

func isValidRegistryType(value string) bool {
	for _, validType := range validRegistryTypes {
		if strings.EqualFold(value, validType) {
			return true
		}
	}

	return false
}
