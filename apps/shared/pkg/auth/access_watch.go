package auth

import (
	"context"
	"time"
)

// WatchAccess closes otherwise idle connections when their principal or resource
// is restricted. The returned cleanup waits for the guard to exit.
func WatchAccess(ctx context.Context, check func(context.Context) error, revoke func()) func() {
	return watchAccess(ctx, time.Second, check, revoke)
}

func watchAccess(ctx context.Context, interval time.Duration, check func(context.Context) error, revoke func()) func() {
	guardCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-guardCtx.Done():
				return
			case <-ticker.C:
				checkCtx, checkCancel := context.WithTimeout(guardCtx, 3*time.Second)
				err := check(checkCtx)
				checkCancel()
				if err != nil {
					if guardCtx.Err() == nil {
						revoke()
					}
					return
				}
			}
		}
	}()
	return func() { cancel(); <-done }
}
