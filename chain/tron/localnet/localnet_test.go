package localnet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStartLocalNetwork verifies that the local Tron network can be started successfully.
func TestStartLocalNetwork(t *testing.T) {
	bc, accounts, err := StartLocalNetwork(t)
	require.NoError(t, err, "StartLocalNetwork should not return an error")
	require.NotNil(t, bc, "Blockchain output should not be nil")
	require.NotNil(t, accounts, "Accounts should not be nil")
	require.Greater(t, len(accounts.PrivateKeys), 0, "There should be at least one account")
}
