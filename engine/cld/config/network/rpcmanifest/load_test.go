package rpcmanifest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func Test_fetchHTTP(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"ChainSelector":"1"}]`))
		}))
		defer srv.Close()

		data, err := fetchHTTP(context.Background(), srv.URL)
		require.NoError(t, err)
		assert.JSONEq(t, `[{"ChainSelector":"1"}]`, string(data))
	})

	t.Run("non-200 status includes body", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		defer srv.Close()

		_, err := fetchHTTP(context.Background(), srv.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("oversize response is rejected", func(t *testing.T) {
		t.Parallel()

		big := strings.Repeat("a", maxManifestBytes+1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(big))
		}))
		defer srv.Close()

		_, err := fetchHTTP(context.Background(), srv.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds max size")
	})

	t.Run("context cancellation surfaces as error", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		}))
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := fetchHTTP(ctx, srv.URL)
		require.Error(t, err)
	})

	t.Run("invalid URL fails to build request", func(t *testing.T) {
		t.Parallel()

		_, err := fetchHTTP(context.Background(), "http://[::1]:namedport/bad")
		require.Error(t, err)
	})
}

func Test_urlPathFromFileURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    string
		wantErr string
	}{
		{
			name: "simple absolute path",
			give: "file:///tmp/rpcs.json",
			want: filepath.FromSlash("/tmp/rpcs.json"),
		},
		{
			name: "localhost host is allowed",
			give: "file://localhost/tmp/rpcs.json",
			want: filepath.FromSlash("/tmp/rpcs.json"),
		},
		{
			name: "percent-encoded path is unescaped",
			give: "file:///tmp/rpc%20manifest.json",
			want: filepath.FromSlash("/tmp/rpc manifest.json"),
		},
		{
			name:    "non-empty non-localhost host is rejected",
			give:    "file://otherhost/tmp/rpcs.json",
			wantErr: "must not specify a host",
		},
		{
			name:    "empty path is rejected",
			give:    "file://",
			wantErr: "empty path",
		},
		{
			name:    "invalid URI fails to parse",
			give:    "file://%zz",
			wantErr: "parse file URI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := urlPathFromFileURI(tt.give)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_readSource(t *testing.T) {
	t.Parallel()

	t.Run("empty source is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := readSource(context.Background(), "   ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("file:// URI reads local file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "rpcs.json")
		require.NoError(t, os.WriteFile(path, []byte(`[]`), 0o600))

		data, err := readSource(context.Background(), "file://"+filepath.ToSlash(path))
		require.NoError(t, err)
		assert.JSONEq(t, `[]`, string(data))
	})

	t.Run("http source dispatches to fetchHTTP", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		}))
		defer srv.Close()

		data, err := readSource(context.Background(), srv.URL)
		require.NoError(t, err)
		assert.JSONEq(t, `[]`, string(data))
	})

	t.Run("bare local path is read directly", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "rpcs.json")
		require.NoError(t, os.WriteFile(path, []byte(`[]`), 0o600))

		data, err := readSource(context.Background(), path)
		require.NoError(t, err)
		assert.JSONEq(t, `[]`, string(data))
	})

	t.Run("relative local path is resolved against cwd", func(t *testing.T) {
		t.Parallel()

		// Verify expandPath's cwd-joining logic directly rather than via os.Chdir, which would
		// mutate global process state and race with sibling tests reading their own relative paths.
		cwd, err := os.Getwd()
		require.NoError(t, err)

		got, err := expandPath("rpcs.json")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(cwd, "rpcs.json"), got)
	})

	t.Run("missing local file surfaces an error", func(t *testing.T) {
		t.Parallel()

		_, err := readSource(context.Background(), filepath.Join(t.TempDir(), "missing.json"))
		require.Error(t, err)
	})

	t.Run("oversize local file is rejected", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "rpcs.json")
		require.NoError(t, os.WriteFile(path, []byte(strings.Repeat("a", maxManifestBytes+1)), 0o600))

		_, err := readSource(context.Background(), path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds max size")
	})
}

func Test_Load(t *testing.T) {
	t.Parallel()

	t.Run("malformed JSON is an error", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "rpcs.json")
		require.NoError(t, os.WriteFile(path, []byte(`not json`), 0o600))

		_, err := Load(context.Background(), path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal rpc manifest")
	})

	t.Run("invalid and duplicate selectors are skipped with a warning, valid ones kept", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "rpcs.json")
		require.NoError(t, os.WriteFile(path, []byte(`[
			{"ChainSelector": "not-a-number", "Names": {"Canonical": "bad-chain"}},
			{"ChainSelector": "", "Names": {"Canonical": "empty-chain"}},
			{"ChainSelector": "1", "Names": {"Canonical": "chain-one"}},
			{"ChainSelector": "1", "Names": {"Canonical": "chain-one-dup"}}
		]`), 0o600))

		manifest, err := Load(context.Background(), path, WithLogger(logger.Test(t)))
		require.NoError(t, err)

		entry, ok := manifest.Chain(1)
		require.True(t, ok)
		assert.Equal(t, "chain-one", entry.Names.Canonical)

		_, ok = manifest.Chain(2)
		assert.False(t, ok)
	})

	t.Run("works without a logger option", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "rpcs.json")
		require.NoError(t, os.WriteFile(path, []byte(`[{"ChainSelector": "-1"}]`), 0o600))

		manifest, err := Load(context.Background(), path)
		require.NoError(t, err)
		assert.Empty(t, manifest.chains)
	})

	t.Run("nil logger option does not panic", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "rpcs.json")
		require.NoError(t, os.WriteFile(path, []byte(`[{"ChainSelector": "not-a-number"}]`), 0o600))

		// WithLogger(nil) must not replace the default no-op logger, or the warning logged for
		// the invalid selector above would panic on a nil logger.Logger interface value.
		require.NotPanics(t, func() {
			_, err := Load(context.Background(), path, WithLogger(nil))
			require.NoError(t, err)
		})
	})
}
