package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// Ssh operations

func formatSSHKeysForProxmox(sshKeys []database.SSHKey) string {
	if len(sshKeys) == 0 {
		return ""
	}

	// Deduplicate keys by fingerprint (prefer VPS-specific over org-wide)
	seenFingerprints := make(map[string]bool)
	deduplicatedKeys := make([]database.SSHKey, 0)
	for _, key := range sshKeys {
		if !seenFingerprints[key.Fingerprint] {
			seenFingerprints[key.Fingerprint] = true
			deduplicatedKeys = append(deduplicatedKeys, key)
		} else {
			// Duplicate fingerprint - prefer VPS-specific over org-wide
			for i, existingKey := range deduplicatedKeys {
				if existingKey.Fingerprint == key.Fingerprint {
					// If the existing key is org-wide and the new one is VPS-specific, replace it
					if existingKey.VPSID == nil && key.VPSID != nil {
						deduplicatedKeys[i] = key
						logger.Debug("[ProxmoxClient] Preferring VPS-specific key %s over org-wide key %s (fingerprint: %s) for Proxmox", key.ID, existingKey.ID, key.Fingerprint)
					}
					break
				}
			}
		}
	}

	var sshKeysStr strings.Builder
	keyCount := 0
	for _, key := range deduplicatedKeys {
		// Aggressively clean the key: remove ALL whitespace, newlines, carriage returns
		trimmedKey := strings.TrimSpace(key.PublicKey)
		// Remove ALL newlines and carriage returns (keys must be single-line)
		trimmedKey = strings.ReplaceAll(trimmedKey, "\n", "")
		trimmedKey = strings.ReplaceAll(trimmedKey, "\r", "")
		trimmedKey = strings.ReplaceAll(trimmedKey, "\t", "")
		// Remove any other control characters
		trimmedKey = strings.TrimSpace(trimmedKey)
		if trimmedKey == "" {
			continue // Skip empty keys
		}

		// Check if key already has a comment (SSH keys can have format: "key-type key-data comment")
		// If it doesn't have a comment, add the key name as a comment
		keyParts := strings.Fields(trimmedKey)
		if len(keyParts) >= 2 {
			// Key has at least type and data, check if it has a comment
			if len(keyParts) == 2 {
				// No comment, add the key name as comment
				// Clean the key name to remove any characters that might cause issues
				cleanName := strings.TrimSpace(key.Name)
				// Remove spaces and special characters that might break the key format
				cleanName = strings.ReplaceAll(cleanName, " ", "-")
				cleanName = strings.ReplaceAll(cleanName, "\n", "")
				cleanName = strings.ReplaceAll(cleanName, "\r", "")
				if cleanName != "" {
					trimmedKey = fmt.Sprintf("%s %s", trimmedKey, cleanName)
				}
			}
			// If key already has a comment (len > 2), keep it as-is
		}

		if keyCount > 0 {
			// Only add newline BETWEEN keys, not after the last one
			sshKeysStr.WriteString("\n")
		}
		// Use raw SSH public key with name as comment
		sshKeysStr.WriteString(trimmedKey)
		keyCount++
	}

	// Get the final string and AGGRESSIVELY ensure no trailing newline
	sshKeysValue := sshKeysStr.String()
	// Multiple passes to ensure absolutely no trailing newlines
	for strings.HasSuffix(sshKeysValue, "\r\n") || strings.HasSuffix(sshKeysValue, "\n") || strings.HasSuffix(sshKeysValue, "\r") {
		sshKeysValue = strings.TrimSuffix(sshKeysValue, "\r\n")
		sshKeysValue = strings.TrimSuffix(sshKeysValue, "\n")
		sshKeysValue = strings.TrimSuffix(sshKeysValue, "\r")
	}
	// Final trim of any trailing whitespace
	sshKeysValue = strings.TrimRight(sshKeysValue, " \t\n\r")

	return sshKeysValue
}

