package database

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const vpsSSHHostKeyFingerprintMetadataKey = "ssh_host_key_fingerprint_sha256"
const vpsSSHHostKeyPinnedAtMetadataKey = "ssh_host_key_pinned_at"

// VerifyOrPinVPSSSHHostKey implements TOFU for guest VPS SSH host keys.
// On first successful connection it stores the fingerprint in VPS metadata.
// Future connections must present the same fingerprint.
func VerifyOrPinVPSSSHHostKey(vpsID, fingerprint string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var vps VPSInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "metadata").
			Where("id = ? AND deleted_at IS NULL", vpsID).
			First(&vps).Error; err != nil {
			return err
		}

		metadata := map[string]any{}
		rawMetadata := strings.TrimSpace(vps.Metadata)
		if rawMetadata != "" && rawMetadata != "null" {
			if err := json.Unmarshal([]byte(vps.Metadata), &metadata); err != nil {
				return fmt.Errorf("parse VPS metadata: %w", err)
			}
		}

		existing, _ := metadata[vpsSSHHostKeyFingerprintMetadataKey].(string)
		if existing != "" {
			if existing != fingerprint {
				return fmt.Errorf("guest SSH host key mismatch")
			}
			return nil
		}

		metadata[vpsSSHHostKeyFingerprintMetadataKey] = fingerprint
		metadata[vpsSSHHostKeyPinnedAtMetadataKey] = time.Now().UTC().Format(time.RFC3339Nano)
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal VPS metadata: %w", err)
		}

		return tx.Model(&vps).Updates(map[string]any{
			"metadata":   string(metadataJSON),
			"updated_at": time.Now(),
		}).Error
	})
}
