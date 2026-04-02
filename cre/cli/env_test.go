package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
)

func TestWriteCREEnvFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       cfgenv.CREConfig
		donFamily string
		hasCtx    bool
		check     func(t *testing.T, path string)
	}{
		{
			name: "empty config returns no path",
			cfg:  cfgenv.CREConfig{},
			check: func(t *testing.T, path string) {
				t.Helper()

				require.Empty(t, path)
			},
		},
		{
			name: "deterministic key order",
			cfg: cfgenv.CREConfig{
				Auth:           cfgenv.CREAuthConfig{TenantID: "t", OrgID: "o"},
				StorageAddress: "s", TLS: "1", Timeout: "30s",
			},
			donFamily: "d",
			hasCtx:    true,
			check: func(t *testing.T, path string) {
				t.Helper()

				b, err := os.ReadFile(path)
				require.NoError(t, err)
				s := string(b)
				require.True(t, strings.HasPrefix(s, "CRE_TENANT_ID="))
				idxTenant := strings.Index(s, "CRE_TENANT_ID")
				idxOrg := strings.Index(s, "CRE_ORG_ID")
				idxStorage := strings.Index(s, "CRE_STORAGE_ADDR")
				require.True(t, idxTenant < idxOrg && idxOrg < idxStorage)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			var ctxPath string
			if tc.hasCtx {
				ctxPath = filepath.Join(dir, "ctx.yaml")
				require.NoError(t, os.WriteFile(ctxPath, []byte("x"), 0o600))
			}
			path, err := WriteCREEnvFile(dir, ctxPath, tc.cfg, tc.donFamily)
			require.NoError(t, err)
			tc.check(t, path)
		})
	}
}

func TestWriteCREEnvFile_DoesNotWritePrivateKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ctxPath := filepath.Join(dir, "ctx.yaml")
	require.NoError(t, os.WriteFile(ctxPath, []byte("x"), 0o600))
	path, err := WriteCREEnvFile(dir, ctxPath, cfgenv.CREConfig{
		Auth: cfgenv.CREAuthConfig{TenantID: "tenant"},
	}, "d")
	require.NoError(t, err)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(b), "CRE_ETH_PRIVATE_KEY")
}
