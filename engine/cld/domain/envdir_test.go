package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/nodes"
)

func Test_EnvDir_RemoveMigrationAddressBook(t *testing.T) {
	t.Parallel()

	var (
		addrBook1 = createAddressBookMap(t,
			"Contract", version1_0_0,
			chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, "0x5B5BBb15ECE0a4Ed8cDab22F902e83F66aBe848f",
		)
		timestamp = "1234567890123456789"
	)

	tests := []struct {
		name              string
		beforeFunc        func(*testing.T, EnvDir)
		giveMigrationName string
		timestamp         string
		want              *cldf.AddressBookMap
		wantErr           string
	}{
		{
			name: "success with removing an address book",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration
				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					AddressBook: addrBook1,
				})
				require.NoError(t, err)

				err = envdir.MergeMigrationAddressBook("0001_initial", "")
				require.NoError(t, err)
			},
			giveMigrationName: "0001_initial",
			want: cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {},
			}),
		},
		{
			name: "success with removing a durable pipeline address book",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration
				arts := envdir.ArtifactsDir()
				err := arts.SetDurablePipelines(timestamp)
				require.NoError(t, err)

				err = arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					AddressBook: addrBook1,
				})
				require.NoError(t, err)

				err = envdir.MergeMigrationAddressBook("0001_initial", arts.timestamp)
				require.NoError(t, err)
			},
			giveMigrationName: "0001_initial",
			timestamp:         timestamp,
			want: cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {},
			}),
		},
		{
			name: "success skips with no migration address book found",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{})
				require.NoError(t, err)
			},
			giveMigrationName: "0001_initial",
			want:              cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{}),
		},
		{
			name:              "failure when no migration artifacts directory exists",
			giveMigrationName: "0001_invalid",
			wantErr:           "error finding files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				fixture = setupTestDomainsFS(t)
				envDir  = fixture.envDir
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, envDir)
			}

			// Merge the migration's address book into the existing address book
			err := envDir.RemoveMigrationAddressBook(tt.giveMigrationName, tt.timestamp)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				// Check the merged address book
				got, err := envDir.AddressBook()

				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_EnvDir_MigrateAddressBook(t *testing.T) {
	t.Parallel()

	var (
		addr1 = "0x5B5BBb15ECE0a4Ed8cDab22F902e83F66aBe848f"
		addr2 = "0x6619Bad7fadbc282B1EF2F6cC078fCbE61478792"

		csel1 = chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
		csel2 = chainsel.ETHEREUM_TESTNET_SEPOLIA_ARBITRUM_1.Selector

		ctype = datastore.ContractType("Contract")

		addrBook1 = createAddressBookMap(t,
			"Contract",
			version1_0_0,
			csel1,
			addr1,
		)
		addrBook2 = createAddressBookMap(t,
			"Contract",
			version1_0_0,
			csel2,
			addr2,
		)
	)
	addrs, err := addrBook1.Addresses()
	require.NoError(t, err)

	addrs[csel1][addr1].Labels.Add("label1")
	addrBook1 = cldf.NewMemoryAddressBookFromMap(addrs)

	convDataStore := datastore.NewMemoryDataStore()

	err = convDataStore.Addresses().Add(
		datastore.AddressRef{
			Address:       addr1,
			ChainSelector: csel1,
			Type:          ctype,
			Version:       &version1_0_0,
			Qualifier:     fmt.Sprintf("%s-%s", addr1, "Contract"),
			Labels:        datastore.NewLabelSet("label1"),
		},
	)
	require.NoError(t, err)

	err = convDataStore.Addresses().Add(
		datastore.AddressRef{
			Address:       addr2,
			ChainSelector: csel2,
			Type:          ctype,
			Version:       &version1_0_0,
			Qualifier:     fmt.Sprintf("%s-%s", addr2, "Contract"),
			Labels:        datastore.NewLabelSet(),
		},
	)
	require.NoError(t, err)

	tests := []struct {
		name              string
		beforeFunc        func(*testing.T, EnvDir)
		giveMigrationName string
		want              datastore.DataStore
	}{
		{
			name: "success when converting empty address book",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration
				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{})
				require.NoError(t, err)
			},
			giveMigrationName: "0001_initial",
			want:              datastore.NewMemoryDataStore().Seal(),
		},
		{
			name: "success when converting non empty address book",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration
				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					AddressBook: addrBook1,
				})
				require.NoError(t, err)

				err = envdir.MergeMigrationAddressBook("0001_initial", "")
				require.NoError(t, err)

				err = arts.SaveChangesetOutput("0002_second", cldf.ChangesetOutput{
					AddressBook: addrBook2,
				})
				require.NoError(t, err)
			},
			giveMigrationName: "0002_second",
			want:              convDataStore.Seal(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				fixture = setupTestDomainsFS(t)
				envDir  = fixture.envDir
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, envDir)
			}

			err := envDir.MergeMigrationAddressBook(tt.giveMigrationName, "")
			require.NoError(t, err)
			// convert the address book to a datastore
			err = envDir.MigrateAddressBook()
			require.NoError(t, err)

			// load the datastore
			got, err := envDir.DataStore()
			require.NoError(t, err)

			gotRefs, err := got.Addresses().Fetch()
			require.NoError(t, err)

			wantRefs, err := tt.want.Addresses().Fetch()
			require.NoError(t, err)

			require.ElementsMatch(t, wantRefs, gotRefs)
		})
	}
}

