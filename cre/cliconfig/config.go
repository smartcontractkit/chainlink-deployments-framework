package cliconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// UserWorkflow is the user-workflow section of workflow.yaml.
type UserWorkflow struct {
	DeploymentRegistry string `json:"deploymentRegistry,omitempty" yaml:"deployment-registry,omitempty"`
	WorkflowName       string `json:"workflowName" yaml:"workflow-name"`
}

// WorkflowArtifacts is the workflow-artifacts section of workflow.yaml.
type WorkflowArtifacts struct {
	WorkflowPath string `json:"workflowPath" yaml:"workflow-path"`
	ConfigPath   string `json:"configPath" yaml:"config-path"`
	SecretsPath  string `json:"secretsPath,omitempty" yaml:"secrets-path,omitempty"`
}

// WorkflowTarget is one target block in workflow.yaml.
type WorkflowTarget struct {
	UserWorkflow      UserWorkflow      `json:"userWorkflow" yaml:"user-workflow"`
	WorkflowArtifacts WorkflowArtifacts `json:"workflowArtifacts" yaml:"workflow-artifacts"`
}

// WorkflowConfig is the full workflow.yaml document.
type WorkflowConfig map[string]WorkflowTarget

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
	if strings.TrimSpace(r.Type) == "" {
		return fmt.Errorf("registry %q: type is required", r.ID)
	}
	return nil
}

// ContextOverrides holds optional user-level overrides for the generated context.yaml.
// When fields are empty, values fall back to CRE_* process environment variables.
type ContextOverrides struct {
	TenantID   string                 `json:"tenantId,omitempty" yaml:"tenantId,omitempty"`
	GatewayURL string                 `json:"gatewayUrl,omitempty" yaml:"gatewayUrl,omitempty"`
	Registries []ContextRegistryEntry `json:"registries,omitempty" yaml:"registries,omitempty"`
}

// ContextEnvironment is one environment block (e.g. PRODUCTION) in context.yaml.
type ContextEnvironment struct {
	TenantID   string                 `json:"tenantId" yaml:"tenant_id"`
	DonFamily  string                 `json:"donFamily" yaml:"don_family"`
	GatewayURL string                 `json:"gatewayUrl" yaml:"gateway_url"`
	Registries []ContextRegistryEntry `json:"registries,omitempty" yaml:"registries,omitempty"`
}

// ContextConfig is the full context.yaml document (environment name → config).
type ContextConfig map[string]ContextEnvironment

// WriteWorkflowYAML writes workflow.yaml to dir and returns the file path.
func WriteWorkflowYAML(dir string, cfg WorkflowConfig) (string, error) {
	return writeYAMLFile(dir, "workflow.yaml", cfg)
}

// WriteContextYAML writes context.yaml to dir and returns the file path.
func WriteContextYAML(dir string, cfg ContextConfig) (string, error) {
	return writeYAMLFile(dir, "context.yaml", cfg)
}

func writeYAMLFile(dir, name string, v any) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	out, err := yaml.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal %s: %w", name, err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}
