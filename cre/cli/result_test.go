package cli

import (
	"testing"

	"github.com/stretchr/testify/require"

	fcre "github.com/smartcontractkit/chainlink-deployments-framework/cre"
)

func TestExitError(t *testing.T) {
	t.Parallel()
	e := &fcre.ExitError{ExitCode: 3, Stderr: []byte("failed")}
	require.ErrorContains(t, e, "code 3")
	var out *fcre.ExitError
	require.ErrorAs(t, e, &out)
	require.Equal(t, 3, out.ExitCode)
	require.Equal(t, "failed", string(out.Stderr))
}
