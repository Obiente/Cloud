package registry

import (
	"testing"

	"github.com/moby/moby/api/types/swarm"
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
