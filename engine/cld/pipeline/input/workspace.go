package input

import (
	"errors"
	"os"
	"path/filepath"
)

// FindWorkspaceRoot finds the root of the workspace by looking for the domains directory.
func FindWorkspaceRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "domains")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errors.New("could not find workspace root (directory with domains/)")
}
