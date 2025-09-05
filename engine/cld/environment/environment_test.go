package environment

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	foperations "github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

func Test_WithAnvilKeyAsDeployer(t *testing.T) {
	t.Parallel()

	opts := &LoadEnvironmentOptions{}
	require.False(t, opts.anvilKeyAsDeployer)

	option := WithAnvilKeyAsDeployer()
	option(opts)

	assert.True(t, opts.anvilKeyAsDeployer)
}

func Test_WithReporter(t *testing.T) {
	t.Parallel()

	opts := &LoadEnvironmentOptions{}
	assert.Nil(t, opts.reporter)

	reporter := foperations.NewMemoryReporter()
	option := WithReporter(reporter)
	option(opts)

	assert.Equal(t, reporter, opts.reporter)
}

func Test_WithOutJD(t *testing.T) {
	t.Parallel()

	opts := &LoadEnvironmentOptions{}
	require.False(t, opts.withoutJD)

	option := WithoutJD()
	option(opts)

	assert.True(t, opts.withoutJD)
}

func Test_OnlyLoadChainsFor(t *testing.T) {
	t.Parallel()

	opts := &LoadEnvironmentOptions{}
	assert.Empty(t, opts.migrationString)
	assert.Nil(t, opts.chainSelectorsToLoad)

	migrationKey := "test_migration"
	chainSelectors := []uint64{1, 2, 3}

	option := OnlyLoadChainsFor(migrationKey, chainSelectors)
	option(opts)

	assert.Equal(t, migrationKey, opts.migrationString)
	assert.Equal(t, chainSelectors, opts.chainSelectorsToLoad)
}

func Test_WithOperationRegistry(t *testing.T) {
	t.Parallel()

	opts := &LoadEnvironmentOptions{}
	assert.Nil(t, opts.operationRegistry)

	registry := foperations.NewOperationRegistry()
	option := WithOperationRegistry(registry)
	option(opts)

	assert.Equal(t, registry, opts.operationRegistry)
}

func Test_Load_InvalidEnvironment(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := fdomain.NewDomain("dummy", "test")

	lggr := logger.Test(t)
	getCtx := func() context.Context { return context.Background() }

	_, err := Load(getCtx, lggr, "non_existent_env", domain, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load networks")
}

func Test_Load_AddressBookFailure(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig)

	lggr := logger.Test(t)
	getCtx := func() context.Context { return context.Background() }

	_, err := Load(getCtx, lggr, "staging", domain, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "addresses.json: no such file or directory")
}

func Test_Load_LoadNodesFailure(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook)

	lggr := logger.Test(t)
	getCtx := func() context.Context { return context.Background() }

	_, err := Load(getCtx, lggr, "staging", domain, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nodes.json: no such file or directory")
}

func Test_Load_LoadOffchainClientFailure(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)

	lggr := logger.Test(t)
	getCtx := func() context.Context { return context.Background() }

	assert.Panics(t, func() {
		_, err := Load(getCtx, lggr, "staging", domain, false)
		require.NoError(t, err)
	})
}

func Test_Load_NoError(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupNodes)

	lggr := logger.Test(t)
	getCtx := func() context.Context { return context.Background() }

	_, err := Load(getCtx, lggr, "staging", domain, false, WithoutJD())
	require.NoError(t, err)
}

func setupTest(t *testing.T, setupFnc ...func(t *testing.T, domain fdomain.Domain)) fdomain.Domain {
	t.Helper()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a minimal domain structure
	domainsDir := filepath.Join(tempDir, "domains")
	require.NoError(t, os.MkdirAll(domainsDir, 0755))

	testDomainDir := filepath.Join(domainsDir, "test")
	require.NoError(t, os.MkdirAll(testDomainDir, 0755))

	// Create environments directory
	envsDir := filepath.Join(testDomainDir, "staging")
	require.NoError(t, os.MkdirAll(envsDir, 0755))

	// Set up domain
	domain := fdomain.NewDomain(domainsDir, "test")

	for _, fn := range setupFnc {
		fn(t, domain)
	}

	return domain
}

func setupTestConfig(t *testing.T, domain fdomain.Domain) {
	t.Helper()

	// Create a minimal config directory
	configDir := filepath.Join(domain.DirPath(), ".config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	networksDir := filepath.Join(configDir, "networks")
	require.NoError(t, os.MkdirAll(networksDir, 0755))

	// Create network configuration file
	input, err := os.ReadFile(filepath.Join("testdata", "networks.yaml"))
	require.NoError(t, err)

	networksPath := filepath.Join(networksDir, "networks-testnet.yaml")
	require.NoError(t, os.WriteFile(networksPath, input, 0600))

	// Create local configuration file
	localDir := filepath.Join(configDir, "local")
	require.NoError(t, os.MkdirAll(localDir, 0755))

	input, err = os.ReadFile(filepath.Join("testdata", "config.staging.yaml"))
	require.NoError(t, err)

	localPath := filepath.Join(localDir, "config.staging.yaml")
	require.NoError(t, os.WriteFile(localPath, input, 0600))

	// Create domains configuration file
	input, err = os.ReadFile(filepath.Join("testdata", "domain.yaml"))
	require.NoError(t, err)

	domainPath := filepath.Join(configDir, "domain.yaml")
	require.NoError(t, os.WriteFile(domainPath, input, 0600))
}

func setupAddressbook(t *testing.T, domain fdomain.Domain) {
	t.Helper()

	env := domain.EnvDir("staging")
	addressbookConfig := `{}`

	// Create address book file
	addressBookPath := filepath.Join(env.DirPath(), "addresses.json")
	require.NoError(t, os.WriteFile(addressBookPath, []byte(addressbookConfig), 0600))
}

func setupNodes(t *testing.T, domain fdomain.Domain) {
	t.Helper()

	env := domain.EnvDir("staging")
	nodesConfig := `{}`

	// Create nodes file
	nodesPath := filepath.Join(env.DirPath(), "nodes.json")
	require.NoError(t, os.WriteFile(nodesPath, []byte(nodesConfig), 0600))
}
