package fileutils

import (
	"os"
	"path/filepath"
)

// WriteFileGitKeep writes a .gitkeep file to the path.
func WriteFileGitKeep(path string) error {
	file, err := os.Create(filepath.Join(path, ".gitkeep"))
	if err != nil {
		return err
	}

	defer file.Close()

	return nil
}

// MkdirAllGitKeep creates a directory with a .gitkeep file. This will create all parent
// directories if they do not already exist.
func MkdirAllGitKeep(path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	return WriteFileGitKeep(path)
}
