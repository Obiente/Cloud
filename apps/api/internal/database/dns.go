package database

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// GetDeploymentTraefikIP returns the Traefik IP for a deployment based on where it's running
// This queries the deployment_locations table to find which node/region the deployment is in,
// then maps that to the appropriate Traefik IP from environment configuration
func GetDeploymentTraefikIP(deploymentID string, traefikIPMap map[string][]string) ([]string, error) {
	// Get deployment locations (where deployment is actually running)
	var locations []DeploymentLocation
	result := DB.Where("deployment_id = ? AND status = ?", deploymentID, "running").
		Find(&locations)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query deployment locations: %w", result.Error)
	}

	if len(locations) == 0 {
		return nil, fmt.Errorf("no running deployment found for deployment_id: %s", deploymentID)
	}

	// Get the first location's node to determine region
	location := locations[0]
	var node NodeMetadata
	var nodeRegion string
	
	if err := DB.First(&node, "id = ?", location.NodeID).Error; err != nil {
		// If node doesn't exist (e.g., was deleted), fall back to default region
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Node not found - use fallback logic
			if ips, ok := traefikIPMap["default"]; ok && len(ips) > 0 {
				return ips, nil
			}
			// Try to find any region in the map as fallback
			for region := range traefikIPMap {
				if ips := traefikIPMap[region]; len(ips) > 0 {
					return ips, nil
				}
			}
			return nil, fmt.Errorf("node %s not found and no default Traefik IP configured", location.NodeID)
		}
		return nil, fmt.Errorf("failed to find node %s: %w", location.NodeID, err)
	}

	nodeRegion = node.Region

	// If node has no region, try to find a default or return error
	if nodeRegion == "" {
		// Try to find "default" region first, then any region as fallback
		if ips, ok := traefikIPMap["default"]; ok && len(ips) > 0 {
			return ips, nil
		}
		// Try to find any region in the map as fallback
		for region := range traefikIPMap {
			if ips := traefikIPMap[region]; len(ips) > 0 {
				return ips, nil
			}
		}
		return nil, fmt.Errorf("node %s has no region configured and no default region found", location.NodeID)
	}

	// Get Traefik IPs for this region
	ips, ok := traefikIPMap[nodeRegion]
	if !ok || len(ips) == 0 {
		// Fallback to "default" region if the node's region doesn't exist
		if defaultIPs, defaultOk := traefikIPMap["default"]; defaultOk && len(defaultIPs) > 0 {
			return defaultIPs, nil
		}
		return nil, fmt.Errorf("no Traefik IP configured for region: %s", nodeRegion)
	}

	return ips, nil
}

// GetDeploymentRegion returns the region where a deployment is running
func GetDeploymentRegion(deploymentID string) (string, error) {
	// Get deployment locations
	var locations []DeploymentLocation
	result := DB.Where("deployment_id = ? AND status = ?", deploymentID, "running").
		Find(&locations)
	if result.Error != nil {
		return "", fmt.Errorf("failed to query deployment locations: %w", result.Error)
	}

	if len(locations) == 0 {
		return "", fmt.Errorf("no running deployment found for deployment_id: %s", deploymentID)
	}

	// Get the first location's node to determine region
	location := locations[0]
	var node NodeMetadata
	if err := DB.First(&node, "id = ?", location.NodeID).Error; err != nil {
		// If node doesn't exist (e.g., was deleted), return empty region
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("node %s not found", location.NodeID)
		}
		return "", fmt.Errorf("failed to find node %s: %w", location.NodeID, err)
	}

	return node.Region, nil
}

