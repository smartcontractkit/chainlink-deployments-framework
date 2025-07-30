package localnet

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/require"
)

// CfgTron holds the configuration for the Tron local network.
// It defines the required blockchain input, loaded from a TOML config.
type CfgTron struct {
	BlockchainTron *blockchain.Input `toml:"blockchain_tron" validate:"required"`
}

// StartLocalNetwork starts a local Tron network for testing purposes.
//
// It loads the network configuration using the Chainlink Testing Framework (CTF),
// initializes the blockchain network, and returns the created blockchain instance
// along with predefined Tron accounts for use in tests.
//
// This is intended to be reused across multiple test files to ensure consistent
// setup of the local Tron environment.
func StartLocalNetwork(t *testing.T) (*blockchain.Output, blockchain.Accounts, error) {
	gitRoot, err := findGitRoot()
	require.NoError(t, err, "Failed to find Git repository root")

	configPath := filepath.Join(gitRoot, "chain/tron/localnet/ctf-config.toml")
	os.Setenv("CTF_CONFIGS", configPath)

	// Load the Tron config from the test environment
	in, err := framework.Load[CfgTron](t)
	require.NoError(t, err, "Failed to load configuration")

	// Use the default test accounts for Tron
	accounts := blockchain.TRONAccounts

	// Create and start the blockchain network based on the loaded config
	bc, err := blockchain.NewBlockchainNetwork(in.BlockchainTron)
	require.NoError(t, err, "Failed to create blockchain network")

	return bc, accounts, nil
}

func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return filepath.Abs(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no .git directory found in %s or its parents", dir)
}
