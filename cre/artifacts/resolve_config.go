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
	"path/filepath"
	"strings"
)

func resolveConfigHttp(ctx context.Context, src ConfigSource, httpClient *http.Client, workDir string) (string, error) {
	if err := src.Validate(); err != nil {
		return "", err
	}

	if src.IsLocal() {
		return resolveLocalArtifactPath(src.LocalPath)
	}

	if err := ensureDownloadWorkDir(workDir); err != nil {
		return "", err
	}

	ref := src.ExternalRef

	if ref.IsURL() {
		plain := plainHTTPClient(httpClient)
		resp, err := httpGet(ctx, plain, strings.TrimSpace(ref.URL), "download config")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		return writeConfigToWorkDir(resp.Body, workDir)
	}

	if ref.IsGitHubFile() {
		gh := githubHTTPClientOrDefault(httpClient)
		r, err := fetchGitHubFileContent(ctx, gh, ref.Repo, ref.Ref, ref.Path)
		if err != nil {
			return "", err
		}

		return writeConfigToWorkDir(r, workDir)
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
		normalizeGitHubConfigPath(e.Path),
	)
}

// githubFileAPIResponse matches GitHub Contents API for a file (not directory).
type githubFileAPIResponse struct {
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

// fetchGitHubFileContent fetches a file via the GitHub Contents API and returns the decoded content
// as a reader. workDir validation and file writing are the caller's responsibility.
func fetchGitHubFileContent(ctx context.Context, client *http.Client, repo, ref, path string) (io.Reader, error) {
	owner, name, err := parseGitHubRepo(repo)
	if err != nil {
		return nil, err
	}
	path = normalizeGitHubConfigPath(path)
	if path == "" {
		return nil, errors.New("cre: github config: path is required")
	}
	apiPath := encodeGitHubPath(path)
	refQ := url.QueryEscape(strings.TrimSpace(ref))
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, name, apiPath, refQ)
	body, err := githubGet(ctx, client, apiURL, "github config")
	if err != nil {
		return nil, fmt.Errorf("cre: github config %s/%s ref %q path %q: %w", owner, name, strings.TrimSpace(ref), path, err)
	}
	var file githubFileAPIResponse
	if unmarshalErr := json.Unmarshal(body, &file); unmarshalErr != nil {
		return nil, fmt.Errorf("cre: github config decode: %w", unmarshalErr)
	}
	if file.Type != "" && file.Type != "file" {
		return nil, fmt.Errorf("cre: github config: path is not a file (type=%s)", file.Type)
	}
	if !strings.EqualFold(strings.TrimSpace(file.Encoding), "base64") {
		return nil, fmt.Errorf("cre: github config: unexpected encoding %q", file.Encoding)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("cre: github config base64: %w", err)
	}

	return bytes.NewReader(raw), nil
}

func encodeGitHubPath(p string) string {
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}

	return strings.Join(parts, "/")
}

// writeConfigToWorkDir writes reader content to a new config file in workDir. The caller must
// ensure workDir exists (via ensureDownloadWorkDir).
func writeConfigToWorkDir(r io.Reader, workDir string) (string, error) {
	path := filepath.Join(strings.TrimSpace(workDir), newWorkDirConfigFileName())

	return path, writeToFile(path, r)
}
