package artifacts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewArtifactsResolver_requiresWorkDir(t *testing.T) {
	t.Parallel()
	_, err := NewArtifactsResolver("")
	require.Error(t, err)
	_, err = NewArtifactsResolver("   ")
	require.Error(t, err)
}

func TestResolveBinary_local(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	good := filepath.Join(dir, "workflow.wasm")
	require.NoError(t, os.WriteFile(good, []byte("wasm"), 0o600))

	badExt := filepath.Join(dir, "x.txt")
	require.NoError(t, os.WriteFile(badExt, []byte("x"), 0o600))

	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0o700))

	tests := []struct {
		name    string
		src     BinarySource
		wantErr string
	}{
		{
			name: "ok",
			src:  BinarySource{LocalPath: good},
		},
		{
			name:    "missing",
			src:     BinarySource{LocalPath: filepath.Join(dir, "missing.wasm")},
			wantErr: "does not exist",
		},
		{
			name:    "directory",
			src:     BinarySource{LocalPath: sub},
			wantErr: "directory",
		},
		{
			name:    "wrong_extension",
			src:     BinarySource{LocalPath: badExt},
			wantErr: ".wasm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			r, err := NewArtifactsResolver(t.TempDir())
			require.NoError(t, err)
			path, err := r.ResolveBinary(ctx, tt.src)
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

func TestResolveBinary_workDir(t *testing.T) {
	t.Parallel()
	payload := []byte("hello wasm")
	sum := sha256.Sum256(payload)
	validHex := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	workDir := t.TempDir()
	src := BinarySource{
		ExternalRef: &ExternalBinaryRef{
			URL:    srv.URL + "/binary.wasm",
			SHA256: validHex,
		},
	}
	ctx := t.Context()
	r, err := NewArtifactsResolver(workDir, WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	path, err := r.ResolveBinary(ctx, src)
	require.NoError(t, err)
	base := filepath.Base(path)
	require.True(t, strings.HasPrefix(base, "workflow-"), base)
	require.True(t, strings.HasSuffix(base, ".wasm"), base)
	require.Equal(t, workDir, filepath.Dir(path))
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, payload, b)
}

func TestResolveBinary_downloadURL(t *testing.T) {
	t.Parallel()
	payload := []byte("hello wasm")
	sum := sha256.Sum256(payload)
	validHex := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "want GET", http.StatusMethodNotAllowed)
			return
		}
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	tests := []struct {
		name    string
		sha     string
		wantErr string
		urlFn   func() string
	}{
		{name: "ok", sha: validHex, urlFn: func() string { return srv.URL + "/binary.wasm" }},
		{name: "bad_sha", sha: strings.Repeat("ab", 32), wantErr: "sha256 mismatch", urlFn: func() string { return srv.URL + "/binary.wasm" }},
		{
			name: "bad_status", sha: validHex, wantErr: "unexpected status",
			urlFn: func() string {
				failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				t.Cleanup(failSrv.Close)

				return failSrv.URL + "/x"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := BinarySource{
				ExternalRef: &ExternalBinaryRef{
					URL:    tt.urlFn(),
					SHA256: tt.sha,
				},
			}
			ctx := t.Context()
			workDir := t.TempDir()
			r, err := NewArtifactsResolver(workDir, WithHTTPClient(srv.Client()))
			require.NoError(t, err)
			path, err := r.ResolveBinary(ctx, src)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)

				return
			}
			require.NoError(t, err)
			require.Equal(t, ".wasm", filepath.Ext(path))
			b, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, payload, b)
		})
	}
}

