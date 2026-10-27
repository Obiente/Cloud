package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type recoveryInspectClient struct {
	client.APIClient
	result container.InspectResponse
	err    error
}

func (c *recoveryInspectClient) ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{Container: c.result}, c.err
}

func TestRecoveryPreservesExistingContainerAndFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name       string
		owner      string
		inspectErr error
		wantErr    bool
	}{
		{"healthy replica", "dep", nil, false}, {"other tenant", "other", nil, true}, {"daemon unavailable", "dep", errors.New("Docker unavailable"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &recoveryInspectClient{result: container.InspectResponse{ID: "existing", Config: &container.Config{Labels: map[string]string{"cloud.obiente.managed": "true", "cloud.obiente.deployment_id": tc.owner}}}, err: tc.inspectErr}
			manager := &DeploymentManager{dockerClient: c}
			id, changed, err := manager.createContainer(context.Background(), &DeploymentConfig{DeploymentID: "dep", RecoveryOnly: true}, "dep-default-replica-0", 0, "default", nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
			if changed {
				t.Fatal("recovery replaced an existing container")
			}
			if !tc.wantErr && id != "existing" {
				t.Fatalf("ID=%s", id)
			}
		})
	}
}

func TestRuntimeLockSerializesAndCancelsWaitingOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := withDeploymentRuntimeLock(ctx, "lock-fixture", func(locked context.Context) error {
		if err := withDeploymentRuntimeLock(locked, "lock-fixture", func(context.Context) error { return nil }); err != nil {
			t.Fatal(err)
		}
		short, stop := context.WithTimeout(ctx, 20*time.Millisecond)
		defer stop()
		if err := withDeploymentRuntimeLock(short, "lock-fixture", func(context.Context) error { t.Error("concurrent lifecycle operation entered"); return nil }); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("waiter err=%v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := withDeploymentRuntimeLock(ctx, "lock-fixture", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
}
