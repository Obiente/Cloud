package database

import (
	"fmt"
	"os"
	"strings"
)

// VPSRegion represents a VPS region from environment configuration
type VPSRegion struct {
	ID        string
	Name      string
	Available bool
}

// ParseVPSRegionsFromEnv parses the VPS_REGIONS environment variable
// Format: "region1:Name 1;region2:Name 2"
// Also supports simple format: "region1" (defaults to region ID as name)
// Returns a slice of VPSRegion
func ParseVPSRegionsFromEnv(vpsRegionsEnv string) ([]VPSRegion, error) {
	var regions []VPSRegion

	if vpsRegionsEnv == "" {
		return nil, fmt.Errorf("VPS_REGIONS environment variable is required")
	}

	// Check if format contains semicolons (multi-region format)
	if !strings.Contains(vpsRegionsEnv, ";") && !strings.Contains(vpsRegionsEnv, ":") {
		// Simple format: just region IDs separated by commas
		regionIDs := strings.Split(vpsRegionsEnv, ",")
		for _, regionID := range regionIDs {
			regionID = strings.TrimSpace(regionID)
			if regionID != "" {
				// Use region ID as name (capitalize first letter of each word)
				name := formatRegionName(regionID)
				regions = append(regions, VPSRegion{
					ID:        regionID,
					Name:      name,
					Available: true,
				})
			}
		}
		return regions, nil
	}

	// Parse semicolon-separated regions
	regionStrings := strings.Split(vpsRegionsEnv, ";")
	for _, regionStr := range regionStrings {
		regionStr = strings.TrimSpace(regionStr)
		if regionStr == "" {
			continue
		}

		// Parse "regionID:Region Name" format
		if strings.Contains(regionStr, ":") {
			parts := strings.SplitN(regionStr, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid format in VPS_REGIONS: %s (expected 'regionID:Region Name')", regionStr)
			}

			regionID := strings.TrimSpace(parts[0])
			regionName := strings.TrimSpace(parts[1])

			if regionID == "" {
				return nil, fmt.Errorf("empty region ID in VPS_REGIONS: %s", regionStr)
			}

			if regionName == "" {
				regionName = formatRegionName(regionID)
			}

			regions = append(regions, VPSRegion{
				ID:        regionID,
				Name:      regionName,
				Available: true,
			})
		} else {
			// No colon - treat as region ID only
			regionID := strings.TrimSpace(regionStr)
			if regionID != "" {
				regions = append(regions, VPSRegion{
					ID:        regionID,
					Name:      formatRegionName(regionID),
					Available: true,
				})
			}
		}
	}

	if len(regions) == 0 {
		return nil, fmt.Errorf("no valid regions found in VPS_REGIONS: %s", vpsRegionsEnv)
	}

	return regions, nil
}

// formatRegionName formats a region ID into a readable name
// e.g., "us-illinois" -> "US Illinois", "us-east-1" -> "US East 1"
func formatRegionName(regionID string) string {
	// Split by hyphens and capitalize each word
	parts := strings.Split(regionID, "-")
	formatted := make([]string, len(parts))
	for i, part := range parts {
		if len(part) > 0 {
			// Capitalize first letter
			formatted[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(formatted, " ")
}

// GetVPSRegionsFromEnv gets VPS regions from environment variable
// Returns an error if VPS_REGIONS is not set or empty
func GetVPSRegionsFromEnv() ([]VPSRegion, error) {
	vpsRegionsEnv := os.Getenv("VPS_REGIONS")
	vpsRegionsEnv = strings.TrimSpace(vpsRegionsEnv)

	if vpsRegionsEnv == "" {
		return nil, fmt.Errorf("VPS_REGIONS environment variable is required but not set or empty. Please set it in your environment or docker-compose file (e.g., VPS_REGIONS=\"us-illinois\")")
	}

	return ParseVPSRegionsFromEnv(vpsRegionsEnv)
}
