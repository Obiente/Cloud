package proxy

import (
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
)

func TestDatabaseLocationStatus(t *testing.T) {
	tests := map[int32]string{
		0:  "created",
		1:  "creating",
		2:  "starting",
		3:  "running",
		4:  "stopping",
		5:  "stopped",
		6:  "backing_up",
		7:  "restoring",
		8:  "failed",
		9:  "deleting",
		10: "deleted",
		11: "suspended",
		12: "sleeping",
	}

	for status, want := range tests {
		if got := databaseLocationStatus(status); got != want {
			t.Errorf("databaseLocationStatus(%d) = %q, want %q", status, got, want)
		}
	}
}

func TestDatabaseLocationNeedsReconciliation(t *testing.T) {
	nodeID := "node-routing-test"
	instance := &database.DatabaseInstance{NodeID: &nodeID, Status: 3}
	localNode := docker.NodeIdentity{
		ID:       nodeID,
		IP:       "192.0.2.10",
		Hostname: "routing-host",
	}
	current := databaseLocationSnapshot{
		NodeID:       nodeID,
		NodeIP:       localNode.IP,
		NodeHostname: localNode.Hostname,
		Status:       "running",
	}

	if databaseLocationNeedsReconciliation(instance, current, localNode) {
		t.Fatal("current location unexpectedly needs reconciliation")
	}

	tests := map[string]databaseLocationSnapshot{
		"changed address": {
			NodeID:       nodeID,
			NodeIP:       "198.51.100.20",
			NodeHostname: localNode.Hostname,
			Status:       "running",
		},
		"changed hostname": {
			NodeID:       nodeID,
			NodeIP:       localNode.IP,
			NodeHostname: "old-routing-host",
			Status:       "running",
		},
		"stale status": {
			NodeID:       nodeID,
			NodeIP:       localNode.IP,
			NodeHostname: localNode.Hostname,
			Status:       "starting",
		},
	}
	for name, location := range tests {
		t.Run(name, func(t *testing.T) {
			if !databaseLocationNeedsReconciliation(instance, location, localNode) {
				t.Fatal("stale location did not request reconciliation")
			}
		})
	}
}