func Test_EnvDir_MutableDataStore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    func(*testing.T, testDomainFS) EnvDir
		want    datastore.MutableDataStore
		wantErr string
	}{
		{
			name: "success",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				return fixture.envDir
			},
			want: datastore.NewMemoryDataStore(),
		},
		{
			name: "missing file will return new empty datastore",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				return fixture.domain.EnvDir("invalid")
			},
			want: datastore.NewMemoryDataStore(),
		},
		{
			name: "empty file will return new empty datastore",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				// Create empty files in a new environment to simulate a corrupted datastore.
				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)

				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)

				ar, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), AddressRefsFileName))
				require.NoError(t, err)
				defer ar.Close()

				ch, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), ChainMetadataFileName))
				require.NoError(t, err)
				defer ch.Close()

				cm, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), ContractMetadataFileName))
				require.NoError(t, err)
				defer cm.Close()

				em, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), EnvMetadataFileName))
				require.NoError(t, err)
				defer em.Close()

				return fixture.domain.EnvDir("test")
			},
			want: datastore.NewMemoryDataStore(),
		},
		{
			name: "failed to unmarshal address ref JSON",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)
				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)
				ar, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), AddressRefsFileName))
				require.NoError(t, err)
				defer ar.Close()
				// Write invalid JSON to the address_refs file
				_, err = ar.WriteString("invalid json")
				require.NoError(t, err)

				return fixture.domain.EnvDir("test")
			},
			wantErr: "failed to unmarshal address refs JSON",
		},
		{
			name: "failed to unmarshal chain metadata JSON",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)
				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)
				ch, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), ChainMetadataFileName))
				require.NoError(t, err)
				defer ch.Close()
				// Write invalid JSON to the chain_metadata file
				_, err = ch.WriteString("invalid json")
				require.NoError(t, err)

				return fixture.domain.EnvDir("test")
			},
			wantErr: "failed to unmarshal chain metadata JSON",
		},
		{
			name: "failed to unmarshal contract metadata JSON",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)
				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)
				cm, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), ContractMetadataFileName))
				require.NoError(t, err)
				defer cm.Close()
				// Write invalid JSON to the contract_metadata file
				_, err = cm.WriteString("invalid json")
				require.NoError(t, err)

				return fixture.domain.EnvDir("test")
			},
			wantErr: "failed to unmarshal contract metadata JSON",
		},
		{
			name: "failed to unmarshal env metadata JSON",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)
				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)
				em, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), EnvMetadataFileName))
				require.NoError(t, err)
				defer em.Close()
				// Write invalid JSON to the env_metadata file
				_, err = em.WriteString("invalid json")
				require.NoError(t, err)

				return fixture.domain.EnvDir("test")
			},
			wantErr: "failed to unmarshal env metadata JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tt.give)

			fixture := setupTestDomainsFS(t)
			envdir := tt.give(t, fixture)

			got, err := envdir.MutableDataStore()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_EnvDir_Artifacts(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)

	got := fixture.envDir.ArtifactsDir()

	assert.Equal(t, fixture.artifactsDir, got)
}

