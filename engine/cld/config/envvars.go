package config

import "os"

// isCI returns true if we are running in CI. This env var is set by Github Actions.
func isCI() bool {
	return os.Getenv("CI") == "true"
}
