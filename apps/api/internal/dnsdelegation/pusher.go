package dnsdelegation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"api/internal/database"
	"api/internal/logger"
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
	sourceAPI := os.Getenv("DASHBOARD_URL")
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

			// Get organization ID from deployment
			var organizationID string
			if err := database.DB.Table("deployments").
				Select("organization_id").
				Where("id = ?", deploymentID).
				Pluck("organization_id", &organizationID).Error; err != nil {
				logger.Warn("[DNS Pusher] Failed to get organization ID for deployment %s: %v", deploymentID, err)
				// Continue anyway - we'll try to store without org ID
			}

			ips, err := database.GetDeploymentTraefikIP(deploymentID, traefikIPMap)
			if err != nil {
				logger.Warn("[DNS Pusher] Failed to get IPs for deployment %s: %v", deploymentID, err)
				continue
			}

			if len(ips) > 0 {
				domain := deploymentID + ".my.obiente.cloud"
				record := map[string]interface{}{
					"domain":      domain,
					"record_type": "A",
					"records":     ips,
					"ttl":         config.TTL,
				}
				if organizationID != "" {
					record["organization_id"] = organizationID
				}
				records = append(records, record)
			}
		}
		deploymentLocations.Close()
	}

	// Push game server A records and SRV records
	gameServerLocations, err := database.DB.Table("game_server_locations").
		Where("status = ?", "running").
		Select("game_server_id, node_ip, port").
		Rows()
	if err == nil {
		for gameServerLocations.Next() {
			var gameServerID, nodeIP string
			var port int32
			if err := gameServerLocations.Scan(&gameServerID, &nodeIP, &port); err != nil {
				continue
			}

			// Get organization ID and game type from game server
			var gameServerInfo struct {
				OrganizationID string
				GameType       int32
			}
			if err := database.DB.Table("game_servers").
				Select("organization_id, game_type").
				Where("id = ?", gameServerID).
				First(&gameServerInfo).Error; err != nil {
				logger.Warn("[DNS Pusher] Failed to get organization ID and game type for game server %s: %v", gameServerID, err)
				// Continue anyway - we'll try to store without org ID
			}

			// Push A record for {id}.my.obiente.cloud format (e.g., gs-123.my.obiente.cloud)
			if nodeIP != "" {
				// Game server IDs are in gs-{id} format, use directly
				domain := fmt.Sprintf("%s.my.obiente.cloud", gameServerID)
				record := map[string]interface{}{
					"domain":      domain,
					"record_type": "A",
					"records":     []string{nodeIP},
					"ttl":         config.TTL,
				}
				if gameServerInfo.OrganizationID != "" {
					record["organization_id"] = gameServerInfo.OrganizationID
				}
				records = append(records, record)
			}

			// Push SRV records based on game type
			// GameType enum values:
			// MINECRAFT = 1, MINECRAFT_JAVA = 2, MINECRAFT_BEDROCK = 3, RUST = 6
			if port > 0 {
				// Use {id} format for SRV records to match A record format
				// The target hostname in SRV records should point to the A record
				targetHostname := fmt.Sprintf("%s.my.obiente.cloud", gameServerID)
				srvRecordValue := fmt.Sprintf("0 0 %d %s", port, targetHostname)

				// Minecraft Java Edition (TCP) - gameType 1 or 2
				if gameServerInfo.GameType == 1 || gameServerInfo.GameType == 2 {
					srvDomain := fmt.Sprintf("_minecraft._tcp.%s", targetHostname)
					record := map[string]interface{}{
						"domain":      srvDomain,
						"record_type": "SRV",
						"records":     []string{srvRecordValue},
						"ttl":         config.TTL,
					}
					if gameServerInfo.OrganizationID != "" {
						record["organization_id"] = gameServerInfo.OrganizationID
					}
					records = append(records, record)
				}

				// Minecraft Bedrock Edition (UDP) - gameType 1 or 3
				if gameServerInfo.GameType == 1 || gameServerInfo.GameType == 3 {
					srvDomain := fmt.Sprintf("_minecraft._udp.%s", targetHostname)
					record := map[string]interface{}{
						"domain":      srvDomain,
						"record_type": "SRV",
						"records":     []string{srvRecordValue},
						"ttl":         config.TTL,
					}
					if gameServerInfo.OrganizationID != "" {
						record["organization_id"] = gameServerInfo.OrganizationID
					}
					records = append(records, record)
				}

				// Rust (UDP) - gameType 6
				if gameServerInfo.GameType == 6 {
					srvDomain := fmt.Sprintf("_rust._udp.%s", targetHostname)
					record := map[string]interface{}{
						"domain":      srvDomain,
						"record_type": "SRV",
						"records":     []string{srvRecordValue},
						"ttl":         config.TTL,
					}
					if gameServerInfo.OrganizationID != "" {
						record["organization_id"] = gameServerInfo.OrganizationID
					}
					records = append(records, record)
				}
			}
		}
		gameServerLocations.Close()
	}

	if len(records) == 0 {
		logger.Info("[DNS Pusher] No DNS records to push")
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
	
	sourceAPI := os.Getenv("DASHBOARD_URL")
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

	logger.Info("[DNS Pusher] Successfully pushed %d DNS records to production", len(records))

	// Also store records locally so they appear in the superadmin DNS page
	// Get API key ID (for tracking, but organization ID comes from each deployment/game server)
	var apiKeyID string
	apiKeyInfo, err := database.GetDNSDelegationAPIKeyByHash(config.APIKey)
	if err == nil {
		apiKeyID = apiKeyInfo.ID
		logger.Info("[DNS Pusher] Found API key in local database: key=%s", apiKeyID)
	}

	// Get source API URL for tracking (reuse the one already set above)
	if sourceAPI == "" {
		sourceAPI = os.Getenv("DASHBOARD_URL")
	}
	if sourceAPI == "" {
		sourceAPI = "local" // Fallback if no URL is set
	}

	// Store each record locally with its own organization ID
	localStoreCount := 0
	for _, recordData := range records {
		domain, _ := recordData["domain"].(string)
		recordType, _ := recordData["record_type"].(string)
		recordsList, _ := recordData["records"].([]string)
		ttl, _ := recordData["ttl"].(int64)
		// Get organization ID from the record (set when building the record)
		recordOrgID, _ := recordData["organization_id"].(string)

		if domain == "" || recordType == "" || len(recordsList) == 0 {
			continue
		}

		// Convert records to JSON
		recordsJSON, err := json.Marshal(recordsList)
		if err != nil {
			logger.Warn("[DNS Pusher] Failed to marshal records for local storage (%s): %v", domain, err)
			continue
		}

		// Store locally with the organization ID from the deployment/game server
		if err := database.UpsertDelegatedDNSRecordWithAPIKey(domain, recordType, string(recordsJSON), sourceAPI, apiKeyID, recordOrgID, ttl); err != nil {
			logger.Warn("[DNS Pusher] Failed to store record locally (%s): %v", domain, err)
			continue
		}

		localStoreCount++
		if recordOrgID != "" {
			logger.Debug("[DNS Pusher] Stored record %s for organization %s", domain, recordOrgID)
		}
	}

	if localStoreCount > 0 {
		logger.Info("[DNS Pusher] Stored %d DNS records locally", localStoreCount)
	}

	return nil
}

// StartDNSPusher starts a background goroutine that periodically pushes DNS records
func StartDNSPusher(config PusherConfig) {
	if config.ProductionAPIURL == "" || config.APIKey == "" {
		logger.Info("[DNS Pusher] DNS delegation not configured, pusher not started")
		return
	}

	logger.Info("[DNS Pusher] Starting DNS pusher (interval: %v, TTL: %d)", config.PushInterval, config.TTL)

	go func() {
		ticker := time.NewTicker(config.PushInterval)
		defer ticker.Stop()

		// Push immediately on start
		if err := PushAllDNSRecords(config); err != nil {
			logger.Warn("[DNS Pusher] Initial push failed: %v", err)
		}

		for range ticker.C {
			if err := PushAllDNSRecords(config); err != nil {
				logger.Warn("[DNS Pusher] Failed to push DNS records: %v", err)
			}
		}
	}()
}

