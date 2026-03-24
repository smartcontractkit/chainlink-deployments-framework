package artifacts

import (
	"net/http"
	"os"
	"strings"
)

// GitHub PAT env var names (not part of CRE deploy / cfg.CRE).
//
//nolint:gosec // G101: env var name strings, not credentials
const (
	envGitHubToken = "GITHUB_TOKEN"
	envGHToken     = "GH_TOKEN"
)

// githubTokenFromEnv returns the first non-empty token from GITHUB_TOKEN or GH_TOKEN.
func githubTokenFromEnv() string {
	for _, k := range []string{envGitHubToken, envGHToken} {
		if t := strings.TrimSpace(os.Getenv(k)); t != "" {
			return t
		}
	}

	return ""
}

// gitHubBearerTransport sets Bearer when token set; skips if Authorization already present.
type gitHubBearerTransport struct {
	base  http.RoundTripper
	token string
}

func (t *gitHubBearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if t.token == "" || req.Header.Get("Authorization") != "" {
		return base.RoundTrip(req)
	}
	r2 := req.Clone(req.Context())
	r2.Header.Set("Authorization", "Bearer "+t.token)

	return base.RoundTrip(r2)
}

// githubHTTPClientOrDefault wraps c with Bearer from env when GITHUB_TOKEN/GH_TOKEN set; else returns c (nil → DefaultClient).
func githubHTTPClientOrDefault(c *http.Client) *http.Client {
	if c == nil {
		c = http.DefaultClient
	}
	token := githubTokenFromEnv()
	if token == "" {
		return c
	}
	base := c.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	out := *c
	out.Transport = &gitHubBearerTransport{base: base, token: token}

	return &out
}
