package lifecycle

import (
	"context"
	"time"
)

// UntilReady retries initialization until it succeeds or shutdown is requested.
// Each attempt receives a deadline so dependency outages cannot hang shutdown.
func UntilReady[T any](ctx context.Context, interval, timeout time.Duration, initialize func(context.Context) (T, error), onError func(error)) (T, error) {
	var zero T
	for {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		value, err := initialize(attemptCtx)
		cancel()
		if err == nil {
			return value, nil
		}
		if onError != nil {
			onError(err)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
}
