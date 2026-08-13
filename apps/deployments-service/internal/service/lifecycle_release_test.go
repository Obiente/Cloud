package deployments

import (
	"context"
	"testing"
)

func TestWaitForDeploymentBuildReleaseContinuesAfterInitialTimeout(t *testing.T) {
	initialCtx, initialCancel := context.WithCancel(context.Background())
	initialCancel()
	continuationCtx, continuationCancel := context.WithCancel(context.Background())
	defer continuationCancel()

	attempts := 0
	released := waitForDeploymentBuildReleaseWithContinuation(initialCtx, continuationCtx, func(ctx context.Context) bool {
		attempts++
		if attempts == 1 {
			if ctx.Err() == nil {
				t.Fatal("initial release attempt did not receive its expired context")
			}
			return false
		}
		if ctx.Err() != nil {
			t.Fatalf("continuation release context was already canceled: %v", ctx.Err())
		}
		return true
	})

	if !released {
		t.Fatal("release did not continue to a successful retry")
	}
	if attempts != 2 {
		t.Fatalf("release attempts = %d, want 2", attempts)
	}
}
