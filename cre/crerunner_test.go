package cre

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCRERunner(t *testing.T) {
	t.Parallel()

	cli := NewCLIRunner("/bin/sh")
	var wf stubWorkflowRunner

	r := NewCRERunner(WithCLI(cli), WithClient(&wf))
	require.NotNil(t, r)
	require.Equal(t, cli, r.CLI())
	require.Equal(t, &wf, r.Client())
}

func TestNewCRERunner_Empty(t *testing.T) {
	t.Parallel()

	r := NewCRERunner()
	require.NotNil(t, r)
	require.Nil(t, r.CLI())
	require.Nil(t, r.Client())
}

func TestCRERunner_NilInterface(t *testing.T) {
	t.Parallel()

	var r CRERunner
	require.Nil(t, r)
}

type stubWorkflowRunner struct{}
