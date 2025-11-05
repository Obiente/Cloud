package dnsdelegation

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"api/internal/database"
)

// HandleDNSQuery handles DNS query requests from remote DNS servers
// This endpoint is public but requires API key authentication
func HandleDNSQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check API key authentication
	apiKey := r.Header.Get("Authorization")
	expectedAPIKey := os.Getenv("DNS_DELEGATION_API_KEY")
	if expectedAPIKey == "" {
		http.Error(w, "DNS delegation not configured", http.StatusServiceUnavailable)
		return
	}

	// Remove "Bearer " prefix if present
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	apiKey = strings.TrimSpace(apiKey)

	if apiKey != expectedAPIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		Domain     string `json:"domain"`
		RecordType string `json:"record_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	domain := strings.TrimSpace(req.Domain)
	recordType := strings.ToUpper(strings.TrimSpace(req.RecordType))
	if recordType == "" {
		recordType = "A"
	}

	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	// Validate domain format
	if !strings.HasSuffix(strings.ToLower(domain), ".my.obiente.cloud") {
		http.Error(w, "Domain must be a *.my.obiente.cloud domain", http.StatusBadRequest)
		return
	}

	// Support A and SRV record types
	if recordType != "A" && recordType != "SRV" {
		http.Error(w, "Only A and SRV record types are supported", http.StatusBadRequest)
		return
	}

	// Extract resource ID from domain
	parts := strings.Split(strings.ToLower(domain), ".")

	// Handle SRV queries
	if recordType == "SRV" {
		if len(parts) < 4 {
			http.Error(w, "Invalid SRV domain format", http.StatusBadRequest)
			return
		}

		gameServerID := parts[2]
		if !strings.HasPrefix(gameServerID, "gameserver-") {
			http.Error(w, "Invalid game server ID format", http.StatusBadRequest)
			return
		}

		// Get game server type to validate SRV service matches
		gameType, err := database.GetGameServerType(gameServerID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"domain":      domain,
				"record_type": recordType,
				"error":       err.Error(),
				"ttl":         60,
			})
			return
		}

		service := parts[0]
		protocol := parts[1]

		// Validate SRV service/protocol matches game type
		isValid := false
		if service == "_minecraft" {
			if protocol == "_tcp" && (gameType == 1 || gameType == 2) {
				isValid = true
			} else if protocol == "_udp" && (gameType == 1 || gameType == 3) {
				isValid = true
			}
		} else if service == "_rust" && protocol == "_udp" && gameType == 6 {
			isValid = true
		}

		if !isValid {
			http.Error(w, "Unsupported SRV service/protocol for this game type", http.StatusBadRequest)
			return
		}

		// Get game server location
		_, port, err := database.GetGameServerLocation(gameServerID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"domain":      domain,
				"record_type": recordType,
				"error":       err.Error(),
				"ttl":         60,
			})
			return
		}

		targetHostname := gameServerID + ".my.obiente.cloud"
		srvRecord := fmt.Sprintf("0 0 %d %s", port, targetHostname)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domain":      domain,
			"record_type": recordType,
			"records":     []string{srvRecord},
			"ttl":         60,
		})
		return
	}

	// Handle A record queries
	if len(parts) < 3 {
		http.Error(w, "Invalid domain format", http.StatusBadRequest)
		return
	}

	resourceID := parts[0]

	// Check if this is a game server
	if strings.HasPrefix(resourceID, "gameserver-") {
		nodeIP, err := database.GetGameServerIP(resourceID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"domain":      domain,
				"record_type": recordType,
				"error":       err.Error(),
				"ttl":         60,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domain":      domain,
			"record_type": recordType,
			"records":     []string{nodeIP},
			"ttl":         60,
		})
		return
	}

	// Otherwise, treat as deployment
	deploymentID := resourceID

	// Get Traefik IPs from environment
	traefikIPsEnv := os.Getenv("TRAEFIK_IPS")
	if traefikIPsEnv == "" {
		http.Error(w, "TRAEFIK_IPS not configured", http.StatusInternalServerError)
		return
	}

	traefikIPMap, err := database.ParseTraefikIPsFromEnv(traefikIPsEnv)
	if err != nil {
		log.Printf("[DNS Delegation] Failed to parse TRAEFIK_IPS: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse TRAEFIK_IPS: %v", err), http.StatusInternalServerError)
		return
	}

	// Query database for deployment location
	ips, err := database.GetDeploymentTraefikIP(deploymentID, traefikIPMap)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domain":      domain,
			"record_type": recordType,
			"error":       err.Error(),
			"ttl":         60,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"domain":      domain,
		"record_type": recordType,
		"records":     ips,
		"ttl":         60,
	})
}

