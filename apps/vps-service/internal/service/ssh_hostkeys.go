package vps

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	vpsSSHHostKeyFingerprintMetadataKey = "ssh_host_key_sha256"
	vpsSSHHostKeyPinnedAtMetadataKey    = "ssh_host_key_pinned_at"
	vpsSSHHostKeyCheckTimeout           = 5 * time.Second
)

func newVPSHostKeyCallback(parentCtx context.Context, vpsID string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		ctx := context.Background()
		if parentCtx != nil {
			ctx = parentCtx
		}

		checkCtx, cancel := context.WithTimeout(ctx, vpsSSHHostKeyCheckTimeout)
		defer cancel()

		fingerprint := ssh.FingerprintSHA256(key)
		return validateOrPinVPSHostKey(checkCtx, vpsID, hostname, remote, fingerprint)
	}
}

func validateOrPinVPSHostKey(ctx context.Context, vpsID, hostname string, remote net.Addr, fingerprint string) error {
	return database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var vps database.VPSInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "metadata").
			Where("id = ? AND deleted_at IS NULL", vpsID).
			First(&vps).Error; err != nil {
			return fmt.Errorf("load VPS metadata: %w", err)
		}

		metadata, err := parseVPSMetadata(vps.Metadata)
		if err != nil {
			return err
		}

		pinnedFingerprint, _ := metadata[vpsSSHHostKeyFingerprintMetadataKey].(string)
		if pinnedFingerprint == "" {
			metadata[vpsSSHHostKeyFingerprintMetadataKey] = fingerprint
			metadata[vpsSSHHostKeyPinnedAtMetadataKey] = time.Now().UTC().Format(time.RFC3339Nano)

			metadataJSON, err := json.Marshal(metadata)
			if err != nil {
				return fmt.Errorf("marshal VPS metadata: %w", err)
			}

			if err := tx.Model(&database.VPSInstance{}).
				Where("id = ?", vpsID).
				Updates(map[string]interface{}{
					"metadata":   string(metadataJSON),
					"updated_at": time.Now(),
				}).Error; err != nil {
				return fmt.Errorf("store pinned SSH host key fingerprint: %w", err)
			}

			logger.Info("[VPS SSH] Pinned guest SSH host key for VPS %s (%s via %s remote=%v)", vpsID, fingerprint, hostname, remote)
			return nil
		}

		if pinnedFingerprint != fingerprint {
			return fmt.Errorf("SSH host key mismatch for VPS %s: expected %s, got %s", vpsID, pinnedFingerprint, fingerprint)
		}

		return nil
	})
}

func parseVPSMetadata(raw string) (map[string]interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return map[string]interface{}{}, nil
	}

	metadata := make(map[string]interface{})
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, fmt.Errorf("parse VPS metadata JSON: %w", err)
	}

	return metadata, nil
}
