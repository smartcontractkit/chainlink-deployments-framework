package input

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

type generateStubChangeset struct{}

func (g *generateStubChangeset) Apply(_ fdeployment.Environment, _ any) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}
func (g *generateStubChangeset) VerifyPreconditions(_ fdeployment.Environment, _ any) error {
	return nil
}

var _ fdeployment.ChangeSetV2[any] = (*generateStubChangeset)(nil)

func generateTestResolver(m map[string]any) (any, error) {
	return map[string]any{"resolved": true, "v": m["v"]}, nil
}

//nolint:paralleltest
func TestGenerate_ObjectFormat(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "domains"), 0o755))
	inputsDir := filepath.Join(dir, "domains", "mydomain", "testnet", "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	inputsContent := `environment: testnet
domain: mydomain
changesets:
  0001_cs1:
    payload:
      v: 1
`
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "in.yaml"), []byte(inputsContent), 0o644)) //nolint:gosec

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	rm := fresolvers.NewConfigResolverManager()
	rm.Register(generateTestResolver, fresolvers.ResolverInfo{Description: "X"})

	reg := cs.NewChangesetsRegistry()
	reg.Add("0001_cs1", cs.Configure(&generateStubChangeset{}).WithConfigResolver(generateTestResolver))

	dom := domain.NewDomain(dir, "mydomain")
	opts := GenerateOptions{
		InputsFileName:  "in.yaml",
		Domain:          dom,
		EnvKey:          "testnet",
		Registry:        reg,
		ResolverManager: rm,
		FormatAsJSON:    false,
	}

	got, err := Generate(opts)
	require.NoError(t, err)
	require.Equal(t, "environment: testnet\ndomain: mydomain\nchangesets:\n    0001_cs1:\n        payload:\n            resolved: true\n            v: 1\n", got)
}

//nolint:paralleltest
func TestGenerate_ArrayFormat(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "domains"), 0o755))
	inputsDir := filepath.Join(dir, "domains", "mydomain", "testnet", "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	inputsContent := `environment: testnet
domain: mydomain
changesets:
  - 0001_cs1:
      payload:
        x: 1
`
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "in.yaml"), []byte(inputsContent), 0o644)) //nolint:gosec

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	arrayFormatResolver := func(m map[string]any) (any, error) { return m, nil }
	rm := fresolvers.NewConfigResolverManager()
	rm.Register(arrayFormatResolver, fresolvers.ResolverInfo{Description: "X"})

	reg := cs.NewChangesetsRegistry()
	reg.Add("0001_cs1", cs.Configure(&generateStubChangeset{}).WithConfigResolver(arrayFormatResolver))

	dom := domain.NewDomain(dir, "mydomain")
	opts := GenerateOptions{
		InputsFileName:  "in.yaml",
		Domain:          dom,
		EnvKey:          "testnet",
		Registry:        reg,
		ResolverManager: rm,
		FormatAsJSON:    true,
	}

	got, err := Generate(opts)
	require.NoError(t, err)
	require.JSONEq(t, "{\n  \"changesets\": [\n    {\n      \"0001_cs1\": {\n        \"payload\": {\n          \"x\": 1\n        }\n      }\n    }\n  ],\n  \"domain\": \"mydomain\",\n  \"environment\": \"testnet\"\n}", got)
}

//nolint:paralleltest
func TestGenerate_InvalidChangesetsFormat(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "domains"), 0o755))
	inputsDir := filepath.Join(dir, "domains", "mydomain", "testnet", "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	inputsContent := `environment: testnet
domain: mydomain
changesets: "not-object-or-array"
`
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "in.yaml"), []byte(inputsContent), 0o644)) //nolint:gosec

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	reg := cs.NewChangesetsRegistry()
	dom := domain.NewDomain(dir, "mydomain")
	opts := GenerateOptions{
		InputsFileName:  "in.yaml",
		Domain:          dom,
		EnvKey:          "testnet",
		Registry:        reg,
		ResolverManager: fresolvers.NewConfigResolverManager(),
	}

	_, err := Generate(opts)
	require.Error(t, err)
	require.Equal(t, "changesets must be either an object (mapping) or an array (sequence), got 8", err.Error())
}

//nolint:paralleltest
func TestGenerate_ResolverNotRegistered(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "domains"), 0o755))
	inputsDir := filepath.Join(dir, "domains", "mydomain", "testnet", "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	inputsContent := `environment: testnet
domain: mydomain
changesets:
  0001_cs1:
    payload:
      x: 1
`
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "in.yaml"), []byte(inputsContent), 0o644)) //nolint:gosec

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	resolver := func(m map[string]any) (any, error) { return m, nil }
	reg := cs.NewChangesetsRegistry()
	reg.Add("0001_cs1", cs.Configure(&generateStubChangeset{}).WithConfigResolver(resolver))
	// rm does NOT register resolver
	rm := fresolvers.NewConfigResolverManager()

	dom := domain.NewDomain(dir, "mydomain")
	opts := GenerateOptions{
		InputsFileName:  "in.yaml",
		Domain:          dom,
		EnvKey:          "testnet",
		Registry:        reg,
		ResolverManager: rm,
	}

	_, err := Generate(opts)
	require.Error(t, err)
	require.Equal(t, "resolver for changeset \"0001_cs1\" is not registered with the resolver manager", err.Error())
}

//nolint:paralleltest
func TestGenerate_ArrayItemInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "domains"), 0o755))
	inputsDir := filepath.Join(dir, "domains", "mydomain", "testnet", "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	inputsContent := `environment: testnet
domain: mydomain
changesets:
  - "not-a-mapping"
`
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "in.yaml"), []byte(inputsContent), 0o644)) //nolint:gosec

	originalWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	reg := cs.NewChangesetsRegistry()
	dom := domain.NewDomain(dir, "mydomain")
	opts := GenerateOptions{
		InputsFileName:  "in.yaml",
		Domain:          dom,
		EnvKey:          "testnet",
		Registry:        reg,
		ResolverManager: fresolvers.NewConfigResolverManager(),
	}

	_, err := Generate(opts)
	require.Error(t, err)
	require.Equal(t, "invalid changeset array item format - expected mapping with at least one key-value pair", err.Error())
}
