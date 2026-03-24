package artifacts

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
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

type githubRelease struct {
	Assets []struct {
		URL                string `json:"url"`
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func resolveBinaryHttp(ctx context.Context, src BinarySource, httpClient *http.Client, workDir string) (string, error) {
	if err := src.Validate(); err != nil {
		return "", err
	}

	if src.IsLocal() {
		return resolveBinaryLocal(src.LocalPath)
	}

	if strings.TrimSpace(workDir) == "" {
		return "", errors.New("cre: WorkDir is required for external binary")
	}

	ref := src.ExternalRef
	if ref.IsURL() {
		plain := plainHTTPClient(httpClient)
		return downloadAndVerify(ctx, plain, ref.URL, ref.SHA256, workDir)
	}
	if ref.IsGitHubRelease() {
		gh := githubHTTPClientOrDefault(httpClient)
		downloadURL, err := resolveGitHubAssetURL(ctx, gh, ref.Repo, ref.ReleaseTag, ref.AssetName)
		if err != nil {
			return "", err
		}
		if isGitHubAPIReleaseAssetURL(downloadURL) {
			return downloadGitHubReleaseAssetAndVerify(ctx, gh, downloadURL, ref.SHA256, workDir)
		}

		return downloadAndVerify(ctx, gh, downloadURL, ref.SHA256, workDir)
	}

	return "", fmt.Errorf("cre: resolve binary: unsupported external binary ref (%s)", binaryExternalRefSummary(ref))
}

func isGitHubAPIReleaseAssetURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	if u.Host != "api.github.com" {
		return false
	}

	return strings.Contains(u.Path, "/releases/assets/")
}

func binaryExternalRefSummary(e *ExternalBinaryRef) string {
	if e == nil {
		return "externalRef=<nil>"
	}

	return fmt.Sprintf("url=%q repo=%q releaseTag=%q assetName=%q sha256=%q",
		strings.TrimSpace(e.URL),
		strings.TrimSpace(e.Repo),
		strings.TrimSpace(e.ReleaseTag),
		strings.TrimSpace(e.AssetName),
		strings.TrimSpace(e.SHA256),
	)
}

func resolveBinaryLocal(p string) (string, error) {
	clean, err := resolveLocalArtifactPath(p)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(clean), ".wasm") {
		if p == clean {
			return "", fmt.Errorf("cre: binary local path %q must have .wasm extension", p)
		}

		return "", fmt.Errorf("cre: binary local path %q must have .wasm extension (resolved %q)", p, clean)
	}

	return clean, nil
}

func downloadAndVerify(ctx context.Context, client *http.Client, downloadURL, expectedSHA256Hex, workDir string) (path string, err error) {
	resp, err := httpGet(ctx, client, downloadURL, "download binary")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return writeStreamAndVerifySHA256(resp.Body, expectedSHA256Hex, workDir)
}

// downloadGitHubReleaseAssetAndVerify downloads via the GitHub API asset URL (not browser_download_url)
// so Bearer auth stays on api.github.com and is not stripped on redirect to the object CDN.
func downloadGitHubReleaseAssetAndVerify(ctx context.Context, client *http.Client, apiAssetURL, expectedSHA256Hex, workDir string) (path string, err error) {
	resp, err := httpGetGitHubReleaseAsset(ctx, client, apiAssetURL, "download binary")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return writeStreamAndVerifySHA256(resp.Body, expectedSHA256Hex, workDir)
}

func writeStreamAndVerifySHA256(body io.Reader, expectedSHA256Hex, workDir string) (path string, err error) {
	wd := strings.TrimSpace(workDir)
	if wd == "" {
		return "", errors.New("cre: workDir is required for binary download")
	}
	expected, err := parseSHA256Hex(expectedSHA256Hex)
	if err != nil {
		return "", err
	}

	if mkErr := os.MkdirAll(wd, 0o700); mkErr != nil {
		return "", fmt.Errorf("cre: download binary work dir: %w", mkErr)
	}
	tmpPath := filepath.Join(wd, newWorkDirBinaryFileName())
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("cre: download binary temp file: %w", err)
	}

	defer func() {
		if err != nil {
			if remErr := os.Remove(tmpPath); remErr != nil {
				err = errors.Join(err, fmt.Errorf("cre: remove temp wasm: %w", remErr))
			}
		}
	}()

	h := sha256.New()
	if _, copyErr := io.Copy(io.MultiWriter(f, h), body); copyErr != nil {
		closeErr := f.Close()
		err = errors.Join(fmt.Errorf("cre: download binary write: %w", copyErr), closeErr)

		return "", err
	}

	if closeErr := f.Close(); closeErr != nil {
		err = fmt.Errorf("cre: download binary close: %w", closeErr)
		return "", err
	}
	sum := h.Sum(nil)
	if subtle.ConstantTimeCompare(sum, expected) != 1 {
		err = errors.New("cre: download binary: sha256 mismatch")
		return "", err
	}

	return tmpPath, nil
}

func parseSHA256Hex(s string) ([]byte, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return nil, errors.New("cre: sha256 is empty")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("cre: invalid sha256 hex: %w", err)
	}
	if len(b) != sha256.Size {
		return nil, fmt.Errorf("cre: sha256 must be %d bytes, got %d", sha256.Size, len(b))
	}

	return b, nil
}

func resolveGitHubAssetURL(ctx context.Context, client *http.Client, repo, tag, assetName string) (string, error) {
	owner, name, err := parseGitHubRepo(repo)
	if err != nil {
		return "", err
	}
	tagClean := strings.TrimSpace(tag)
	want := strings.TrimSpace(assetName)
	tagEnc := url.PathEscape(tagClean)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, name, tagEnc)
	body, err := githubGet(ctx, client, apiURL, "github release")
	if err != nil {
		return "", fmt.Errorf("cre: github release %s/%s tag %q asset %q: %w", owner, name, tagClean, want, err)
	}
	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", fmt.Errorf("cre: github release decode %s/%s tag %q: %w", owner, name, tagClean, err)
	}
	for _, a := range rel.Assets {
		if a.Name != want {
			continue
		}
		if u := strings.TrimSpace(a.URL); u != "" {
			return u, nil
		}
		if u := strings.TrimSpace(a.BrowserDownloadURL); u != "" {
			return u, nil
		}
	}

	return "", fmt.Errorf("cre: github release %s/%s tag %q: asset %q not found in release assets", owner, name, tagClean, want)
}
