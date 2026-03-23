package artifacts

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_gitHubBearerTransport(t *testing.T) {
	t.Run("adds bearer to every request when token set", func(t *testing.T) {
		t.Setenv(envGitHubToken, "secret-token")

		var inner *http.Request
		mockRT := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			inner = req
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		})

		rt := &gitHubBearerTransport{base: mockRT, token: githubTokenFromEnv()}
		req, err := http.NewRequest(http.MethodGet, "https://example.com/x", nil)
		require.NoError(t, err)
		_, err = rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, inner)
		require.Equal(t, "Bearer secret-token", inner.Header.Get("Authorization"))
		require.Equal(t, "", req.Header.Get("Authorization"))
	})

	t.Run("does not override existing authorization", func(t *testing.T) {
		t.Setenv(envGitHubToken, "secret-token")

		var inner *http.Request
		mockRT := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			inner = req
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		})

		rt := &gitHubBearerTransport{base: mockRT, token: githubTokenFromEnv()}
		req, err := http.NewRequest(http.MethodGet, "https://api.github.com/foo", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer existing")
		_, err = rt.RoundTrip(req)
		require.NoError(t, err)
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
