package authentication

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAuthorizationServerMetadata(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/.well-known/oauth-authorization-server", r.URL.Path)

		payload, err := json.Marshal(AuthorizationServerMetadata{
			Issuer:                        serverURL(r),
			AuthorizationEndpoint:         serverURL(r) + "/authorize",
			TokenEndpoint:                 serverURL(r) + "/token",
			CodeChallengeMethodsSupported: []string{"S256"},
		})
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)

	metadata, err := GetAuthorizationServerMetadata(ctx, server.URL)
	require.NoError(t, err)
	require.Equal(t, server.URL, metadata.Issuer)
	require.Equal(t, server.URL+"/authorize", metadata.AuthorizationEndpoint)
	require.Equal(t, server.URL+"/token", metadata.TokenEndpoint)
}

func TestGetAuthorizationServerMetadata_UnexpectedIssuer(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := json.Marshal(AuthorizationServerMetadata{
			Issuer: "https://other.example.com",
		})
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)

	_, err := GetAuthorizationServerMetadata(ctx, server.URL)
	require.Error(t, err)
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}
