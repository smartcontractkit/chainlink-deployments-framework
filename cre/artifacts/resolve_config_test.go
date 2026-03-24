package artifacts

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveConfig_local(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	good := filepath.Join(dir, "cfg.json")
	require.NoError(t, os.WriteFile(good, []byte(`{"a":1}`), 0o600))

	sub := filepath.Join(dir, "d")
	require.NoError(t, os.Mkdir(sub, 0o700))

	tests := []struct {
		name    string
		src     ConfigSource
		wantErr string
	}{
		{name: "ok", src: ConfigSource{LocalPath: good}},
		{name: "missing", src: ConfigSource{LocalPath: filepath.Join(dir, "nope.json")}, wantErr: "does not exist"},
		{name: "dir", src: ConfigSource{LocalPath: sub}, wantErr: "directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			r, err := NewArtifactsResolver(t.TempDir())
			require.NoError(t, err)
			path, err := r.ResolveConfig(ctx, tt.src)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)

				return
			}
			require.NoError(t, err)
			require.Equal(t, good, path)
		})
	}
}

func TestResolveConfig_URL(t *testing.T) {
	t.Parallel()
	want := []byte(`{"env":"staging"}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "want GET", http.StatusMethodNotAllowed)
			return
		}
		_, _ = w.Write(want)
	}))
	t.Cleanup(srv.Close)

	src := ConfigSource{
		ExternalRef: &ExternalConfigRef{URL: srv.URL + "/config.json"},
	}
	ctx := t.Context()
	workDir := t.TempDir()
	r, err := NewArtifactsResolver(workDir, WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	path, err := r.ResolveConfig(ctx, src)
	require.NoError(t, err)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestResolveConfig_gitHubFile(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"x":true}`)
	encoded := base64.StdEncoding.EncodeToString(raw)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "want GET", http.StatusMethodNotAllowed)
			return
		}
		if !strings.Contains(r.URL.Path, "/contents/") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("ref") != "main" {
			http.Error(w, "bad ref", http.StatusBadRequest)
			return
		}
		resp := map[string]any{
			"type":     "file",
			"encoding": "base64",
			"content":  encoded,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{
		Transport: rewriteGitHubAPIToTestServer(srv.URL, http.DefaultTransport),
	}

	src := ConfigSource{
		ExternalRef: &ExternalConfigRef{
			Repo: "org/repo",
			Ref:  "main",
			Path: "workflows/config.json",
		},
	}
	ctx := t.Context()
	workDir := t.TempDir()
	r, err := NewArtifactsResolver(workDir, WithHTTPClient(client))
	require.NoError(t, err)
	path, err := r.ResolveConfig(ctx, src)
	require.NoError(t, err)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, raw, got)
}
