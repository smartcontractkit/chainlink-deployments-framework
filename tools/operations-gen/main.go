package main

import (
	"embed"
	"flag"
	"fmt"
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

// chainFamilies is the single registration point for all supported chain families.
var chainFamilies = map[string]core.ChainFamilyHandler{
	"evm": evm.Handler{},
}

func main() {
	configPath := flag.String("config", "operations_gen_config.yaml", "Path to config file")
	flag.Parse()

	configData, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}

	var config core.Config
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

	config.ConfigDir = absConfigDir

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
