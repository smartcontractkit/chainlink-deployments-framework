package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/families/evm"
)

//go:embed templates
var templatesFS embed.FS

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// chainFamilies is the single registration point for all supported chain families.
var chainFamilies = map[string]core.ChainFamilyHandler{
	"evm": evm.Handler{},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entrypoint. It uses a dedicated FlagSet so it can be
// called multiple times (e.g. in tests) without conflicting with the global
// flag.CommandLine that the testing harness parses before any test runs.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("operations-gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "operations_gen_config.yaml", "Path to config file")
	showVersion := fs.Bool("version", false, "Print version information and exit")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *showVersion {
		fmt.Fprintf(stdout, "operations-gen version=%s commit=%s date=%s\n", version, commit, date)
		return 0
	}

	configData, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading config: %v\n", err)
		return 1
	}

	var config core.Config
	if err = yaml.Unmarshal(configData, &config); err != nil {
		fmt.Fprintf(stderr, "Error parsing config: %v\n", err)
		return 1
	}

	chainFamily := config.ChainFamily
	if chainFamily == "" {
		chainFamily = "evm"
	}

	handler, ok := chainFamilies[chainFamily]
	if !ok {
		fmt.Fprintf(stderr, "Unsupported chain_family %q (supported: %s)\n",
			chainFamily, supportedFamilies())

		return 1
	}

	tmpl, err := loadTemplate(chainFamily)
	if err != nil {
		fmt.Fprintf(stderr, "Error loading template for chain family %q: %v\n", chainFamily, err)
		return 1
	}

	configDir := filepath.Dir(*configPath)
	absConfigDir, err := filepath.Abs(configDir)
	if err != nil {
		fmt.Fprintf(stderr, "Error resolving config directory: %v\n", err)
		return 1
	}

	config.ConfigDir = absConfigDir

	if err := handler.Generate(config, tmpl); err != nil {
		fmt.Fprintf(stderr, "Error generating operations: %v\n", err)
		return 1
	}

	return 0
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
