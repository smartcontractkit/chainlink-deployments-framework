package cli

import (
	"fmt"
	"os"
	"path/filepath"

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

// WriteWorkflowYAML writes workflow.yaml to dir and returns the file path.
func WriteWorkflowYAML(dir string, cfg WorkflowConfig) (string, error) {
	return writeYAMLFile(dir, "workflow.yaml", cfg)
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
