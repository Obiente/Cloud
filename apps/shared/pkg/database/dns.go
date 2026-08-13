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
	return resolveNodeIPs(nodeID, explicitNodeIP, nodeIPMap, true)
}

func resolveAuthoritativeNodeIPs(nodeID, explicitNodeIP string, nodeIPMap map[string][]string) ([]string, error) {
	return resolveNodeIPs(nodeID, explicitNodeIP, nodeIPMap, false)
}

func resolveNodeIPs(nodeID, explicitNodeIP string, nodeIPMap map[string][]string, allowCompatibilityFallback bool) ([]string, error) {
	if explicitNodeIP = strings.TrimSpace(explicitNodeIP); configuredNodeIP(explicitNodeIP, nodeIPMap) {
		return []string{explicitNodeIP}, nil
	}

	var node NodeMetadata
	var nodeRegion string

	if err := DB.First(&node, "id = ?", nodeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if !allowCompatibilityFallback {
				return nil, fmt.Errorf("node %s not found and authoritative DNS fallback is disabled", nodeID)
			}
			// Older records may refer to a node that predates node metadata. Only use
			// an unassigned compatibility fallback when it is unambiguous.
			return compatibilityNodeIPs(nodeIPMap, fmt.Sprintf("node %s not found", nodeID))
		}
		return nil, fmt.Errorf("failed to find node %s: %w", nodeID, err)
	}

	if node.IP = strings.TrimSpace(node.IP); configuredNodeIP(node.IP, nodeIPMap) {
		return []string{node.IP}, nil
	}
	if ips, found, err := nodeSpecificIPs(node, nodeIPMap); err != nil {
		return nil, err
	} else if found {
		return ips, nil
	}

	nodeRegion = node.Region

	if nodeRegion == "" {
		if !allowCompatibilityFallback {
			return nil, fmt.Errorf("node %s has no configured IP or region and authoritative DNS fallback is disabled", nodeID)
		}
		// If node has no region, only use an unambiguous compatibility fallback.
		return compatibilityNodeIPs(nodeIPMap, fmt.Sprintf("node %s has no IP or region", nodeID))
	}

	// Get node IPs for this region
	ips := cleanNodeIPs(nodeIPMap[nodeRegion])
	if len(ips) == 0 {
		// Compatibility callers may use the historical default region. An
		// authoritative database owner must never be replaced by another node.
		if allowCompatibilityFallback {
			defaultIPs := cleanNodeIPs(nodeIPMap["default"])
			if len(defaultIPs) > 0 {
				return defaultIPs, nil
			}
		}
		return nil, fmt.Errorf("no node IP configured for region: %s", nodeRegion)
	}
	if !allowCompatibilityFallback && len(ips) != 1 {
		return nil, fmt.Errorf("region %s contains %d node IPs; configure exactly one NODE_IPS entry for node %s or hostname %s", nodeRegion, len(ips), node.ID, node.Hostname)
	}

	return ips, nil
}

func nodeSpecificIPs(node NodeMetadata, nodeIPMap map[string][]string) ([]string, bool, error) {
	keys := []string{strings.TrimSpace(node.ID), strings.TrimSpace(node.Hostname)}
	for _, key := range keys {
		if key == "" {
			continue
		}
		ips := cleanNodeIPs(nodeIPMap[key])
		if len(ips) == 0 {
			continue
		}
		if len(ips) != 1 {
			return nil, false, fmt.Errorf("NODE_IPS entry %s contains %d addresses; node-specific entries must contain exactly one address", key, len(ips))
		}
		return ips, true, nil
	}
	return nil, false, nil
}

func configuredNodeIP(candidate string, nodeIPMap map[string][]string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return false
	}
	for _, rawIPs := range nodeIPMap {
		for _, configuredIP := range rawIPs {
			if strings.TrimSpace(configuredIP) == candidate {
				return true
			}
		}
	}
	return false
}

