package database

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// GetDeploymentNodeIP returns the node IP for a deployment based on where it's running
// This queries the deployment_locations table to find which node/region the deployment is in,
// then maps that to the appropriate node IP from environment configuration
func GetDeploymentNodeIP(deploymentID string, nodeIPMap map[string][]string) ([]string, error) {
	// Get deployment locations (where deployment is actually running)
	var locations []DeploymentLocation
	preferredStatuses := []string{"running", "restarting", "starting", "created"}
	result := DB.Where("deployment_id = ? AND status IN ?", deploymentID, preferredStatuses).
		Find(&locations)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query deployment locations: %w", result.Error)
	}

	if len(locations) == 0 {
		// Fallback: allow any status (deployments might still be starting/recovering)
		result = DB.Where("deployment_id = ?", deploymentID).
			Order("updated_at DESC").
			Find(&locations)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to query deployment locations (fallback): %w", result.Error)
		}
		if len(locations) == 0 {
			return nil, fmt.Errorf("no deployment locations found for deployment_id: %s", deploymentID)
		}
		// Prefer the first non-stopped location if available
		for i, loc := range locations {
			if !isStoppedStatus(loc.Status) {
				if i != 0 {
					locations[0], locations[i] = locations[i], locations[0]
				}
				break
			}
		}
	}

	// Get the first location's node to determine region
	location := locations[0]
	var node NodeMetadata
	var nodeRegion string

	if err := DB.First(&node, "id = ?", location.NodeID).Error; err != nil {
		// If node doesn't exist (e.g., was deleted), fall back to default region
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Node not found - use fallback logic
			if ips, ok := nodeIPMap["default"]; ok && len(ips) > 0 {
				return ips, nil
			}
			// Try to find any region in the map as fallback
			for region := range nodeIPMap {
				if ips := nodeIPMap[region]; len(ips) > 0 {
					return ips, nil
				}
			}
			return nil, fmt.Errorf("node %s not found and no default node IP configured", location.NodeID)
		}
		return nil, fmt.Errorf("failed to find node %s: %w", location.NodeID, err)
	}

	nodeRegion = node.Region

	// If node has no region, try to find a default or return error
	if nodeRegion == "" {
		// Try to find "default" region first, then any region as fallback
		if ips, ok := nodeIPMap["default"]; ok && len(ips) > 0 {
			return ips, nil
		}
		// Try to find any region in the map as fallback
		for region := range nodeIPMap {
			if ips := nodeIPMap[region]; len(ips) > 0 {
				return ips, nil
			}
		}
		return nil, fmt.Errorf("node %s has no region configured and no default region found", location.NodeID)
	}

	// Get node IPs for this region
	ips, ok := nodeIPMap[nodeRegion]
	if !ok || len(ips) == 0 {
		// Fallback to "default" region if the node's region doesn't exist
		if defaultIPs, defaultOk := nodeIPMap["default"]; defaultOk && len(defaultIPs) > 0 {
			return defaultIPs, nil
		}
		return nil, fmt.Errorf("no node IP configured for region: %s", nodeRegion)
	}

	return ips, nil
}

