package artifacts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactsResolver_WorkDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, err := NewArtifactsResolver(dir)
	require.NoError(t, err)
	require.Equal(t, dir, r.WorkDir())

	var nilR *ArtifactsResolver
	require.Equal(t, "", nilR.WorkDir())
}
