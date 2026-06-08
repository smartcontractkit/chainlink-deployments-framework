package authorizationcode

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type safeBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

var stdoutMu sync.Mutex

func (s *safeBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.b.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.b.String()
}

func captureStdout(t *testing.T) (*safeBuffer, func()) {
	t.Helper()

	stdoutMu.Lock()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	buffer := &safeBuffer{}
	done := make(chan struct{})

	os.Stdout = writer
	go func() {
		_, _ = io.Copy(buffer, reader)
		close(done)
	}()

	return buffer, func() {
		_ = writer.Close()
		os.Stdout = original
		<-done
		stdoutMu.Unlock()
	}
}

func freePort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx
	require.NoError(t, err)
	addr := listener.Addr().String()
	_ = listener.Close()

	return addr
}

func extractFirstURL(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		candidate := strings.TrimSpace(line)
		if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
			if parsed, err := url.Parse(candidate); err == nil && parsed.Scheme != "" && parsed.Host != "" {
				return candidate
			}
		}
	}

	return ""
}

func newTokenServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.NotEmpty(t, r.Form.Get("code"))
		assert.NotEmpty(t, r.Form.Get("code_verifier"))

		payload, err := json.Marshal(map[string]any{ //nolint:gosec // G101: OAuth test fixture, not a real credential
			"access_token": "auth-code-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
}

func TestNewProvider_ValidatesInputs(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		name     string
		authURL  string
		tokenURL string
		clientID string
	}{
		//nolint:gosec // G101: test fixture
		{name: "missing auth url", authURL: "", tokenURL: "https://example.test/token", clientID: "client-id"},
		//nolint:gosec // G101: test fixture
		{name: "missing token url", authURL: "https://example.test/auth", tokenURL: "", clientID: "client-id"},
		//nolint:gosec // G101: test fixture
		{name: "missing client id", authURL: "https://example.test/auth", tokenURL: "https://example.test/token", clientID: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewProvider(ctx, test.authURL, test.tokenURL, test.clientID, WithOpenBrowser(false))
			require.Error(t, err)
		})
	}
}

func TestNewDiscoveryProvider_RequiresS256(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := json.Marshal(map[string]any{ //nolint:gosec // G101: OAuth metadata test fixture
			"issuer":                           "http://" + r.Host,
			"authorization_endpoint":           "https://example.test/auth",
			"token_endpoint":                   "https://example.test/token",
			"code_challenge_methods_supported": []string{"plain"},
		})
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)

	_, err := NewDiscoveryProvider(ctx, server.URL, "client-id") //nolint:gosec // G101: test fixture
	require.Error(t, err)
}

func TestNewProvider_FlowCompletes(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	callbackHost := freePort(t)

	tokenServer := newTokenServer(t)
	t.Cleanup(tokenServer.Close)

	output, restore := captureStdout(t)
	defer restore()

	resultCh := make(chan struct {
		provider *Provider
		err      error
	}, 1)

	go func() {
		provider, err := NewProvider(
			ctx,
			tokenServer.URL+"/auth",
			tokenServer.URL+"/token",
			"client-id",
			WithCallbackURL("http://"+callbackHost+"/callback"),
			WithOpenBrowser(false),
			WithTimeout(5*time.Second),
		)
		resultCh <- struct {
			provider *Provider
			err      error
		}{provider: provider, err: err}
	}()

	var authCodeURL string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		authCodeURL = extractFirstURL(output.String())
		if authCodeURL != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.NotEmpty(t, authCodeURL)

	parsed, err := url.Parse(authCodeURL)
	require.NoError(t, err)
	state := parsed.Query().Get("state")
	require.NotEmpty(t, state)

	callbackURL := "http://" + callbackHost + "/callback?code=code123&state=" + url.QueryEscape(state)
	response, err := http.Get(callbackURL) //nolint:noctx,gosec // G107: test hits local callback server
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())

	result := <-resultCh
	require.NoError(t, result.err)
	require.NotNil(t, result.provider)

	token, err := result.provider.TokenSource().Token()
	require.NoError(t, err)
	require.Equal(t, "auth-code-token", token.AccessToken)
}
