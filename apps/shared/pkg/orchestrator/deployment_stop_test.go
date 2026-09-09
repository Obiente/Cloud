package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStopDeploymentPropagatesSwarmFailures(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Docker command stub requires a POSIX shell")
	}
	for _, stage := range []string{"discover", "remove"} {
		t.Run(stage, func(t *testing.T) {
			t.Setenv("ENABLE_SWARM", "true")
			dir := t.TempDir()
			script := "#!/bin/sh\nexit 1\n"
			if stage == "remove" {
				script = "#!/bin/sh\nif [ \"$2\" = ls ]; then echo managed-service; exit 0; fi\nexit 1\n"
			}
			if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			// No database is configured: a failed runtime stop must return before
			// attempting to mark any deployment locations as stopped.
			err := (&DeploymentManager{}).StopDeployment(context.Background(), "test-deployment")
			if err == nil || !strings.Contains(err.Error(), stage+" managed Swarm services") {
				t.Fatalf("expected %s error before status update, got %v", stage, err)
			}
		})
	}
}
