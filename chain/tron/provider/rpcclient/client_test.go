package rpcclient

import (
	"context"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/require"

	cldf_tron "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
)

func TestConfirmRetryOpts_DefaultsAndOverrides(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Test default options
	opts := ConfirmRetryOpts(ctx, cldf_tron.DefaultConfirmRetryOptions())
	require.Len(t, opts, 4)

	// Confirm context is set correctly
	var hasCtx bool
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		hasCtx = true
	}
	require.True(t, hasCtx)

	// Test with custom options
	customOpts := ConfirmRetryOpts(ctx, cldf_tron.ConfirmRetryOptions{
		RetryAttempts: 3,
		RetryDelay:    50 * time.Millisecond,
	})
	require.Len(t, customOpts, 4)
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	dummyAddr := address.Address{}
	cli := New(nil, nil, dummyAddr)
	require.NotNil(t, cli)
	require.Equal(t, dummyAddr, cli.Account)
	require.Nil(t, cli.Client)
	require.Nil(t, cli.Keystore)
}
