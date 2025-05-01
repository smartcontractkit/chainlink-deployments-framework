package deployment

import (
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
)

// TODO(giogam): This test is incomplete, it should be completed with support for websockets URLS
func TestMultiClient(t *testing.T) {
	var (
		lggr                 = logger.Test(t)
		chainSelector uint64 = 16015286601757825753 // "ethereum-testnet-sepolia"
		wsURL                = "ws://example.com"
		httpURL              = "http://example.com"
	)

	// Expect defaults to be set if not provided.
	mc, err := NewMultiClient(lggr, RPCConfig{ChainSelector: chainSelector, RPCs: []RPC{
		{Name: "test-rpc", WSURL: wsURL, HTTPURL: httpURL, PreferredURLScheme: URLSchemePreferenceHTTP},
	}})

	require.NoError(t, err)
	require.NotNil(t, mc)

	assert.Equal(t, "ethereum-testnet-sepolia", mc.chainName)
	assert.Equal(t, mc.RetryConfig.Attempts, uint(RPCDefaultRetryAttempts))
	assert.Equal(t, RPCDefaultRetryDelay, mc.RetryConfig.Delay)

	// Expect error if no RPCs provided.
	_, err = NewMultiClient(lggr, RPCConfig{ChainSelector: chainSelector, RPCs: []RPC{}})
	require.Error(t, err)

	// Expect second client to be set as backup.
	mc, err = NewMultiClient(lggr, RPCConfig{ChainSelector: chainSelector, RPCs: []RPC{
		{Name: "test-rpc", WSURL: wsURL, HTTPURL: httpURL, PreferredURLScheme: URLSchemePreferenceHTTP},
		{Name: "test-rpc", WSURL: wsURL, HTTPURL: httpURL, PreferredURLScheme: URLSchemePreferenceHTTP},
	}})
	require.NoError(t, err)
	require.Len(t, mc.Backups, 1)
}

func TestMultiClient_retryWithBackups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		URL        string
		retryDelay time.Duration
		opName     string
		op         func(client *ethclient.Client) error
		wantErr    string
	}{
		{
			name:       "All retries fail with http",
			URL:        "http://example.com",
			retryDelay: 100,
			opName:     "test-operation",
			op: func(client *ethclient.Client) error {
				return errors.New("operation failed")
			},
			wantErr: "operation failed\nall backup clients failed for chain ethereum-testnet-sepolia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lggr := logger.Test(t)
			chainSelector := uint64(16015286601757825753) // "ethereum-testnet-sepolia"

			// Create MultiClient with retry configuration
			mc, err := NewMultiClient(lggr, RPCConfig{
				ChainSelector: chainSelector,
				RPCs: []RPC{
					{Name: "test-rpc", HTTPURL: tt.URL, PreferredURLScheme: URLSchemePreferenceHTTP},
					{Name: "test-rpc-2", HTTPURL: tt.URL, PreferredURLScheme: URLSchemePreferenceHTTP},
				},
			})

			require.NoError(t, err)
			require.NotNil(t, mc)

			mc.RetryConfig.Delay = tt.retryDelay

			// Run operation and check expectations
			err = mc.retryWithBackups(tt.opName, tt.op)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
