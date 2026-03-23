package artifacts

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_httpGet(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(srv.Close)

	ctx := t.Context()
	client := srv.Client()
	resp, err := httpGet(ctx, client, srv.URL+"/x", "test op")
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(b))

	_, err = httpGet(ctx, client, srv.URL+"/missing", "test op")
	require.Error(t, err)
	require.Contains(t, err.Error(), "404")
}

func Test_githubRESTGETBytes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
		_, _ = io.WriteString(w, `{"x":1}`)
	}))
	t.Cleanup(srv.Close)

	ctx := t.Context()
	client := srv.Client()
	body, err := githubGet(ctx, client, srv.URL+"/api", "test gh")
	require.NoError(t, err)
	require.Equal(t, `{"x":1}`, string(body))
}

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
