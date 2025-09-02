package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

var (
	// Directory permissions for test setup
	dirPerms = os.FileMode(0700)
	// File permissions for test setup
	filePerms = os.FileMode(0600)
)

// Network fixtures which map exactly to the network data in testdata files.
// Used for asserting against the expected network configuration.
var (
	testnetNetwork1 = config_network.Network{
		Type:          "testnet",
		ChainSelector: 16015286601757825753,
		RPCs: []config_network.RPC{
			{
				RPCName:            "sepolia-rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://sepolia.infura.io/v3/test",
				WSURL:              "wss://sepolia.infura.io/ws/v3/test",
			},
		},
	}

	testnetNetwork2 = config_network.Network{
		Type:          "testnet",
		ChainSelector: 12532609583862916517,
		RPCs: []config_network.RPC{
			{
				RPCName:            "mumbai-rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://polygon-mumbai.infura.io/v3/test",
				WSURL:              "wss://polygon-mumbai.infura.io/ws/v3/test",
			},
		},
	}

	mainnetNetwork1 = config_network.Network{
		Type:          "mainnet",
		ChainSelector: 5009297550715157269,
		RPCs: []config_network.RPC{
			{
				RPCName:            "mainnet-rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://mainnet.infura.io/v3/test",
				WSURL:              "wss://mainnet.infura.io/ws/v3/test",
			},
		},
	}
)

// setupConfigDirs sets up a minimal domain structure with a .config directory structure and returns
// the domain and environment key.
func setupConfigDirs(t *testing.T) (domain.Domain, string) {
	t.Helper()

	// Create a temporary directory structure for testing
	rootDir := t.TempDir()
	domainKey := "test-domain"
	envKey := "staging_testnet"

	// Set up minimal domain structure
	domainDir := filepath.Join(rootDir, domainKey)
	require.NoError(t, os.MkdirAll(domainDir, dirPerms))

	// Create .config directory structure
	configDir := filepath.Join(domainDir, ".config")
	networksDir := filepath.Join(configDir, "networks")
	require.NoError(t, os.MkdirAll(networksDir, dirPerms))

	// Create local directory
	localDir := filepath.Join(configDir, "local")
	require.NoError(t, os.MkdirAll(localDir, dirPerms))

	return domain.NewDomain(rootDir, domainKey), envKey
}

// writeConfigLocalFile writes a config file to the domain's local directory with testdata.
func writeConfigLocalFile(t *testing.T, dom domain.Domain, envKey, testdataFileName string) {
	t.Helper()

	// Create .config.testnet.yaml file for the new config format
	input, err := os.ReadFile(filepath.Join("testdata", testdataFileName))
	require.NoError(t, err)

	err = os.WriteFile(dom.ConfigLocalFilePath(envKey), input, filePerms)
	require.NoError(t, err)
}

// writeConfigNetworksFile writes a config file to the domain's networks directory with testdata.
func writeConfigNetworksFile(t *testing.T, dom domain.Domain, filename, testdataFileName string) {
	t.Helper()

	// Create network configuration file
	input, err := os.ReadFile(filepath.Join("testdata", testdataFileName))
	require.NoError(t, err)

	err = os.WriteFile(dom.ConfigNetworksFilePath(filename), input, filePerms)
	require.NoError(t, err)
}

// writeConfigDomainFile writes a config file to the domain's directory with testdata.
func writeConfigDomainFile(t *testing.T, dom domain.Domain, testdataFileName string) { //nolint:unparam // testdataFileName could be different for any future tests
	t.Helper()

	// Create network configuration file
	input, err := os.ReadFile(filepath.Join("testdata", testdataFileName))
	require.NoError(t, err)

	err = os.WriteFile(dom.ConfigDomainFilePath(), input, filePerms)
	require.NoError(t, err)
}
