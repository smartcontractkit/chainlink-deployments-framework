package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fcre "github.com/smartcontractkit/chainlink-deployments-framework/cre"

	"gopkg.in/yaml.v3"
)

func TestContextRegistryEntry_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entry   fcre.ContextRegistryEntry
		wantErr string
	}{
		{
			name:  "valid entry",
			entry: fcre.ContextRegistryEntry{ID: "private", Label: "Private", Type: "off-chain"},
		},
		{
			name:    "missing id",
			entry:   fcre.ContextRegistryEntry{Label: "Private", Type: "off-chain"},
			wantErr: "registry id is required",
		},
		{
			name:    "blank id",
			entry:   fcre.ContextRegistryEntry{ID: "  ", Label: "Private", Type: "off-chain"},
			wantErr: "registry id is required",
		},
		{
			name:    "missing label",
			entry:   fcre.ContextRegistryEntry{ID: "private", Type: "off-chain"},
			wantErr: `registry "private": label is required`,
		},
		{
			name:    "missing type",
			entry:   fcre.ContextRegistryEntry{ID: "private", Label: "Private"},
			wantErr: `registry "private": type is required`,
		},
		{
			name:    "invalid type",
			entry:   fcre.ContextRegistryEntry{ID: "private", Label: "Private", Type: "private"},
			wantErr: `registry "private": invalid type "private" (allowed: on-chain, off-chain)`,
		},
		{
			name:  "valid type is case insensitive",
			entry: fcre.ContextRegistryEntry{ID: "private", Label: "Private", Type: "On-Chain"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.entry.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestWriteWorkflowYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := WorkflowConfig{
		"staging": {
			UserWorkflow:      UserWorkflow{DeploymentRegistry: "private", WorkflowName: "wf"},
			WorkflowArtifacts: WorkflowArtifacts{WorkflowPath: ".", ConfigPath: "c.json"},
		},
	}
	path, err := WriteWorkflowYAML(dir, cfg)
	require.NoError(t, err)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var got WorkflowConfig
	require.NoError(t, yaml.Unmarshal(raw, &got))
	require.Equal(t, "private", got["staging"].UserWorkflow.DeploymentRegistry)
}
