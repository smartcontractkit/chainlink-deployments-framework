package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	validCfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return changeset.NewChangesetsRegistry(), nil },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name:    "valid config",
			cfg:     validCfg,
			wantErr: "",
		},
		{
			name: "missing logger",
			cfg: &Config{
				Logger:                nil,
				Domain:                validCfg.Domain,
				LoadChangesets:        validCfg.LoadChangesets,
				ConfigResolverManager: validCfg.ConfigResolverManager,
			},
			wantErr: "pipeline.Config: missing required fields: Logger",
		},
		{
			name: "missing domain",
			cfg: &Config{
				Logger:                validCfg.Logger,
				Domain:                domain.Domain{},
				LoadChangesets:        validCfg.LoadChangesets,
				ConfigResolverManager: validCfg.ConfigResolverManager,
			},
			wantErr: "pipeline.Config: missing required fields: Domain",
		},
		{
			name: "missing LoadChangesets",
			cfg: &Config{
				Logger:                validCfg.Logger,
				Domain:                validCfg.Domain,
				LoadChangesets:        nil,
				ConfigResolverManager: validCfg.ConfigResolverManager,
			},
			wantErr: "pipeline.Config: missing required fields: LoadChangesets",
		},
		{
			name: "missing ConfigResolverManager",
			cfg: &Config{
				Logger:                validCfg.Logger,
				Domain:                validCfg.Domain,
				LoadChangesets:        validCfg.LoadChangesets,
				ConfigResolverManager: nil,
			},
			wantErr: "pipeline.Config: missing required fields: ConfigResolverManager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewCommand(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return changeset.NewChangesetsRegistry(), nil },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "pipeline", cmd.Use)
	require.Equal(t, []string{"durable-pipeline"}, cmd.Aliases)
	require.Len(t, cmd.Commands(), 4) // run, input-generate, list, template-input
}

func TestNewCommand_InvalidConfig(t *testing.T) {
	t.Parallel()

	_, err := NewCommand(&Config{})
	require.Error(t, err)
	require.Equal(t, "pipeline.Config: missing required fields: Logger, Domain, LoadChangesets, ConfigResolverManager", err.Error())
}