// ParseTraefikIPsFromEnv parses the TRAEFIK_IPS environment variable
// Format: "region1:ip1,ip2;region2:ip3,ip4"
// Also supports simple format: "ip1,ip2" (defaults to "default" region)
// Returns a map of region -> []IP addresses
func ParseTraefikIPsFromEnv(traefikIPsEnv string) (map[string][]string, error) {
	result := make(map[string][]string)

	if traefikIPsEnv == "" {
		return result, nil
	}

	// Check if the format contains semicolons (multi-region format)
	if !strings.Contains(traefikIPsEnv, ";") && !strings.Contains(traefikIPsEnv, ":") {
		// Simple format: just IPs without region (e.g., "ip1,ip2" or "ip1")
		ips := strings.Split(traefikIPsEnv, ",")
		var cleanedIPs []string
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				cleanedIPs = append(cleanedIPs, ip)
			}
		}
		if len(cleanedIPs) > 0 {
			result["default"] = cleanedIPs
		}
		return result, nil
	}

	// Split by semicolon to get regions
	regions := strings.Split(traefikIPsEnv, ";")
	for _, regionStr := range regions {
		regionStr = strings.TrimSpace(regionStr)
		if regionStr == "" {
			continue
		}

		// Split by colon to separate region name from IPs
		parts := strings.SplitN(regionStr, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format in TRAEFIK_IPS: %s (expected 'region:ip1,ip2')", regionStr)
		}

		region := strings.TrimSpace(parts[0])
		ipsStr := strings.TrimSpace(parts[1])

		if region == "" {
			return nil, fmt.Errorf("empty region name in TRAEFIK_IPS: %s", regionStr)
		}

		// Split IPs by comma
		ips := strings.Split(ipsStr, ",")
		var cleanedIPs []string
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				cleanedIPs = append(cleanedIPs, ip)
			}
		}

		if len(cleanedIPs) == 0 {
			return nil, fmt.Errorf("no IPs found for region %s in TRAEFIK_IPS", region)
		}

		result[region] = cleanedIPs
	}

	return result, nil
}

// GetGameServerLocation returns the IP and port for a game server
// This queries the game_server_locations table to find where the game server is running
func GetGameServerLocation(gameServerID string) (string, int32, error) {
	// Get game server locations (where game server is actually running)
	var locations []GameServerLocation
	result := DB.Where("game_server_id = ? AND status = ?", gameServerID, "running").
		Find(&locations)
	if result.Error != nil {
		return "", 0, fmt.Errorf("failed to query game server locations: %w", result.Error)
	}

	if len(locations) == 0 {
		return "", 0, fmt.Errorf("no running game server found for game_server_id: %s", gameServerID)
	}

	// Get the first location (game servers typically run on one node)
	location := locations[0]
	
	// If NodeIP is not set, try to get it from NodeMetadata
	if location.NodeIP == "" {
		var node NodeMetadata
		if err := DB.First(&node, "id = ?", location.NodeID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", 0, fmt.Errorf("node %s not found for game server %s", location.NodeID, gameServerID)
			}
			return "", 0, fmt.Errorf("failed to find node %s: %w", location.NodeID, err)
		}
		
		// NodeMetadata.Address is a JSONB field that might contain IP
		// For now, use NodeHostname if available, or try to parse Address
		if location.NodeHostname != "" {
			// Try to resolve hostname to IP (this is a fallback - ideally NodeIP should be populated)
			// For now, return hostname and let DNS resolve it
			return location.NodeHostname, location.Port, nil
		}
		
		// If Address is set, try to extract IP from it
		// Address is JSONB, so it might be a JSON object or string
		// For now, return error if we can't find IP
		return "", 0, fmt.Errorf("node %s has no IP address configured for game server %s", location.NodeID, gameServerID)
	}

	return location.NodeIP, location.Port, nil
}

// GetGameServerType returns the game type for a game server
func GetGameServerType(gameServerID string) (int32, error) {
	var gameServer struct {
		GameType int32
	}
	result := DB.Table("game_servers").
		Select("game_type").
		Where("id = ?", gameServerID).
		First(&gameServer)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("game server %s not found", gameServerID)
		}
		return 0, fmt.Errorf("failed to query game server: %w", result.Error)
	}
	return gameServer.GameType, nil
}

// GetGameServerIP returns the IP address for a game server (for A record queries)
func GetGameServerIP(gameServerID string) (string, error) {
	nodeIP, _, err := GetGameServerLocation(gameServerID)
	return nodeIP, err
}