func Test_EnvDir_MergeMigrationAddressBook(t *testing.T) {
	t.Parallel()

	var (
		addrBook1 = createAddressBookMap(t,
			"Contract", version1_0_0,
			chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, "0x5B5BBb15ECE0a4Ed8cDab22F902e83F66aBe848f",
		)

		addrBook2 = createAddressBookMap(t,
			"Contract", version1_0_0,
			chainsel.ETHEREUM_TESTNET_SEPOLIA_ARBITRUM_1.Selector, "0x7719BAd7FadbC282B1Ef2f6cC078FcbE61478792",
		)
	)

	tests := []struct {
		name              string
		beforeFunc        func(*testing.T, EnvDir)
		giveMigrationName string
		want              *cldf.AddressBookMap
		wantErr           string
	}{
		{
			name: "success with merging to empty address book",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration
				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					AddressBook: addrBook1,
				})
				require.NoError(t, err)
			},
			giveMigrationName: "0001_initial",
			want: cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
					"0x5B5BBb15ECE0a4Ed8cDab22F902e83F66aBe848f": cldf.TypeAndVersion{
						Type:    "Contract",
						Version: version1_0_0,
						Labels:  nil,
					},
				},
			}),
		},
		{
			name: "success with merging to non-empty address book",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration and merge to the address book
				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					AddressBook: addrBook1,
				})
				require.NoError(t, err)

				err = envdir.MergeMigrationAddressBook("0001_initial", "")
				require.NoError(t, err)

				// Create a migration with another address book
				err = arts.SaveChangesetOutput("0002_second", cldf.ChangesetOutput{
					AddressBook: addrBook2,
				})
				require.NoError(t, err)
			},
			giveMigrationName: "0002_second",
			want: cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
					"0x5B5BBb15ECE0a4Ed8cDab22F902e83F66aBe848f": {
						Type:    "Contract",
						Version: version1_0_0,
						Labels:  nil,
					},
				},
				chainsel.ETHEREUM_TESTNET_SEPOLIA_ARBITRUM_1.Selector: {
					"0x7719BAd7FadbC282B1Ef2f6cC078FcbE61478792": {
						Type:    "Contract",
						Version: version1_0_0,
						Labels:  nil,
					},
				},
			}),
		},
		{
			name: "success with merging non-empty durable pipeline address book",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration and merge to the address book
				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					AddressBook: addrBook1,
				})
				require.NoError(t, err)

				err = envdir.MergeMigrationAddressBook("0001_initial", "")
				require.NoError(t, err)

				// Create a durable pipeline artifact with another address book and merge to the address book
				err = arts.SetDurablePipelines("1742316304198171000")
				require.NoError(t, err)

				err = arts.SaveChangesetOutput("durable_pipeline", cldf.ChangesetOutput{
					AddressBook: addrBook2,
				})
				require.NoError(t, err)

				err = envdir.MergeMigrationAddressBook("durable_pipeline", arts.timestamp)
				require.NoError(t, err)
			},
			giveMigrationName: "durable_pipeline",
			want: cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
					"0x5B5BBb15ECE0a4Ed8cDab22F902e83F66aBe848f": {
						Type:    "Contract",
						Version: version1_0_0,
						Labels:  nil,
					},
				},
				chainsel.ETHEREUM_TESTNET_SEPOLIA_ARBITRUM_1.Selector: {
					"0x7719BAd7FadbC282B1Ef2f6cC078FcbE61478792": {
						Type:    "Contract",
						Version: version1_0_0,
						Labels:  nil,
					},
				},
			}),
		},
		{
			name: "success skips with no migration address book found",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{})
				require.NoError(t, err)
			},
			giveMigrationName: "0001_initial",
			want:              cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{}),
		},
		{
			name:              "failure when no migration artifacts directory exists",
			giveMigrationName: "0001_invalid",
			wantErr:           "error finding files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				fixture = setupTestDomainsFS(t)
				envDir  = fixture.envDir
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, envDir)
			}

			// Merge the migration's address book into the existing address book
			err := envDir.MergeMigrationAddressBook(tt.giveMigrationName, "")

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				// Check the merged address book
				got, err := envDir.AddressBook()

				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_EnvDir_MergeMigrationDataStore(t *testing.T) {
	t.Parallel()

	var (
		dataStore1 = createDataStore(t,
			"Contract", version1_0_0,
			chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
			"0x5B5BBb15ECE0a4Ed8cDab22F902e83F66aBe848f",
			"qtest1",
		)

		dataStore2 = createDataStore(t,
			"Contract", version1_0_0,
			chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
			"0x5B5BBb15ECE0a4Ed8cDab22F902e83F66aBe848f",
			"qtest2",
		)
	)

	mergeDatastore := datastore.NewMemoryDataStore()

	err := mergeDatastore.Merge(dataStore1.Seal())
	require.NoError(t, err)

	err = mergeDatastore.Merge(dataStore2.Seal())
	require.NoError(t, err)

	tests := []struct {
		name              string
		beforeFunc        func(*testing.T, EnvDir)
		giveMigrationName string
		want              datastore.DataStore
		wantErr           string
	}{
		{
			name: "success with merging to empty datastore",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration
				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					DataStore: dataStore1,
				})
				require.NoError(t, err)
			},
			giveMigrationName: "0001_initial",
			want:              dataStore1.Seal(),
		},
		{
			name: "success with merging to non-empty datastore",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration and merge to the address book
				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					DataStore: dataStore1,
				})
				require.NoError(t, err)

				err = envdir.MergeMigrationDataStore("0001_initial", "")
				require.NoError(t, err)

				// Create a migration with another datastore
				err = arts.SaveChangesetOutput("0002_second", cldf.ChangesetOutput{
					DataStore: dataStore2,
				})
				require.NoError(t, err)
			},
			giveMigrationName: "0002_second",
			want:              mergeDatastore.Seal(),
		},
		{
			name: "success with merging non-empty durable pipeline datastore",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				// Create the artifacts for the migration and merge to the address book
				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					DataStore: dataStore1,
				})
				require.NoError(t, err)

				err = envdir.MergeMigrationDataStore("0001_initial", "")
				require.NoError(t, err)

				// Create a durable pipeline artifact with another address book and merge to the address book
				err = arts.SetDurablePipelines("1742316304198171000")
				require.NoError(t, err)

				err = arts.SaveChangesetOutput("durable_pipeline", cldf.ChangesetOutput{
					DataStore: dataStore2,
				})
				require.NoError(t, err)

				err = envdir.MergeMigrationDataStore("durable_pipeline", arts.timestamp)
				require.NoError(t, err)
			},
			giveMigrationName: "durable_pipeline",
			want:              mergeDatastore.Seal(),
		},
		{
			name: "success skips with no migration datastore found",
			beforeFunc: func(t *testing.T, envdir EnvDir) {
				t.Helper()

				arts := envdir.ArtifactsDir()
				err := arts.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{})
				require.NoError(t, err)
			},
			giveMigrationName: "0001_initial",
			want:              datastore.NewMemoryDataStore().Seal(),
		},
		{
			name:              "failure when no migration artifacts directory exists",
			giveMigrationName: "0001_invalid",
			wantErr:           "error finding files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				fixture = setupTestDomainsFS(t)
				envDir  = fixture.envDir
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, envDir)
			}

			// Merge the migration's address book into the existing address book
			err := envDir.MergeMigrationDataStore(tt.giveMigrationName, "")

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				// Check the merged address book
				got, err := envDir.DataStore()

				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_EnvDir_DataStoreDirPath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/datastore", envdir.DataStoreDirPath())
}

