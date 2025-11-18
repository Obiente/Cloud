package health

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
)

var (
	replicaID     string
	replicaIDOnce sync.Once
)

// GetReplicaID returns a unique replica ID for this instance
// Uses hostname + container ID (if available) to generate a stable but unique ID
// The ID is generated once and cached for the lifetime of the process
func GetReplicaID() string {
	replicaIDOnce.Do(func() {
		// Try to get container ID from environment (Docker/Kubernetes)
		containerID := os.Getenv("HOSTNAME")
		if containerID == "" {
			// Fallback to hostname
			hostname, err := os.Hostname()
			if err != nil {
				hostname = "unknown"
			}
			containerID = hostname
		}

		// Generate replica ID: service-name-container-id
		// This ensures uniqueness across replicas while being stable for the same container
		replicaID = containerID
	})
	return replicaID
}

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status    string                 `json:"status"`
	Service   string                 `json:"service"`
	ReplicaID string                 `json:"replica_id"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// HandleHealth creates a health check handler that includes replica ID
// serviceName: name of the service (e.g., "auth-service")
// healthCheck: optional function to perform additional health checks (database, etc.)
//
//	returns (isHealthy, message, extraData)
func HandleHealth(serviceName string, healthCheck func() (bool, string, map[string]interface{})) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// Perform health check if provided
		var isHealthy bool = true
		var extra map[string]interface{} = make(map[string]interface{})

		if healthCheck != nil {
			var healthMsg string
			var healthExtra map[string]interface{}
			isHealthy, healthMsg, healthExtra = healthCheck()
			if healthExtra != nil {
				extra = healthExtra
			}
			// Add message to extra if provided
			if healthMsg != "" && healthMsg != "healthy" {
				if extra == nil {
					extra = make(map[string]interface{})
				}
				extra["message"] = healthMsg
			}
		}

		// Build response
		response := HealthResponse{
			Status:    "healthy",
			Service:   serviceName,
			ReplicaID: GetReplicaID(),
			Extra:     extra,
		}

		if !isHealthy {
			response.Status = "unhealthy"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// SimpleHealth creates a basic health check handler without additional checks
func SimpleHealth(serviceName string) http.HandlerFunc {
	return HandleHealth(serviceName, nil)
}
