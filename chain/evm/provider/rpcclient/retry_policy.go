package rpcclient

import (
	"time"

	"github.com/avast/retry-go/v4"
)

// ErrorMatcher returns true when the error should use a policy's retry delay.
type ErrorMatcher func(error) bool

// ErrorRetryPolicy configures a longer wait before retrying when Match returns true.
type ErrorRetryPolicy struct {
	Match ErrorMatcher
	Delay time.Duration
}

// WithErrorRetryPolicies configures error-specific retry delays on a MultiClient.
func WithErrorRetryPolicies(policies ...ErrorRetryPolicy) func(*MultiClient) {
	return func(mc *MultiClient) {
		mc.RetryConfig.ErrorPolicies = policies
	}
}

func matchErrorPolicy(err error, policies []ErrorRetryPolicy) (ErrorRetryPolicy, bool) {
	for _, policy := range policies {
		if policy.Match != nil && policy.Match(err) {
			return policy, true
		}
	}

	return ErrorRetryPolicy{}, false
}

func (rc RetryConfig) rpcRetryOptions() []retry.Option {
	opts := []retry.Option{
		retry.Attempts(rc.Attempts),
		retry.Delay(rc.Delay),
		retry.DelayType(rc.delayForError),
	}

	return opts
}

func (rc RetryConfig) delayForError(n uint, err error, cfg *retry.Config) time.Duration {
	if policy, ok := matchErrorPolicy(err, rc.ErrorPolicies); ok {
		return policy.Delay
	}

	return retry.FixedDelay(n, err, cfg)
}
