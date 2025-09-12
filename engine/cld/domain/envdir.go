package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/fileutils"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/jsonutils"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/nodes"
)

// EnvDir represents a specific environment directory within a domain.
type EnvDir struct {
	// rootPath is absolute path of the domains filesystem. e.g. if the domain directory is in the
	// project root directory, this would be the project root directory.
	// TODO: unexport after scaffolding logic moved to cld
	rootPath string

	// The key of the domain that the environment belongs to. e.g. "ccip", "keystone"
	domainKey string

	// The name of the environment. e.g. "mainnet", "testnet", "staging"
	key string
}

// NewEnvDir creates a new DomainEnv.
func NewEnvDir(rootPath, domainKey, key string) EnvDir {
	return EnvDir{
		rootPath:  rootPath,
		domainKey: domainKey,
		key:       key,
	}
}

// String returns the domain and key of the environment.
func (d EnvDir) String() string {
	return fmt.Sprintf("%s/%s", d.domainKey, d.key)
}

// DirPath returns the path to the environment directory within the domain.
func (d EnvDir) DirPath() string {
	return filepath.Join(d.rootPath, d.domainKey, d.key)
}

// RootPath returns the root path of the environment directory.
func (d EnvDir) RootPath() string { return d.rootPath }

// DomainKey returns the domain key that the environment belongs to.
func (d EnvDir) DomainKey() string {
	return d.domainKey
}

// DomainDirPath returns the path to the domain directory within the environment.
func (d EnvDir) DomainDirPath() string {
	return filepath.Join(d.rootPath, d.domainKey)
}

// Key returns the environment key.
func (d EnvDir) Key() string {
	return d.key
}

// PipelinesFilePath returns the path to the Pipeline file for the domain's environment
// directory.
func (d EnvDir) PipelinesFilePath() string {
	return filepath.Join(d.DirPath(), PipelinesFileName)
}

// MigrationsFilePath returns the path to the migrations file for the domain's environment
// directory.
func (d EnvDir) MigrationsFilePath() string {
	return filepath.Join(d.DirPath(), MigrationsFileName)
}

// MigrationsArchiveFilePath returns the path to the migrations archive file for the domain's
// environment directory.
func (d EnvDir) MigrationsArchiveFilePath() string {
	return filepath.Join(d.DirPath(), MigrationsArchiveFileName)
}

// AddressBookFilePath returns the path to the address book file for the domain's environment
// directory.
func (d EnvDir) AddressBookFilePath() string {
	return filepath.Join(d.DirPath(), AddressBookFileName)
}

// AddressRefsFilePath returns the path to the address refs store file for the domain's
// environment directory.
func (d EnvDir) AddressRefsFilePath() string {
	return filepath.Join(d.DirPath(), DatastoreDirName, AddressRefsFileName)
}

// ChainMetadataFilePath returns the path to the chain metadata store file for the
// domain's environment directory.
func (d EnvDir) ChainMetadataFilePath() string {
	return filepath.Join(d.DirPath(), DatastoreDirName, ChainMetadataFileName)
}

// ContractMetadataFilePath returns the path to the contract metadata store file for the
// domain's environment directory.
func (d EnvDir) ContractMetadataFilePath() string {
	return filepath.Join(d.DirPath(), DatastoreDirName, ContractMetadataFileName)
}

// EnvMetadataFilePath returns the path to the environment metadata file for the domain's
// environment directory.
func (d EnvDir) EnvMetadataFilePath() string {
	return filepath.Join(d.DirPath(), DatastoreDirName, EnvMetadataFileName)
}

// AddressBook returns the address book for the domain's environment directory.
func (d EnvDir) AddressBook() (fdeployment.AddressBook, error) {
	filePath := d.AddressBookFilePath()
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	addrsByChain := make(map[uint64]map[string]fdeployment.TypeAndVersion)
	if err = json.Unmarshal(b, &addrsByChain); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return fdeployment.NewMemoryAddressBookFromMap(addrsByChain), nil
}

// DataStore returns the datastore for the domain's environment directory as read-only.
func (d EnvDir) DataStore() (fdatastore.DataStore, error) {
	ds, err := d.loadDataStore()
	if err != nil {
		return nil, fmt.Errorf("failed to load datastore: %w", err)
	}

	return ds.Seal(), nil
}

// MutableDataStore returns the datastore for the domain's environment directory as mutable.
func (d EnvDir) MutableDataStore() (fdatastore.MutableDataStore, error) {
	ds, err := d.loadDataStore()
	if err != nil {
		return nil, fmt.Errorf("failed to load mutable datastore: %w", err)
	}

	return ds, nil
}