// GetGameServerNodeIP returns the node IP for a game server based on where it's running
// This queries the game_server_locations table to find which node/region the game server is in,
// then maps that to the appropriate node IP from environment configuration
func GetGameServerNodeIP(gameServerID string, nodeIPMap map[string][]string) ([]string, error) {
	// Get game server locations (where game server is actually running)
	var locations []GameServerLocation
	preferredStatuses := []string{"running", "restarting", "starting", "created"}
	result := DB.Where("game_server_id = ? AND status IN ?", gameServerID, preferredStatuses).
		Find(&locations)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query game server locations: %w", result.Error)
	}

	if len(locations) == 0 {
		result = DB.Where("game_server_id = ?", gameServerID).
			Order("updated_at DESC").
			Find(&locations)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to query game server locations (fallback): %w", result.Error)
		}
		if len(locations) == 0 {
			return nil, fmt.Errorf("no game server locations found for game_server_id: %s", gameServerID)
		}
		for i, loc := range locations {
			if !isStoppedStatus(loc.Status) {
				if i != 0 {
					locations[0], locations[i] = locations[i], locations[0]
				}
				break
			}
		}
	}

	// Get the first location's node to determine region
	location := locations[0]
	var node NodeMetadata
	var nodeRegion string

	if err := DB.First(&node, "id = ?", location.NodeID).Error; err != nil {
		// If node doesn't exist (e.g., was deleted), fall back to default region
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Node not found - use fallback logic
			if ips, ok := nodeIPMap["default"]; ok && len(ips) > 0 {
				return ips, nil
			}
			// Try to find any region in the map as fallback
			for region := range nodeIPMap {
				if ips := nodeIPMap[region]; len(ips) > 0 {
					return ips, nil
				}
			}
			return nil, fmt.Errorf("node %s not found and no default node IP configured", location.NodeID)
		}
		return nil, fmt.Errorf("failed to find node %s: %w", location.NodeID, err)
	}

	nodeRegion = node.Region

	// If node has no region, try to find a default or return error
	if nodeRegion == "" {
		// Try to find "default" region first, then any region as fallback
		if ips, ok := nodeIPMap["default"]; ok && len(ips) > 0 {
			return ips, nil
		}
		// Try to find any region in the map as fallback
		for region := range nodeIPMap {
			if ips := nodeIPMap[region]; len(ips) > 0 {
				return ips, nil
			}
		}
		return nil, fmt.Errorf("node %s has no region configured and no default region found", location.NodeID)
	}

	// Get node IPs for this region
	ips, ok := nodeIPMap[nodeRegion]
	if !ok || len(ips) == 0 {
		// Fallback to "default" region if the node's region doesn't exist
		if defaultIPs, defaultOk := nodeIPMap["default"]; defaultOk && len(defaultIPs) > 0 {
			return defaultIPs, nil
		}
		return nil, fmt.Errorf("no node IP configured for region: %s", nodeRegion)
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

// ParseNodeIPsFromEnv parses the NODE_IPS environment variable
// Format: "region1:ip1,ip2;region2:ip3,ip4"
// Also supports simple format: "ip1,ip2" (defaults to "default" region)
// Also supports space-separated format: "region1:ip1,ip2 region2:ip3,ip4" (when semicolons are not present)
// Returns a map of region -> []IP addresses
func ParseNodeIPsFromEnv(nodeIPsEnv string) (map[string][]string, error) {
	result := make(map[string][]string)

	if nodeIPsEnv == "" {
		return result, nil
	}

	// Check if the format contains semicolons (multi-region format)
	if !strings.Contains(nodeIPsEnv, ";") && !strings.Contains(nodeIPsEnv, ":") {
		// Simple format: just IPs without region (e.g., "ip1,ip2" or "ip1")
		ips := strings.Split(nodeIPsEnv, ",")
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

	// Determine separator: prefer semicolon, but fall back to space if no semicolons present
	var regions []string
	if strings.Contains(nodeIPsEnv, ";") {
		// Standard format: semicolon-separated
		regions = strings.Split(nodeIPsEnv, ";")
	} else {
		// Space-separated format: use regex to find all "region:ip" patterns
		// Pattern matches: word characters (region name), colon, then IP address(es) optionally separated by commas
		// This handles formats like:
		// - "us:1.2.3.4 nl:5.6.7.8"
		// - "us 1.2.3.4 nl:5.6.7.8" (region name followed by space and IP)
		// - "us:1.2.3.4,9.10.11.12 nl:5.6.7.8"

		// First, try to find all patterns that match "region:ip" or "region:ip1,ip2"
		// Pattern: one or more word chars, colon, then IP addresses (dots and numbers) possibly separated by commas
		re := regexp.MustCompile(`\w+:\d+\.\d+\.\d+\.\d+(?:,\d+\.\d+\.\d+\.\d+)*`)
		matches := re.FindAllString(nodeIPsEnv, -1)

		if len(matches) > 0 {
			// Found region:ip patterns
			regions = matches
		} else {
			// Fallback: try to handle "region IP" format (region name followed by space and IP)
			// Pattern: word chars (region), space, IP address
			re2 := regexp.MustCompile(`(\w+)\s+(\d+\.\d+\.\d+\.\d+)`)
			matches2 := re2.FindAllStringSubmatch(nodeIPsEnv, -1)
			for _, match := range matches2 {
				if len(match) >= 3 {
					// Convert "region IP" to "region:IP" format
					regions = append(regions, match[1]+":"+match[2])
				}
			}

			// If still no matches, fall back to simple splitting
			if len(regions) == 0 {
				parts := strings.Fields(nodeIPsEnv)
				var currentRegion string
				for i, part := range parts {
					if strings.Contains(part, ":") {
						if currentRegion != "" {
							regions = append(regions, currentRegion)
						}
						currentRegion = part
					} else if currentRegion != "" {
						if strings.Contains(currentRegion, ":") {
							// Append IP to existing region:ip
							parts := strings.SplitN(currentRegion, ":", 2)
							if len(parts) == 2 {
								currentRegion = parts[0] + ":" + parts[1] + "," + part
							}
						} else {
							// Region name without colon, add colon and IP
							currentRegion += ":" + part
						}
					} else {
						currentRegion = part
					}
					if i == len(parts)-1 && currentRegion != "" {
						regions = append(regions, currentRegion)
					}
				}
			}
		}
	}

	for _, regionStr := range regions {
		regionStr = strings.TrimSpace(regionStr)
		if regionStr == "" {
			continue
		}

		// Split by colon to separate region name from IPs
		parts := strings.SplitN(regionStr, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format in NODE_IPS: %s (expected 'region:ip1,ip2')", regionStr)
		}

		region := strings.TrimSpace(parts[0])
		ipsStr := strings.TrimSpace(parts[1])

		if region == "" {
			return nil, fmt.Errorf("empty region name in NODE_IPS: %s", regionStr)
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
			return nil, fmt.Errorf("no IPs found for region %s in NODE_IPS", region)
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

		// First try to use node.IP from NodeMetadata
		if node.IP != "" {
			return node.IP, location.Port, nil
		}

		// Fallback to hostname if IP is not available
		if location.NodeHostname != "" {
			// Try to resolve hostname to IP (this is a fallback - ideally NodeIP should be populated)
			// For now, return hostname and let DNS resolve it
			return location.NodeHostname, location.Port, nil
		}

		// If neither IP nor hostname is available, return error
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

func isStoppedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "stopped", "exited", "dead", "removing":
		return true
	default:
		return false
	}
}
