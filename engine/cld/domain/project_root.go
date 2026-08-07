package domain

import (
	"fmt"
	"os"
	"path/filepath"
)

// ProjectRootEnvVar is the name of the environment variable that, when set, overrides the
// project root resolution. Its value must point to the root of a chainlink-deployments
// checkout (i.e. the directory that contains a "domains" subdirectory). This is primarily
// intended for domain CLI binaries that are executed outside of the chainlink-deployments
// repository (e.g. downloaded from S3), where the filesystem-based discovery cannot locate
// the repo layout.
const ProjectRootEnvVar = "CLD_PROJECT_ROOT"

// Defines root paths for the project and domains.
var (
	ProjectRoot = ResolveProjectRoot()
	DomainsRoot = filepath.Join(ProjectRoot, DomainsDirName)
)

// ResolveProjectRoot determines the project root path at runtime. It tries multiple
// strategies in order:
//  0. Use the CLD_PROJECT_ROOT environment variable if it is set.
//  1. Search upward from the executable location.
//  2. Search upward from the current working directory.
//
// If CLD_PROJECT_ROOT is set but does not point to a valid project root, it panics rather
// than silently falling back, so that an explicit misconfiguration fails fast.
func ResolveProjectRoot() string {
	// Strategy 0: Explicit override via environment variable.
	if v, ok := os.LookupEnv(ProjectRootEnvVar); ok && v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			panic(fmt.Sprintf("%s=%q could not be resolved to an absolute path: %v", ProjectRootEnvVar, v, err))
		}
		if !isValidProjectRoot(abs) {
			panic(fmt.Sprintf(
				"%s=%q is not a valid project root: no %q subdirectory found",
				ProjectRootEnvVar, abs, DomainsDirName,
			))
		}

		return abs
	}

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
	_, err := os.Stat(filepath.Join(dir, DomainsDirName))
	return err == nil
}