// loadDataStore is a helper function that loads the datastore for the domain's environment
// directory from the address_refs.json, contract_metadata.json, and env_metadata.json files.
func (d EnvDir) loadDataStore() (fdatastore.MutableDataStore, error) {
	addrRefsPath := d.AddressRefsFilePath()
	refs, err := os.ReadFile(addrRefsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read address_refs file %s: %w", addrRefsPath, err)
	}

	chainMetaPath := d.ChainMetadataFilePath()
	chainMeta, err := os.ReadFile(chainMetaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chain_metadata file %s: %w", chainMetaPath, err)
	}

	ctrMetaPath := d.ContractMetadataFilePath()
	ctrMeta, err := os.ReadFile(ctrMetaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read contract_metadata file %s: %w", ctrMetaPath, err)
	}

	envMetaPath := d.EnvMetadataFilePath()
	envMeta, err := os.ReadFile(envMetaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read env_metadata file %s: %w", envMetaPath, err)
	}

	var ds = fdatastore.NewMemoryDataStore()

	if len(refs) > 0 {
		if err = json.Unmarshal(refs, &ds.AddressRefStore.Records); err != nil {
			return nil, fmt.Errorf("failed to unmarshal address refs JSON: %w", err)
		}
	}

	if len(chainMeta) > 0 {
		if err = json.Unmarshal(chainMeta, &ds.ChainMetadataStore.Records); err != nil {
			return nil, fmt.Errorf("failed to unmarshal chain metadata JSON: %w", err)
		}
	}
	if len(ctrMeta) > 0 {
		if err = json.Unmarshal(ctrMeta, &ds.ContractMetadataStore.Records); err != nil {
			return nil, fmt.Errorf("failed to unmarshal contract metadata JSON: %w", err)
		}
	}

	if len(envMeta) > 0 {
		if err = json.Unmarshal(envMeta, &ds.EnvMetadataStore.Record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal env metadata JSON: %w", err)
		}
	}

	return ds, nil
}

// ArtifactsDir returns the artifacts for the domain's environment directory.
func (d EnvDir) ArtifactsDir() *ArtifactsDir {
	return NewArtifactsDir(d.rootPath, d.domainKey, d.key)
}

func (d EnvDir) MergeMigrationDataStore(migkey, timestamp string) error {
	// Get the artifacts directory for the environment
	artDir := d.ArtifactsDir()

	// Load the migration datastore for the migration key and timestamp
	migrDataStore, err := loadDataStoreByMigrationKey(artDir, migkey, timestamp)
	if err != nil {
		return err
	}

	// Merge the migration datastore into the existing datastore
	dataStore, err := d.MutableDataStore()
	if err != nil {
		return err
	}

	if err = dataStore.Merge(migrDataStore); err != nil {
		return err
	}

	// Cast the datastore to the concrete type and write it to the file
	dataStoreConcrete, ok := dataStore.(*fdatastore.MemoryDataStore)
	if !ok {
		return errors.New("failed to cast dataStore to concrete type MemoryDataStore")
	}

	err = jsonutils.WriteFile(d.AddressRefsFilePath(), dataStoreConcrete.AddressRefStore.Records)
	if err != nil {
		return errors.New("failed to write address refs store file")
	}

	err = jsonutils.WriteFile(d.ChainMetadataFilePath(), dataStoreConcrete.ChainMetadataStore.Records)
	if err != nil {
		return errors.New("failed to write chain metadata store file")
	}

	err = jsonutils.WriteFile(d.ContractMetadataFilePath(), dataStoreConcrete.ContractMetadataStore.Records)
	if err != nil {
		return errors.New("failed to write contract metadata store file")
	}

	err = jsonutils.WriteFile(d.EnvMetadataFilePath(), dataStoreConcrete.EnvMetadataStore.Record)
	if err != nil {
		return errors.New("failed to write environment datastore file")
	}

	return nil
}

// MergeMigrationAddressBook merges a migration's address book into an existing address book for
// the given domain environment. It reads the existing address book and the migration's address
// book, merges the latter into the former, and then writes the updated address book back to the
// domain environment address book.
func (d EnvDir) MergeMigrationAddressBook(migKey, timestamp string) error {
	addrBook, err := d.AddressBook()
	if err != nil {
		return err
	}

	// Get the artifacts directory for the environment
	artDir := d.ArtifactsDir()

	// Load the migration address book for the migration key and timestamp
	migAddrBook, err := loadAddressBookByMigrationKey(artDir, migKey, timestamp)
	if err != nil {
		return err
	}

	if err = addrBook.Merge(migAddrBook); err != nil {
		return err
	}

	addrs, err := addrBook.Addresses()
	if err != nil {
		return err
	}

	// Use marshalIndentAndSort to ensure deterministic output
	sortedBytes, err := marshalIndentAndSort(addrs)
	if err != nil {
		return err
	}

	return os.WriteFile(d.AddressBookFilePath(), sortedBytes, 0600)
}

