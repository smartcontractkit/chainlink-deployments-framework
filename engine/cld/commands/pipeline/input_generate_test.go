package pipeline

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func inputGenerateResolver(input map[string]any) (any, error) {
	chain := "resolved"
	if c, ok := input["chain"].(string); ok {
		chain = c + "_resolved"
	}

	return map[string]any{"resolvedChain": chain}, nil
}

//nolint:paralleltest
func TestInputGenerateCmd_Success(t *testing.T) {
	env := "testnet"
	testDomain := domain.NewDomain(t.TempDir(), "test")
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	inputsContent := `environment: testnet
domain: test
changesets:
  - 0001_test_changeset:
      payload:
        chain: optimism_sepolia
        value: 100`
	inputsFile := "inputs.yaml"
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, inputsFile), []byte(inputsContent), 0o644)) //nolint:gosec

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	resolverManager := fresolvers.NewConfigResolverManager()
	resolverManager.Register(inputGenerateResolver, fresolvers.ResolverInfo{
		Description: "Test",
		ExampleYAML: "chain: x",
	})

	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		reg := changeset.NewChangesetsRegistry()
		reg.Add("0001_test_changeset", changeset.Configure(&stubChangeset{}).WithConfigResolver(inputGenerateResolver))

		return reg, nil
	}

	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                testDomain,
		LoadChangesets:        loadChangesets,
		ConfigResolverManager: resolverManager,
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{
		"input-generate",
		"--environment", env,
		"--inputs", inputsFile,
	})

	err = cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, buf.String(), "environment: testnet")
	require.Contains(t, buf.String(), "resolvedChain")
}

//nolint:paralleltest
func TestInputGenerateCmd_WithOutputFile(t *testing.T) {
	env := "testnet"
	testDomain := domain.NewDomain(t.TempDir(), "test")
	workspaceRoot := t.TempDir()
	inputsDir := filepath.Join(workspaceRoot, "domains", testDomain.String(), env, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	inputsContent := `environment: testnet
domain: test
changesets:
  - 0001_cs:
      payload:
        x: 1`
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "in.yaml"), []byte(inputsContent), 0o644)) //nolint:gosec

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(originalWd)) })

	withOutputResolver := func(m map[string]any) (any, error) { return m, nil }
	resolverManager := fresolvers.NewConfigResolverManager()
	resolverManager.Register(withOutputResolver, fresolvers.ResolverInfo{Description: "X"})

	loadChangesets := func(string) (*changeset.ChangesetsRegistry, error) {
		reg := changeset.NewChangesetsRegistry()
		reg.Add("0001_cs", changeset.Configure(&stubChangeset{}).WithConfigResolver(withOutputResolver))

		return reg, nil
	}

	outPath := filepath.Join(t.TempDir(), "out.yaml")
	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                testDomain,
		LoadChangesets:        loadChangesets,
		ConfigResolverManager: resolverManager,
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"input-generate",
		"--environment", env,
		"--inputs", "in.yaml",
		"--output", outPath,
	})

	err = cmd.Execute()
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "environment: testnet")
}

//nolint:paralleltest
func TestInputGenerateCmd_LoadError(t *testing.T) {
	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return nil, errors.New("load failed") },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{"input-generate", "--environment", "testnet", "--inputs", "x.yaml"})

	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, "load changesets registry: load failed", err.Error())
}

//nolint:paralleltest
func TestInputGenerateCmd_MissingEnvironment(t *testing.T) {
	cfg := &Config{
		Logger:                logger.Test(t),
		Domain:                domain.NewDomain(t.TempDir(), "test"),
		LoadChangesets:        func(string) (*changeset.ChangesetsRegistry, error) { return changeset.NewChangesetsRegistry(), nil },
		ConfigResolverManager: fresolvers.NewConfigResolverManager(),
	}

	cmd, err := NewCommand(cfg)
	require.NoError(t, err)

	cmd.SetArgs([]string{"input-generate", "--inputs", "x.yaml"})

	err = cmd.Execute()
	require.Error(t, err)
	require.Equal(t, `required flag(s) "environment" not set`, err.Error())
}