func compatibilityNodeIPs(nodeIPMap map[string][]string, reason string) ([]string, error) {
	if ips := cleanNodeIPs(nodeIPMap["default"]); len(ips) > 0 {
		return ips, nil
	}

	var onlyRegionIPs []string
	regions := 0
	for region, rawIPs := range nodeIPMap {
		if region == "default" {
			continue
		}
		ips := cleanNodeIPs(rawIPs)
		if len(ips) == 0 {
			continue
		}
		regions++
		onlyRegionIPs = ips
	}
	if regions == 1 {
		return onlyRegionIPs, nil
	}
	if regions > 1 {
		return nil, fmt.Errorf("%s and NODE_IPS contains %d regions; refusing ambiguous cross-node DNS fallback", reason, regions)
	}
	return nil, fmt.Errorf("%s and no fallback node IP is configured", reason)
}

func cleanNodeIPs(rawIPs []string) []string {
	ips := make([]string, 0, len(rawIPs))
	for _, ip := range rawIPs {
		if ip = strings.TrimSpace(ip); ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}

// GetDatabaseNodeIP returns the node IPs for a managed database domain.
// Database domains point to proxy/ingress node IPs and should resolve for any
// provisioned (non-deleted) database.
func GetDatabaseNodeIP(databaseID string, nodeIPMap map[string][]string) ([]string, error) {
	resolvedID, err := ResolveDatabaseIDByLabel(databaseID)
	if err == nil {
		databaseID = resolvedID
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to resolve database label %s: %w", databaseID, err)
	}

	var dbInstance DatabaseInstance
	if err := DB.Where("id = ? AND deleted_at IS NULL", databaseID).First(&dbInstance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("database %s not found", databaseID)
		}
		return nil, fmt.Errorf("failed to query database %s: %w", databaseID, err)
	}

	var locations []DatabaseLocation
	if err := DB.Where("database_id = ? AND status IN ?", databaseID, []string{"running", "restarting", "starting", "created", "sleeping"}).
		Order("updated_at DESC").
		Find(&locations).Error; err != nil {
		return nil, fmt.Errorf("failed to query database locations: %w", err)
	}
	if len(locations) > 0 {
		location := locations[0]
		ips, err := resolveAuthoritativeNodeIPs(location.NodeID, location.NodeIP, nodeIPMap)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve active database location on node %s: %w", location.NodeID, err)
		}
		return ips, nil
	}

	if dbInstance.NodeID != nil && *dbInstance.NodeID != "" {
		return resolveAuthoritativeNodeIPs(*dbInstance.NodeID, "", nodeIPMap)
	}

	return compatibilityNodeIPs(nodeIPMap, fmt.Sprintf("database %s has no recorded host node", databaseID))
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
// Format: "key1:ip1,ip2;key2:ip3,ip4", where keys may identify regions or nodes.
// Also supports simple format: "ip1,ip2" (defaults to "default" region)
// Also supports space-separated format: "region1:ip1,ip2 region2:ip3,ip4" (when semicolons are not present)
// Returns a map of region or node key -> []IP addresses
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
		// Pattern matches region/node keys, a colon, then IP address(es) optionally separated by commas.
		// This handles formats like:
		// - "us:1.2.3.4 nl:5.6.7.8"
		// - "us 1.2.3.4 nl:5.6.7.8" (region name followed by space and IP)
		// - "us:1.2.3.4,9.10.11.12 nl:5.6.7.8"

		// First, try to find all patterns that match "region:ip" or "region:ip1,ip2"
		// Pattern: one or more word chars, colon, then IP addresses (dots and numbers) possibly separated by commas
		re := regexp.MustCompile(`[A-Za-z0-9_.-]+:\d+\.\d+\.\d+\.\d+(?:,\d+\.\d+\.\d+\.\d+)*`)
		matches := re.FindAllString(nodeIPsEnv, -1)

		if len(matches) > 0 {
			// Found region:ip patterns
			regions = matches
		} else {
			// Fallback: try to handle "region IP" format (region name followed by space and IP)
			// Pattern: word chars (region), space, IP address
			re2 := regexp.MustCompile(`([A-Za-z0-9_.-]+)\s+(\d+\.\d+\.\d+\.\d+)`)
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
