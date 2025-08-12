package domain

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// GetDomain returns a Domain for the specified key based on the available dirs in the domains root.
// If the key is not recognized, it will return an error
func GetDomain(key string) (Domain, error) {
	entries, err := os.ReadDir(filepath.Join(ProjectRoot, "domains"))
	if err != nil {
		return Domain{}, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == key {
			return NewDomain(DomainsRoot, key), nil
		}
	}

	return Domain{}, fmt.Errorf("domain not found: %s", key)
}

// MustGetDomain returns a Domain for the specified key. If the key is not recognized, it will
// panic.
func MustGetDomain(key string) Domain {
	d, err := GetDomain(key)
	if err != nil {
		panic(err)
	}

	return d
}

// Domain represents a specific domain that is operated by a team. Each domain corresponds to a
// team's ownership of a set of migrations that span multiple environments.
type Domain struct {
	// rootPath is absolute path of the domains filesystem. e.g. if the domain directory is in the
	// project root directory, this would be the project root directory.
	rootPath string
	// key is the name of the domain. e.g. "ccip", "keystone"
	key string
}

// NewDomain creates a new Domain.
func NewDomain(rootPath, key string) Domain {
	return Domain{
		rootPath: rootPath,
		key:      key,
	}
}

// DirPath returns the path to the domain directory.
func (d Domain) DirPath() string {
	return filepath.Join(d.rootPath, d.key)
}

// String returns the key of the domain.
func (d Domain) String() string {
	return d.key
}

// Key returns the key of the domain.
func (d Domain) Key() string {
	return d.key
}

// EnvDir returns a DomainEnv for the specified environment.
func (d Domain) EnvDir(env string) EnvDir {
	return NewEnvDir(d.rootPath, d.key, env)
}

// AddressBookByEnv returns the address book for the specified environment.
func (d Domain) AddressBookByEnv(env string) (cldf.AddressBook, error) {
	return d.EnvDir(env).AddressBook()
}

// DataStoreByEnv returns the datastore for the specified environment.
func (d Domain) DataStoreByEnv(env string) (datastore.DataStore, error) {
	return d.EnvDir(env).DataStore()
}

// ArtifactsDirByEnv returns the artifacts directory for the specified environment.
func (d Domain) ArtifactsDirByEnv(env string) *ArtifactsDir {
	return d.EnvDir(env).ArtifactsDir()
}

// LibDirPath returns the path to the lib directory within the domain. This is where shared
// libraries and packages should be placed.
func (d Domain) LibDirPath() string {
	return filepath.Join(d.DirPath(), LibDirName)
}

// InternalDirPath returns the path to the internal directory within the domain. This is where
// internal packages should be placed.
func (d Domain) InternalDirPath() string {
	return filepath.Join(d.DirPath(), InternalDirName)
}

// CmdDirPath returns the path to the cmd directory within the domain. This is where the domain's
// command line tools should be placed.
func (d Domain) CmdDirPath() string {
	return filepath.Join(d.DirPath(), CmdDirName)
}

// todo: uncomment after viewState is moved over to cldf
// SaveViewState saves the view state of the domain's environment with the default filename.
// func (d EnvDir) SaveViewState(v json.Marshaler) error {
// 	return SaveViewState(d.ViewStateFilePath(), v)
// }
