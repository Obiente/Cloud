package database

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// GetDeploymentNodeIP returns the preferred IPs for a deployment based on where it's running.
// It prefers the actual node IP recorded on the selected location, then falls back to the
// node metadata IP, and only uses region->NODE_IPS mapping as a compatibility fallback.
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
	}

	sortDeploymentLocations(locations)
	location := locations[0]

	ips, err := resolvePreferredNodeIPs(location.NodeID, location.NodeIP, nodeIPMap)
	if err != nil {
		return nil, err
	}

	return ips, nil
}

// GetGameServerNodeIP returns the preferred IPs for a game server based on where it's running.
// It prefers the actual node IP recorded on the selected location, then falls back to the
// node metadata IP, and only uses region->NODE_IPS mapping as a compatibility fallback.
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
	}

	sortGameServerLocations(locations)
	location := locations[0]

	ips, err := resolvePreferredNodeIPs(location.NodeID, location.NodeIP, nodeIPMap)
	if err != nil {
		return nil, err
	}

	return ips, nil
}

func resolvePreferredNodeIPs(nodeID, explicitNodeIP string, nodeIPMap map[string][]string) ([]string, error) {
	if explicitNodeIP = strings.TrimSpace(explicitNodeIP); explicitNodeIP != "" {
		return []string{explicitNodeIP}, nil
	}

	var node NodeMetadata
	var nodeRegion string

	if err := DB.First(&node, "id = ?", nodeID).Error; err != nil {
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
			return nil, fmt.Errorf("node %s not found and no default node IP configured", nodeID)
		}
		return nil, fmt.Errorf("failed to find node %s: %w", nodeID, err)
	}

	if node.IP = strings.TrimSpace(node.IP); node.IP != "" {
		return []string{node.IP}, nil
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
		return nil, fmt.Errorf("node %s has no region configured and no default region found", nodeID)
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

// GetDatabaseNodeIP returns the node IPs for a managed database domain.
// Database domains point to proxy/ingress node IPs and should resolve for any
// provisioned (non-deleted) database.
func GetDatabaseNodeIP(databaseID string, nodeIPMap map[string][]string) ([]string, error) {
	var dbInstance DatabaseInstance
	if err := DB.Where("id = ? AND deleted_at IS NULL", databaseID).First(&dbInstance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("database %s not found", databaseID)
		}
		return nil, fmt.Errorf("failed to query database %s: %w", databaseID, err)
	}

	if dbInstance.NodeID != nil && *dbInstance.NodeID != "" {
		var node NodeMetadata
		if err := DB.First(&node, "id = ?", *dbInstance.NodeID).Error; err == nil {
			nodeRegion := node.Region
			if nodeRegion != "" {
				if ips, ok := nodeIPMap[nodeRegion]; ok && len(ips) > 0 {
					return ips, nil
				}
			}
		}
	}

	if ips, ok := nodeIPMap["default"]; ok && len(ips) > 0 {
		return ips, nil
	}

	for _, ips := range nodeIPMap {
		if len(ips) > 0 {
			return ips, nil
		}
	}

	return nil, fmt.Errorf("no node IPs configured for database %s", databaseID)
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

// GetGameServerLocationForDNS returns the best game server location for DNS resolution.
// It first prefers active locations, then optionally falls back to a recently updated
// stale location for a short grace period after stop/restart transitions.
// Returns:
//   - nodeIP: resolved node IP/hostname
//   - port: game server port
//   - isStale: true when serving a stale (non-active) location
//   - staleRemaining: remaining grace period when isStale=true
func GetGameServerLocationForDNS(gameServerID string, staleGracePeriod time.Duration) (nodeIP string, port int32, isStale bool, staleRemaining time.Duration, err error) {
	preferredStatuses := []string{"running", "restarting", "starting", "created"}

	// 1) Prefer active/recovering locations first.
	var activeLocation GameServerLocation
	activeResult := DB.Where("game_server_id = ? AND status IN ?", gameServerID, preferredStatuses).
		Order("updated_at DESC").
		First(&activeLocation)
	if activeResult.Error == nil {
		nodeIP, err = resolveGameServerLocationNodeIP(activeLocation, gameServerID)
		if err != nil {
			return "", 0, false, 0, err
		}
		return nodeIP, activeLocation.Port, false, 0, nil
	}
	if activeResult.Error != nil && !errors.Is(activeResult.Error, gorm.ErrRecordNotFound) {
		return "", 0, false, 0, fmt.Errorf("failed to query active game server location: %w", activeResult.Error)
	}

	// 2) Optional stale fallback (for short DNS continuity after stop).
	if staleGracePeriod <= 0 {
		return "", 0, false, 0, fmt.Errorf("no active game server found for game_server_id: %s", gameServerID)
	}

	cutoff := time.Now().Add(-staleGracePeriod)
	var recentLocation GameServerLocation
	recentResult := DB.Where("game_server_id = ? AND updated_at >= ?", gameServerID, cutoff).
		Order("updated_at DESC").
		First(&recentLocation)
	if recentResult.Error != nil {
		if errors.Is(recentResult.Error, gorm.ErrRecordNotFound) {
			return "", 0, false, 0, fmt.Errorf("no recent game server location found for game_server_id: %s", gameServerID)
		}
		return "", 0, false, 0, fmt.Errorf("failed to query recent game server location: %w", recentResult.Error)
	}

	nodeIP, err = resolveGameServerLocationNodeIP(recentLocation, gameServerID)
	if err != nil {
		return "", 0, false, 0, err
	}

	remaining := recentLocation.UpdatedAt.Add(staleGracePeriod).Sub(time.Now())
	if remaining < time.Second {
		remaining = time.Second
	}

	return nodeIP, recentLocation.Port, true, remaining, nil
}

func resolveGameServerLocationNodeIP(location GameServerLocation, gameServerID string) (string, error) {
	// If NodeIP is not set, try to get it from NodeMetadata
	if location.NodeIP == "" {
		var node NodeMetadata
		if err := DB.First(&node, "id = ?", location.NodeID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", fmt.Errorf("node %s not found for game server %s", location.NodeID, gameServerID)
			}
			return "", fmt.Errorf("failed to find node %s: %w", location.NodeID, err)
		}

		if node.IP != "" {
			return node.IP, nil
		}

		if location.NodeHostname != "" {
			return location.NodeHostname, nil
		}

		return "", fmt.Errorf("node %s has no IP address configured for game server %s", location.NodeID, gameServerID)
	}

	return location.NodeIP, nil
}

func isStoppedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "stopped", "exited", "dead", "removing":
		return true
	default:
		return false
	}
}

func locationStatusPriority(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return 0
	case "restarting":
		return 1
	case "starting":
		return 2
	case "created":
		return 3
	default:
		return 4
	}
}

func sortDeploymentLocations(locations []DeploymentLocation) {
	sort.SliceStable(locations, func(i, j int) bool {
		left := locations[i]
		right := locations[j]

		leftPriority := locationStatusPriority(left.Status)
		rightPriority := locationStatusPriority(right.Status)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.After(right.CreatedAt)
		}
		return left.ID < right.ID
	})
}

func sortGameServerLocations(locations []GameServerLocation) {
	sort.SliceStable(locations, func(i, j int) bool {
		left := locations[i]
		right := locations[j]

		leftPriority := locationStatusPriority(left.Status)
		rightPriority := locationStatusPriority(right.Status)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.After(right.CreatedAt)
		}
		return left.ID < right.ID
	})
}
