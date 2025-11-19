package vps

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsorch "vps-service/orchestrator"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"connectrpc.com/connect"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// ListSSHKeys lists SSH keys for an organization or a specific VPS
// If vps_id is provided, returns VPS-specific keys + org-wide keys
// If vps_id is not provided, returns only org-wide keys
func (s *Service) ListSSHKeys(ctx context.Context, req *connect.Request[vpsv1.ListSSHKeysRequest]) (*connect.Response[vpsv1.ListSSHKeysResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	var keys []database.SSHKey
	vpsID := req.Msg.GetVpsId()

	if vpsID != "" {
		// List keys for a specific VPS (includes VPS-specific and org-wide keys)
		// Also verify the VPS belongs to the organization
		var vps database.VPSInstance
		if err := database.DB.Where("id = ? AND organization_id = ? AND deleted_at IS NULL", vpsID, orgID).First(&vps).Error; err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS not found or access denied"))
		}

		// Seed SSH keys from Proxmox if VPS has an instance ID
		if vps.InstanceID != nil {
			proxmoxConfig, err := vpsorch.GetProxmoxConfig()
			if err == nil {
				proxmoxClient, err := vpsorch.NewProxmoxClient(proxmoxConfig)
				if err == nil {
					vmIDInt := 0
					fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
					if vmIDInt > 0 {
						// Find the node where the VM is running
						nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
						if err == nil {
							// Get existing SSH keys from Proxmox and seed them
							// After deleting a key, we update Proxmox first, so Proxmox is the source of truth
							existingSSHKeysRaw, err := proxmoxClient.GetVMSSHKeys(ctx, nodeName, vmIDInt)
							if err == nil && existingSSHKeysRaw != "" {
								// Seed all keys from Proxmox into database
								// We trust Proxmox as the source of truth after updates
								if seedErr := proxmoxClient.SeedSSHKeysFromProxmox(ctx, existingSSHKeysRaw, orgID, vpsID); seedErr != nil {
									logger.Warn("[VPS] Failed to seed SSH keys from Proxmox for VPS %s: %v", vpsID, seedErr)
									// Don't fail the request if seeding fails
								}
							}
						}
					}
				}
			}
		}

		keys, err = database.GetSSHKeysForVPS(orgID, vpsID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list SSH keys: %w", err))
		}
		// Don't deduplicate - show both org-wide and VPS-specific keys even if they have the same fingerprint
	} else {
		// List only org-wide keys
		keys, err = database.GetSSHKeysForOrganization(orgID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list SSH keys: %w", err))
		}
	}

	keyProtos := make([]*vpsv1.SSHKey, len(keys))
	for i, key := range keys {
		keyProto := &vpsv1.SSHKey{
			Id:          key.ID,
			Name:        key.Name,
			PublicKey:   key.PublicKey,
			Fingerprint: key.Fingerprint,
			CreatedAt:   timestamppb.New(key.CreatedAt),
			UpdatedAt:   timestamppb.New(key.UpdatedAt),
		}
		if key.VPSID != nil {
			keyProto.VpsId = key.VPSID
		}
		keyProtos[i] = keyProto
	}

	return connect.NewResponse(&vpsv1.ListSSHKeysResponse{
		Keys: keyProtos,
	}), nil
}

