package cre

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRunners(t *testing.T) {
	t.Parallel()

	cli := NewCLIRunner("/bin/sh")
	var api stubWorkflowAPI

	r := NewRunners(WithCLI(cli), WithWorkflowAPI(&api))
	require.NotNil(t, r)
	require.Equal(t, cli, r.CLI())
	require.Equal(t, &api, r.Workflow())
}

func TestNewRunners_Empty(t *testing.T) {
	t.Parallel()

	r := NewRunners()
	require.NotNil(t, r)
	require.Nil(t, r.CLI())
	require.Nil(t, r.Workflow())
}

func TestRunners_CLI_Workflow_NilReceiver(t *testing.T) {
	t.Parallel()

	var r *Runners
	require.Nil(t, r.CLI())
	require.Nil(t, r.Workflow())
}

// stubWorkflowAPI implements [WorkflowAPI] for tests.
type stubWorkflowAPI struct{}

func (stubWorkflowAPI) DeployWorkflow(context.Context, DeployWorkflowConfig) error { return nil }