func Test_EnvDir_DurablePipelinesDirPath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/durable_pipelines", envdir.DurablePipelinesDirPath())
}

func Test_EnvDir_DurablePipelinesInputsDirPath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/durable_pipelines/inputs", envdir.DurablePipelinesInputsDirPath())
}

func Test_EnvDir_CreateDurablePipelinesDir(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)
	envdir := fixture.envDir

	err := envdir.CreateDurablePipelinesDir()
	require.NoError(t, err)

	// Check if the directories exist
	_, err = os.Stat(envdir.DurablePipelinesDirPath())
	require.NoError(t, err)

	_, err = os.Stat(envdir.DurablePipelinesInputsDirPath())
	require.NoError(t, err)

	// Check if .gitkeep files exist
	_, err = os.Stat(filepath.Join(envdir.DurablePipelinesDirPath(), ".gitkeep"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(envdir.DurablePipelinesInputsDirPath(), ".gitkeep"))
	require.NoError(t, err)
}

func Test_EnvDir_String(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "ccip/staging", envdir.String())
}

func Test_EnvDir_DirPath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging", envdir.DirPath())
}

func Test_EnvDir_DomainDirPath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip", envdir.DomainDirPath())
}

func Test_EnvDir_Key(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "staging", envdir.Key())
}

