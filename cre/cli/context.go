package cli

import (
	"errors"
	"strings"

	fcre "github.com/smartcontractkit/chainlink-deployments-framework/cre"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
)

const defaultEnvName = "PRODUCTION"

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
