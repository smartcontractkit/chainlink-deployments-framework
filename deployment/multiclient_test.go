package deployment

import (
	"context"
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
		name      string
		URL       string
		retryConf RetryConfig
		call      func(ctx context.Context, client *ethclient.Client) error
	}{
		{
			// this test case triggers a context timeout error for all dial attempts.
			// without proper timeout the dial logic inside  will hang forever and the test
			// will timeout.
			name: "All dial attempts fail due to context timeout",
			URL:  "http://rpcs.cldev.sh/avalanche/test",
			retryConf: RetryConfig{
				Attempts: 2,
				Delay:    10 * time.Millisecond,
				Timeout:  3 * time.Second,
			},
			call: func(ctx context.Context, client *ethclient.Client) error {
				return errors.New("operation failed")
			},
		},
		{
			name: "All dial attempts fail due to malformed URL",
			URL:  "http://rpcs.cldev.sh/avalanche/test",
			retryConf: RetryConfig{
				Attempts: 2,
				Delay:    10 * time.Millisecond,
				Timeout:  3 * time.Second,
			},
			call: func(ctx context.Context, client *ethclient.Client) error {
				// Simulate a long-running operation that will cause the context to timeout
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Second):
					return errors.New("operation failed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lggr := logger.Test(t)

			client, err := ethclient.Dial(tt.URL)
			require.NoError(t, err)

			mc := MultiClient{
				Client:      client,
				chainName:   "ethereum-testnet-sepolia",
				RetryConfig: tt.retryConf,
				lggr:        lggr,
			}

			err = mc.retryWithBackups(
				context.Background(),
				"test-operation",
				tt.call,
			)

			require.Error(t, err)
		})
	}
}
