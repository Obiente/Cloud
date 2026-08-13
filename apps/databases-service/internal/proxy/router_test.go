package proxy

import "testing"

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