func Test_EnvDir_DurablePipelineFilePath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/durable_pipelines.go", envdir.DurablePipelineFilePath())
}

func Test_EnvDir_AddressRefsFilePath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/datastore/address_refs.json", envdir.AddressRefsFilePath())
}

func Test_EnvDir_ChainMetadataFilePath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/datastore/chain_metadata.json", envdir.ChainMetadataFilePath())
}

func Test_EnvDir_ContractMetadataFilePath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/datastore/contract_metadata.json", envdir.ContractMetadataFilePath())
}

func Test_EnvDir_EnvMetadataFilePath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/datastore/env_metadata.json", envdir.EnvMetadataFilePath())
}

func Test_EnvDir_MigrationsFilePath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/migrations.go", envdir.MigrationsFilePath())
}

func Test_EnvDir_MigrationsArchiveFilePath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/migrations_archive.go", envdir.MigrationsArchiveFilePath())
}

func Test_EnvDir_InputsDirPath(t *testing.T) {
	t.Parallel()

	envdir := NewEnvDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/inputs", envdir.InputsDirPath())
}

func Test_EnvDir_AddressBook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    func(*testing.T, testDomainFS) EnvDir
		want    cldf.AddressBook
		wantErr string
	}{
		{
			name: "success",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				return fixture.envDir
			},
			want: cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{}),
		},
		{
			name: "failed to read file: missing file",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				return fixture.domain.EnvDir("invalid")
			},
			wantErr: "failed to read file",
		},
		{
			name: "failed to unmarshal JSON",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				// Create an empty file in a new environment to simulate a corrupted address book.
				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)

				ab, err := os.Create(filepath.Join(fixture.domain.DirPath(), "test", AddressBookFileName))
				require.NoError(t, err)
				defer ab.Close()

				return fixture.domain.EnvDir("test")
			},
			wantErr: "failed to unmarshal JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tt.give)

			fixture := setupTestDomainsFS(t)
			envdir := tt.give(t, fixture)

			got, err := envdir.AddressBook()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_EnvDir_DataStore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    func(*testing.T, testDomainFS) EnvDir
		want    datastore.DataStore
		wantErr string
	}{
		{
			name: "success",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				return fixture.envDir
			},
			want: datastore.NewMemoryDataStore().Seal(),
		},
		{
			name: "missing file will return new empty datastore",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				return fixture.domain.EnvDir("invalid")
			},
			want: datastore.NewMemoryDataStore().Seal(),
		},
		{
			name: "empty file will return new empty datastore",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				// Create empty files in a new environment to simulate a corrupted datastore.
				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)

				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)

				ar, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), AddressRefsFileName))
				require.NoError(t, err)
				defer ar.Close()

				ch, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), ChainMetadataFileName))
				require.NoError(t, err)
				defer ch.Close()

				cm, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), ContractMetadataFileName))
				require.NoError(t, err)
				defer cm.Close()

				em, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), EnvMetadataFileName))
				require.NoError(t, err)
				defer em.Close()

				return fixture.domain.EnvDir("test")
			},
			want: datastore.NewMemoryDataStore().Seal(),
		},
		{
			name: "failed to unmarshal address ref JSON",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)
				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)
				ar, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), AddressRefsFileName))
				require.NoError(t, err)
				defer ar.Close()
				// Write invalid JSON to the address_refs file
				_, err = ar.WriteString("invalid json")
				require.NoError(t, err)

				return fixture.domain.EnvDir("test")
			},
			wantErr: "failed to unmarshal address refs JSON",
		},
		{
			name: "failed to unmarshal chain metadata JSON",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)
				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)
				ch, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), ChainMetadataFileName))
				require.NoError(t, err)
				defer ch.Close()
				// Write invalid JSON to the chain_metadata file
				_, err = ch.WriteString("invalid json")
				require.NoError(t, err)

				return fixture.domain.EnvDir("test")
			},
			wantErr: "failed to unmarshal chain metadata JSON",
		},
		{
			name: "failed to unmarshal contract metadata JSON",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)
				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)
				cm, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), ContractMetadataFileName))
				require.NoError(t, err)
				defer cm.Close()
				// Write invalid JSON to the contract_metadata file
				_, err = cm.WriteString("invalid json")
				require.NoError(t, err)

				return fixture.domain.EnvDir("test")
			},
			wantErr: "failed to unmarshal contract metadata JSON",
		},
		{
			name: "failed to unmarshal env metadata JSON",
			give: func(t *testing.T, fixture testDomainFS) EnvDir {
				t.Helper()

				err := os.Mkdir(filepath.Join(fixture.domain.DirPath(), "test"), 0755)
				require.NoError(t, err)
				err = os.Mkdir(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath()), 0755)
				require.NoError(t, err)
				em, err := os.Create(filepath.Join(fixture.domain.EnvDir("test").DataStoreDirPath(), EnvMetadataFileName))
				require.NoError(t, err)
				defer em.Close()
				// Write invalid JSON to the env_metadata file
				_, err = em.WriteString("invalid json")
				require.NoError(t, err)

				return fixture.domain.EnvDir("test")
			},
			wantErr: "failed to unmarshal env metadata JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tt.give)

			fixture := setupTestDomainsFS(t)
			envdir := tt.give(t, fixture)

			got, err := envdir.DataStore()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_EnvDir_LoadNodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		beforeFunc func(*testing.T, testDomainFS)
		want       *nodes.Nodes
		wantErr    string
	}{
		{
			name: "success with no nodes in file",
			want: nodes.NewNodes([]string{}),
		},
		{
			name: "success with nodes in file",
			beforeFunc: func(t *testing.T, fixture testDomainFS) {
				t.Helper()

				nodesFile, err := os.Create(fixture.envDir.NodesFilePath())
				require.NoError(t, err)
				defer nodesFile.Close()

				_, err = nodesFile.WriteString(`{
					"nodes": {
						"node1": {},
						"node2": {}
					}
				}`)
				require.NoError(t, err)

				err = nodesFile.Sync()
				require.NoError(t, err)
			},
			want: nodes.NewNodes([]string{"node1", "node2"}),
		},
		{
			name: "failure with file read error",
			beforeFunc: func(t *testing.T, fixture testDomainFS) {
				t.Helper()

				err := os.Remove(fixture.envDir.NodesFilePath())
				require.NoError(t, err)
			},
			wantErr: "failed to read",
		},
		{
			name: "failure with file unmarshal error",
			beforeFunc: func(t *testing.T, fixture testDomainFS) {
				t.Helper()

				// Truncates the file
				nodesFile, err := os.Create(fixture.envDir.NodesFilePath())
				require.NoError(t, err)
				defer nodesFile.Close()
			},
			wantErr: "failed to unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, fixture)
			}

			nodes, err := fixture.envDir.LoadNodes()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, nodes)
			}
		})
	}
}

