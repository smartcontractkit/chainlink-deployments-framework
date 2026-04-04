package main

import (
	"embed"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed templates
var templatesFS embed.FS

// ChainFamilyHandler abstracts all chain-specific generation logic.
// Each implementation owns its own contract config schema, type mappings,
// template data preparation, and method body generation.
//
// To add a new chain family:
//  1. Implement this interface in a new <family>.go file.
//  2. Add a template under templates/<family>/operations.tmpl.
//  3. Register the handler in chainFamilies below.
type ChainFamilyHandler interface {
	// Generate parses the raw YAML contract nodes and writes an operations
	// file for each contract using the provided template.
	// The node format is chain-family-specific; each handler decodes its own schema.
	Generate(config Config, tmpl *template.Template) error
}

// chainFamilies is the single registration point for all supported chain families.
var chainFamilies = map[string]ChainFamilyHandler{
	"evm": evmHandler{},
}

// Config holds the top-level generator configuration.
// Contracts is kept as raw YAML nodes so each handler can decode
// its own chain-specific contract schema.
type Config struct {
	Version     string       `yaml:"version"`
	ChainFamily string       `yaml:"chain_family"` // defaults to "evm"
	Input       InputConfig  `yaml:"input"`
	Output      OutputConfig `yaml:"output"`
	Contracts   yaml.Node    `yaml:"contracts"`
}

type InputConfig struct {
	BasePath string `yaml:"base_path"`
}

type OutputConfig struct {
	BasePath string `yaml:"base_path"`
}

func main() {
	configPath := flag.String("config", "operations_gen_config.yaml", "Path to config file")
	flag.Parse()

	configData, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}

	var config Config
	if err = yaml.Unmarshal(configData, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	chainFamily := config.ChainFamily
	if chainFamily == "" {
		chainFamily = "evm"
	}

	handler, ok := chainFamilies[chainFamily]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unsupported chain_family %q (supported: %s)\n",
			chainFamily, supportedFamilies())
		os.Exit(1)
	}

	tmpl, err := loadTemplate(chainFamily)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading template for chain family %q: %v\n", chainFamily, err)
		os.Exit(1)
	}

	configDir := filepath.Dir(*configPath)
	absConfigDir, err := filepath.Abs(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving config directory: %v\n", err)
		os.Exit(1)
	}

	config.Input.BasePath = filepath.Join(absConfigDir, config.Input.BasePath)
	config.Output.BasePath = filepath.Join(absConfigDir, config.Output.BasePath)

	if err := handler.Generate(config, tmpl); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating operations: %v\n", err)
		os.Exit(1)
	}
}

// loadTemplate loads the code generation template for the given chain family.
func loadTemplate(chainFamily string) (*template.Template, error) {
	path := fmt.Sprintf("templates/%s/operations.tmpl", chainFamily)
	content, err := templatesFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("template not found at %s: %w", path, err)
	}

	return template.New("operations").Parse(string(content))
}

// supportedFamilies returns a sorted, comma-separated list of supported chain families.
func supportedFamilies() string {
	families := make([]string, 0, len(chainFamilies))
	for k := range chainFamilies {
		families = append(families, k)
	}
	sort.Strings(families)

	return strings.Join(families, ", ")
}

// writeGoFile formats src as Go source and writes it to path, creating parent directories.
// Shared utility available to all chain-family handlers.
func writeGoFile(path string, src []byte) error {
	formatted, err := format.Source(src)
	if err != nil {
		return fmt.Errorf("formatting error: %w\n%s", err, src)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(path, formatted, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// versionToPath converts a semver string to a directory path segment.
// e.g. "1.2.3" → "v1_2_3"
func versionToPath(version string) string {
	return "v" + strings.ReplaceAll(version, ".", "_")
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}

	return strings.ToUpper(s[:1]) + s[1:]
}
