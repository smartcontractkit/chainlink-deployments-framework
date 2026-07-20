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

func TestGenerate_ArrayFormat(t *testing.T) {
	t.Parallel()

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

	arrayFormatResolver := func(m map[string]any) (any, error) { return m, nil }
	rm := fresolvers.NewConfigResolverManager()
	rm.Register(arrayFormatResolver, fresolvers.ResolverInfo{Description: "X"})

	reg := cs.NewChangesetsRegistry()
	reg.Add("0001_cs1", cs.Configure(&generateStubChangeset{}).WithConfigResolver(arrayFormatResolver))

	dom := domain.NewDomain(filepath.Join(dir, domain.DomainsDirName), "mydomain")
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

func TestGenerate_ObjectFormatRejected(t *testing.T) {
	t.Parallel()

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

	reg := cs.NewChangesetsRegistry()
	dom := domain.NewDomain(filepath.Join(dir, domain.DomainsDirName), "mydomain")
	opts := GenerateOptions{
		InputsFileName:  "in.yaml",
		Domain:          dom,
		EnvKey:          "testnet",
		Registry:        reg,
		ResolverManager: fresolvers.NewConfigResolverManager(),
	}

	_, err := Generate(opts)
	require.EqualError(t, err, "changesets must be an array (sequence)")
}

func TestGenerate_InvalidChangesetsFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "domains"), 0o755))
	inputsDir := filepath.Join(dir, "domains", "mydomain", "testnet", "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	inputsContent := `environment: testnet
domain: mydomain
changesets: "not-object-or-array"
`
	require.NoError(t, os.WriteFile(filepath.Join(inputsDir, "in.yaml"), []byte(inputsContent), 0o644)) //nolint:gosec

	reg := cs.NewChangesetsRegistry()
	dom := domain.NewDomain(filepath.Join(dir, domain.DomainsDirName), "mydomain")
	opts := GenerateOptions{
		InputsFileName:  "in.yaml",
		Domain:          dom,
		EnvKey:          "testnet",
		Registry:        reg,
		ResolverManager: fresolvers.NewConfigResolverManager(),
	}

	_, err := Generate(opts)
	require.Error(t, err)
	require.Equal(t, "changesets must be an array (sequence)", err.Error())
}

func TestGenerate_ResolverNotRegistered(t *testing.T) {
	t.Parallel()

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

	resolver := func(m map[string]any) (any, error) { return m, nil }
	reg := cs.NewChangesetsRegistry()
	reg.Add("0001_cs1", cs.Configure(&generateStubChangeset{}).WithConfigResolver(resolver))
	// rm does NOT register resolver
	rm := fresolvers.NewConfigResolverManager()

	dom := domain.NewDomain(filepath.Join(dir, domain.DomainsDirName), "mydomain")
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

func TestGenerate_ArrayItemInvalidFormat(t *testing.T) {
	t.Parallel()

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

	reg := cs.NewChangesetsRegistry()
	dom := domain.NewDomain(filepath.Join(dir, domain.DomainsDirName), "mydomain")
	opts := GenerateOptions{
		InputsFileName:  "in.yaml",
		Domain:          dom,
		EnvKey:          "testnet",
		Registry:        reg,
		ResolverManager: fresolvers.NewConfigResolverManager(),
	}

	_, err := Generate(opts)
	require.Error(t, err)
	require.Equal(t, "invalid changeset array item at index 0: expected a mapping with exactly one key-value pair", err.Error())
}
