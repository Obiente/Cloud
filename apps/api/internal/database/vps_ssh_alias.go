package database

// ResolveVPSIDFromSSHIdentifier resolves a VPS ID from either a full VPS ID or an SSH alias.
// Returns the actual VPS ID if found, or an error if not found.
// The identifier can be:
// - A full VPS ID (e.g., "vps-1763004383291811536")
// - An SSH alias (e.g., "prod-db", "web-1")
func ResolveVPSIDFromSSHIdentifier(identifier string, organizationID string) (string, error) {
	// If it starts with "vps-", it's already a VPS ID
	if len(identifier) > 4 && identifier[:4] == "vps-" {
		// Verify it exists
		var vps VPSInstance
		if err := DB.Where("id = ? AND organization_id = ? AND deleted_at IS NULL", identifier, organizationID).First(&vps).Error; err != nil {
			return "", err
		}
		return identifier, nil
	}

	// Otherwise, treat it as an alias
	var vps VPSInstance
	if err := DB.Where("ssh_alias = ? AND organization_id = ? AND deleted_at IS NULL", identifier, organizationID).First(&vps).Error; err != nil {
		return "", err
	}
	return vps.ID, nil
}

// ResolveVPSIDFromSSHIdentifierAnyOrg resolves a VPS ID from either a full VPS ID or an SSH alias,
// without requiring organization ID (for use in SSH proxy where org might not be known yet).
// Returns the actual VPS ID if found, or an error if not found.
func ResolveVPSIDFromSSHIdentifierAnyOrg(identifier string) (string, error) {
	// If it starts with "vps-", it's already a VPS ID
	if len(identifier) > 4 && identifier[:4] == "vps-" {
		// Verify it exists
		var vps VPSInstance
		if err := DB.Where("id = ? AND deleted_at IS NULL", identifier).First(&vps).Error; err != nil {
			return "", err
		}
		return identifier, nil
	}

	// Otherwise, treat it as an alias
	var vps VPSInstance
	if err := DB.Where("ssh_alias = ? AND deleted_at IS NULL", identifier).First(&vps).Error; err != nil {
		return "", err
	}
	return vps.ID, nil
}
