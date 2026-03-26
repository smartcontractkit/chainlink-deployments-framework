package artifacts

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_gitHubBearerAllowedHost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		host string
		want bool
	}{
		{"", false},
		{"example.com", false},
		{"127.0.0.1", false},
		{"localhost", false},
		{"api.github.com", true},
		{"API.GITHUB.COM", true},
		{"github.com", true},
		{"gist.github.com", true},
		{"uploads.github.com", true},
		{"objects.githubusercontent.com", true},
		{"raw.githubusercontent.com", true},
		{"githubusercontent.com", true},
		{"evilgithub.com", false},
		{"notgithub.com", false},
	}
	for _, tt := range tests {
		name := tt.host
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, gitHubBearerAllowedHost(tt.host), "host=%q", tt.host)
		})
	}
}

func Test_gitHubBearerAllowedHost_withPort(t *testing.T) {
	t.Parallel()
	require.True(t, gitHubBearerAllowedHost("api.github.com:443"))
	require.True(t, gitHubBearerAllowedHost("github.com:443"))
	require.False(t, gitHubBearerAllowedHost("example.com:443"))
}

func Test_gitHubBearerTransport(t *testing.T) {
	t.Run("does_not_add_bearer_to_non_github_host", func(t *testing.T) {
		t.Setenv(envGitHubToken, "secret-token")

		var inner *http.Request
		mockRT := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			inner = req
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		})

		rt := &gitHubBearerTransport{base: mockRT, token: githubTokenFromEnv()}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/x", nil)
		require.NoError(t, err)
		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Cleanup(func() { _ = resp.Body.Close() })
		require.NotNil(t, inner)
		require.Empty(t, inner.Header.Get("Authorization"))
		require.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("adds_bearer_for_api_github_com", func(t *testing.T) {
		t.Setenv(envGitHubToken, "secret-token")

		var inner *http.Request
		mockRT := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			inner = req
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		})

		rt := &gitHubBearerTransport{base: mockRT, token: githubTokenFromEnv()}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.github.com/foo", nil)
		require.NoError(t, err)
		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Cleanup(func() { _ = resp.Body.Close() })
		require.NotNil(t, inner)
		require.Equal(t, "Bearer secret-token", inner.Header.Get("Authorization"))
		require.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("adds_bearer_for_github_com_release_download", func(t *testing.T) {
		t.Setenv(envGitHubToken, "secret-token")

		var inner *http.Request
		mockRT := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			inner = req
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		})

		rt := &gitHubBearerTransport{base: mockRT, token: githubTokenFromEnv()}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://github.com/org/repo/releases/download/v1/a.wasm", nil)
		require.NoError(t, err)
		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Cleanup(func() { _ = resp.Body.Close() })
		require.Equal(t, "Bearer secret-token", inner.Header.Get("Authorization"))
	})

	t.Run("does_not_override_existing_authorization", func(t *testing.T) {
		t.Setenv(envGitHubToken, "secret-token")

		var inner *http.Request
		mockRT := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			inner = req
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		})

		rt := &gitHubBearerTransport{base: mockRT, token: githubTokenFromEnv()}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.github.com/foo", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer existing")
		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Cleanup(func() { _ = resp.Body.Close() })
		require.Equal(t, "Bearer existing", inner.Header.Get("Authorization"))
	})
}

func Test_githubHTTPClientOrDefault(t *testing.T) {
	t.Run("no env returns same client pointer when token absent", func(t *testing.T) {
		t.Setenv(envGitHubToken, "")
		t.Setenv(envGHToken, "")

		custom := &http.Client{}
		got := githubHTTPClientOrDefault(custom)
		require.Equal(t, custom, got)
	})

	t.Run("wraps when GITHUB_TOKEN set", func(t *testing.T) {
		t.Setenv(envGitHubToken, "tok")

		base := &http.Client{}
		got := githubHTTPClientOrDefault(base)
		require.NotSame(t, base, got)
		require.NotNil(t, got.Transport)
	})
}

func Test_githubTokenFromEnv_precedence(t *testing.T) {
	t.Setenv(envGitHubToken, "a")
	t.Setenv(envGHToken, "b")
	require.Equal(t, "a", githubTokenFromEnv())

	t.Setenv(envGitHubToken, "")
	require.Equal(t, "b", githubTokenFromEnv())
}
