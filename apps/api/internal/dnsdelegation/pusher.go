package dnsdelegation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"api/internal/database"
)

// PusherConfig represents configuration for DNS record pusher
type PusherConfig struct {
	ProductionAPIURL string        // URL of production API (e.g., "https://api.obiente.cloud")
	APIKey           string        // API key for authentication
	PushInterval     time.Duration // How often to push records (default: 2 minutes)
	TTL              int64         // TTL for pushed records (default: 300 seconds)
}

// ParsePusherConfig parses DNS pusher configuration from environment variables
func ParsePusherConfig() PusherConfig {
	config := PusherConfig{
		PushInterval: 2 * time.Minute,
		TTL:          300, // 5 minutes default
	}

	// Parse production API URL
	config.ProductionAPIURL = strings.TrimSpace(os.Getenv("DNS_DELEGATION_PRODUCTION_API_URL"))

	// Parse API key
	config.APIKey = strings.TrimSpace(os.Getenv("DNS_DELEGATION_API_KEY"))

	// Parse push interval
	intervalStr := strings.TrimSpace(os.Getenv("DNS_DELEGATION_PUSH_INTERVAL"))
	if intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			config.PushInterval = interval
		}
	}

	// Parse TTL
	ttlStr := strings.TrimSpace(os.Getenv("DNS_DELEGATION_TTL"))
	if ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			config.TTL = int64(ttl.Seconds())
		}
	}

	return config
}

// PushDNSRecord pushes a single DNS record to production API
func PushDNSRecord(config PusherConfig, domain, recordType string, records []string) error {
	if config.ProductionAPIURL == "" || config.APIKey == "" {
		return fmt.Errorf("DNS delegation not configured (missing PRODUCTION_API_URL or API_KEY)")
	}

	url := strings.TrimSuffix(config.ProductionAPIURL, "/") + "/dns/push"

	reqBody := map[string]interface{}{
		"domain":      domain,
		"record_type": recordType,
		"records":     records,
		"ttl":         config.TTL,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	
	// Include source API URL for tracking
	sourceAPI := os.Getenv("CONSOLE_URL")
	if sourceAPI == "" {
		sourceAPI = os.Getenv("DASHBOARD_URL")
	}
	if sourceAPI != "" {
		req.Header.Set("X-Source-API", sourceAPI)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to push DNS record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// PushAllDNSRecords pushes all DNS records for deployments and game servers
func PushAllDNSRecords(config PusherConfig) error {
	if config.ProductionAPIURL == "" || config.APIKey == "" {
		return fmt.Errorf("DNS delegation not configured")
	}

	// Get Traefik IPs for deployments
	traefikIPsEnv := os.Getenv("TRAEFIK_IPS")
	if traefikIPsEnv == "" {
		return fmt.Errorf("TRAEFIK_IPS not configured")
	}

	traefikIPMap, err := database.ParseTraefikIPsFromEnv(traefikIPsEnv)
	if err != nil {
		return fmt.Errorf("failed to parse TRAEFIK_IPS: %w", err)
	}

	records := make([]map[string]interface{}, 0)

	// Push deployment A records
	deploymentLocations, err := database.DB.Table("deployment_locations").
		Where("status = ?", "running").
		Select("deployment_id").
		Distinct().
		Rows()
	if err == nil {
		for deploymentLocations.Next() {
			var deploymentID string
			if err := deploymentLocations.Scan(&deploymentID); err != nil {
				continue
			}

			ips, err := database.GetDeploymentTraefikIP(deploymentID, traefikIPMap)
			if err != nil {
				log.Printf("[DNS Pusher] Failed to get IPs for deployment %s: %v", deploymentID, err)
				continue
			}

			if len(ips) > 0 {
				domain := deploymentID + ".my.obiente.cloud"
				records = append(records, map[string]interface{}{
					"domain":      domain,
					"record_type": "A",
					"records":     ips,
					"ttl":         config.TTL,
				})
			}
		}
		deploymentLocations.Close()
	}

	// Push game server A records
	gameServerLocations, err := database.DB.Table("game_server_locations").
		Where("status = ?", "running").
		Select("game_server_id, node_ip").
		Rows()
	if err == nil {
		for gameServerLocations.Next() {
			var gameServerID, nodeIP string
			if err := gameServerLocations.Scan(&gameServerID, &nodeIP); err != nil {
				continue
			}

			if nodeIP != "" {
				domain := gameServerID + ".my.obiente.cloud"
				records = append(records, map[string]interface{}{
					"domain":      domain,
					"record_type": "A",
					"records":     []string{nodeIP},
					"ttl":         config.TTL,
				})
			}
		}
		gameServerLocations.Close()
	}

	if len(records) == 0 {
		log.Printf("[DNS Pusher] No DNS records to push")
		return nil
	}

	// Push all records in batch
	url := strings.TrimSuffix(config.ProductionAPIURL, "/") + "/dns/push/batch"

	reqBody := map[string]interface{}{
		"records": records,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	
	sourceAPI := os.Getenv("CONSOLE_URL")
	if sourceAPI == "" {
		sourceAPI = os.Getenv("DASHBOARD_URL")
	}
	if sourceAPI != "" {
		req.Header.Set("X-Source-API", sourceAPI)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to push DNS records: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("[DNS Pusher] Successfully pushed %d DNS records", len(records))
	return nil
}

// StartDNSPusher starts a background goroutine that periodically pushes DNS records
func StartDNSPusher(config PusherConfig) {
	if config.ProductionAPIURL == "" || config.APIKey == "" {
		log.Printf("[DNS Pusher] DNS delegation not configured, pusher not started")
		return
	}

	log.Printf("[DNS Pusher] Starting DNS pusher (interval: %v, TTL: %d)", config.PushInterval, config.TTL)

	go func() {
		ticker := time.NewTicker(config.PushInterval)
		defer ticker.Stop()

		// Push immediately on start
		if err := PushAllDNSRecords(config); err != nil {
			log.Printf("[DNS Pusher] Initial push failed: %v", err)
		}

		for range ticker.C {
			if err := PushAllDNSRecords(config); err != nil {
				log.Printf("[DNS Pusher] Failed to push DNS records: %v", err)
			}
		}
	}()
}

