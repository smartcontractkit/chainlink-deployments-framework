package artifacts

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
)

// ArtifactsResolver resolves workflow WASM and config paths from [BinarySource] and [ConfigSource]
// via local files or remote fetch.
type ArtifactsResolver struct {
	HTTPClient *http.Client
	// WorkDir is required: downloads use unique workflow-<hex>.wasm / workflow-config-<hex>.json names; caller owns the tree (e.g. defer os.RemoveAll).
	WorkDir string
}

// NewArtifactsResolver returns a resolver for workDir (non-empty after trim). GitHub: GITHUB_TOKEN/GH_TOKEN (github_http.go).
func NewArtifactsResolver(workDir string) (*ArtifactsResolver, error) {
	wd := strings.TrimSpace(workDir)
	if wd == "" {
		return nil, errors.New("cre: WorkDir is required")
	}
	return &ArtifactsResolver{WorkDir: wd}, nil
}

// HTTPClientFromCREConfig returns a client with Timeout from cre.Timeout, or nil if unset/invalid.
func HTTPClientFromCREConfig(cre cfgenv.CREConfig) *http.Client {
	s := strings.TrimSpace(cre.Timeout)
	if s == "" {
		return nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return nil
	}
	return &http.Client{Timeout: d}
}

// ResolveBinary resolves a .wasm path. Cleanup: defer os.RemoveAll(resolver.WorkDir) (downloads only).
func (r *ArtifactsResolver) ResolveBinary(ctx context.Context, src BinarySource) (string, error) {
	if r == nil {
		return "", errors.New("cre: resolver is nil")
	}
	wd := strings.TrimSpace(r.WorkDir)
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
	wd := strings.TrimSpace(r.WorkDir)
	if wd == "" {
		return "", errors.New("cre: WorkDir is required")
	}
	return resolveConfigHttp(ctx, src, r.httpClient(), wd)
}

func (r *ArtifactsResolver) httpClient() *http.Client {
	if r == nil {
		return nil
	}
	return r.HTTPClient
}