func encodeSSHKeysForProxmox(sshKeysValue string) string {
	if sshKeysValue == "" {
		return ""
	}

	// Clean the value: split by newlines (for multiple keys), clean each, rejoin
	// This preserves newlines BETWEEN keys but removes trailing ones
	keyLines := strings.Split(sshKeysValue, "\n")
	var cleanedLines []string
	for _, line := range keyLines {
		// Clean each line: remove carriage returns and trim
		line = strings.ReplaceAll(line, "\r", "")
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}
	// Rejoin with newlines (only between keys, NOT at the end)
	cleanValue := strings.Join(cleanedLines, "\n")
	// Remove trailing newline if present
	cleanValue = strings.TrimRight(cleanValue, " \t\n\r")

	// Proxmox v8.4 requires sshkeys to be DOUBLE URL-encoded
	// First encode: spaces become %20, + becomes %2B, / becomes %2F
	firstEncoded := url.QueryEscape(cleanValue)
	firstEncoded = strings.ReplaceAll(firstEncoded, "+", "%20")
	// Second encode: %20 becomes %2520, %2B becomes %252B, %2F becomes %252F
	encodedValue := url.QueryEscape(firstEncoded)
	// Replace + with %20 in the double-encoded value
	encodedValue = strings.ReplaceAll(encodedValue, "+", "%20")

	return encodedValue
}

func (pc *ProxmoxClient) GetVMSSHKeys(ctx context.Context, nodeName string, vmID int) (string, error) {
	vmConfig, err := pc.GetVMConfig(ctx, nodeName, vmID)
	if err != nil {
		return "", fmt.Errorf("failed to get VM config: %w", err)
	}

	if sshKeysRaw, ok := vmConfig["sshkeys"].(string); ok && sshKeysRaw != "" {
		return sshKeysRaw, nil
	}

	return "", nil // No SSH keys configured
}

