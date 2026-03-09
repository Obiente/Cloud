package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the cost factor for bcrypt (10-14 recommended for production)
	BcryptCost = 12
	// MinPasswordLength is minimum password length
	MinPasswordLength = 12
)

// SecretManager handles encryption and secure password operations
type SecretManager struct {
	encryptionKey []byte
}

// NewSecretManager creates a production-ready secret manager
func NewSecretManager() (*SecretManager, error) {
	keyStr := os.Getenv("DATABASE_ENCRYPTION_KEY")
	if keyStr == "" {
		return nil, fmt.Errorf("DATABASE_ENCRYPTION_KEY environment variable is required")
	}

	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (256 bits), got %d bytes", len(key))
	}

	return &SecretManager{
		encryptionKey: key,
	}, nil
}

// EncryptPassword encrypts a password using AES-256-GCM for storage
// Use this for database connection passwords that need to be retrieved
func (sm *SecretManager) EncryptPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptPassword decrypts a password encrypted with EncryptPassword
func (sm *SecretManager) DecryptPassword(encrypted string) (string, error) {
	if encrypted == "" {
		return "", fmt.Errorf("encrypted password cannot be empty")
	}

	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// HashPassword creates a bcrypt hash of a password for verification
// Use this for user authentication passwords that only need comparison
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against a bcrypt hash
func VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateSecurePassword generates a cryptographically secure random password
func GenerateSecurePassword(length int) (string, error) {
	if length < MinPasswordLength {
		length = MinPasswordLength
	}

	const (
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars   = "0123456789"
		specialChars = "!@#$%^&*()_+-=[]{}|;:,.<>?"
		allChars     = lowerChars + upperChars + digitChars + specialChars
	)

	password := make([]byte, length)

	// Ensure at least one of each type
	if length >= 4 {
		password[0] = lowerChars[secureRandomInt(len(lowerChars))]
		password[1] = upperChars[secureRandomInt(len(upperChars))]
		password[2] = digitChars[secureRandomInt(len(digitChars))]
		password[3] = specialChars[secureRandomInt(len(specialChars))]

		// Fill the rest randomly
		for i := 4; i < length; i++ {
			password[i] = allChars[secureRandomInt(len(allChars))]
		}

		// Shuffle to avoid predictable pattern
		for i := len(password) - 1; i > 0; i-- {
			j := secureRandomInt(i + 1)
			password[i], password[j] = password[j], password[i]
		}
	} else {
		// For very short passwords, just use random chars
		for i := 0; i < length; i++ {
			password[i] = allChars[secureRandomInt(len(allChars))]
		}
	}

	return string(password), nil
}

// secureRandomInt returns a cryptographically secure random int < max
func secureRandomInt(max int) int {
	if max <= 0 {
		return 0
	}

	b := make([]byte, 1)
	for {
		if _, err := rand.Read(b); err != nil {
			// Fallback to 0 on error (shouldn't happen)
			return 0
		}
		if int(b[0]) < (256 - (256 % max)) {
			return int(b[0]) % max
		}
	}
}

// GenerateSecureToken generates a cryptographically secure URL-safe token
func GenerateSecureToken(byteLength int) (string, error) {
	if byteLength < 16 {
		byteLength = 16 // Minimum 128 bits
	}

	token := make([]byte, byteLength)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(token), nil
}

// GenerateAPIKey generates a secure API key with a prefix
func GenerateAPIKey(prefix string) (string, error) {
	token, err := GenerateSecureToken(32) // 256 bits
	if err != nil {
		return "", err
	}

	if prefix == "" {
		prefix = "db"
	}

	return fmt.Sprintf("%s_%s", prefix, token), nil
}
