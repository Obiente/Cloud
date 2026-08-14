package registry

import (
	"context"
	"testing"

	"github.com/moby/moby/api/types/swarm"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStableSwarmTaskSlot(t *testing.T) {
	tests := []struct {
		name string
		mode swarm.ServiceMode
		task swarm.Task
		want string
	}{
		{
			name: "replicated slot",
			mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{}},
			task: swarm.Task{Slot: 3, NodeID: "worker-one"},
			want: "3",
		},
		{
			name: "global node",
			mode: swarm.ServiceMode{Global: &swarm.GlobalService{}},
			task: swarm.Task{NodeID: "worker-two"},
			want: "worker-two",
		},
		{
			name: "missing stable identity",
			mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{}},
			task: swarm.Task{NodeID: "worker-three"},
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := stableSwarmTaskSlot(test.mode, test.task); got != test.want {
				t.Fatalf("stableSwarmTaskSlot() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestShouldRecordSwarmTaskPrefersRunningReplacement(t *testing.T) {
	containerStatus := &swarm.ContainerStatus{ContainerID: "replacement-container"}
	tests := []struct {
		name string
		task swarm.Task
		want bool
	}{
		{
			name: "running replacement",
			task: swarm.Task{
				DesiredState: swarm.TaskStateRunning,
				Status:       swarm.TaskStatus{State: swarm.TaskStateRunning, ContainerStatus: containerStatus},
			},
			want: true,
		},
		{
			name: "old running task marked for shutdown",
			task: swarm.Task{
				DesiredState: swarm.TaskStateShutdown,
				Status:       swarm.TaskStatus{State: swarm.TaskStateRunning, ContainerStatus: containerStatus},
			},
			want: false,
		},
		{
			name: "replacement without container",
			task: swarm.Task{
				DesiredState: swarm.TaskStateRunning,
				Status:       swarm.TaskStatus{State: swarm.TaskStatePreparing},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldRecordSwarmTask(test.task); got != test.want {
				t.Fatalf("shouldRecordSwarmTask() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestUnregisterDeploymentIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:registry-unregister-idempotent?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open registry test database: %v", err)
	}
	if err := db.AutoMigrate(&database.DeploymentLocation{}); err != nil {
		t.Fatalf("migrate deployment locations: %v", err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	registry := &ServiceRegistry{}
	if err := registry.UnregisterDeployment(context.Background(), "already-removed-container"); err != nil {
		t.Fatalf("unregister already-removed deployment location: %v", err)
	}
}
