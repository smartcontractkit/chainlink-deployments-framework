package cre

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	t.Parallel()

	cli := NewCLIRunner("/bin/sh")
	var cr stubClientRunner

	r := NewRunner(WithCLI(cli), WithClient(&cr))
	require.NotNil(t, r)
	require.Equal(t, cli, r.CLI())
	require.Equal(t, &cr, r.Client())
}

func TestNewRunner_Empty(t *testing.T) {
	t.Parallel()

	r := NewRunner()
	require.NotNil(t, r)
	require.Nil(t, r.CLI())
	require.Nil(t, r.Client())
}

func TestRunner_NilInterface(t *testing.T) {
	t.Parallel()

	var r Runner
	require.Nil(t, r)
	// A nil Runner interface must not have methods called on it; check non-nil before CLI() / Client().
}

// stubClientRunner implements [Client] for tests (empty interface: any concrete type will do).
type stubClientRunner struct{}
