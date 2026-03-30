package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

const encryptedValuePrefix = "enc:v1:"

// TokenCipher encrypts and decrypts stored tokens using AES-256-GCM.
type TokenCipher struct {
	key []byte
}

// NewTokenCipherFromEnv derives an encryption key from configured environment secrets.
func NewTokenCipherFromEnv() (*TokenCipher, error) {
	for _, envName := range []string{
		"GITHUB_TOKEN_ENCRYPTION_KEY",
		"DATABASE_ENCRYPTION_KEY",
		"API_SECRET",
		"SECRET",
		"ZITADEL_CLIENT_SECRET",
		"ZITADEL_MANAGEMENT_TOKEN",
		"GITHUB_CLIENT_SECRET",
	} {
		if secret := strings.TrimSpace(os.Getenv(envName)); secret != "" {
			return &TokenCipher{key: deriveEncryptionKey(secret)}, nil
		}
	}

	return nil, fmt.Errorf("token encryption key is not configured")
}

func (c *TokenCipher) EncryptString(value string) (string, error) {
	if c == nil || len(c.key) == 0 {
		return "", fmt.Errorf("token cipher is not initialized")
	}
	if value == "" {
		return "", fmt.Errorf("value cannot be empty")
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)
	return encryptedValuePrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *TokenCipher) DecryptString(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if !IsEncryptedString(value) {
		return value, nil
	}
	if c == nil || len(c.key) == 0 {
		return "", fmt.Errorf("token cipher is not initialized")
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encryptedValuePrefix))
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(payload) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := payload[:nonceSize], payload[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt token: %w", err)
	}

	return string(plaintext), nil
}

func IsEncryptedString(value string) bool {
	return strings.HasPrefix(value, encryptedValuePrefix)
}

func deriveEncryptionKey(secret string) []byte {
	if decoded, err := base64.StdEncoding.DecodeString(secret); err == nil && len(decoded) == 32 {
		return decoded
	}

	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
