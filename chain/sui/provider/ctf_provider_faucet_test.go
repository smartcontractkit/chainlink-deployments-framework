package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_fundAccount_Success verifies a ready faucet funds the account on the
// first attempt.
func Test_fundAccount_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	require.NoError(t, fundAccount(t.Context(), srv.URL, "0xabc"))
}

// Test_fundAccount_RetryableThenSuccess verifies that a transient 5xx is retried
// and funding eventually succeeds once the faucet becomes ready.
func Test_fundAccount_RetryableThenSuccess(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable) // faucet warming up
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	require.NoError(t, fundAccount(t.Context(), srv.URL, "0xabc"))
	assert.GreaterOrEqual(t, calls.Load(), int64(2), "5xx must be retried")
}

// Test_fundAccount_TooManyRequests_Retried verifies 429 is treated as retryable.
func Test_fundAccount_TooManyRequests_Retried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	require.NoError(t, fundAccount(t.Context(), srv.URL, "0xabc"))
	assert.GreaterOrEqual(t, calls.Load(), int64(2), "429 must be retried")
}

// Test_fundAccount_NonRetryableClientError_StopsImmediately verifies a 4xx
// client error (other than 429) is not retried, so it can't burn the budget.
func Test_fundAccount_NonRetryableClientError_StopsImmediately(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "bad recipient", http.StatusBadRequest)
	}))
	defer srv.Close()

	err := fundAccount(t.Context(), srv.URL, "0xabc")
	require.Error(t, err)
	require.ErrorContains(t, err, "fund account via Sui faucet")
	require.ErrorContains(t, err, "faucet returned status 400")
	require.ErrorContains(t, err, "bad recipient") // response body surfaced for debugging
	assert.Equal(t, int64(1), calls.Load(), "non-retryable 4xx must not be retried")
}

// Test_fundAccount_ContextDeadline_BoundsHangingServer verifies the total
// context timeout bounds a faucet that hangs at the TCP/HTTP layer, returning
// promptly instead of blocking indefinitely / burning all attempts.
func Test_fundAccount_ContextDeadline_BoundsHangingServer(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		// Hold the connection open past the caller's deadline so the request
		// can only complete via context cancellation. The wait is short so
		// srv.Close does not stall teardown if the server-side request context
		// is not cancelled promptly when the client abandons the call.
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	// Tighten the total budget via the caller's context; the child created
	// inside fundAccount inherits the earlier deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := fundAccount(ctx, srv.URL, "0xabc")
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Less(t, elapsed, 3*time.Second, "must stop shortly after the deadline, not burn all attempts")
}

// Test_isRetryableFaucetErr covers the error classification directly.
func Test_isRetryableFaucetErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline exceeded", context.DeadlineExceeded, false},
		{"500 internal server error", &faucetStatusError{status: http.StatusInternalServerError, body: []byte("boom")}, true},
		{"502 bad gateway", &faucetStatusError{status: http.StatusBadGateway}, true},
		{"503 service unavailable", &faucetStatusError{status: http.StatusServiceUnavailable}, true},
		{"429 too many requests", &faucetStatusError{status: http.StatusTooManyRequests}, true},
		{"400 bad request", &faucetStatusError{status: http.StatusBadRequest, body: []byte("bad recipient")}, false},
		{"401 unauthorized", &faucetStatusError{status: http.StatusUnauthorized}, false},
		{"403 forbidden", &faucetStatusError{status: http.StatusForbidden}, false},
		{"404 not found", &faucetStatusError{status: http.StatusNotFound}, false},
		{"422 unprocessable entity", &faucetStatusError{status: http.StatusUnprocessableEntity}, false},
		{"generic network error", errors.New("connection refused"), true},
		{"transport reset", errors.New("read tcp: connection reset by peer"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isRetryableFaucetErr(tt.err))
		})
	}
}

// Test_faucetStatusError_Error verifies the body is surfaced for debugging and
// omitted when empty.
func Test_faucetStatusError_Error(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"faucet returned status 400: bad recipient",
		(&faucetStatusError{status: 400, body: []byte("bad recipient")}).Error(),
	)
	assert.Equal(t,
		"faucet returned status 400: bad recipient",
		(&faucetStatusError{status: 400, body: []byte("  bad recipient\n")}).Error(),
	)
	assert.Equal(t,
		"faucet returned status 503",
		(&faucetStatusError{status: 503}).Error(),
	)
}
