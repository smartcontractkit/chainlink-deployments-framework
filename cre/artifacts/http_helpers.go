package artifacts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// plainHTTPClient returns c or [http.DefaultClient] if c is nil. Use for arbitrary URLs (not GitHub-specific refs).
func plainHTTPClient(c *http.Client) *http.Client {
	if c == nil {
		return http.DefaultClient
	}
	return c
}

// httpGet performs GET and returns the response only when status is 200. The caller must close resp.Body.
// op labels errors (e.g. "download config", "download binary").
func httpGet(ctx context.Context, client *http.Client, rawURL string, op string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cre: %s: %w", op, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cre: %s: %w", op, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("cre: %s: unexpected status %d: %s", op, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp, nil
}

// httpGetGitHubReleaseAsset GETs a release asset via the GitHub API asset URL
// (…/repos/{owner}/{repo}/releases/assets/{id}). GitHub requires Accept: application/octet-stream;
// using browser_download_url instead redirects to another host and drops Bearer auth on redirect.
func httpGetGitHubReleaseAsset(ctx context.Context, client *http.Client, apiAssetURL string, op string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(apiAssetURL), nil)
	if err != nil {
		return nil, fmt.Errorf("cre: %s: %w", op, err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cre: %s: %w", op, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("cre: %s: %s: unexpected status %d: %s", op, strings.TrimSpace(apiAssetURL), resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp, nil
}

// githubGet calls the GitHub REST API (Accept: application/vnd.github+json) and returns the body on 200.
func githubGet(ctx context.Context, client *http.Client, apiURL string, op string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cre: %s: %w", op, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cre: %s: %w", op, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cre: %s read body: %w", op, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cre: %s: %s: status %d: %s", op, apiURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// resolveLocalArtifactPath returns a cleaned path to an existing non-directory file, or an error.
func resolveLocalArtifactPath(path string) (string, error) {
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("cre: local file does not exist: %w", err)
		}
		return "", fmt.Errorf("cre: local file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("cre: local path is a directory: %s", clean)
	}
	return clean, nil
}
