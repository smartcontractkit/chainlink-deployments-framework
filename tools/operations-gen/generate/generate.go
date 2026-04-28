package generate

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
)

//go:embed templates
var templatesFS embed.FS

// Config holds the top-level generator configuration.
// Input/Output/Contracts are raw YAML nodes so chain-family handlers own their
// own schemas.
type Config = core.Config

// GenerateFile reads an operations-gen YAML config file and generates operations.
//
// Relative output paths and package loading are resolved from the config file's
// directory, which makes this safe to call from another repository without
// depending on the process working directory.
func GenerateFile(configPath string) error {
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	configDir := filepath.Dir(configPath)
	absConfigDir, err := filepath.Abs(configDir)
	if err != nil {
		return fmt.Errorf("resolve config directory: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	config.ConfigDir = absConfigDir

	return Generate(config)
}

// Generate generates operations from a decoded config.
func Generate(config Config) error {
	chainFamily := config.ChainFamily
	if chainFamily == "" {
		chainFamily = "evm"
	}

	handler, ok := chainFamilies[chainFamily]
	if !ok {
		return fmt.Errorf("unsupported chain_family %q (supported: %s)", chainFamily, supportedFamilies())
	}

	tmpl, err := LoadTemplate(chainFamily)
	if err != nil {
		return fmt.Errorf("load template for chain family %q: %w", chainFamily, err)
	}

	if err := handler.Generate(config, tmpl); err != nil {
		return fmt.Errorf("generate operations: %w", err)
	}

	return nil
}

// LoadTemplate loads the code generation template for the given chain family.
func LoadTemplate(chainFamily string) (*template.Template, error) {
	path := fmt.Sprintf("templates/%s/operations.tmpl", chainFamily)
	content, err := templatesFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("template not found at %s: %w", path, err)
	}

	return template.New("operations").Parse(string(content))
}
