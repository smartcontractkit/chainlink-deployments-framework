package rpcclient

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestMatchErrorPolicy(t *testing.T) {
	t.Parallel()

	first := ErrorRetryPolicy{
		Match: func(err error) bool { return err != nil && err.Error() == "nonce too low" },
		Delay: time.Second,
	}
	second := ErrorRetryPolicy{
		Match: func(err error) bool { return err != nil && err.Error() == "no contract code" },
		Delay: 2 * time.Second,
	}

	tests := []struct {
		name      string
		policies  []ErrorRetryPolicy
		err       error
		wantOK    bool
		wantDelay time.Duration
	}{
		{
			name:     "returns false when no policies configured",
			policies: nil,
			err:      errors.New("nonce too low"),
			wantOK:   false,
		},
		{
			name:      "matches first policy",
			policies:  []ErrorRetryPolicy{first, second},
			err:       errors.New("nonce too low"),
			wantOK:    true,
			wantDelay: time.Second,
		},
		{
			name:      "matches second policy when first does not match",
			policies:  []ErrorRetryPolicy{first, second},
			err:       errors.New("no contract code"),
			wantOK:    true,
			wantDelay: 2 * time.Second,
		},
		{
			name: "skips nil matcher",
			policies: []ErrorRetryPolicy{
				{Match: nil, Delay: 5 * time.Second},
				second,
			},
			err:       errors.New("no contract code"),
			wantOK:    true,
			wantDelay: 2 * time.Second,
		},
		{
			name: "uses custom matcher",
			policies: []ErrorRetryPolicy{
				{
					Match: func(err error) bool { return err != nil && err.Error() == "custom" },
					Delay: 3 * time.Second,
				},
			},
			err:       errors.New("custom"),
			wantOK:    true,
			wantDelay: 3 * time.Second,
		},
		{
			name:     "returns false when no policy matches",
			policies: []ErrorRetryPolicy{first, second},
			err:      errors.New("unrelated"),
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			policy, ok := matchErrorPolicy(tt.err, tt.policies)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantDelay, policy.Delay)
			}
		})
	}
}

func TestRetryConfig_delayForError(t *testing.T) {
	t.Parallel()

	rc := RetryConfig{
		Delay: 10 * time.Millisecond,
		ErrorPolicies: []ErrorRetryPolicy{
			{
				Match: func(err error) bool { return err != nil && err.Error() == "nonce too low" },
				Delay: 2 * time.Second,
			},
		},
	}

	t.Run("returns policy delay for matched error", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 2*time.Second, rc.delayForError(1, errors.New("nonce too low"), nil))
	})

	t.Run("returns default delay for unmatched error", func(t *testing.T) {
		t.Parallel()

		var cfg retry.Config
		retry.Delay(10 * time.Millisecond)(&cfg)

		assert.Equal(t, 10*time.Millisecond, rc.delayForError(1, errors.New("connection reset"), &cfg))
	})
}

func TestWithErrorRetryPolicies_setsPoliciesOnMultiClient(t *testing.T) {
	t.Parallel()

	mc := &MultiClient{}
	policies := []ErrorRetryPolicy{
		{Match: func(error) bool { return true }, Delay: time.Second},
		{Match: func(error) bool { return false }, Delay: 2 * time.Second},
	}

	WithErrorRetryPolicies(policies...)(mc)
	assert.Equal(t, policies, mc.RetryConfig.ErrorPolicies)
}

func TestMultiClient_retryWithBackups_errorPolicyDelay(t *testing.T) {
	t.Parallel()

	const (
		defaultDelay = 5 * time.Millisecond
		policyDelay  = 100 * time.Millisecond
	)

	lggr := logger.Test(t)

	mc := MultiClient{
		Client:    nil,
		chainName: "ethereum-testnet-sepolia",
		RetryConfig: RetryConfig{
			Attempts: 2,
			Delay:    defaultDelay,
			Timeout:  3 * time.Second,
			ErrorPolicies: []ErrorRetryPolicy{
				{
					Match: func(err error) bool {
						return err != nil && strings.Contains(err.Error(), "nonce too low")
					},
					Delay: policyDelay,
				},
			},
		},
		lggr: lggr,
	}

	start := time.Now()
	err = mc.retryWithBackups(
		context.Background(),
		"test-operation",
		func(context.Context, *ethclient.Client) error {
			return errors.New("nonce too low")
		},
	)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.ErrorContains(t, err, "nonce too low")
	assert.GreaterOrEqual(t, elapsed, policyDelay, "expected at least one policy delay between retries")
}
