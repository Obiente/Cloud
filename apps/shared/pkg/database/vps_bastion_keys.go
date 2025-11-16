package database

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// GenerateVPSBastionKeyPair generates a new SSH key pair for bastion host access
// Returns: public key (PEM), private key (PEM), fingerprint, error
func GenerateVPSBastionKeyPair() (string, string, string, error) {
	// Generate Ed25519 key pair (recommended for SSH)
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Encode private key as PEM
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Generate SSH public key from Ed25519 public key
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	// Format public key as OpenSSH format
	publicKeyBytes := ssh.MarshalAuthorizedKey(sshPublicKey)
	publicKeyStr := string(publicKeyBytes)

	// Calculate fingerprint
	fingerprint := ssh.FingerprintSHA256(sshPublicKey)

	return publicKeyStr, string(privateKeyPEM), fingerprint, nil
}

// CreateVPSBastionKey creates a new bastion key for a VPS
func CreateVPSBastionKey(vpsID, orgID string) (*VPSBastionKey, error) {
	// Generate key pair
	publicKey, privateKey, fingerprint, err := GenerateVPSBastionKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Generate ID
	keyID := fmt.Sprintf("bastion-%d", time.Now().UnixNano())

	// Create key record
	bastionKey := &VPSBastionKey{
		ID:             keyID,
		VPSID:          vpsID,
		OrganizationID: orgID,
		PublicKey:      publicKey,
		PrivateKey:     privateKey,
		Fingerprint:    fingerprint,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := DB.Create(bastionKey).Error; err != nil {
		return nil, fmt.Errorf("failed to create bastion key: %w", err)
	}

	return bastionKey, nil
}

// GetVPSBastionKey retrieves the bastion key for a VPS
func GetVPSBastionKey(vpsID string) (*VPSBastionKey, error) {
	var key VPSBastionKey
	if err := DB.Where("vps_id = ?", vpsID).First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// DeleteVPSBastionKey deletes the bastion key for a VPS
func DeleteVPSBastionKey(vpsID string) error {
	return DB.Where("vps_id = ?", vpsID).Delete(&VPSBastionKey{}).Error
}

// RotateVPSBastionKey generates a new key pair and updates the existing record
// If the key doesn't exist, it creates a new one (requires orgID)
func RotateVPSBastionKey(vpsID string, orgID string) (*VPSBastionKey, error) {
	// Get existing key to preserve ID and timestamps
	var existingKey VPSBastionKey
	err := DB.Where("vps_id = ?", vpsID).First(&existingKey).Error
	if err != nil {
		// Key doesn't exist, create a new one
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreateVPSBastionKey(vpsID, orgID)
		}
		return nil, fmt.Errorf("failed to get bastion key: %w", err)
	}

	// Generate new key pair
	publicKey, privateKey, fingerprint, err := GenerateVPSBastionKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Update existing record
	existingKey.PublicKey = publicKey
	existingKey.PrivateKey = privateKey
	existingKey.Fingerprint = fingerprint
	existingKey.UpdatedAt = time.Now()

	if err := DB.Save(&existingKey).Error; err != nil {
		return nil, fmt.Errorf("failed to update bastion key: %w", err)
	}

	return &existingKey, nil
}

