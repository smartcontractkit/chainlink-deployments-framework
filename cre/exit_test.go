package cre

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExitError(t *testing.T) {
	t.Parallel()
	e := &ExitError{ExitCode: 3, Stderr: []byte("failed")}
	require.ErrorContains(t, e, "code 3")
	var out *ExitError
	require.ErrorAs(t, e, &out)
	require.Equal(t, 3, out.ExitCode)
	require.Equal(t, "failed", string(out.Stderr))
}
