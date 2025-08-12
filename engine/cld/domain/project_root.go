package domain

import (
	"os"
	"path/filepath"
)

// Defines root paths for the project and domains.
var (
	ProjectRoot = getProjectRoot()
	DomainsRoot = filepath.Join(ProjectRoot, "domains")
)

// getProjectRoot dynamically determines the project root path at runtime.
// It tries multiple strategies in order:
// 1. Search upward from executable location
// 2. Search upward from current working directory
func getProjectRoot() string {
	// Strategy 1: Search upward from executable location
	if execPath, err := os.Executable(); err == nil {
		if root := searchUpwardForProjectRoot(filepath.Dir(execPath)); root != "" {
			return root
		}
	}

	// Strategy 2: Search upward from current working directory
	cwd, err := os.Getwd()
	if err == nil {
		if root := searchUpwardForProjectRoot(cwd); root != "" {
			return root
		}
	}

	// Last resort - return cwd if search fails
	return cwd
}

// searchUpwardForProjectRoot searches upward from the given starting directory
// looking for a directory that contains a "domains" subdirectory.
func searchUpwardForProjectRoot(startDir string) string {
	current := startDir
	for range 10 { // limit search to avoid infinite loops
		if isValidProjectRoot(current) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current { // reached filesystem root
			break
		}
		current = parent
	}

	return ""
}

// isValidProjectRoot checks if the given directory is a valid project root
// by verifying it contains a "domains" subdirectory.
func isValidProjectRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "domains"))
	return err == nil
}
