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

	if err := ensureDownloadWorkDir(workDir); err != nil {
		return "", err
	}

	ref := src.ExternalRef
	expectedSHA, err := parseSHA256Hex(ref.SHA256)
	if err != nil {
		return "", err
	}

	if ref.IsURL() {
		plain := plainHTTPClient(httpClient)
		resp, err := httpGet(ctx, plain, ref.URL, "download binary")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		return writeBinaryAndVerifySHA256(resp.Body, expectedSHA, workDir)
	}

	if ref.IsGitHubRelease() {
		gh := githubHTTPClientOrDefault(httpClient)
		downloadURL, err := resolveGitHubAssetURL(ctx, gh, ref.Repo, ref.ReleaseTag, ref.AssetName)
		if err != nil {
			return "", err
		}

		var resp *http.Response
		if isGitHubAPIReleaseAssetURL(downloadURL) {
			resp, err = httpGetGitHubReleaseAsset(ctx, gh, downloadURL, "download binary")
		} else {
			resp, err = httpGet(ctx, gh, downloadURL, "download binary")
		}
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		return writeBinaryAndVerifySHA256(resp.Body, expectedSHA, workDir)
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
		e.URL, e.Repo, e.ReleaseTag, e.AssetName, e.SHA256,
	)
}

func resolveBinaryLocal(p string) (string, error) {
	clean, err := resolveLocalArtifactPath(p)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(clean), ".wasm") {
		return "", fmt.Errorf("cre: binary local path %q must have .wasm extension (resolved %q)", p, clean)
	}

	return clean, nil
}

// writeBinaryAndVerifySHA256 streams body to a temp file while computing SHA-256, then verifies
// the digest matches expected. The file is removed on any error (write failure or sha256 mismatch).
func writeBinaryAndVerifySHA256(body io.Reader, expected []byte, workDir string) (string, error) {
	tmpPath := filepath.Join(strings.TrimSpace(workDir), newWorkDirBinaryFileName())
	h := sha256.New()
	if err := writeToFile(tmpPath, io.TeeReader(body, h)); err != nil {
		return "", err
	}
	if subtle.ConstantTimeCompare(h.Sum(nil), expected) != 1 {
		remErr := os.Remove(tmpPath)

		return "", errors.Join(
			errors.New("cre: download binary: sha256 mismatch"),
			remErr,
		)
	}

	return tmpPath, nil
}

// parseSHA256Hex decodes a 64-character lowercase hex SHA-256 digest. An optional "0x" or "0X"
// prefix (after trimming) is accepted for convenience; typical shasum/openssl output has no prefix.
func parseSHA256Hex(s string) ([]byte, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "0x")
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
	tagEnc := url.PathEscape(tag)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, name, tagEnc)
	body, err := githubGet(ctx, client, apiURL, "github release")
	if err != nil {
		return "", fmt.Errorf("cre: github release %s/%s tag %q asset %q: %w", owner, name, tag, assetName, err)
	}
	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", fmt.Errorf("cre: github release decode %s/%s tag %q: %w", owner, name, tag, err)
	}
	for _, a := range rel.Assets {
		if a.Name != assetName {
			continue
		}
		if u := strings.TrimSpace(a.URL); u != "" {
			return u, nil
		}
		if u := strings.TrimSpace(a.BrowserDownloadURL); u != "" {
			return u, nil
		}
	}

	return "", fmt.Errorf("cre: github release %s/%s tag %q: asset %q not found in release assets", owner, name, tag, assetName)
}
