package artifacts

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// resolveLocalArtifactPath returns a cleaned path to an existing non-directory file, or an error.
func resolveLocalArtifactPath(path string) (string, error) {
	s := strings.TrimSpace(path)
	if s == "" {
		return "", errors.New("cre: local path is empty")
	}

	clean := filepath.Clean(s)
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

// ensureDownloadWorkDir validates workDir and creates it when missing.
func ensureDownloadWorkDir(workDir string) error {
	wd := strings.TrimSpace(workDir)
	if wd == "" {
		return errors.New("cre: WorkDir is required for download")
	}
	if err := os.MkdirAll(wd, 0o700); err != nil {
		return fmt.Errorf("cre: download WorkDir: %w", err)
	}

	return nil
}

// maxDownloadSize is the upper bound for any single artifact download (3 GiB).
const maxDownloadSize int64 = 3 << 30

// writeToFile writes r to path with mode 0o600 and a hard cap of maxSize bytes.
// The file is removed on any failure (write error, close error, or size exceeded).
func writeToFile(path string, r io.Reader) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("cre: create file: %w", err)
	}

	n, copyErr := io.Copy(f, io.LimitReader(r, maxDownloadSize+1))
	if copyErr != nil {
		closeErr := f.Close()
		remErr := os.Remove(path)

		return errors.Join(
			fmt.Errorf("cre: write file: %w", copyErr),
			closeErr,
			remErr,
		)
	}
	if n > maxDownloadSize {
		closeErr := f.Close()
		remErr := os.Remove(path)

		return errors.Join(
			fmt.Errorf("cre: download exceeds maximum size (%d bytes)", maxDownloadSize),
			closeErr,
			remErr,
		)
	}
	if closeErr := f.Close(); closeErr != nil {
		remErr := os.Remove(path)

		return errors.Join(
			fmt.Errorf("cre: close file: %w", closeErr),
			remErr,
		)
	}

	return nil
}
