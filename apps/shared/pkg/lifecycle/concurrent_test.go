package lifecycle

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentProbesAreBoundedAndCancelQueuedWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, done := make(chan struct{}, 2), make(chan struct{})
	var active, calls atomic.Int32
	go func() {
		defer close(done)
		EachConcurrent(ctx, []int{1, 2, 3, 4, 5}, 2, func(ctx context.Context, _ int) {
			if active.Add(1) > 2 {
				t.Error("worker bound exceeded")
			}
			defer active.Add(-1)
			calls.Add(1)
			started <- struct{}{}
			<-ctx.Done()
		})
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("slow probe blocked another worker")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("workers did not stop")
	}
	if calls.Load() != 2 {
		t.Fatalf("cancelled queue ran %d probes", calls.Load())
	}
}