func (pc *ProxmoxClient) SeedSSHKeysFromProxmox(ctx context.Context, sshKeysRaw string, organizationID string, vpsID string) error {
	// Build a map of fingerprints that exist in Proxmox
	proxmoxFingerprints := make(map[string]bool)
	seededCount := 0
	deletedCount := 0

	// If Proxmox has keys, parse them
	if sshKeysRaw != "" {
		// URL-decode the value (Proxmox stores it URL-encoded)
		decoded, err := url.QueryUnescape(sshKeysRaw)
		if err != nil {
			// If decoding fails, try using it as-is (might already be decoded)
			decoded = sshKeysRaw
			logger.Debug("[ProxmoxClient] Failed to URL-decode sshkeys, using as-is: %v", err)
		}

		// Split by newlines to get individual keys
		keyLines := strings.Split(decoded, "\n")

		for _, keyLine := range keyLines {
			// Clean the key line
			keyLine = strings.TrimSpace(keyLine)
			keyLine = strings.ReplaceAll(keyLine, "\r", "")
			if keyLine == "" {
				continue
			}

			// Parse the SSH key to validate it and get fingerprint
			parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyLine))
			if err != nil {
				logger.Debug("[ProxmoxClient] Failed to parse SSH key from Proxmox (skipping): %v", err)
				continue
			}

			// Calculate fingerprint
			fingerprint := ssh.FingerprintSHA256(parsedKey)

			// Track this fingerprint as existing in Proxmox
			proxmoxFingerprints[fingerprint] = true

			// Extract comment from key if available (for name matching)
			_, comment, _, _, _ := ssh.ParseAuthorizedKey([]byte(keyLine))

			// Check if key already exists in database - check both VPS-specific and org-wide
			// We need to find the key that matches the scope we're seeding for
			var existingKey database.SSHKey
			var foundKey bool

			if vpsID != "" {
				// Seeding for a specific VPS - first check for VPS-specific key
				err = database.DB.Where("organization_id = ? AND fingerprint = ? AND vps_id = ?", organizationID, fingerprint, vpsID).First(&existingKey).Error
				if err == nil {
					foundKey = true
				} else if errors.Is(err, gorm.ErrRecordNotFound) {
					// VPS-specific key doesn't exist - check for org-wide key
					err = database.DB.Where("organization_id = ? AND fingerprint = ? AND vps_id IS NULL", organizationID, fingerprint).First(&existingKey).Error
					if err == nil {
						foundKey = true
						// Found org-wide key - don't update its name from VPS seeding
						// The org-wide key should keep its own name
						logger.Debug("[ProxmoxClient] Key with fingerprint %s exists as org-wide key %s - skipping name update (VPS-specific seeding)", fingerprint, existingKey.ID)
					}
				}
			} else {
				// Seeding for org-wide - only check for org-wide key
				err = database.DB.Where("organization_id = ? AND fingerprint = ? AND vps_id IS NULL", organizationID, fingerprint).First(&existingKey).Error
				if err == nil {
					foundKey = true
				}
			}

			if foundKey {
				// Key exists - update name only if it matches the scope
				// Don't update org-wide key name when seeding from VPS-specific context
				shouldUpdateName := true
				if vpsID != "" && existingKey.VPSID == nil {
					// We're seeding for a VPS, but found an org-wide key
					// Don't update the org-wide key's name - it should keep its own name
					shouldUpdateName = false
				}

				if shouldUpdateName && comment != "" {
					// Proxmox has a comment - use it as the name (remove "Imported: " prefix if present)
					oldName := existingKey.Name
					needsUpdate := false

					if strings.HasPrefix(existingKey.Name, "Imported: ") {
						// If current name starts with "Imported: ", compare without that prefix
						currentNameWithoutPrefix := strings.TrimPrefix(existingKey.Name, "Imported: ")
						if currentNameWithoutPrefix != comment {
							needsUpdate = true
						}
					} else if existingKey.Name != comment {
						needsUpdate = true
					}

					if needsUpdate {
						// Name in Proxmox differs from DB - update DB to match Proxmox
						existingKey.Name = comment
						if err := database.DB.Save(&existingKey).Error; err != nil {
							logger.Warn("[ProxmoxClient] Failed to update SSH key name from Proxmox comment: %v", err)
						} else {
							logger.Info("[ProxmoxClient] Updated SSH key %s name from '%s' to '%s' (from Proxmox comment)", existingKey.ID, oldName, comment)
						}
					}
				}
				// Key exists, skip seeding
				continue
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				// Unexpected error
				logger.Warn("[ProxmoxClient] Error checking for existing SSH key: %v", err)
				continue
			}

			// Key doesn't exist in database - seed it
			// Comment was already extracted above

			// Generate a name for the key (use comment if available, otherwise use fingerprint)
			seedName := "Imported from Proxmox"
			if comment != "" {
				seedName = fmt.Sprintf("Imported: %s", comment)
			}

			keyID := fmt.Sprintf("ssh-%d", time.Now().UnixNano())
			var vpsIDPtr *string
			if vpsID != "" {
				vpsIDPtr = &vpsID
			}

			sshKey := database.SSHKey{
				ID:             keyID,
				OrganizationID: organizationID,
				VPSID:          vpsIDPtr,
				Name:           seedName,
				PublicKey:      keyLine,
				Fingerprint:    fingerprint,
			}

			if err := database.DB.Create(&sshKey).Error; err != nil {
				logger.Warn("[ProxmoxClient] Failed to seed SSH key to database: %v", err)
				continue
			}

			// Create audit log entry for seeded key (system action)
			go createSeededKeyAuditLog(organizationID, vpsID, keyID, fingerprint)

			seededCount++
			logger.Info("[ProxmoxClient] Seeded SSH key %s from Proxmox to database (fingerprint: %s)", keyID, fingerprint)
		}

		if seededCount > 0 {
			logger.Info("[ProxmoxClient] Seeded %d SSH key(s) from Proxmox to database", seededCount)
		}
	}

	// Delete keys from database that are NOT in Proxmox (Proxmox is the source of truth)
	// Get all keys for this organization/VPS from database
	var dbKeys []database.SSHKey
	query := database.DB.Where("organization_id = ?", organizationID)
	if vpsID != "" {
		query = query.Where("vps_id = ? OR vps_id IS NULL", vpsID)
	} else {
		query = query.Where("vps_id IS NULL")
	}
	if err := query.Find(&dbKeys).Error; err != nil {
		logger.Warn("[ProxmoxClient] Failed to fetch keys from database for cleanup: %v", err)
	} else {
		// Check each DB key - if it's not in Proxmox, delete it
		for _, dbKey := range dbKeys {
			if !proxmoxFingerprints[dbKey.Fingerprint] {
				// Key exists in DB but not in Proxmox - delete it
				if err := database.DB.Delete(&dbKey).Error; err != nil {
					logger.Warn("[ProxmoxClient] Failed to delete key %s from database (not in Proxmox): %v", dbKey.ID, err)
				} else {
					deletedCount++
					logger.Info("[ProxmoxClient] Deleted SSH key %s from database (fingerprint: %s) - it no longer exists in Proxmox", dbKey.ID, dbKey.Fingerprint)
				}
			}
		}
	}

	if deletedCount > 0 {
		logger.Info("[ProxmoxClient] Deleted %d SSH key(s) from database that no longer exist in Proxmox", deletedCount)
	}

	return nil
}

