package pipeline

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestListCmd_Success(t *testing.T) {
	t.Parallel()

	testResolver := func(map[string]any) (any, error) { return map[string]any{}, nil }
	resolverManager := fresolvers.NewConfigResolverManager()
	resolverManager.Register(testResolver, fresolvers.ResolverInfo{
		Description: "Test Resolver",
		ExampleYAML: "x: 1",
	})

	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		reg := changeset.NewChangesetsRegistry()
		reg.Add("0001_static_cs", changeset.Configure(&stubChangeset{}).With(1))
		reg.Add("0002_dynamic_cs", changeset.Configure(&stubChangeset{}).WithConfigResolver(testResolver))

		return reg, nil
	}

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        loadChangesets,
		ConfigResolverManager: resolverManager,
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"list", "--environment", "testnet"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "Durable Pipeline Info")
	require.Contains(t, output, "STATIC")
	require.Contains(t, output, "0001_static_cs")
	require.Contains(t, output, "DYNAMIC")
	require.Contains(t, output, "0002_dynamic_cs")
	require.Contains(t, output, "Available Config Resolvers")
}

func TestListCmd_LoadError(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("load failed")
	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return nil, loadErr },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{"list", "--environment", "testnet"})

	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, "failed to load changesets registry: load failed", err.Error())
}
