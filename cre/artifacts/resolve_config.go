package artifacts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func resolveConfigHttp(ctx context.Context, src ConfigSource, httpClient *http.Client, workDir string) (string, error) {
	if err := src.Validate(); err != nil {
		return "", err
	}

	if src.IsLocal() {
		return resolveConfigLocal(src.LocalPath)
	}

	if strings.TrimSpace(workDir) == "" {
		return "", errors.New("cre: WorkDir is required for external config")
	}

	ref := src.ExternalRef

	if ref.IsURL() {
		plain := plainHTTPClient(httpClient)
		return downloadConfigURL(ctx, plain, ref.URL, workDir)
	}

	if ref.IsGitHubFile() {
		gh := githubHTTPClientOrDefault(httpClient)
		return fetchGitHubFileContent(ctx, gh, ref.Repo, ref.Ref, ref.Path, workDir)
	}

	return "", fmt.Errorf("cre: resolve config: unsupported external config ref (%s)", configExternalRefSummary(ref))
}

func configExternalRefSummary(e *ExternalConfigRef) string {
	if e == nil {
		return "externalRef=<nil>"
	}

	return fmt.Sprintf("url=%q repo=%q ref=%q path=%q",
		strings.TrimSpace(e.URL),
		strings.TrimSpace(e.Repo),
		strings.TrimSpace(e.Ref),
		strings.TrimSpace(e.Path),
	)
}

func resolveConfigLocal(p string) (string, error) {
	return resolveLocalArtifactPath(p)
}

func downloadConfigURL(ctx context.Context, client *http.Client, rawURL, workDir string) (string, error) {
	resp, err := httpGet(ctx, client, strings.TrimSpace(rawURL), "download config")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return writeConfigFile(workDir, resp.Body)
}

// githubFileAPIResponse matches GitHub Contents API for a file (not directory).
type githubFileAPIResponse struct {
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

func fetchGitHubFileContent(ctx context.Context, client *http.Client, repo, ref, path, workDir string) (string, error) {
	owner, name, err := parseGitHubRepo(repo)
	if err != nil {
		return "", err
	}
	path = strings.TrimPrefix(strings.TrimSpace(path), "/")
	apiPath := encodeGitHubPath(path)
	refQ := url.QueryEscape(strings.TrimSpace(ref))
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, name, apiPath, refQ)
	body, err := githubGet(ctx, client, apiURL, "github config")
	if err != nil {
		return "", fmt.Errorf("cre: github config %s/%s ref %q path %q: %w", owner, name, strings.TrimSpace(ref), path, err)
	}
	var file githubFileAPIResponse
	if err := json.Unmarshal(body, &file); err != nil {
		return "", fmt.Errorf("cre: github config decode: %w", err)
	}
	if file.Type != "" && file.Type != "file" {
		return "", fmt.Errorf("cre: github config: path is not a file (type=%s)", file.Type)
	}
	if !strings.EqualFold(strings.TrimSpace(file.Encoding), "base64") {
		return "", fmt.Errorf("cre: github config: unexpected encoding %q", file.Encoding)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
	if err != nil {
		return "", fmt.Errorf("cre: github config base64: %w", err)
	}

	return writeConfigFile(workDir, bytes.NewReader(raw))
}

func encodeGitHubPath(p string) string {
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}

	return strings.Join(parts, "/")
}

func writeConfigFile(workDir string, r io.Reader) (string, error) {
	wd := strings.TrimSpace(workDir)
	if wd == "" {
		return "", errors.New("cre: workDir is required for config download")
	}
	if err := os.MkdirAll(wd, 0o700); err != nil {
		return "", fmt.Errorf("cre: config work dir: %w", err)
	}
	path := filepath.Join(wd, newWorkDirConfigFileName())
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("cre: config file: %w", err)
	}
	if _, copyErr := io.Copy(f, r); copyErr != nil {
		closeErr := f.Close()
		remErr := os.Remove(path)

		return "", errors.Join(
			fmt.Errorf("cre: config write: %w", copyErr),
			closeErr,
			remErr,
		)
	}
	if closeErr := f.Close(); closeErr != nil {
		remErr := os.Remove(path)
		return "", errors.Join(
			fmt.Errorf("cre: config close: %w", closeErr),
			remErr,
		)
	}

	return path, nil
}
