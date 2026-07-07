package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestLoadBinaryConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		beforeFunc func(t *testing.T, dom fdomain.Domain)
		wantErrMsg string
		validate   func(t *testing.T, cfg *cfgdomain.BinaryConfig)
	}{
		{
			name: "successfully loads binary config",
			beforeFunc: func(t *testing.T, dom fdomain.Domain) {
				t.Helper()

				domainYAML := `environments:
  staging_testnet:
    network_types:
      - testnet

binary:
  provider: s3
  version: v1.2.3
`
				err := os.WriteFile(dom.ConfigDomainFilePath(), []byte(domainYAML), filePerms)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, cfg *cfgdomain.BinaryConfig) {
				t.Helper()

				assert.Equal(t, cfgdomain.BinaryProviderS3, cfg.Provider)
				assert.Equal(t, "v1.2.3", cfg.Version)
			},
		},
		{
			name: "defaults version to latest when not specified",
			beforeFunc: func(t *testing.T, dom fdomain.Domain) {
				t.Helper()

				domainYAML := `environments:
  staging_testnet:
    network_types:
      - testnet

binary:
  provider: source
`
				err := os.WriteFile(dom.ConfigDomainFilePath(), []byte(domainYAML), filePerms)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, cfg *cfgdomain.BinaryConfig) {
				t.Helper()

				assert.Equal(t, cfgdomain.BinaryProviderSource, cfg.Provider)
				assert.Equal(t, cfgdomain.DefaultBinaryVersion, cfg.Version)
			},
		},
		{
			name: "defaults to source when binary section is absent",
			beforeFunc: func(t *testing.T, dom fdomain.Domain) {
				t.Helper()

				writeConfigDomainFile(t, dom, "domain.yaml")
			},
			validate: func(t *testing.T, cfg *cfgdomain.BinaryConfig) {
				t.Helper()

				assert.Equal(t, cfgdomain.BinaryProviderSource, cfg.Provider)
				assert.Equal(t, cfgdomain.DefaultBinaryVersion, cfg.Version)
			},
		},
		{
			name: "fails when domain config file does not exist",
			beforeFunc: func(t *testing.T, dom fdomain.Domain) {
				t.Helper()
			},
			wantErrMsg: "no such file or directory",
		},
		{
			name: "fails when binary provider is invalid",
			beforeFunc: func(t *testing.T, dom fdomain.Domain) {
				t.Helper()

				domainYAML := `environments:
  staging_testnet:
    network_types:
      - testnet

binary:
  provider: artifact-registry
`
				err := os.WriteFile(dom.ConfigDomainFilePath(), []byte(domainYAML), filePerms)
				require.NoError(t, err)
			},
			wantErrMsg: "invalid binary provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dom, _ := setupConfigDirs(t)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, dom)
			}

			cfg, err := LoadBinaryConfig(dom)

			if tt.wantErrMsg != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErrMsg)
				require.Nil(t, cfg)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}
