package utils

import (
	"os"
	"strconv"
	"strings"
)

// IsSwarmModeEnabled checks if Swarm mode is enabled via ENABLE_SWARM environment variable
// The compose files set defaults:
//   - docker-compose.swarm.yml: ENABLE_SWARM=${ENABLE_SWARM:-true} (defaults to "true")
//   - docker-compose.yml: uses common-orchestrator with ENABLE_SWARM=${ENABLE_SWARM:-false} (defaults to "false")
//
// Returns true if ENABLE_SWARM is "true", "1", "yes", "on"
// Returns false if ENABLE_SWARM is "false", "0", "no", "off", or empty string
func IsSwarmModeEnabled() bool {
	enableSwarm := os.Getenv("ENABLE_SWARM")
	if enableSwarm == "" {
		// If not set at all (shouldn't happen with compose defaults, but handle gracefully)
		// Default to false for safety (non-swarm mode)
		return false
	}

	// Parse as boolean
	lower := strings.ToLower(strings.TrimSpace(enableSwarm))
	if lower == "true" || lower == "1" || lower == "yes" || lower == "on" {
		return true
	}
	if lower == "false" || lower == "0" || lower == "no" || lower == "off" {
		return false
	}

	// Try parsing as boolean
	if enabled, err := strconv.ParseBool(lower); err == nil {
		return enabled
	}

	// Default to false if unparseable (safer default - non-swarm mode)
	return false
}