func TestResolveBinary_gitHubRelease(t *testing.T) {
	t.Parallel()
	payload := []byte("asset-bytes")
	sum := sha256.Sum256(payload)
	validHex := hex.EncodeToString(sum[:])

	tests := []struct {
		name       string
		repo       string
		releaseTag string
		assetName  string
		sha        string
		wantErr    string
		handler    func(srv **httptest.Server, payload []byte) http.HandlerFunc
	}{
		{
			name: "success", repo: "org/repo", releaseTag: "v1.0.0", assetName: "binary.wasm", sha: validHex,
			handler: func(srv **httptest.Server, payload []byte) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					s := *srv
					switch {
					case strings.HasPrefix(r.URL.Path, "/repos/") && strings.Contains(r.URL.Path, "/releases/tags/"):
						w.Header().Set("Content-Type", "application/json")
						rel := map[string]any{
							"assets": []map[string]string{
								{
									"name":                 "binary.wasm",
									"url":                  "https://api.github.com/repos/org/repo/releases/assets/123",
									"browser_download_url": s.URL + "/download/binary.wasm",
								},
							},
						}
						if err := json.NewEncoder(w).Encode(rel); err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
					case strings.Contains(r.URL.Path, "/releases/assets/"):
						if r.Header.Get("Accept") != "application/octet-stream" {
							http.Error(w, "bad Accept", http.StatusBadRequest)
							return
						}
						_, _ = w.Write(payload)
					default:
						w.WriteHeader(http.StatusNotFound)
					}
				}
			},
		},
		{
			name: "missing_asset", repo: "o/r", releaseTag: "v1", assetName: "want.wasm",
			sha:     strings.Repeat("ab", 32),
			wantErr: `o/r tag "v1": asset "want.wasm" not found`,
			handler: func(srv **httptest.Server, _ []byte) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					s := *srv
					if strings.Contains(r.URL.Path, "/releases/tags/") {
						_ = json.NewEncoder(w).Encode(map[string]any{
							"assets": []map[string]string{
								{"name": "other.wasm", "browser_download_url": s.URL + "/x"},
							},
						})

						return
					}
					w.WriteHeader(http.StatusNotFound)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var srv *httptest.Server
			srv = httptest.NewServer(tt.handler(&srv, payload))
			t.Cleanup(srv.Close)

			client := &http.Client{
				Transport: rewriteGitHubAPIToTestServer(srv.URL, http.DefaultTransport),
			}

			src := BinarySource{
				ExternalRef: &ExternalBinaryRef{
					Repo:       tt.repo,
					ReleaseTag: tt.releaseTag,
					AssetName:  tt.assetName,
					SHA256:     tt.sha,
				},
			}

			workDir := t.TempDir()
			r, err := NewArtifactsResolver(workDir, WithHTTPClient(client))
			require.NoError(t, err)
			path, err := r.ResolveBinary(t.Context(), src)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)

				return
			}
			require.NoError(t, err)
			require.Equal(t, ".wasm", filepath.Ext(path))
			b, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, payload, b)
		})
	}
}

func TestResolveBinary_parseSHA256Errors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("x"))
	}))
	t.Cleanup(srv.Close)

	baseURL := srv.URL + "/b.wasm"
	tests := []struct {
		name    string
		sha256  string
		wantErr string
	}{
		{name: "invalid_hex", sha256: "not-hex", wantErr: "invalid sha256"},
		{name: "wrong_length", sha256: "abcd", wantErr: "sha256 must"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := BinarySource{
				ExternalRef: &ExternalBinaryRef{
					URL:    baseURL,
					SHA256: tt.sha256,
				},
			}
			r, err := NewArtifactsResolver(t.TempDir(), WithHTTPClient(srv.Client()))
			require.NoError(t, err)
			_, err = r.ResolveBinary(t.Context(), src)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestResolveBinary_contextCanceled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	payload := []byte("x")
	sum := sha256.Sum256(payload)
	validHex := hex.EncodeToString(sum[:])

	src := BinarySource{
		ExternalRef: &ExternalBinaryRef{
			URL:    srv.URL + "/slow",
			SHA256: validHex,
		},
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r, err := NewArtifactsResolver(t.TempDir(), WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	_, err = r.ResolveBinary(ctx, src)
	require.Error(t, err)
}

// rewriteGitHubAPIToTestServer maps https://api.github.com/... to the httptest server for integration tests.
func rewriteGitHubAPIToTestServer(testBase string, rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api.github.com" {
			u, err := url.Parse(testBase + req.URL.Path)
			if err != nil {
				return nil, err
			}
			u.RawQuery = req.URL.RawQuery
			req2 := req.Clone(req.Context())
			req2.URL = u

			return rt.RoundTrip(req2)
		}

		return rt.RoundTrip(req)
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
