package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func Test_Load(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		beforeFunc func(t *testing.T, dom fdomain.Domain, envKey string)
		wantErr    string
	}{
		{
			name: "Loads config",
			beforeFunc: func(t *testing.T, dom fdomain.Domain, envKey string) {
				t.Helper()

				writeConfigNetworksFile(t, dom, "networks.yaml", "networks-testnet.yaml")
				writeConfigLocalFile(t, dom, envKey, "config.testnet.yaml")
				writeConfigDomainFile(t, dom, "domain.yaml")
			},
		},
		{
			name: "fails to load networks - domain config does not exist",
			beforeFunc: func(t *testing.T, dom fdomain.Domain, envKey string) {
				t.Helper()

				writeConfigNetworksFile(t, dom, "networks.yaml", "networks-testnet.yaml")
			},
			wantErr: "failed to load networks",
		},
		{
			name: "loads config without local config file (falls back to env vars)",
			beforeFunc: func(t *testing.T, dom fdomain.Domain, envKey string) {
				t.Helper()

				writeConfigNetworksFile(t, dom, "networks.yaml", "networks-testnet.yaml")
				writeConfigDomainFile(t, dom, "domain.yaml")
				// Note: not creating a local config file - it should fall back to env vars
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				dom, envKey = setupConfigDirs(t)
				lggr        = logger.Test(t)
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, dom, envKey)
			}

			got, err := Load(dom, envKey, lggr)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, got.Networks)
				require.NotNil(t, got.Env)
				require.NotEmpty(t, got.DatastoreType, "DatastoreType should be loaded")
			}
		})
	}
}
