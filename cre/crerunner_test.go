package cre

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCRERunner(t *testing.T) {
	t.Parallel()

	cli := NewCLIRunner("/bin/sh")
	var cr stubClientRunner

	r := NewCRERunner(WithCLI(cli), WithClient(&cr))
	require.NotNil(t, r)
	require.Equal(t, cli, r.CLI())
	require.Equal(t, &cr, r.Client())
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

// stubClientRunner implements [ClientRunner] for tests (empty interface: any concrete type will do).
type stubClientRunner struct{}
