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
	t.Parallel()
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
	assert.Equal(t, uint(RPCDefaultRetryAttempts), mc.RetryConfig.Attempts)
	assert.Equal(t, RPCDefaultRetryDelay, mc.RetryConfig.Delay)
	assert.Equal(t, uint(RPCDefaultDialRetryAttempts), mc.RetryConfig.DialAttempts)
	assert.Equal(t, RPCDefaultDialRetryDelay, mc.RetryConfig.DialDelay)

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

func TestMultiClient_dialWithRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		URL       string
		retryConf RetryConfig
	}{
		{
			// this test case that triggers a context timeout error for all dial attempts.
			// without proper timeout the dial logic inside  will hang forever and the test
			// will timeout.
			name: "All dial attempts fail due to context timeout",
			URL:  "wss://rpcs.cldev.sh/avalanche/test",
			retryConf: RetryConfig{
				DialAttempts: 2,
				DialDelay:    10 * time.Millisecond,
				DialTimeout:  3 * time.Second,
			},
		},
		{
			name: "All dial attempts fail due to malformed URL",
			URL:  "wxz://malformed/avalanche/test",
			retryConf: RetryConfig{
				DialAttempts: 2,
				DialDelay:    10 * time.Millisecond,
				DialTimeout:  3 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lggr := logger.Test(t)

			mc := MultiClient{
				chainName:   "ethereum-testnet-sepolia",
				RetryConfig: tt.retryConf,
				lggr:        lggr,
			}

			_, err := mc.dialWithRetry(RPC{
				Name:               "test-rpc",
				WSURL:              tt.URL,
				PreferredURLScheme: URLSchemePreferenceWS,
			}, lggr)

			require.Error(t, err)
		})
	}
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
			wantErr: "operation failed\nall backup clients failed for chain \"ethereum-testnet-sepolia\"",
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
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
