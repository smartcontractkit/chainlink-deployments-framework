package rpcclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// Helper RPC server that always answers with a valid eth_blockNumber response
func newMockRPCServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a valid eth_blockNumber response
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	})

	return httptest.NewServer(handler)
}

// Helper RPC server that always answers with a JSON-RPC error payload
func newBadRPCServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Standard JSON-RPC error payload
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"internal error"}}`))
	})

	return httptest.NewServer(handler)
}

// TODO(giogam): This test is incomplete, it should be completed with support for websockets URLS
func TestMultiClient(t *testing.T) {
	t.Parallel()

	mockSrv := newMockRPCServer(t)
	defer mockSrv.Close()

	var (
		lggr                 = logger.Test(t)
		chainSelector uint64 = 16015286601757825753 // "ethereum-testnet-sepolia"
		wsURL                = ""                   // WS unused in this test
		httpURL              = mockSrv.URL          // use mock server for health-check
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
		{Name: "test-rpc", WSURL: wsURL, HTTPURL: httpURL, PreferredURLScheme: URLSchemePreferenceHTTP}, //preferred
		{Name: "test-rpc", WSURL: wsURL, HTTPURL: httpURL, PreferredURLScheme: URLSchemePreferenceHTTP}, //backup
	}})
	require.NoError(t, err)
	require.Len(t, mc.Backups, 1)
}

// Verifies that a bad eth_blockNumber response causes MultiClient to skip the
// first RPC and succeed with the next one.
func TestMultiClient_healthCheckSkipsBadRPC(t *testing.T) {
	t.Parallel()

	badSrv := newBadRPCServer(t)
	defer badSrv.Close()

	goodSrv := newMockRPCServer(t)
	defer goodSrv.Close()

	var (
		lggr                 = logger.Test(t)
		chainSelector uint64 = 16015286601757825753
	)

	mc, err := NewMultiClient(lggr, RPCConfig{ChainSelector: chainSelector, RPCs: []RPC{
		// first RPC -> health-check fails
		{Name: "bad-rpc", WSURL: "", HTTPURL: badSrv.URL, PreferredURLScheme: URLSchemePreferenceHTTP},
		// second RPC -> health-check passes
		{Name: "good-rpc", WSURL: "", HTTPURL: goodSrv.URL, PreferredURLScheme: URLSchemePreferenceHTTP},
	}})
	require.NoError(t, err)

	// Only the good RPC should remain (primary) and there should be no backups.
	require.NotNil(t, mc.Client)
	require.Empty(t, mc.Backups)

	// Sanity-check: calling BlockNumber on the surviving client should succeed.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	blockNum, err := mc.BlockNumber(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), blockNum)
}

func TestMultiClient_dialWithRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		URL       string
		retryConf RetryConfig
		wantErr   string
	}{
		{
			// this test case that triggers a context timeout error for all dial attempts.
			// without proper timeout the dial logic inside  will hang forever and the test
			// will timeout.
			name: "All dial attempts fail due to context timeout",
			URL:  "wss://rpcs.cldev.sh/avalanche/fuji",
			retryConf: RetryConfig{
				DialAttempts: 2,
				DialDelay:    10 * time.Millisecond,
				DialTimeout:  3 * time.Microsecond,
			},
			wantErr: "i/o timeout",
		},
		{
			name: "All dial attempts fail due to malformed URL",
			URL:  "wxz://malformed/avalanche/test",
			retryConf: RetryConfig{
				DialAttempts: 2,
				DialDelay:    10 * time.Millisecond,
				DialTimeout:  3 * time.Second,
			},
			wantErr: "no known transport for URL scheme \"wxz\"",
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
			require.Contains(t, err.Error(), tt.wantErr)
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
		wantErr   string
	}{
		{
			// This test simulates a consistently failing RPC call to exhaust the retry attempts.
			name: "All retry attempts fail due to failing RPC call",
			URL:  "http://rpcs.cldev.sh/avalanche/fuji",
			retryConf: RetryConfig{
				Attempts: 2,
				Delay:    10 * time.Millisecond,
				Timeout:  3 * time.Second,
			},
			call: func(ctx context.Context, client *ethclient.Client) error {
				return errors.New("operation failed")
			},
			wantErr: "operation failed",
		},
		{
			name: "All retry attempts fail due to context timeout",
			URL:  "http://rpcs.cldev.sh/avalanche/fuji",
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
			wantErr: "context deadline exceeded",
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
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestEnsureTimeout(t *testing.T) {
	t.Parallel()

	var (
		ctxNoTimeout           = context.Background()
		ctxWithTimeout, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	)

	defer cancel()

	tests := []struct {
		name          string
		parentContext context.Context //nolint:containedctx
		timeout       time.Duration
	}{
		{
			name:          "Parent context with deadline",
			parentContext: ctxWithTimeout,
			timeout:       1 * time.Minute,
		},
		{
			name:          "Parent context without deadline",
			parentContext: ctxNoTimeout,
			timeout:       1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancelFunc := ensureTimeout(tt.parentContext, tt.timeout)
			defer cancelFunc()

			deadline, hasDeadline := ctx.Deadline()
			require.True(t, hasDeadline, "Expected context to have a deadline")

			if parentDeadline, hasParentDeadline := tt.parentContext.Deadline(); hasParentDeadline {
				require.WithinDuration(t, parentDeadline, deadline, 0, "Deadline should match parent's deadline")
			} else {
				require.WithinDuration(t, time.Now().Add(tt.timeout), deadline, 50*time.Millisecond, "Deadline should be approximately the specified timeout")
			}
		})
	}
}
func TestMultiClient_reorderRPCs(t *testing.T) {
	t.Parallel()

	// Create some test clients with different memory addresses for identification
	client0 := ethclient.NewClient(nil) // primary
	client1 := ethclient.NewClient(nil) // backup 0
	client2 := ethclient.NewClient(nil) // backup 1
	client3 := ethclient.NewClient(nil) // backup 2

	rpcClients := []*ethclient.Client{
		client1, // backup 0
		client2, // backup 1
		client3, // backup 2
	}

	tests := []struct {
		name                string
		backups             []*ethclient.Client
		newDefaultClientIdx int
		expectedClient      *ethclient.Client
		expectedBackups     []*ethclient.Client
	}{
		{
			name:                "Move first backup to primary",
			backups:             rpcClients,
			newDefaultClientIdx: 1,
			expectedClient:      client1,
			expectedBackups: []*ethclient.Client{
				client2,
				client3,
				client0,
			},
		},
		{
			name:                "Move middle backup to primary",
			backups:             rpcClients,
			newDefaultClientIdx: 2,
			expectedClient:      client2,
			expectedBackups: []*ethclient.Client{
				client3,
				client1,
				client0,
			},
		},
		{
			name:                "Move last backup to primary",
			backups:             rpcClients,
			newDefaultClientIdx: 3,
			expectedClient:      client3,
			expectedBackups: []*ethclient.Client{
				client1,
				client2,
				client0,
			},
		},
		{
			name:                "Keep primary unchanged",
			backups:             rpcClients,
			newDefaultClientIdx: 0,
			expectedClient:      client0,
			expectedBackups: []*ethclient.Client{
				client1,
				client2,
				client3,
			},
		},
		{
			name:                "Keep primary unchanged when no backups",
			backups:             []*ethclient.Client{},
			newDefaultClientIdx: 1,
			expectedClient:      client0,
			expectedBackups:     []*ethclient.Client{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &MultiClient{
				Client:  client0,
				Backups: tt.backups,
				lggr:    logger.Test(t),
			}

			// Call the method being tested
			mc.reorderRPCs(tt.newDefaultClientIdx)

			// Verify the results
			assert.Same(t, tt.expectedClient, mc.Client, "Primary client should be the selected backup")
			require.Len(t, mc.Backups, len(tt.expectedBackups), "Backup count should remain the same")

			// Check that backups are in the expected order
			for i, expected := range tt.expectedBackups {
				assert.Same(t, expected, mc.Backups[i],
					"Backup at position %d should be as expected", i)
			}
		})
	}
}