// RemoveAddressBooks removes a migration's address book from an existing address book for a given
// domain environment. It reads the existing address book and the migration's address book, removes
// the latter from the former, and then writes the updated address book back to the domain
// environment address book.
//
// This can rollback MergeAddressBooks changes.
func (d EnvDir) RemoveMigrationAddressBook(migKey, timestamp string) error {
	addrBook, err := d.AddressBook()
	if err != nil {
		return err
	}

	// Get the artifacts directory for the environment
	artDir := d.ArtifactsDir()

	// Load the migration address book for the migration key and timestamp
	migAddrBook, err := loadAddressBookByMigrationKey(artDir, migKey, timestamp)
	if err != nil {
		return err
	}

	if err = addrBook.Remove(migAddrBook); err != nil {
		return err
	}

	addrs, err := addrBook.Addresses()
	if err != nil {
		return err
	}

	// Use marshalIndentAndSort to ensure deterministic output
	sortedBytes, err := marshalIndentAndSort(addrs)
	if err != nil {
		return err
	}

	return os.WriteFile(d.AddressBookFilePath(), sortedBytes, 0600)
}

// NodesFilePath returns the path to the file containing node information for the domain's
// environment directory.
func (d EnvDir) NodesFilePath() string {
	return filepath.Join(d.DirPath(), NodesFileName)
}

// LoadNodes loads the nodes in the domain's environment directory from a nodes.json file.
func (d EnvDir) LoadNodes() (*nodes.Nodes, error) {
	return nodes.LoadNodesFromFile(d.NodesFilePath())
}

// SaveNodes saves the node IDs to the domain's environment directory into a nodes.json file.
//
// If the file already exists, the new nodes will be merged with the existing nodes. If a node
// with the same id already exists, it will be overwritten.
func (d EnvDir) SaveNodes(nodeIDs []string) error {
	return nodes.SaveNodeIDsToFile(d.NodesFilePath(), nodeIDs)
}

// ViewStateFilePath returns the path to the file containing view state for the domain's
// environment directory.
func (d EnvDir) ViewStateFilePath() string {
	return filepath.Join(d.DirPath(), ViewStateFileName)
}

// JSONSerializer combines both marshaling and unmarshaling capabilities
type JSONSerializer interface {
	json.Marshaler
	json.Unmarshaler
}

// LoadViewState loads the view domain state as a json.Marshaler
// if the state file does not exist, it will return an empty json.Marshaler by default.
func (d EnvDir) LoadState() (JSONSerializer, error) {
	out := json.RawMessage([]byte("{}")) // default to empty json.Marshaler in case of file not found
	// Load the view state file
	b, err := os.ReadFile(d.ViewStateFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &out, fmt.Errorf("view state file does not exist: %w", err)
		}

		return &out, fmt.Errorf("failed to read view state file: %w", err)
	}
	out = json.RawMessage(b)

	return &out, nil
}

// InputsDirPath returns the path to the inputs directory for the domain's environment directory.
func (d EnvDir) InputsDirPath() string {
	return filepath.Join(d.DirPath(), "inputs")
}

// DurablePipelinesDirPath returns the path to the durable_pipelines directory within the domain's environment.
func (d EnvDir) DurablePipelinesDirPath() string {
	return filepath.Join(d.DirPath(), DurablePipelineDirName)
}

// DurablePipelinesInputsDirPath returns the path to the inputs directory within the durable_pipelines directory.
func (d EnvDir) DurablePipelinesInputsDirPath() string {
	return filepath.Join(d.DurablePipelinesDirPath(), DurablePipelineInputsDirName)
}

// DatastoreDirPath returns the path to the datastore directory within the domain's environment.
func (d EnvDir) DataStoreDirPath() string {
	return filepath.Join(d.DirPath(), DatastoreDirName)
}

// CreateDurablePipelinesDir creates the durable_pipelines directory and inputs subdirectory
// within the domain's environment directory if they don't exist. It also creates .gitkeep files
// within both directories to ensure they are tracked by git.
func (d EnvDir) CreateDurablePipelinesDir() error {
	// Create the durable_pipelines directory
	if err := fileutils.MkdirAllGitKeep(d.DurablePipelinesDirPath()); err != nil {
		return fmt.Errorf("failed to create durable_pipelines directory: %w", err)
	}

	// Create the inputs subdirectory
	if err := fileutils.MkdirAllGitKeep(d.DurablePipelinesInputsDirPath()); err != nil {
		return fmt.Errorf("failed to create durable_pipelines inputs directory: %w", err)
	}

	return nil
}

// SaveViewState saves the view state of the domain's environment with the default filename.
func (d EnvDir) SaveViewState(v json.Marshaler) error {
	return SaveViewState(d.ViewStateFilePath(), v)
}
