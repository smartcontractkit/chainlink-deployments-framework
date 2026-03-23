package contract

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestDeps_ApplyDefaults(t *testing.T) {
	t.Parallel()

	d := &Deps{}
	d.applyDefaults()

	require.NotNil(t, d.NetworkLoader)
	require.NotNil(t, d.DataStoreLoader)
}

func TestDefaultDataStoreLoader_FromLocal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	domainKey := "test"
	envKey := "staging"

	// Create env dir structure with datastore files (required for envdir.DataStore())
	envDirPath := filepath.Join(root, domainKey, envKey)
	datastoreDir := filepath.Join(envDirPath, "datastore")
	require.NoError(t, os.MkdirAll(datastoreDir, 0700))

	writeDatastoreFile(t, datastoreDir, "address_refs.json", "[]")
	writeDatastoreFile(t, datastoreDir, "chain_metadata.json", "[]")
	writeDatastoreFile(t, datastoreDir, "contract_metadata.json", "[]")
	writeDatastoreFile(t, datastoreDir, "env_metadata.json", "null")

	dom := domain.NewDomain(root, domainKey)
	envdir := dom.EnvDir(envKey)

	d := &Deps{}
	d.applyDefaults()

	ds, err := d.DataStoreLoader(context.Background(), envdir, logger.Nop(), DataStoreLoadOptions{FromLocal: true})
	require.NoError(t, err)
	require.NotNil(t, ds)
}

func TestDefaultDataStoreLoader_FromLocal_DataStoreError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dom := domain.NewDomain(root, "test")
	envdir := dom.EnvDir("staging")
	// No datastore files - envdir.DataStore() will fail

	d := &Deps{}
	d.applyDefaults()

	_, err := d.DataStoreLoader(context.Background(), envdir, logger.Nop(), DataStoreLoadOptions{FromLocal: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load datastore")
}

func TestDefaultDataStoreLoader_UseDomainConfig_FileType(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	domainKey := "test"
	envKey := "staging"

	// Full domain config structure
	require.NoError(t, os.MkdirAll(filepath.Join(root, domainKey, envKey, "datastore"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, domainKey, ".config", "networks"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, domainKey, ".config", "local"), 0700))

	writeDatastoreFile(t, filepath.Join(root, domainKey, envKey, "datastore"), "address_refs.json", "[]")
	writeDatastoreFile(t, filepath.Join(root, domainKey, envKey, "datastore"), "chain_metadata.json", "[]")
	writeDatastoreFile(t, filepath.Join(root, domainKey, envKey, "datastore"), "contract_metadata.json", "[]")
	writeDatastoreFile(t, filepath.Join(root, domainKey, envKey, "datastore"), "env_metadata.json", "null")

	// domain.yaml with datastore: file
	domainYaml := `
environments:
  staging:
    network_types: [mainnet]
    datastore: file
`
	require.NoError(t, os.WriteFile(filepath.Join(root, domainKey, ".config", "domain.yaml"), []byte(domainYaml), 0600))

	// networks yaml
	networksYaml := `
networks:
  - type: mainnet
    chain_selector: 5009297550715157269
    rpcs:
      - http_url: https://mainnet.example.com
`
	require.NoError(t, os.WriteFile(filepath.Join(root, domainKey, ".config", "networks", "mainnet.yaml"), []byte(networksYaml), 0600))

	// env config
	envConfig := `{}
`
	require.NoError(t, os.WriteFile(filepath.Join(root, domainKey, ".config", "local", "config.staging.yaml"), []byte(envConfig), 0600))

	dom := domain.NewDomain(root, domainKey)
	envdir := dom.EnvDir(envKey)

	d := &Deps{}
	d.applyDefaults()

	ds, err := d.DataStoreLoader(context.Background(), envdir, logger.Nop(), DataStoreLoadOptions{FromLocal: false})
	require.NoError(t, err)
	require.NotNil(t, ds)
}

func TestDefaultDataStoreLoader_ConfigLoadError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dom := domain.NewDomain(root, "nonexistent-domain")
	envdir := dom.EnvDir("staging")
	// No config files - config.Load will fail

	d := &Deps{}
	d.applyDefaults()

	_, err := d.DataStoreLoader(context.Background(), envdir, logger.Nop(), DataStoreLoadOptions{FromLocal: false})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load config")
}

func writeDatastoreFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0600))
}
