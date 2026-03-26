package artifacts

import (
	"net"
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

	hostGitHubCom           = "github.com"
	hostGitHubUserContent   = "githubusercontent.com"
	suffixGitHubCom         = "." + hostGitHubCom
	suffixGitHubUserContent = "." + hostGitHubUserContent
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

// gitHubBearerAllowedHost reports whether an outgoing request should include a GitHub PAT.
// Hosts under github.com (including api.github.com) and githubusercontent.com are allowed so tokens
// are not sent to arbitrary redirect targets or third-party URLs.
func gitHubBearerAllowedHost(hostPort string) bool {
	if hostPort == "" {
		return false
	}

	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
	}

	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	// check for hosts that do not match the subdomain suffixes (e.g. github.com vs *.github.com).
	return host == hostGitHubCom ||
		strings.HasSuffix(host, suffixGitHubCom) ||
		host == hostGitHubUserContent ||
		strings.HasSuffix(host, suffixGitHubUserContent)
}

// gitHubBearerTransport sets Bearer when token set and the request host is GitHub-owned; skips if
// Authorization already present.
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
	if !gitHubBearerAllowedHost(req.URL.Host) {
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
