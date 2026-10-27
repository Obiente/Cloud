package orchestrator

import (
	"github.com/moby/moby/api/types/swarm"
	"testing"
)

func TestSwarmHealthFindsPausedRolloutEvenWithHealthyReplicas(t *testing.T) {
	service := swarm.Service{UpdateStatus: &swarm.UpdateStatus{State: swarm.UpdateStatePaused}, ServiceStatus: &swarm.ServiceStatus{DesiredTasks: 2, RunningTasks: 2}}
	paused, missing := swarmServiceHealth(service)
	if paused != 1 || missing != 0 {
		t.Fatalf("paused=%v missing=%v", paused, missing)
	}
	service.UpdateStatus.State = swarm.UpdateStateCompleted
	service.ServiceStatus.RunningTasks = 1
	paused, missing = swarmServiceHealth(service)
	if paused != 0 || missing != 1 {
		t.Fatalf("paused=%v missing=%v", paused, missing)
	}
	service.ServiceStatus.RunningTasks = 3
	_, missing = swarmServiceHealth(service)
	if missing != 0 {
		t.Fatal("surplus replicas underflowed")
	}
}
