package artifacts

import (
	"net/http"
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
	require.Empty(t, nilR.WorkDir())
}

func TestWithHTTPClient(t *testing.T) {
	t.Parallel()
	c := &http.Client{}
	r, err := NewArtifactsResolver(t.TempDir(), WithHTTPClient(c))
	require.NoError(t, err)
	require.Equal(t, c, r.client)
}
