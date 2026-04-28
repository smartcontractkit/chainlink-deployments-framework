package core

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// ChainFamilyHandler abstracts all chain-specific generation logic.
type ChainFamilyHandler interface {
	Generate(config Config, tmpl *template.Template) error
}

// Config holds the top-level generator configuration.
// Input/Output/Contracts are raw YAML nodes so handlers own their own schemas.
type Config struct {
	Version     string    `yaml:"version"`
	ChainFamily string    `yaml:"chain_family"` // defaults to "evm"
	Input       yaml.Node `yaml:"input"`
	Output      yaml.Node `yaml:"output"`
	Contracts   yaml.Node `yaml:"contracts"`
	ConfigDir   string    `yaml:"-"`
}

// WriteGoFile formats src as Go source and writes it to path, creating parent directories.
func WriteGoFile(path string, src []byte) error {
	formatted, err := format.Source(src)
	if err != nil {
		return fmt.Errorf("formatting error: %w\n%s", err, src)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(path, formatted, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// VersionToPath converts a semver string to a directory path segment.
func VersionToPath(version string) string {
	return "v" + strings.ReplaceAll(version, ".", "_")
}

// ContractOutputPath builds the output file path for a generated contract operations file.
func ContractOutputPath(basePath, versionPath, packageName string) string {
	return filepath.Join(basePath, versionPath, "operations", packageName, packageName+".go")
}

// Capitalize uppercases the first character of s.
func Capitalize(s string) string {
	if s == "" {
		return ""
	}

	return strings.ToUpper(s[:1]) + s[1:]
}
