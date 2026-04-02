package cli

import (
	"errors"
	"strings"

	fcre "github.com/smartcontractkit/chainlink-deployments-framework/cre"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
)

const defaultEnvName = "PRODUCTION"

// ContextOverrides holds optional user-level overrides for the generated context.yaml.
// When fields are empty, values fall back to CRE_* process environment variables.
type ContextOverrides struct {
	TenantID   string                      `json:"tenantId,omitempty" yaml:"tenantId,omitempty"`
	GatewayURL string                      `json:"gatewayUrl,omitempty" yaml:"gatewayUrl,omitempty"`
	Registries []fcre.ContextRegistryEntry `json:"registries,omitempty" yaml:"registries,omitempty"`
}

// ContextEnvironment is one environment block (e.g. PRODUCTION) in context.yaml.
type ContextEnvironment struct {
	TenantID   string                      `json:"tenantId" yaml:"tenant_id"`
	DonFamily  string                      `json:"donFamily" yaml:"don_family"`
	GatewayURL string                      `json:"gatewayUrl" yaml:"gateway_url"`
	Registries []fcre.ContextRegistryEntry `json:"registries,omitempty" yaml:"registries,omitempty"`
}

// ContextConfig is the full context.yaml document (environment name → config).
type ContextConfig map[string]ContextEnvironment

// BuildContextConfig produces the context.yaml structure from domain defaults, input overrides, and CRE config.
// ContextOverrides take precedence over the domain.yaml configs.
func BuildContextConfig(
	donFamily string,
	contextOverrides ContextOverrides,
	cfg cfgenv.CREConfig,
	domainRegistries []fcre.ContextRegistryEntry,
) (ContextConfig, error) {
	envName := strings.TrimSpace(cfg.CLIEnv)
	if envName == "" {
		envName = defaultEnvName
	}
	tenant := strings.TrimSpace(contextOverrides.TenantID)
	if tenant == "" {
		tenant = cfg.Auth.TenantID
	}
	gateway := strings.TrimSpace(contextOverrides.GatewayURL)
	if gateway == "" {
		gateway = cfg.GatewayURL
	}
	registries := append([]fcre.ContextRegistryEntry{}, domainRegistries...)
	if len(contextOverrides.Registries) > 0 {
		registries = append([]fcre.ContextRegistryEntry{}, contextOverrides.Registries...)
	}
	if len(registries) == 0 {
		return nil, errors.New("CRE context registries: empty after merge (set domain cre_context_defaults.default_registries or input.context.registries)")
	}

	return ContextConfig{
		envName: {
			TenantID:   tenant,
			DonFamily:  donFamily,
			GatewayURL: gateway,
			Registries: registries,
		},
	}, nil
}

// IsOnChainRegistry reports whether the registry matching deploymentRegistryID
// has Type "on-chain" in the given list.
func IsOnChainRegistry(deploymentRegistryID string, registries []fcre.ContextRegistryEntry) bool {
	for _, r := range registries {
		if r.ID == deploymentRegistryID {
			return strings.EqualFold(r.Type, "on-chain")
		}
	}

	return false
}

// FlatRegistries collects all registry entries from a ContextConfig (across all environments).
func FlatRegistries(cfg ContextConfig) []fcre.ContextRegistryEntry {
	total := 0
	for _, env := range cfg {
		total += len(env.Registries)
	}

	out := make([]fcre.ContextRegistryEntry, 0, total)
	for _, env := range cfg {
		out = append(out, env.Registries...)
	}

	return out
}

// WriteContextYAML writes context.yaml to dir and returns the file path.
func WriteContextYAML(dir string, cfg ContextConfig) (string, error) {
	return writeYAMLFile(dir, "context.yaml", cfg)
}
