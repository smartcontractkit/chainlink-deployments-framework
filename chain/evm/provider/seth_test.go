// Contains the test setup for the Seth provider in the EVM chain package.
package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// sethTOML is a valid TOML configuration for the Seth client to be used in tests.
const sethTOML = `
artifacts_dir = "artifacts"

[nonce_manager]
key_sync_rate_limit_per_sec = 10
key_sync_timeout = "20s"
key_sync_retry_delay = "1s"
key_sync_retries = 10

[[networks]]
name = "Test"
chain_id = 1000
dial_timeout="1m"
transaction_timeout = "5m"
eip_1559_dynamic_fees = true

# automated gas estimation
gas_price_estimation_enabled = true
gas_price_estimation_blocks = 20
gas_price_estimation_tx_priority = "standard"

# gas limits
transfer_gas_fee = 21_000
# gas limit should be explicitly set only if you are connecting to a node that's incapable of estimating gas limit itself (should only happen for very old versions)
# gas_limit = 14_000_000

# manual settings, used when gas_price_estimation_enabled is false or when it fails
# legacy transactions
gas_price = 150_000_000_000   #150 gwei
# EIP-1559 transactions
gas_fee_cap = 150_000_000_000 #150 gwei
gas_tip_cap = 100_000_000_000  #100 gwei
`

// writeSethConfigFile creates a temporary seth configuration file for testing purposes.
func writeSethConfigFile(t *testing.T) string {
	t.Helper()

	var (
		// Create a temporary directory for the test
		tmpDir = t.TempDir()
		// Define the path for the config file
		configPath = filepath.Join(tmpDir, "seth.toml")
	)

	err := os.WriteFile(configPath, []byte(sethTOML), 0600)
	require.NoError(t, err)

	return configPath
}

// writeInvalidSethConfigFile creates a temporary invalid seth configuration file for testing error handling.
func writeInvalidSethConfigFile(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bad.toml")
	invalidConfigContent := `not a valid toml`
	err := os.WriteFile(configPath, []byte(invalidConfigContent), 0600)
	require.NoError(t, err)

	return configPath
}
