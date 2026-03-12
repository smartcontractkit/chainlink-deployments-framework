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

func mockTemplateResolver(map[string]any) (any, error) {
	return map[string]any{"resolved": true}, nil
}

func TestTemplateInputCmd_Success(t *testing.T) {
	t.Parallel()

	resolverManager := fresolvers.NewConfigResolverManager()
	resolverManager.Register(mockTemplateResolver, fresolvers.ResolverInfo{
		Description: "MockResolver",
		ExampleYAML: "x: 1",
	})

	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		reg := changeset.NewChangesetsRegistry()
		reg.Add("0001_test_cs", changeset.Configure(&stubChangeset{}).WithConfigResolver(mockTemplateResolver))

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
	cmd.SetArgs([]string{
		"template-input",
		"--environment", "testnet",
		"--changeset", "0001_test_cs",
	})

	err = cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, buf.String(), "environment: testnet")
	require.Contains(t, buf.String(), "0001_test_cs")
	require.Contains(t, buf.String(), "payload:")
}

func TestTemplateInputCmd_MultipleChangesets(t *testing.T) {
	t.Parallel()

	resolverManager := fresolvers.NewConfigResolverManager()
	resolverManager.Register(mockTemplateResolver, fresolvers.ResolverInfo{Description: "Mock"})

	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		reg := changeset.NewChangesetsRegistry()
		reg.Add("0001_cs1", changeset.Configure(&stubChangeset{}).WithConfigResolver(mockTemplateResolver))
		reg.Add("0002_cs2", changeset.Configure(&stubChangeset{}).WithConfigResolver(mockTemplateResolver))

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
	cmd.SetArgs([]string{
		"template-input",
		"--environment", "testnet",
		"--changeset", "0001_cs1,0002_cs2",
	})

	err = cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, buf.String(), "0001_cs1")
	require.Contains(t, buf.String(), "0002_cs2")
}

func TestTemplateInputCmd_UnknownChangeset(t *testing.T) {
	t.Parallel()

	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		reg := changeset.NewChangesetsRegistry()
		reg.Add("0001_known_cs", changeset.Configure(&stubChangeset{}).With(1))

		return reg, nil
	}

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        loadChangesets,
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"template-input",
		"--environment", "testnet",
		"--changeset", "unknown_cs",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, "generate YAML template: get configurations for changeset unknown_cs: changeset 'unknown_cs' not found", err.Error())
}

func TestTemplateInputCmd_LoadError(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return nil, errors.New("load failed") },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"template-input",
		"--environment", "testnet",
		"--changeset", "0001_cs1",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, "load registry: load failed", err.Error())
}

func TestTemplateInputCmd_MissingChangesetFlag(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return changeset.NewChangesetsRegistry(), nil },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"template-input",
		"--environment", "testnet",
	})

	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, `required flag(s) "changeset" not set`, err.Error())
}
