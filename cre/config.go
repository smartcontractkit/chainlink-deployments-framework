package cre

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RPCEntry is one RPC endpoint for CRE project.yaml.
type RPCEntry struct {
	ChainName string `json:"chainName" yaml:"chain-name"`
	URL       string `json:"url" yaml:"url"`
}

// ProjectTargetCRECLI holds cre-cli settings under a project.yaml target.
type ProjectTargetCRECLI struct {
	DonFamily string `json:"donFamily" yaml:"don-family"`
}

// ProjectTarget is one target block in project.yaml.
type ProjectTarget struct {
	CreCLI ProjectTargetCRECLI `json:"cre-cli" yaml:"cre-cli"`
	RPCs   []RPCEntry          `json:"rpcs,omitempty" yaml:"rpcs,omitempty"`
}

// ProjectConfig is the full project.yaml document (map: target name -> target config).
type ProjectConfig map[string]ProjectTarget

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

// ContextRegistryEntry is one registry in context.yaml (CRE CLI).
type ContextRegistryEntry struct {
	ID               string   `json:"id" yaml:"id"`
	Label            string   `json:"label" yaml:"label"`
	Type             string   `json:"type" yaml:"type"` // "on-chain" or "off-chain"
	Address          string   `json:"address,omitempty" yaml:"address,omitempty"`
	ChainName        string   `json:"chainName,omitempty" yaml:"chain_name,omitempty"`
	SecretsAuthFlows []string `json:"secretsAuthFlows,omitempty" yaml:"secrets_auth_flows,omitempty"`
}

// ContextEnvironment is one environment block (e.g. PRODUCTION) in context.yaml.
type ContextEnvironment struct {
	TenantID   string                 `json:"tenantId" yaml:"tenant_id"`
	DonFamily  string                 `json:"donFamily" yaml:"don_family"`
	GatewayURL string                 `json:"gatewayUrl" yaml:"gateway_url"`
	Registries []ContextRegistryEntry `json:"registries,omitempty" yaml:"registries,omitempty"`
}

// ContextConfig is the full context.yaml document (environment name -> config).
type ContextConfig map[string]ContextEnvironment

// WriteProjectYAML writes project.yaml to dir and returns the file path.
func WriteProjectYAML(dir string, cfg ProjectConfig) (string, error) {
	return writeYAMLFile(dir, "project.yaml", cfg)
}

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
