package artifacts

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_writeConfigFile(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		want := []byte(`{"ok":true}`)
		path, err := writeConfigFile(dir, bytes.NewReader(want))
		require.NoError(t, err)
		require.Equal(t, dir, filepath.Dir(path))
		base := filepath.Base(path)
		require.True(t, strings.HasPrefix(base, "workflow-config-"), base)
		require.True(t, strings.HasSuffix(base, ".json"), base)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("empty_work_dir", func(t *testing.T) {
		t.Parallel()
		_, err := writeConfigFile("", strings.NewReader("{}"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "workDir is required")
	})

	t.Run("whitespace_work_dir", func(t *testing.T) {
		t.Parallel()
		_, err := writeConfigFile("   ", strings.NewReader("{}"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "workDir is required")
	})

	t.Run("mkdir_fails_when_path_is_file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		blocksDir := filepath.Join(dir, "not-a-dir")
		require.NoError(t, os.WriteFile(blocksDir, []byte("x"), 0o600))
		_, err := writeConfigFile(blocksDir, strings.NewReader("{}"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "config work dir")
	})

	t.Run("copy_error_removes_partial_file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, err := writeConfigFile(dir, errReader{err: errors.New("read boom")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "config write")
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Empty(t, entries, "partial config file should be removed on write failure")
	})
}

type errReader struct {
	err error
}

func (e errReader) Read(p []byte) (int, error) {
	return 0, e.err
}

func Test_configExternalRefSummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ref  *ExternalConfigRef
		want string
	}{
		{
			name: "nil",
			ref:  nil,
			want: "externalRef=<nil>",
		},
		{
			name: "empty_fields",
			ref:  &ExternalConfigRef{},
			want: `url="" repo="" ref="" path=""`,
		},
		{
			name: "all_fields",
			ref: &ExternalConfigRef{
				URL:  "https://example/cfg.json",
				Repo: "org/repo",
				Ref:  "main",
				Path: "path/to/config.json",
			},
			want: `url="https://example/cfg.json" repo="org/repo" ref="main" path="path/to/config.json"`,
		},
		{
			name: "trims_whitespace",
			ref: &ExternalConfigRef{
				URL:  "  https://x  ",
				Repo: " org/r ",
				Ref:  " v1 ",
				Path: " a/b.json ",
			},
			want: `url="https://x" repo="org/r" ref="v1" path="a/b.json"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, configExternalRefSummary(tt.ref))
		})
	}
}

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
