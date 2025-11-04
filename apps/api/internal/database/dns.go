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
