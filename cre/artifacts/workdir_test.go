package artifacts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newWorkDirBinaryFileName_unique(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{}, 200)
	for range 200 {
		n := newWorkDirBinaryFileName()
		require.NotContains(t, seen, n)
		seen[n] = struct{}{}
	}
}

func Test_newWorkDirConfigFileName_unique(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{}, 200)
	for range 200 {
		n := newWorkDirConfigFileName()
		require.NotContains(t, seen, n)
		seen[n] = struct{}{}
	}
}
