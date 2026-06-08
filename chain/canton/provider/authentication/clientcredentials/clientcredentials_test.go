package clientcredentials

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
)

const testBearerToken = "test-access-token"

func tokenResponsePayload() map[string]any { //nolint:gosec // G101: OAuth test fixture, not a real credential
	return map[string]any{
		"access_token": testBearerToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
	}
}

func newTokenServer(t *testing.T, expectedScope string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		values, err := url.ParseQuery(string(body))
		assert.NoError(t, err)
		assert.Equal(t, "client_credentials", values.Get("grant_type"))

		if expectedScope != "" {
			assert.Equal(t, expectedScope, values.Get("scope"))
		}

		payload, err := json.Marshal(tokenResponsePayload())
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
}

func TestNewProvider_ValidatesInputs(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		name         string
		tokenURL     string
		clientID     string
		clientSecret string
	}{
		{name: "missing token url", tokenURL: "", clientID: "client-id", clientSecret: "client-secret"},                  //nolint:gosec // G101: test fixture
		{name: "missing client id", tokenURL: "https://example.test/token", clientID: "", clientSecret: "client-secret"}, //nolint:gosec // G101: test fixture
		{name: "missing client secret", tokenURL: "https://example.test/token", clientID: "client-id", clientSecret: ""}, //nolint:gosec // G101: test fixture
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewProvider(ctx, test.tokenURL, test.clientID, test.clientSecret)
			require.Error(t, err)
		})
	}
}

func TestNewProvider_UsesOptionsAndTokenSource(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	server := newTokenServer(t, "scope-a scope-b")
	t.Cleanup(server.Close)

	customCreds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec // G402: intentional for transport-credentials test

	provider, err := NewProvider(
		ctx,
		server.URL,
		"client-id",
		"client-secret", //nolint:gosec // G101: test fixture
		WithScopes("scope-a", "scope-b"),
		WithTransportCredentials(customCreds),
	)
	require.NoError(t, err)
	require.Same(t, customCreds, provider.TransportCredentials())

	token, err := provider.TokenSource().Token()
	require.NoError(t, err)
	require.Equal(t, testBearerToken, token.AccessToken)
}

func TestNewDiscoveryProvider_UsesMetadataTokenEndpoint(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	metadataPath := "/.well-known/oauth-authorization-server"
	tokenPath := "/token"

	mux.HandleFunc(metadataPath, func(w http.ResponseWriter, r *http.Request) {
		payload, err := json.Marshal(map[string]string{
			"issuer":         server.URL,
			"token_endpoint": server.URL + tokenPath,
		})
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	})
	mux.HandleFunc(tokenPath, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		values, err := url.ParseQuery(string(body))
		assert.NoError(t, err)
		assert.Equal(t, "client_credentials", values.Get("grant_type"))
		assert.Equal(t, "daml_ledger_api", values.Get("scope"))

		payload, err := json.Marshal(tokenResponsePayload())
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	})

	provider, err := NewDiscoveryProvider(ctx, server.URL, "client-id", "client-secret") //nolint:gosec // G101: test fixture
	require.NoError(t, err)

	token, err := provider.TokenSource().Token()
	require.NoError(t, err)
	require.Equal(t, testBearerToken, token.AccessToken)
}

func TestNewDiscoveryProvider_RequiresMetadataEndpoint(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/.well-known/oauth-authorization-server") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	_, err := NewDiscoveryProvider(ctx, server.URL, "client-id", "client-secret") //nolint:gosec // G101: test fixture
	require.Error(t, err)
}
