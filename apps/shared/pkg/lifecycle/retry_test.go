package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUntilReadyRecoversAfterOutage(t *testing.T) {
	attempts := 0
	got, err := UntilReady(context.Background(), time.Millisecond, time.Second, func(ctx context.Context) (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("dependency unavailable")
		}
		return "ready", nil
	}, nil)
	if err != nil || got != "ready" || attempts != 3 {
		t.Fatalf("value=%s attempts=%d error=%v", got, attempts, err)
	}
}

func TestUntilReadyCancelsAttemptAndBackoff(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		_, err := UntilReady(ctx, time.Hour, time.Hour, func(ctx context.Context) (int, error) {
			if blocked {
				<-ctx.Done()
			}
			return 0, errors.New("offline")
		}, nil)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}
