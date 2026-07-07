package config

import (
	"fmt"
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
		name              string
		setup             func(t *testing.T, dom fdomain.Domain)
		wantProvider      cfgdomain.BinaryProvider
		wantVersion       string
		wantErr           string
		missingDomainFile bool
	}{
		{
			name: "successfully loads binary config",
			setup: func(t *testing.T, dom fdomain.Domain) {
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
			wantProvider: cfgdomain.BinaryProviderS3,
			wantVersion:  "v1.2.3",
		},
		{
			name: "defaults version to latest when not specified",
			setup: func(t *testing.T, dom fdomain.Domain) {
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
			wantProvider: cfgdomain.BinaryProviderSource,
			wantVersion:  cfgdomain.DefaultBinaryVersion,
		},
		{
			name: "defaults to source when binary section is absent",
			setup: func(t *testing.T, dom fdomain.Domain) {
				t.Helper()

				writeConfigDomainFile(t, dom, "domain.yaml")
			},
			wantProvider: cfgdomain.BinaryProviderSource,
			wantVersion:  cfgdomain.DefaultBinaryVersion,
		},
		{
			name:              "fails when domain config file does not exist",
			missingDomainFile: true,
		},
		{
			name: "fails when binary provider is invalid",
			setup: func(t *testing.T, dom fdomain.Domain) {
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
			wantErr: "invalid binary provider: artifact-registry (must be 'source' or 's3')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dom, _ := setupConfigDirs(t)

			if tt.setup != nil {
				tt.setup(t, dom)
			}

			cfg, err := LoadBinaryConfig(dom)

			if tt.missingDomainFile {
				require.EqualError(t, err, fmt.Sprintf("open %s: no such file or directory", dom.ConfigDomainFilePath()))
				require.Nil(t, cfg)

				return
			}

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, cfg)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Equal(t, tt.wantProvider, cfg.Provider)
			assert.Equal(t, tt.wantVersion, cfg.Version)
		})
	}
}