// AddSSHKey adds a new SSH public key for an organization
func (s *Service) AddSSHKey(ctx context.Context, req *connect.Request[vpsv1.AddSSHKeyRequest]) (*connect.Response[vpsv1.AddSSHKeyResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	publicKey := strings.TrimSpace(req.Msg.GetPublicKey())
	if publicKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("public_key is required"))
	}

	// Validate and parse SSH public key
	parsedKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid SSH public key: %w", err))
	}

	// Calculate fingerprint (SHA256)
	fingerprint := ssh.FingerprintSHA256(parsedKey)

	vpsID := req.Msg.GetVpsId()
	var vpsIDPtr *string

	// If VPS ID is provided, verify the VPS belongs to the organization
	if vpsID != "" {
		var vps database.VPSInstance
		if err := database.DB.Where("id = ? AND organization_id = ? AND deleted_at IS NULL", vpsID, orgID).First(&vps).Error; err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS not found or access denied"))
		}
		vpsIDPtr = &vpsID
	}

	// Check if key already exists (by fingerprint) in the same scope
	// Allow the same key to be both org-wide and VPS-specific, but prevent duplicates within the same scope
	var existingKey database.SSHKey
	query := database.DB.Where("organization_id = ? AND fingerprint = ?", orgID, fingerprint)
	if vpsID != "" {
		// For VPS-specific keys, check if it already exists for THIS specific VPS
		// Allow it to exist as org-wide, but not as VPS-specific for the same VPS
		query = query.Where("vps_id = ?", vpsID)
	} else {
		// For org-wide keys, check if it already exists as org-wide
		// Allow it to exist as VPS-specific, but not as org-wide again
		query = query.Where("vps_id IS NULL")
	}

	err = query.First(&existingKey).Error
	if err == nil {
		// Key exists in the same scope - duplicate
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("SSH key with this fingerprint already exists: %s", existingKey.Name))
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Unexpected error (not just "not found")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check for existing SSH key: %w", err))
	}
	// err is ErrRecordNotFound - key doesn't exist, we can proceed

	// Generate ID (format: ssh-{timestamp}-{random})
	keyID := fmt.Sprintf("ssh-%d", time.Now().UnixNano())

	// Prepare SSH key record
	sshKey := database.SSHKey{
		ID:             keyID,
		OrganizationID: orgID,
		VPSID:          vpsIDPtr,
		Name:           name,
		PublicKey:      publicKey,
		Fingerprint:    fingerprint,
	}

	// IMPORTANT: Create database record FIRST so UpdateVPSSSHKeys can see it
	// If Proxmox update fails, we'll delete the DB record
	if err := database.DB.Create(&sshKey).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create SSH key record: %w", err))
	}

	// Now update Proxmox with the new key (it's now in the database)
	if vpsID != "" {
		// Update only this specific VPS
		if err := s.vpsManager.UpdateVPSSSHKeys(ctx, vpsID); err != nil {
			// Rollback: delete the database record since Proxmox update failed
			database.DB.Delete(&sshKey)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update SSH keys in Proxmox: %w", err))
		}
	} else {
		// Update all VPS instances in the organization (org-wide key)
		if err := s.vpsManager.UpdateOrganizationVPSSSHKeys(ctx, orgID); err != nil {
			// Rollback: delete the database record since Proxmox update failed
			database.DB.Delete(&sshKey)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update SSH keys in Proxmox: %w", err))
		}
	}

	// Log comment if available (usually contains email or identifier)
	if vpsID != "" {
		if comment != "" {
			logger.Info("[VPS] Added SSH key %s for VPS %s (org: %s, comment: %s)", keyID, vpsID, orgID, comment)
		} else {
			logger.Info("[VPS] Added SSH key %s for VPS %s (org: %s)", keyID, vpsID, orgID)
		}
	} else {
		if comment != "" {
			logger.Info("[VPS] Added organization-wide SSH key %s for organization %s (comment: %s)", keyID, orgID, comment)
		} else {
			logger.Info("[VPS] Added organization-wide SSH key %s for organization %s", keyID, orgID)
		}
	}

	keyProto := &vpsv1.SSHKey{
		Id:          sshKey.ID,
		Name:        sshKey.Name,
		PublicKey:   sshKey.PublicKey,
		Fingerprint: sshKey.Fingerprint,
		CreatedAt:   timestamppb.New(sshKey.CreatedAt),
		UpdatedAt:   timestamppb.New(sshKey.UpdatedAt),
	}
	if sshKey.VPSID != nil {
		keyProto.VpsId = sshKey.VPSID
	}

	return connect.NewResponse(&vpsv1.AddSSHKeyResponse{
		Key: keyProto,
	}), nil
}

