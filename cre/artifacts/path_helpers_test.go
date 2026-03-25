package artifacts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_resolveLocalArtifactPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	good := filepath.Join(dir, "f.json")
	require.NoError(t, os.WriteFile(good, []byte("{}"), 0o600))

	p, err := resolveLocalArtifactPath(good)
	require.NoError(t, err)
	require.Equal(t, good, p)

	_, err = resolveLocalArtifactPath(filepath.Join(dir, "nope"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")

	sub := filepath.Join(dir, "d")
	require.NoError(t, os.Mkdir(sub, 0o700))
	_, err = resolveLocalArtifactPath(sub)
	require.Error(t, err)
	require.Contains(t, err.Error(), "directory")
}
