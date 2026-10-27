package lifecycle

import (
	"context"
	"sync"
)

// EachConcurrent bounds background probes so a slow resource does not serialize
// the entire fleet. Cancellation stops queued work and reaches active probes.
func EachConcurrent[T any](ctx context.Context, items []T, workers int, run func(context.Context, T)) {
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan T)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				if ctx.Err() != nil {
					return
				}
				run(ctx, item)
			}
		}()
	}
	for _, item := range items {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case jobs <- item:
		}
	}
	close(jobs)
	wg.Wait()
}
