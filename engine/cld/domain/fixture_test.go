package domain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

var (
	// A version constant fixture for testing purposes.
	version1_0_0 = *semver.MustParse("1.0.0")
)

// testDomainFS contains the paths to the domain directory structure for testing.
type testDomainFS struct {
	rootDirPath  string // Path to the root directory
	domain       Domain
	envDir       EnvDir
	artifactsDir *ArtifactsDir
}

// TODO: unexport after migrations/archive.go is extracted from cldf
// setupDomainFS creates the domain directory structure for testing.
func setupTestDomainsFS(t *testing.T) testDomainFS {
	t.Helper()

	// Setup the root directory.
	rootDir := t.TempDir()

	var (
		domDir         = filepath.Join(rootDir, "ccip")
		envDir         = filepath.Join(domDir, "staging")
		artDir         = filepath.Join(envDir, "artifacts")
		propDir        = filepath.Join(envDir, "proposals")
		decodedPropDir = filepath.Join(envDir, "decoded_proposals")
		apropDir       = filepath.Join(envDir, "archived_proposals")
		reportsDir     = filepath.Join(envDir, "operations_reports")
		datastoreDir   = filepath.Join(envDir, "datastore")
	)

	// Create the test domains.
	err := os.Mkdir(domDir, 0755)
	require.NoError(t, err)

	// Create the environments.
	err = os.Mkdir(envDir, 0755)
	require.NoError(t, err)

	// Create the artifacts directory.
	err = os.Mkdir(artDir, 0755)
	require.NoError(t, err)

	// Create the operations reports directory.
	err = os.Mkdir(reportsDir, 0755)
	require.NoError(t, err)

	// Create the proposals directory.
	err = os.Mkdir(propDir, 0755)
	require.NoError(t, err)

	// Create the decoded proposals directory.
	err = os.Mkdir(decodedPropDir, 0755)
	require.NoError(t, err)

	// Create the archived proposals directory.
	err = os.Mkdir(apropDir, 0755)
	require.NoError(t, err)

	// Create the address book file.
	abFile, err := os.Create(filepath.Join(envDir, AddressBookFileName))
	require.NoError(t, err)
	defer abFile.Close()

	_, err = abFile.WriteString(`{}`)
	require.NoError(t, err)

	err = abFile.Sync()
	require.NoError(t, err)

	// Create the datastore
	err = os.Mkdir(datastoreDir, 0755)
	require.NoError(t, err)

	// Create the address refs file
	addrRefsFile, err := os.Create(filepath.Join(datastoreDir, AddressRefsFileName))
	require.NoError(t, err)
	defer addrRefsFile.Close()

	_, err = addrRefsFile.WriteString(`[]`)
	require.NoError(t, err)

	err = addrRefsFile.Sync()
	require.NoError(t, err)

	// Create the chain metadata file
	chainMetadataFile, err := os.Create(filepath.Join(datastoreDir, ChainMetadataFileName))
	require.NoError(t, err)
	defer chainMetadataFile.Close()

	_, err = chainMetadataFile.WriteString(`[]`)
	require.NoError(t, err)

	err = chainMetadataFile.Sync()
	require.NoError(t, err)

	// Create the contract metadata file
	contractMetadataFile, err := os.Create(filepath.Join(datastoreDir, ContractMetadataFileName))
	require.NoError(t, err)
	defer contractMetadataFile.Close()

	_, err = contractMetadataFile.WriteString(`[]`)
	require.NoError(t, err)

	err = contractMetadataFile.Sync()
	require.NoError(t, err)

	// Create the contract metadata file
	envMetadataFile, err := os.Create(filepath.Join(datastoreDir, EnvMetadataFileName))
	require.NoError(t, err)
	defer envMetadataFile.Close()

	_, err = envMetadataFile.WriteString(`null`)
	require.NoError(t, err)

	err = envMetadataFile.Sync()
	require.NoError(t, err)

	// Create the nodes file
	nodesFile, err := os.Create(filepath.Join(envDir, NodesFileName))
	require.NoError(t, err)
	defer nodesFile.Close()

	_, err = nodesFile.WriteString(`{"nodes": {}}`)
	require.NoError(t, err)

	err = nodesFile.Sync()
	require.NoError(t, err)

	dom := NewDomain(rootDir, "ccip")

	return testDomainFS{
		rootDirPath:  rootDir,
		domain:       dom,
		envDir:       dom.EnvDir("staging"),
		artifactsDir: dom.ArtifactsDirByEnv("staging"),
	}
}

// createAddressBook creates an address book with a single entry for testing.
func createAddressBookMap(
	t *testing.T, cType cldf.ContractType, ver semver.Version, csel uint64, addr string, //nolint:unparam // Unused parameters are for testing purposes
) *cldf.AddressBookMap {
	t.Helper()

	// Create a new migration with an address book
	var (
		addrBook = cldf.NewMemoryAddressBook()
		tv       = cldf.NewTypeAndVersion(cType, ver)
	)

	// Save data to the address book
	err := addrBook.Save(csel, addr, tv)
	require.NoError(t, err)

	return addrBook
}

func createDataStore(
	t *testing.T, cType cldf.ContractType, ver semver.Version, csel uint64, addr string, qual string, //nolint:unparam // Unused parameters are for testing purposes
) *datastore.MemoryDataStore {
	t.Helper()

	// Create a new migration with an address book
	ds := datastore.NewMemoryDataStore()

	// Save data to the address book
	err := ds.Addresses().Add(
		datastore.AddressRef{
			ChainSelector: csel,
			Address:       addr,
			Type:          datastore.ContractType(cType),
			Version:       &ver,
			Qualifier:     qual,
			Labels: datastore.NewLabelSet(
				"LinkToken",
				"LinkTokenV1",
			),
		},
	)
	require.NoError(t, err)

	return ds
}
