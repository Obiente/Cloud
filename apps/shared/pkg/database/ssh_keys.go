package database

// GetSSHKeysForOrganization gets all organization-wide SSH keys (internal use)
func GetSSHKeysForOrganization(orgID string) ([]SSHKey, error) {
	var keys []SSHKey
	if err := DB.Where("organization_id = ? AND vps_id IS NULL", orgID).Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// GetSSHKeysForVPS gets all SSH keys for a VPS instance (includes VPS-specific and org-wide keys)
func GetSSHKeysForVPS(orgID string, vpsID string) ([]SSHKey, error) {
	var keys []SSHKey
	// Get both VPS-specific keys and org-wide keys
	if err := DB.Where("organization_id = ? AND (vps_id = ? OR vps_id IS NULL)", orgID, vpsID).Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

