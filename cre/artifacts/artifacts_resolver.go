package artifacts

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// ArtifactsResolver resolves workflow WASM and config paths from [BinarySource] and [ConfigSource]
// via local files or remote fetch.
type ArtifactsResolver struct {
	client  *http.Client
	workDir string
}

// ArtifactsResolverOption configures [NewArtifactsResolver].
type ArtifactsResolverOption func(*ArtifactsResolver)

// WithHTTPClient sets the HTTP client for remote artifact fetches. When omitted, http.DefaultClient is used
func WithHTTPClient(c *http.Client) ArtifactsResolverOption {
	return func(r *ArtifactsResolver) {
		r.client = c
	}
}

// NewArtifactsResolver returns a resolver for workDir (non-empty after trim). GitHub: GITHUB_TOKEN/GH_TOKEN (github_http.go).
func NewArtifactsResolver(workDir string, opts ...ArtifactsResolverOption) (*ArtifactsResolver, error) {
	wd := strings.TrimSpace(workDir)
	if wd == "" {
		return nil, errors.New("cre: WorkDir is required")
	}
	r := &ArtifactsResolver{workDir: wd}
	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// WorkDir returns the directory used for downloads and temp artifacts. Callers may pass it to os.RemoveAll after use.
func (r *ArtifactsResolver) WorkDir() string {
	if r == nil {
		return ""
	}

	return r.workDir
}

// ResolveBinary resolves a .wasm path. Cleanup: defer os.RemoveAll(r.WorkDir()) (downloads only).
func (r *ArtifactsResolver) ResolveBinary(ctx context.Context, src BinarySource) (string, error) {
	if r == nil {
		return "", errors.New("cre: resolver is nil")
	}
	wd := strings.TrimSpace(r.workDir)
	if wd == "" {
		return "", errors.New("cre: WorkDir is required")
	}

	return resolveBinaryHttp(ctx, src, r.httpClient(), wd)
}

// ResolveConfig resolves a config file path. Cleanup same as [ArtifactsResolver.ResolveBinary].
func (r *ArtifactsResolver) ResolveConfig(ctx context.Context, src ConfigSource) (string, error) {
	if r == nil {
		return "", errors.New("cre: resolver is nil")
	}
	wd := strings.TrimSpace(r.workDir)
	if wd == "" {
		return "", errors.New("cre: WorkDir is required")
	}

	return resolveConfigHttp(ctx, src, r.httpClient(), wd)
}

func (r *ArtifactsResolver) httpClient() *http.Client {
	if r == nil {
		return nil
	}

	return r.client
}
