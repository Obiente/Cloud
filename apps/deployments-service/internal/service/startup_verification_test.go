package deployments

import (
	"errors"
	"strings"
	"testing"
)

func TestIsSwarmTaskRunning(t *testing.T) {
	t.Parallel()

	runningStates := []string{
		"Running 3 seconds ago",
		"running 1 minute ago",
		" Running about a minute ago ",
	}
	for _, state := range runningStates {
		state := state
		t.Run("running "+state, func(t *testing.T) {
			t.Parallel()
			if !isSwarmTaskRunning(state) {
				t.Fatalf("isSwarmTaskRunning(%q) = false, want true", state)
			}
		})
	}

	nonRunningStates := []string{
		"",
		"Starting 2 seconds ago",
		"Preparing 4 seconds ago",
		"Shutdown 1 second ago",
		"Failed 5 seconds ago",
		"Complete 1 second ago",
	}
	for _, state := range nonRunningStates {
		state := state
		t.Run("not running "+state, func(t *testing.T) {
			t.Parallel()
			if isSwarmTaskRunning(state) {
				t.Fatalf("isSwarmTaskRunning(%q) = true, want false", state)
			}
		})
	}
}

func TestFormatContainerVerificationErrorIncludesDiagnostics(t *testing.T) {
	t.Parallel()

	err := formatContainerVerificationError(
		errors.New("no running containers found for deployment deploy-123"),
		"abc123 status=exited exit_code=1 error=\"\" oom_killed=false",
		"deploy-deploy-123-default: task.1\tFailed 2 seconds ago\tno such file or directory",
	)

	message := err.Error()
	for _, want := range []string{
		"no running containers found",
		"containers: abc123 status=exited exit_code=1",
		"swarm tasks: deploy-deploy-123-default",
		"no such file or directory",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("formatted error %q does not contain %q", message, want)
		}
	}
}
