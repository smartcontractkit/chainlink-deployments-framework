package environment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func Test_Load_InvalidEnvironment(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := fdomain.NewDomain("dummy", "test")

	_, err := Load(t.Context(), domain, "non_existent_env")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load networks")
}

func Test_Load_AddressBookFailure(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig)

	_, err := Load(t.Context(), domain, "staging")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "addresses.json: no such file or directory")
}

func Test_Load_DataStoreFailure(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook)

	_, err := Load(t.Context(), domain, "staging")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "address_refs.json: no such file or directory")
}

func Test_Load_LoadNodesFailure(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupDataStore)

	_, err := Load(t.Context(), domain, "staging")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nodes.json: no such file or directory")
}

func Test_Load_LoadOffchainClientFailure(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupDataStore, setupNodes)

	assert.Panics(t, func() {
		_, err := Load(t.Context(), domain, "staging")
		require.NoError(t, err)
	})
}

func Test_Load_NoError(t *testing.T) {
	t.Parallel()

	// Set up domain
	domain := setupTest(t, setupTestConfig, setupAddressbook, setupDataStore, setupNodes)

	_, err := Load(t.Context(), domain, "staging", WithoutJD())
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

func setupDataStore(t *testing.T, domain fdomain.Domain) {
	t.Helper()

	env := domain.EnvDir("staging")
	addressRefsConfig := `[]`
	chainMetadataConfig := `[]`
	contractMetadataConfig := `[]`
	envMetadataConfig := `null`

	// Create datastore directory
	require.NoError(t, os.MkdirAll(env.DataStoreDirPath(), 0755))

	// Create address refs file
	addressRefsPath := filepath.Join(env.DataStoreDirPath(), "address_refs.json")
	require.NoError(t, os.WriteFile(addressRefsPath, []byte(addressRefsConfig), 0600))

	// Create chain metadata file
	chainMetadataPath := filepath.Join(env.DataStoreDirPath(), "chain_metadata.json")
	require.NoError(t, os.WriteFile(chainMetadataPath, []byte(chainMetadataConfig), 0600))

	// Create contract metadata file
	contractMetadataPath := filepath.Join(env.DataStoreDirPath(), "contract_metadata.json")
	require.NoError(t, os.WriteFile(contractMetadataPath, []byte(contractMetadataConfig), 0600))

	// Create env metadata file
	envMetadataPath := filepath.Join(env.DataStoreDirPath(), "env_metadata.json")
	require.NoError(t, os.WriteFile(envMetadataPath, []byte(envMetadataConfig), 0600))
}

func setupNodes(t *testing.T, domain fdomain.Domain) {
	t.Helper()

	env := domain.EnvDir("staging")
	nodesConfig := `{}`

	// Create nodes file
	nodesPath := filepath.Join(env.DirPath(), "nodes.json")
	require.NoError(t, os.WriteFile(nodesPath, []byte(nodesConfig), 0600))
}