func Test_EnvDir_SaveFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		beforeFunc  func(*testing.T, testDomainFS)
		giveNodeIDs []string
		want        *nodes.Nodes
		wantErr     string
	}{
		{
			name:        "success with all new nodes",
			giveNodeIDs: []string{"node1", "node2"},
			want:        nodes.NewNodes([]string{"node1", "node2"}),
		},
		{
			name: "success with all adding and overwriting nodes",
			beforeFunc: func(t *testing.T, fixture testDomainFS) {
				t.Helper()

				nodesFile, err := os.Create(fixture.envDir.NodesFilePath())
				require.NoError(t, err)
				defer nodesFile.Close()

				_, err = nodesFile.WriteString(`{
					"nodes": {
						"node1": {},
						"node3": {}
					}
				}`)
				require.NoError(t, err)

				err = nodesFile.Sync()
				require.NoError(t, err)
			},
			giveNodeIDs: []string{"node1", "node2"},
			want:        nodes.NewNodes([]string{"node1", "node2", "node3"}),
		},
		{
			name: "success with non existent nodes file",
			beforeFunc: func(t *testing.T, fixture testDomainFS) {
				t.Helper()

				err := os.Remove(fixture.envDir.NodesFilePath())
				require.NoError(t, err)
			},
			giveNodeIDs: []string{"node1", "node2"},
			want:        nodes.NewNodes([]string{"node1", "node2"}),
		},
		{
			name: "failure with loading existing file",
			beforeFunc: func(t *testing.T, fixture testDomainFS) {
				t.Helper()

				nodesFile, err := os.Create(fixture.envDir.NodesFilePath())
				require.NoError(t, err)
				defer nodesFile.Close()

				_, err = nodesFile.Write([]byte{})
				require.NoError(t, err)

				err = nodesFile.Sync()
				require.NoError(t, err)
			},
			giveNodeIDs: []string{"node1", "node2"},
			wantErr:     "failed to unmarshal JSON",
		},
		{
			name: "failure with writing file due to permissions",
			beforeFunc: func(t *testing.T, fixture testDomainFS) {
				t.Helper()

				err := os.Chmod(fixture.envDir.NodesFilePath(), 0400)
				require.NoError(t, err)
			},
			giveNodeIDs: []string{"node1", "node2"},
			wantErr:     "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, fixture)
			}

			err := fixture.envDir.SaveNodes(tt.giveNodeIDs)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				nodes, err := fixture.envDir.LoadNodes()

				require.NoError(t, err)
				assert.Equal(t, tt.want, nodes)
			}
		})
	}
}

func Test_EnvDir_SaveViewState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		giveState json.Marshaler
		want      string
		wantErr   string
	}{
		{
			name: "success",
			giveState: &testMarshaler{
				Name: "test",
			},
			want: `{"Name":"test"}`,
		},
		{
			name:      "save error",
			giveState: &failedMarshaler{},
			wantErr:   "unable to marshal state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)

			err := fixture.envDir.SaveViewState(tt.giveState)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				b, err := os.ReadFile(fixture.envDir.ViewStateFilePath())
				require.NoError(t, err)

				assert.JSONEq(t, tt.want, string(b))
			}
		})
	}
}
