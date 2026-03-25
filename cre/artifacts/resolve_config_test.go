package artifacts

import (
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

func Test_writeConfigToWorkDir(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		want := []byte(`{"ok":true}`)
		path, err := writeConfigToWorkDir(strings.NewReader(string(want)), dir)
		require.NoError(t, err)
		require.Equal(t, dir, filepath.Dir(path))
		base := filepath.Base(path)
		require.True(t, strings.HasPrefix(base, "workflow-config-"), base)
		require.True(t, strings.HasSuffix(base, ".json"), base)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("copy_error_removes_partial_file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, err := writeConfigToWorkDir(errReader{err: errors.New("read boom")}, dir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "write file")
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
		{
			name: "trims_leading_slashes_on_path",
			ref: &ExternalConfigRef{
				Repo: "org/repo",
				Ref:  "main",
				Path: " /dir/cfg.json ",
			},
			want: `url="" repo="org/repo" ref="main" path="dir/cfg.json"`,
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

func TestResolveConfig_external(t *testing.T) {
	t.Parallel()

	urlPayload := []byte(`{"env":"staging"}`)
	ghPayload := []byte(`{"x":true}`)
	ghEncoded := base64.StdEncoding.EncodeToString(ghPayload)

	tests := []struct {
		name    string
		want    []byte
		setup   func(t *testing.T) (ConfigSource, *http.Client)
		wantErr string
	}{
		{
			name: "url",
			want: urlPayload,
			setup: func(t *testing.T) (ConfigSource, *http.Client) {
				t.Helper()
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet {
						http.Error(w, "want GET", http.StatusMethodNotAllowed)
						return
					}
					_, _ = w.Write(urlPayload)
				}))
				t.Cleanup(srv.Close)

				return ConfigSource{ExternalRef: &ExternalConfigRef{URL: srv.URL + "/config.json"}}, srv.Client()
			},
		},
		{
			name: "github_file",
			want: ghPayload,
			setup: func(t *testing.T) (ConfigSource, *http.Client) {
				t.Helper()
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
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{
						"type":     "file",
						"encoding": "base64",
						"content":  ghEncoded,
					})
				}))
				t.Cleanup(srv.Close)

				src := ConfigSource{
					ExternalRef: &ExternalConfigRef{Repo: "org/repo", Ref: "main", Path: "workflows/config.json"},
				}
				client := &http.Client{
					Transport: rewriteGitHubAPIToTestServer(srv.URL, http.DefaultTransport),
				}

				return src, client
			},
		},
		{
			name:    "url_404",
			wantErr: "404",
			setup: func(t *testing.T) (ConfigSource, *http.Client) {
				t.Helper()
				srv := httptest.NewServer(http.NotFoundHandler())
				t.Cleanup(srv.Close)

				return ConfigSource{ExternalRef: &ExternalConfigRef{URL: srv.URL + "/missing.json"}}, srv.Client()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src, client := tt.setup(t)
			workDir := t.TempDir()
			r, err := NewArtifactsResolver(workDir, WithHTTPClient(client))
			require.NoError(t, err)
			path, err := r.ResolveConfig(t.Context(), src)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)

				return
			}
			require.NoError(t, err)
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
