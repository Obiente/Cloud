package orchestrator

import (
	"errors"
	"fmt"
	"strings"
)

// SwarmRolloutError describes an update that failed after Docker accepted it.
// When PreviousRevisionRunning is true, Swarm completed its rollback and the
// deployment remains available on the recorded task.
type SwarmRolloutError struct {
	ServiceName             string
	State                   string
	Message                 string
	Diagnostics             string
	PreviousRevisionRunning bool
	ServiceID               string
	TaskID                  string
	ContainerID             string
}

func (e *SwarmRolloutError) Error() string {
	if e == nil {
		return "swarm rollout failed"
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = strings.TrimSpace(e.State)
	}
	if message == "" {
		message = "update did not converge"
	}
	result := fmt.Sprintf("swarm rollout for %s did not converge: %s", e.ServiceName, message)
	if e.PreviousRevisionRunning {
		result += "; the previous revision is still running"
	}
	if diagnostics := strings.TrimSpace(e.Diagnostics); diagnostics != "" {
		result += "; diagnostics: " + diagnostics
	}
	return result
}

// RollbackPreserved reports whether Docker restored a running previous revision.
func RollbackPreserved(err error) bool {
	var rolloutErr *SwarmRolloutError
	return errors.As(err, &rolloutErr) && rolloutErr.PreviousRevisionRunning
}
