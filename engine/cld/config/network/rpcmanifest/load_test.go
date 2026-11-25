package rpcmanifest

import (
	"context"
	"net/http"
	"net/http/httptest"
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

	t.Run("non-200 status does not echo the response body", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		defer srv.Close()

		_, err := fetchHTTP(context.Background(), srv.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
		assert.NotContains(t, err.Error(), "boom")
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

	t.Run("errors do not leak the URL", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.NotFoundHandler())
		unreachable := strings.Replace(srv.URL, "http://", "http://user:secret@", 1) + "?token=secret"
		srv.Close()

		for _, manifestURL := range []string{unreachable, "http://user:secret@[::1]:namedport/bad"} {
			_, err := fetchHTTP(context.Background(), manifestURL)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "secret")
		}
	})
}

// serveManifest serves body as the RPC manifest and returns its URL.
func serveManifest(t *testing.T, body string) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv.URL + "/files/rpcs.json"
}

func Test_Load(t *testing.T) {
	t.Parallel()

	t.Run("testdata manifest uses the server origin as base url", func(t *testing.T) {
		t.Parallel()

		manifest, origin := loadTestManifest(t)
		assert.Equal(t, origin, manifest.baseURL)
		assert.Len(t, manifest.chains, 2)

		entry, ok := manifest.chains[sepoliaSelector]
		require.True(t, ok)
		assert.Equal(t, "11155111", entry.ChainID)
		assert.Len(t, entry.Nodes, 2)
	})

	t.Run("base url drops credentials, path, query and fragment; scheme is case-insensitive", func(t *testing.T) {
		t.Parallel()

		manifestURL := strings.Replace(serveManifest(t, `[]`), "http://", "HTTP://user:secret@", 1) + "?token=x#frag"

		manifest, err := Load(context.Background(), " "+manifestURL+" ")
		require.NoError(t, err)
		assert.Regexp(t, `^http://127\.0\.0\.1:\d+$`, manifest.baseURL)
	})

	t.Run("non-http(s) urls are rejected", func(t *testing.T) {
		t.Parallel()

		for _, give := range []string{"", "   ", "ftp://example.com/rpcs.json", "rpcs.json", "https:///rpcs.json"} {
			_, err := Load(context.Background(), give)
			require.ErrorContains(t, err, "must be an http(s) url", "source %q", give)
		}
	})

	t.Run("unparseable url error does not leak the url", func(t *testing.T) {
		t.Parallel()

		_, err := Load(context.Background(), "http://user:secret@[::1]:namedport/bad")
		require.ErrorContains(t, err, "failed to parse rpc manifest url")
		assert.NotContains(t, err.Error(), "secret")
	})

	t.Run("malformed JSON is an error", func(t *testing.T) {
		t.Parallel()

		_, err := Load(context.Background(), serveManifest(t, `not json`))
		require.ErrorContains(t, err, "unmarshal rpc manifest")
	})

	t.Run("invalid and duplicate selectors are skipped with a warning, valid ones kept", func(t *testing.T) {
		t.Parallel()

		manifestURL := serveManifest(t, `[
			{"ChainSelector": "not-a-number", "Names": {"Canonical": "bad-chain"}},
			{"ChainSelector": "", "Names": {"Canonical": "empty-chain"}},
			{"ChainSelector": "1", "Names": {"Canonical": "chain-one"}},
			{"ChainSelector": "1", "Names": {"Canonical": "chain-one-dup"}}
		]`)

		manifest, err := Load(context.Background(), manifestURL, WithLogger(logger.Test(t)))
		require.NoError(t, err)

		require.Len(t, manifest.chains, 1)
		assert.Equal(t, "chain-one", manifest.chains[1].Names.Canonical)
	})

	t.Run("invalid selectors are skipped without a logger option", func(t *testing.T) {
		t.Parallel()

		manifest, err := Load(context.Background(), serveManifest(t, `[{"ChainSelector": "-1"}]`))
		require.NoError(t, err)
		assert.Empty(t, manifest.chains)
	})
}
