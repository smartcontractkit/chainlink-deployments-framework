package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
)

// WriteCREEnvFile writes a .env file with CRE config values for the CRE CLI.
func WriteCREEnvFile(workDir, contextYAMLPath string, cfg cfgenv.CREConfig, donFamily string) (string, error) {
	envPath := filepath.Join(workDir, ".env")

	var pairs []kv

	addVar("CRE_TENANT_ID", cfg.Auth.TenantID, &pairs)
	addVar("CRE_ORG_ID", cfg.Auth.OrgID, &pairs)
	addVar("CRE_STORAGE_ADDR", cfg.StorageAddress, &pairs)
	addVar("CRE_TLS", cfg.TLS, &pairs)
	addVar("CRE_TIMEOUT", cfg.Timeout, &pairs)
	addVar("CRE_DON_FAMILY", donFamily, &pairs)
	addVar("CRE_CONTEXT_YAML", contextYAMLPath, &pairs)
	lines := make([]string, 0, len(pairs))
	for _, p := range pairs {
		lines = append(lines, fmt.Sprintf("%s=%q", p.key, p.value))
	}

	if len(lines) == 0 {
		return "", nil
	}

	content := strings.Join(lines, "\n") + "\n"

	return envPath, os.WriteFile(envPath, []byte(content), 0o600)
}

type kv struct {
	key   string
	value string
}

func addVar(key, value string, pairs *[]kv) {
	if strings.TrimSpace(value) != "" {
		*pairs = append(*pairs, kv{key, value})
	}
}
