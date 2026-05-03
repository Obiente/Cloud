package github

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewInstallationClient(ctx context.Context, installationID int64) (*Client, error) {
	token, err := CreateInstallationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}
	return NewClient(token), nil
}

func CreateInstallationToken(ctx context.Context, installationID int64) (string, error) {
	if installationID <= 0 {
		return "", fmt.Errorf("GitHub App installation ID is required")
	}

	appJWT, err := createGitHubAppJWT(time.Now())
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub App installation token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("GitHub App installation token request failed: %d - %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp installationTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode GitHub App installation token response: %w", err)
	}
	if strings.TrimSpace(tokenResp.Token) == "" {
		return "", fmt.Errorf("GitHub App installation token response did not include a token")
	}

	return tokenResp.Token, nil
}

func createGitHubAppJWT(now time.Time) (string, error) {
	appID := strings.TrimSpace(os.Getenv("GITHUB_APP_ID"))
	if appID == "" {
		return "", fmt.Errorf("GITHUB_APP_ID is required for GitHub App installations")
	}

	key, err := loadGitHubAppPrivateKey()
	if err != nil {
		return "", err
	}

	header, err := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(map[string]interface{}{
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": appID,
	})
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign GitHub App JWT: %w", err)
	}

	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func loadGitHubAppPrivateKey() (*rsa.PrivateKey, error) {
	keyPEM := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY"))
	if keyPEM == "" {
		if encoded := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY_BASE64")); encoded != "" {
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return nil, fmt.Errorf("failed to decode GITHUB_APP_PRIVATE_KEY_BASE64: %w", err)
			}
			keyPEM = string(decoded)
		}
	}
	if keyPEM == "" {
		if path := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")); path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read GITHUB_APP_PRIVATE_KEY_PATH: %w", err)
			}
			keyPEM = string(data)
		}
	}
	if keyPEM == "" {
		return nil, fmt.Errorf("GITHUB_APP_PRIVATE_KEY, GITHUB_APP_PRIVATE_KEY_BASE64, or GITHUB_APP_PRIVATE_KEY_PATH is required for GitHub App installations")
	}

	keyPEM = strings.ReplaceAll(keyPEM, `\n`, "\n")
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode GitHub App private key PEM")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub App private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("GitHub App private key must be an RSA private key")
	}
	return key, nil
}

func ParseInstallationID(value string) (int64, error) {
	installationID, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || installationID <= 0 {
		return 0, fmt.Errorf("invalid GitHub App installation ID %q", value)
	}
	return installationID, nil
}