// RemoveSSHKey removes an SSH public key
func (s *Service) RemoveSSHKey(ctx context.Context, req *connect.Request[vpsv1.RemoveSSHKeyRequest]) (*connect.Response[vpsv1.RemoveSSHKeyResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	keyID := req.Msg.GetKeyId()

	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	// Verify key belongs to organization
	var key database.SSHKey
	if err := database.DB.Where("id = ? AND organization_id = ?", keyID, orgID).First(&key).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("SSH key not found"))
	}

	// Get affected VPS instances if this is an org-wide key (before deletion)
	var affectedVPSIDs []string
	var affectedVPSNames []string
	if key.VPSID == nil {
		// This is an org-wide key - get all VPS instances in the organization that have this key
		// (VPS instances with instance_id set, meaning they're provisioned in Proxmox)
		var vpsInstances []database.VPSInstance
		if err := database.DB.Where("organization_id = ? AND deleted_at IS NULL AND instance_id IS NOT NULL", orgID).
			Select("id, name").
			Order("name ASC").
			Find(&vpsInstances).Error; err == nil {
			for _, vps := range vpsInstances {
				affectedVPSIDs = append(affectedVPSIDs, vps.ID)
				affectedVPSNames = append(affectedVPSNames, vps.Name)
			}
		}
	}

	// IMPORTANT: Update Proxmox FIRST and verify it actually cleared the key before deleting from database
	// If Proxmox doesn't clear the key, we must NOT delete it from our database to keep them in sync
	logger.Info("[VPS] Removing SSH key %s (fingerprint: %s) - updating Proxmox first", keyID, key.Fingerprint)
	var updateErr error
	if key.VPSID != nil {
		// Update only this specific VPS, excluding the key being deleted
		updateErr = s.vpsManager.UpdateVPSSSHKeysExcluding(ctx, *key.VPSID, keyID)
		if updateErr != nil {
			logger.Error("[VPS] Failed to update SSH keys for VPS %s: %v", *key.VPSID, updateErr)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove SSH key from Proxmox: %w. The key was not deleted from the database to keep it in sync with Proxmox", updateErr))
		}
		logger.Info("[VPS] Successfully updated Proxmox to remove key %s from VPS %s", keyID, *key.VPSID)
	} else {
		// Update all VPS instances in the organization (org-wide key was removed)
		updateErr = s.vpsManager.UpdateOrganizationVPSSSHKeysExcluding(ctx, orgID, keyID)
		if updateErr != nil {
			logger.Error("[VPS] Failed to update SSH keys for existing VPS instances: %v", updateErr)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove SSH key from Proxmox: %w. The key was not deleted from the database to keep it in sync with Proxmox", updateErr))
		}
		logger.Info("[VPS] Successfully updated Proxmox to remove org-wide key %s from organization %s", keyID, orgID)
	}

	// Only delete from database if Proxmox successfully cleared the key
	// The UpdateVMSSHKeys function verifies that Proxmox actually cleared it before returning success
	if err := database.DB.Delete(&key).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete SSH key: %w", err))
	}

	if key.VPSID != nil {
		logger.Info("[VPS] Removed SSH key %s from VPS %s (org: %s)", keyID, *key.VPSID, orgID)
	} else {
		logger.Info("[VPS] Removed organization-wide SSH key %s from organization %s (affected %d VPS instances)", keyID, orgID, len(affectedVPSIDs))
	}

	return connect.NewResponse(&vpsv1.RemoveSSHKeyResponse{
		AffectedVpsIds:   affectedVPSIDs,
		AffectedVpsNames: affectedVPSNames,
	}), nil
}

// UpdateSSHKey updates the name of an SSH key
func (s *Service) UpdateSSHKey(ctx context.Context, req *connect.Request[vpsv1.UpdateSSHKeyRequest]) (*connect.Response[vpsv1.UpdateSSHKeyResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	keyID := req.Msg.GetKeyId()
	newName := req.Msg.GetName()

	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	// Validate name is not empty
	if strings.TrimSpace(newName) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("key name cannot be empty"))
	}

	// Verify key belongs to organization
	var key database.SSHKey
	if err := database.DB.Where("id = ? AND organization_id = ?", keyID, orgID).First(&key).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("SSH key not found"))
	}

	// Update the name in database
	key.Name = strings.TrimSpace(newName)
	if err := database.DB.Save(&key).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update SSH key: %w", err))
	}

	// Update Proxmox to reflect the new name (name is used as comment in Proxmox)
	// The name change needs to be synced to Proxmox so the comment matches
	if key.VPSID != nil {
		// Update only this specific VPS
		if err := s.vpsManager.UpdateVPSSSHKeys(ctx, *key.VPSID); err != nil {
			logger.Warn("[VPS] Failed to update SSH keys in Proxmox after name change for VPS %s: %v", *key.VPSID, err)
			// Don't fail the request - name is updated in DB, Proxmox will sync on next update
		} else {
			logger.Info("[VPS] Successfully updated Proxmox SSH keys with new name for VPS %s", *key.VPSID)
		}
	} else {
		// Update all VPS instances in the organization (org-wide key name changed)
		if err := s.vpsManager.UpdateOrganizationVPSSSHKeys(ctx, orgID); err != nil {
			logger.Warn("[VPS] Failed to update SSH keys in Proxmox after name change for organization %s: %v", orgID, err)
			// Don't fail the request - name is updated in DB, Proxmox will sync on next update
		} else {
			logger.Info("[VPS] Successfully updated Proxmox SSH keys with new name for organization %s", orgID)
		}
	}

	// Convert to proto response
	sshKey := &vpsv1.SSHKey{
		Id:          key.ID,
		Name:        key.Name,
		PublicKey:   key.PublicKey,
		Fingerprint: key.Fingerprint,
		CreatedAt:   timestamppb.New(key.CreatedAt),
		UpdatedAt:   timestamppb.New(key.UpdatedAt),
	}
	if key.VPSID != nil {
		sshKey.VpsId = key.VPSID
	}

	logger.Info("[VPS] Updated SSH key %s name to '%s' (org: %s)", keyID, newName, orgID)

	return connect.NewResponse(&vpsv1.UpdateSSHKeyResponse{
		Key: sshKey,
	}), nil
}

// GetSSHKeysForOrganization gets all SSH keys for an organization (internal use)
func GetSSHKeysForOrganization(orgID string) ([]database.SSHKey, error) {
	var keys []database.SSHKey
	if err := database.DB.Where("organization_id = ?", orgID).Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}
