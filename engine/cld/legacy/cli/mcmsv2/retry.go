package mcmsv2

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
)

var (
	defaultRetryMinDelay = 500 * time.Millisecond
	retryContextTimeout  = 30 * time.Second
	defaultRetryOpts     = func(ctx context.Context) []retry.Option {
		retryIncrementalDelays := getRetryDelays()
		return []retry.Option{
			retry.Context(ctx),
			retry.DelayType(func(n uint, _ error, config *retry.Config) time.Duration {
				return time.Duration(retryIncrementalDelays[min(int(n), len(retryIncrementalDelays)-1)]) * time.Millisecond
			}),
			retry.Delay(defaultRetryMinDelay),
			retry.Attempts(uint(len(retryIncrementalDelays) + 1)),
			retry.LastErrorOnly(true),
			// retry.OnRetry(func(attempt uint, err error) {
			// 	fmt.Printf("RETRYING: %d, %s\n", attempt, err)
			// }),
		}
	}
)

type retryCallback[T any] func(ctx context.Context) (T, error)

func Retry[T any](ctx context.Context, callback retryCallback[T], opts ...retry.Option) (T, error) {
	var returnValue T
	var err error

	err = retry.Do(func() error {
		rctx, cancel := context.WithTimeout(ctx, retryContextTimeout)
		defer cancel()

		returnValue, err = callback(rctx)

		return err
	}, append(defaultRetryOpts(ctx), opts...)...)

	return returnValue, err
}

func getRetryDelays() []int {
	defaultRetryDelays := []int{500, 2000, 8000}

	retryDelaysEnvVar := os.Getenv("CLD_MCMS_RETRY_DELAYS")
	retryDelaysStrSlice := strings.Split(retryDelaysEnvVar, ",")
	retryDelays := []int{}
	for _, d := range retryDelaysStrSlice {
		retryDelay, err := strconv.Atoi(d)
		if err != nil {
			return defaultRetryDelays
		}

		retryDelays = append(retryDelays, retryDelay)
	}

	return retryDelays
}
