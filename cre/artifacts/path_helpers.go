package artifacts

import (
	"errors"
	"fmt"
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