func (pc *ProxmoxClient) UpdateVMSSHKeys(ctx context.Context, nodeName string, vmID int, organizationID string, vpsID string, excludeKeyID ...string) error {
	// NOTE: We don't seed keys here because:
	// 1. If we seed before updating, deleted keys will be re-imported
	// 2. If we seed after updating, we'd be seeding the keys we just set
	// Seeding should be done separately, e.g., on VPS creation or explicit sync

	// Fetch SSH keys (VPS-specific + org-wide if vpsID provided, or just org-wide if empty)
	var sshKeys []database.SSHKey
	var err error
	if vpsID != "" {
		sshKeys, err = database.GetSSHKeysForVPS(organizationID, vpsID)
	} else {
		sshKeys, err = database.GetSSHKeysForOrganization(organizationID)
	}
	if err != nil {
		return fmt.Errorf("failed to fetch SSH keys: %w", err)
	}

	// Exclude the specified key ID if provided (e.g., when deleting a key)
	originalKeyCount := len(sshKeys)
	if len(excludeKeyID) > 0 && excludeKeyID[0] != "" {
		filteredKeys := make([]database.SSHKey, 0, len(sshKeys))
		excludedCount := 0
		for _, key := range sshKeys {
			if key.ID != excludeKeyID[0] {
				filteredKeys = append(filteredKeys, key)
			} else {
				excludedCount++
				logger.Info("[ProxmoxClient] Excluding key %s (fingerprint: %s) from Proxmox update (key being deleted)", key.ID, key.Fingerprint)
			}
		}
		sshKeys = filteredKeys
		if excludedCount == 0 {
			logger.Warn("[ProxmoxClient] Key %s was not found in the key list to exclude - it may have already been deleted", excludeKeyID[0])
		}
		logger.Info("[ProxmoxClient] Excluding key %s: %d keys before, %d keys after exclusion", excludeKeyID[0], originalKeyCount, len(sshKeys))
	}

	// Format SSH keys using reusable function
	sshKeysValue := formatSSHKeysForProxmox(sshKeys)

	// Update VM config with SSH keys
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	formData := url.Values{}

	if len(sshKeysValue) > 0 {
		// Encode SSH keys using reusable function (double-encoding for Proxmox v8.4)
		encodedValue := encodeSSHKeysForProxmox(sshKeysValue)

		// Verify decoded value has no newlines (for debugging)
		if decoded, err := url.QueryUnescape(encodedValue); err == nil {
			// Double-decode to get back to original
			if decoded2, err2 := url.QueryUnescape(decoded); err2 == nil {
				if strings.Contains(decoded2, "\n") || strings.Contains(decoded2, "\r") {
					logger.Error("[ProxmoxClient] ERROR: Decoded value contains newlines! Raw: %q, Decoded: %q", sshKeysValue, decoded2)
				}
			}
		}

		formData.Set("sshkeys", encodedValue)
		logger.Info("[ProxmoxClient] Updating SSH keys for VM %d (org: %s) - %d key(s)", vmID, organizationID, len(sshKeys))
		logger.Debug("[ProxmoxClient] SSH keys raw length: %d chars, encoded length: %d chars", len(sshKeysValue), len(encodedValue))
		logger.Debug("[ProxmoxClient] SSH keys ends with newline: %v, contains newline: %v, contains carriage return: %v", strings.HasSuffix(sshKeysValue, "\n"), strings.Contains(sshKeysValue, "\n"), strings.Contains(sshKeysValue, "\r"))
		// Log a preview of the actual value (first 100 chars)
		preview := sshKeysValue
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		logger.Debug("[ProxmoxClient] SSH keys preview (raw): %q", preview)
		logger.Debug("[ProxmoxClient] SSH keys encoded: %s", encodedValue)
	} else {
		// If no SSH keys remain after exclusion, we need to clear the sshkeys parameter
		// Don't include sshkeys in the PUT request - Proxmox should keep existing values if parameter is omitted
		// But we want to clear it, so we need to explicitly delete it
		logger.Info("[ProxmoxClient] Clearing SSH keys for VM %d (org: %s) - no keys remain after exclusion", vmID, organizationID)

		// Use PUT with delete=sshkeys query parameter (this is how Proxmox web UI does it)
		// PUT /nodes/{node}/qemu/{vmid}/config?delete=sshkeys
		deleteEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config?delete=sshkeys", nodeName, vmID)
		logger.Debug("[ProxmoxClient] Attempting PUT with delete=sshkeys query parameter: %s", deleteEndpoint)
		// Use empty form data for the PUT request
		emptyFormData := url.Values{}
		deleteResp, deleteErr := pc.apiRequestForm(ctx, "PUT", deleteEndpoint, emptyFormData)
		if deleteErr == nil && deleteResp != nil {
			defer deleteResp.Body.Close()
			if deleteResp.StatusCode == http.StatusOK {
				logger.Info("[ProxmoxClient] Successfully cleared SSH keys for VM %d using PUT with delete=sshkeys", vmID)
				// Verify the deletion after a short delay
				time.Sleep(500 * time.Millisecond)
				verifyKeys, err := pc.GetVMSSHKeys(ctx, nodeName, vmID)
				if err == nil {
					if verifyKeys == "" {
						logger.Info("[ProxmoxClient] Verified: Proxmox now has no SSH keys configured for VM %d", vmID)
						return nil
					} else {
						logger.Warn("[ProxmoxClient] Verified: Proxmox still has SSH keys after PUT with delete=sshkeys, will try fallback")
					}
				}
			} else {
				body, _ := io.ReadAll(deleteResp.Body)
				logger.Debug("[ProxmoxClient] PUT with delete=sshkeys returned status %d: %s", deleteResp.StatusCode, string(body))
			}
		} else {
			if deleteErr != nil {
				logger.Debug("[ProxmoxClient] PUT with delete=sshkeys failed: %v", deleteErr)
			}
		}

		// Fallback: send PUT with empty sshkeys value (in case delete=sshkeys doesn't work)
		// Proxmox requires at least one parameter, so we must explicitly set sshkeys to empty string
		formData = url.Values{}
		// Set sshkeys to empty string - Proxmox should clear it when it receives an empty value
		// Double-encode as required by Proxmox API
		encodedEmpty := url.QueryEscape("")
		encodedEmpty = url.QueryEscape(encodedEmpty)
		formData.Set("sshkeys", encodedEmpty)
		logger.Info("[ProxmoxClient] PUT with delete=sshkeys didn't work, will try PUT with empty sshkeys value for VM %d", vmID)
	}

	// Use PUT as per Proxmox web UI behavior (they use PUT for config updates)
	logger.Info("[ProxmoxClient] Sending PUT request to %s to update SSH keys (excluded key: %v, sending %d keys)", endpoint, len(excludeKeyID) > 0 && excludeKeyID[0] != "", len(sshKeys))
	resp, err := pc.apiRequestForm(ctx, "PUT", endpoint, formData)
	if err != nil {
		logger.Error("[ProxmoxClient] Failed to send request to Proxmox: %v", err)
		return fmt.Errorf("failed to update SSH keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errorBody := string(body)
		logger.Error("[ProxmoxClient] Proxmox returned non-OK status %d: %s", resp.StatusCode, errorBody)

		// Check if this is the known Proxmox v8.4 sshkeys parsing bug
		// Even though we send valid data, Proxmox may report a false newline error
		if strings.Contains(errorBody, "invalid urlencoded string") && strings.Contains(errorBody, "sshkeys") {
			logger.Warn("[ProxmoxClient] Proxmox v8.4 sshkeys parsing error (possible bug). Error: %s", errorBody)
			return fmt.Errorf("failed to update SSH keys (Proxmox v8.4 sshkeys parsing issue): %s (status: %d)", errorBody, resp.StatusCode)
		}

		return fmt.Errorf("failed to update SSH keys: %s (status: %d)", errorBody, resp.StatusCode)
	}

	logger.Info("[ProxmoxClient] Successfully updated SSH keys for VM %d (org: %s) - %d key(s) sent to Proxmox", vmID, organizationID, len(sshKeys))

	// Verify the update by fetching the keys back from Proxmox
	// If Proxmox has keys that aren't in our database, we need to clear them
	// This ensures Proxmox matches our database (the source of truth)
	verifyKeys, err := pc.GetVMSSHKeys(ctx, nodeName, vmID)
	if err == nil {
		if verifyKeys == "" {
			logger.Info("[ProxmoxClient] Verified: Proxmox now has no SSH keys configured for VM %d", vmID)
		} else {
			// Parse keys from Proxmox and check if they all exist in our database
			decodedVerify, _ := url.QueryUnescape(verifyKeys)
			verifyKeyLines := strings.Split(decodedVerify, "\n")
			proxmoxKeyFingerprints := make(map[string]bool)
			for _, line := range verifyKeyLines {
				line = strings.TrimSpace(line)
				line = strings.ReplaceAll(line, "\r", "")
				if line == "" {
					continue
				}
				// Parse the key to get fingerprint
				parsedKey, _, _, _, parseErr := ssh.ParseAuthorizedKey([]byte(line))
				if parseErr == nil {
					fingerprint := ssh.FingerprintSHA256(parsedKey)
					proxmoxKeyFingerprints[fingerprint] = true
				}
			}

			// Check which Proxmox keys exist in our database
			expectedFingerprints := make(map[string]bool)
			for _, key := range sshKeys {
				expectedFingerprints[key.Fingerprint] = true
			}

			// Find keys in Proxmox that aren't in our database
			extraKeys := make([]string, 0)
			for fp := range proxmoxKeyFingerprints {
				if !expectedFingerprints[fp] {
					extraKeys = append(extraKeys, fp)
				}
			}

			if len(extraKeys) > 0 {
				// Proxmox still has keys that shouldn't be there - deletion failed
				// Return an error so the caller knows the deletion didn't work
				return fmt.Errorf("failed to clear SSH keys from Proxmox: Proxmox still has %d key(s) that should have been deleted (fingerprints: %v). This may be due to a Proxmox v8.4 bug where empty sshkeys parameter doesn't clear the keys", len(extraKeys), extraKeys)
			}

			logger.Info("[ProxmoxClient] Verified: Proxmox SSH keys match our database (%d keys)", len(sshKeys))
		}
	} else {
		logger.Warn("[ProxmoxClient] Failed to verify SSH keys in Proxmox after update: %v", err)
	}

	return nil
}

func createSeededKeyAuditLog(organizationID string, vpsID string, keyID string, fingerprint string) {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[ProxmoxClient] Panic creating audit log for seeded key: %v", r)
		}
	}()

	// Use background context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use MetricsDB (TimescaleDB) for audit logs
	if database.MetricsDB == nil {
		logger.Warn("[ProxmoxClient] Metrics database not available, skipping audit log for seeded key")
		return
	}

	// Determine resource type and ID
	var resourceType *string
	var resourceID *string
	if vpsID != "" {
		rt := "vps"
		resourceType = &rt
		resourceID = &vpsID
	} else {
		rt := "organization"
		resourceType = &rt
		resourceID = &organizationID
	}

	// Create audit log entry
	auditLog := database.AuditLog{
		ID:             fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		UserID:         "system", // System user for seeded keys
		OrganizationID: &organizationID,
		Action:         "SeedSSHKey",
		Service:        "VPSService",
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		IPAddress:      "system",
		UserAgent:      "system",
		RequestData:    fmt.Sprintf(`{"key_id":"%s","fingerprint":"%s","source":"proxmox"}`, keyID, fingerprint),
		ResponseStatus: 200,
		ErrorMessage:   nil,
		DurationMs:     0,
		CreatedAt:      time.Now(),
	}

	if err := database.MetricsDB.WithContext(ctx).Create(&auditLog).Error; err != nil {
		logger.Warn("[ProxmoxClient] Failed to create audit log for seeded key %s: %v", keyID, err)
	} else {
		logger.Debug("[ProxmoxClient] Created audit log for seeded SSH key %s", keyID)
	}
}
