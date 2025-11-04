package deployments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"api/internal/database"
)

// DomainVerificationStatus represents the verification status of a custom domain
type DomainVerificationStatus string

const (
	DomainVerificationStatusPending  DomainVerificationStatus = "pending"
	DomainVerificationStatusVerified DomainVerificationStatus = "verified"
	DomainVerificationStatusFailed   DomainVerificationStatus = "failed"
	DomainVerificationStatusExpired  DomainVerificationStatus = "expired"
)

// DomainVerification represents a domain verification record
type DomainVerification struct {
	DeploymentID string                   `json:"deployment_id"`
	Domain       string                   `json:"domain"`
	Token        string                   `json:"token"`
	Status       DomainVerificationStatus `json:"status"`
	CreatedAt    time.Time                `json:"created_at"`
	VerifiedAt   *time.Time               `json:"verified_at,omitempty"`
}

// verifyDomainOwnershipInternal verifies domain ownership via DNS TXT record (internal method)
func (s *Service) verifyDomainOwnershipInternal(ctx context.Context, deploymentID string, domain string) error {
	// Normalize domain (remove trailing dots, convert to lowercase)
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	// Check if domain is already claimed by another deployment
	if err := s.checkDomainConflict(ctx, deploymentID, domain); err != nil {
		return err
	}

	// Get or create verification record
	verification, err := s.getOrCreateVerification(ctx, deploymentID, domain)
	if err != nil {
		return fmt.Errorf("failed to get verification record: %w", err)
	}

	// If already verified, return success
	if verification.Status == DomainVerificationStatusVerified {
		return nil
	}

	// Check if verification has expired (7 days)
	if time.Since(verification.CreatedAt) > 7*24*time.Hour {
		return fmt.Errorf("verification expired. Please request a new verification token")
	}

	// Perform DNS verification
	verificationToken := fmt.Sprintf("obiente-verification=%s", verification.Token)
	txtRecordName := fmt.Sprintf("_obiente-verification.%s", domain)

	// Look up TXT record
	txtRecords, err := lookupTXT(txtRecordName)
	if err != nil {
		log.Printf("[VerifyDomainOwnership] DNS lookup failed for %s: %v", txtRecordName, err)
		return fmt.Errorf("DNS lookup failed: %w. Please ensure the TXT record is configured correctly", err)
	}

	// Check if verification token is present in TXT records
	found := false
	for _, record := range txtRecords {
		if strings.Contains(record, verificationToken) {
			found = true
			break
		}
	}

	if !found {
		// Update status to failed
		s.updateVerificationStatus(ctx, deploymentID, domain, DomainVerificationStatusFailed)
		return fmt.Errorf("verification failed: TXT record not found or token mismatch. Please add TXT record: %s = %s", txtRecordName, verificationToken)
	}

	// Verification successful
	s.updateVerificationStatus(ctx, deploymentID, domain, DomainVerificationStatusVerified)
	log.Printf("[VerifyDomainOwnership] Domain %s verified successfully for deployment %s", domain, deploymentID)

	// Store verification record in deployment's custom_domains with metadata
	if err := s.storeVerifiedDomain(ctx, deploymentID, domain); err != nil {
		return fmt.Errorf("failed to store verified domain: %w", err)
	}

	return nil
}

// getDomainVerificationTokenInternal retrieves or generates a verification token for a domain (internal method)
func (s *Service) getDomainVerificationTokenInternal(ctx context.Context, deploymentID string, domain string) (string, error) {
	// Normalize domain
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	// Get or create verification record
	verification, err := s.getOrCreateVerification(ctx, deploymentID, domain)
	if err != nil {
		return "", fmt.Errorf("failed to get verification record: %w", err)
	}

	return verification.Token, nil
}

// checkDomainConflict checks if a domain is already claimed by another deployment
func (s *Service) checkDomainConflict(ctx context.Context, deploymentID string, domain string) error {
	// Normalize domain for comparison
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	// Query all deployments to check for domain conflicts
	var deployments []database.Deployment
	if err := database.DB.Find(&deployments).Error; err != nil {
		return fmt.Errorf("failed to query deployments: %w", err)
	}

	for _, dep := range deployments {
		// Skip the current deployment
		if dep.ID == deploymentID {
			continue
		}

		// Check default domain
		if strings.ToLower(strings.TrimSuffix(dep.Domain, ".")) == domain {
			return fmt.Errorf("domain %s is already in use by deployment %s", domain, dep.ID)
		}

		// Check custom domains
		if dep.CustomDomains != "" {
			var customDomains []string
			if err := json.Unmarshal([]byte(dep.CustomDomains), &customDomains); err == nil {
				for _, customDomain := range customDomains {
					// Parse custom domain (may include verification metadata)
					verifiedDomain := extractDomainFromCustomDomainEntry(customDomain)
					if strings.ToLower(strings.TrimSuffix(verifiedDomain, ".")) == domain {
						return fmt.Errorf("domain %s is already in use by deployment %s", domain, dep.ID)
					}
				}
			}
		}
	}

	return nil
}

// getOrCreateVerification retrieves or creates a verification record
// Tokens are stored in the custom_domains field with format: "domain.com:token:abc123:pending"
func (s *Service) getOrCreateVerification(ctx context.Context, deploymentID string, domain string) (*DomainVerification, error) {
	// Get deployment to check for existing verification token
	var deployment database.Deployment
	if err := database.DB.Where("id = ?", deploymentID).First(&deployment).Error; err != nil {
		return nil, fmt.Errorf("deployment not found: %w", err)
	}

	// Parse existing custom domains to find token
	var customDomains []string
	if deployment.CustomDomains != "" {
		if err := json.Unmarshal([]byte(deployment.CustomDomains), &customDomains); err != nil {
			customDomains = []string{}
		}
	}

	// Look for existing token in custom domains
	for _, entry := range customDomains {
		parts := strings.Split(entry, ":")
		entryDomain := extractDomainFromCustomDomainEntry(entry)

		if !strings.EqualFold(entryDomain, domain) {
			continue
		}

		// Found entry for this domain - check if it's a token entry (pending or verified)
		if len(parts) >= 3 && parts[1] == "token" {
			// Format: "domain.com:token:abc123:pending" or "domain.com:token:abc123:verified"
			token := parts[2]
			status := DomainVerificationStatusPending
			if len(parts) >= 4 {
				status = DomainVerificationStatus(parts[3])
			}
			// Return existing token - never create a new one if token exists
			return &DomainVerification{
				DeploymentID: deploymentID,
				Domain:       domain,
				Token:        token,
				Status:       status,
				CreatedAt:    time.Now(),
			}, nil
		}
	}

	// No existing token found - generate deterministic token based on domain + deployment ID
	// This ensures the token never changes for the same domain/deployment combination
	token := generateDeterministicToken(deploymentID, domain)

	verification := &DomainVerification{
		DeploymentID: deploymentID,
		Domain:       domain,
		Token:        token,
		Status:       DomainVerificationStatusPending,
		CreatedAt:    time.Now(),
	}

	// Remove any existing entries for this domain before adding new token entry
	filteredDomains := []string{}
	for _, existingDomain := range customDomains {
		existingDomainName := extractDomainFromCustomDomainEntry(existingDomain)
		if !strings.EqualFold(existingDomainName, domain) {
			filteredDomains = append(filteredDomains, existingDomain)
		}
	}

	// Store token in custom_domains field temporarily
	entry := fmt.Sprintf("%s:token:%s:pending", domain, token)
	filteredDomains = append(filteredDomains, entry)
	customDomainsJSON, err := json.Marshal(filteredDomains)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom domains: %w", err)
	}

	deployment.CustomDomains = string(customDomainsJSON)
	if err := database.DB.Save(&deployment).Error; err != nil {
		return nil, fmt.Errorf("failed to save verification token: %w", err)
	}

	return verification, nil
}

// generateDeterministicToken generates a deterministic token based on deployment ID and domain
// This ensures the same domain+deployment always gets the same token, so it never changes
// Uses SECRET environment variable for HMAC-based hashing
func generateDeterministicToken(deploymentID string, domain string) string {
	// Normalize inputs
	deploymentID = strings.ToLower(strings.TrimSpace(deploymentID))
	domain = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))

	// Get SECRET from environment
	secret := os.Getenv("SECRET")
	if secret == "" {
		log.Printf("[generateDeterministicToken] Warning: SECRET environment variable not set, using fallback")
		// Fallback: still generate deterministic token but log warning
		secret = "fallback-secret-not-configured"
	}

	// Create hash input: deploymentID + domain
	input := fmt.Sprintf("%s:%s", deploymentID, domain)

	// Generate HMAC-SHA256 hash using SECRET
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	hash := mac.Sum(nil)

	// Return first 32 characters (16 bytes) as hex string
	return hex.EncodeToString(hash[:16])
}

// updateVerificationStatus updates the verification status in the custom_domains field
func (s *Service) updateVerificationStatus(ctx context.Context, deploymentID string, domain string, status DomainVerificationStatus) {
	var deployment database.Deployment
	if err := database.DB.Where("id = ?", deploymentID).First(&deployment).Error; err != nil {
		log.Printf("[updateVerificationStatus] Failed to get deployment: %v", err)
		return
	}

	var customDomains []string
	if deployment.CustomDomains != "" {
		if err := json.Unmarshal([]byte(deployment.CustomDomains), &customDomains); err != nil {
			log.Printf("[updateVerificationStatus] Failed to parse custom domains: %v", err)
			return
		}
	}

	// Find and update the token entry, remove all other entries for this domain
	var updatedEntry string
	var foundTokenEntry bool
	filteredDomains := []string{}

	for _, entry := range customDomains {
		parts := strings.Split(entry, ":")
		entryDomain := extractDomainFromCustomDomainEntry(entry)

		// Skip all entries for this domain - we'll add the updated one back
		if strings.EqualFold(entryDomain, domain) {
			// If this is a token entry, update it
			if len(parts) >= 3 && parts[1] == "token" {
				token := parts[2]
				updatedEntry = fmt.Sprintf("%s:token:%s:%s", domain, token, status)
				foundTokenEntry = true
			}
			// Skip all entries for this domain (we'll add updated one back)
			continue
		}

		// Keep entries for other domains
		filteredDomains = append(filteredDomains, entry)
	}

	// Add the updated entry if we found a token entry
	if foundTokenEntry {
		filteredDomains = append(filteredDomains, updatedEntry)
	}

	customDomainsJSON, err := json.Marshal(filteredDomains)
	if err != nil {
		log.Printf("[updateVerificationStatus] Failed to marshal custom domains: %v", err)
		return
	}

	deployment.CustomDomains = string(customDomainsJSON)
	if err := database.DB.Save(&deployment).Error; err != nil {
		log.Printf("[updateVerificationStatus] Failed to save status: %v", err)
		return
	}

	log.Printf("[updateVerificationStatus] Domain %s for deployment %s: %s", domain, deploymentID, status)
}

// storeVerifiedDomain stores a verified domain in the deployment's custom_domains
// IMPORTANT: Preserves the token entry so the token never changes after verification
func (s *Service) storeVerifiedDomain(ctx context.Context, deploymentID string, domain string) error {
	// Get deployment
	var deployment database.Deployment
	if err := database.DB.Where("id = ?", deploymentID).First(&deployment).Error; err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	// Parse existing custom domains
	var customDomains []string
	if deployment.CustomDomains != "" {
		if err := json.Unmarshal([]byte(deployment.CustomDomains), &customDomains); err != nil {
			customDomains = []string{}
		}
	}

	// Find and preserve the token entry for this domain
	var tokenEntry string
	var foundTokenEntry bool
	filteredDomains := []string{}

	for _, existingDomain := range customDomains {
		parts := strings.Split(existingDomain, ":")
		existingDomainName := extractDomainFromCustomDomainEntry(existingDomain)

		if !strings.EqualFold(existingDomainName, domain) {
			// Keep entries for other domains
			filteredDomains = append(filteredDomains, existingDomain)
			continue
		}

		// Found entry for this domain - check if it has a token
		if len(parts) >= 3 && parts[1] == "token" {
			// Preserve the token entry but mark as verified
			token := parts[2]
			tokenEntry = fmt.Sprintf("%s:token:%s:verified", domain, token)
			foundTokenEntry = true
		}
		// Skip other entries (plain domain, old verified entries, etc.)
	}

	// If we found a token entry, use it with verified status
	// Otherwise, create a new deterministic token (shouldn't happen, but safety check)
	if foundTokenEntry {
		filteredDomains = append(filteredDomains, tokenEntry)
	} else {
		// Fallback: generate deterministic token (shouldn't normally happen)
		token := generateDeterministicToken(deploymentID, domain)
		filteredDomains = append(filteredDomains, fmt.Sprintf("%s:token:%s:verified", domain, token))
		log.Printf("[storeVerifiedDomain] Warning: No token entry found for domain %s, generated deterministic token", domain)
	}

	// Save back to database
	customDomainsJSON, err := json.Marshal(filteredDomains)
	if err != nil {
		return fmt.Errorf("failed to marshal custom domains: %w", err)
	}

	deployment.CustomDomains = string(customDomainsJSON)
	if err := database.DB.Save(&deployment).Error; err != nil {
		return fmt.Errorf("failed to save custom domains: %w", err)
	}

	return nil
}

// extractDomainFromCustomDomainEntry extracts the domain name from a custom domain entry
// Entry format: "domain.com" or "domain.com:verified" or "domain.com:token:abc123:pending"
func extractDomainFromCustomDomainEntry(entry string) string {
	parts := strings.Split(entry, ":")
	return parts[0]
}

// DeduplicateCustomDomains removes duplicate domain entries (case-insensitive)
// Keeps the most recent/relevant entry for each domain (prefers verified over pending)
// This is a standalone function so it can be used from converters
func DeduplicateCustomDomains(customDomains []string) []string {
	domainMap := make(map[string]string) // domain -> entry

	for _, entry := range customDomains {
		domainName := extractDomainFromCustomDomainEntry(entry)
		domainLower := strings.ToLower(domainName)

		// Determine priority: verified > verified with token > pending with token > pending
		existingEntry, exists := domainMap[domainLower]
		if !exists {
			domainMap[domainLower] = entry
		} else {
			// If we already have this domain, keep the one with higher priority
			existingParts := strings.Split(existingEntry, ":")
			currentParts := strings.Split(entry, ":")

			existingPriority := getDomainEntryPriority(existingParts)
			currentPriority := getDomainEntryPriority(currentParts)

			// Keep the entry with higher priority (higher number = more important)
			if currentPriority > existingPriority {
				domainMap[domainLower] = entry
			}
		}
	}

	// Convert map back to slice
	result := make([]string, 0, len(domainMap))
	for _, entry := range domainMap {
		result = append(result, entry)
	}

	return result
}

// deduplicateCustomDomains is a wrapper for the Service method
func (s *Service) deduplicateCustomDomains(customDomains []string) []string {
	return DeduplicateCustomDomains(customDomains)
}

// getDomainEntryPriority returns a priority number for domain entries
// Higher number = higher priority
// verified = 3, verified with token = 2, pending with token = 1, plain = 0
func getDomainEntryPriority(parts []string) int {
	if len(parts) >= 4 && parts[1] == "token" && parts[3] == "verified" {
		return 2 // verified with token
	}
	if len(parts) >= 2 && parts[1] == "verified" {
		return 3 // verified
	}
	if len(parts) >= 4 && parts[1] == "token" {
		return 1 // pending with token
	}
	return 0 // plain domain
}

// extractDomainFromCustomDomains extracts only verified domains from custom_domains array
// This filters out pending verification entries
func (s *Service) extractVerifiedDomains(customDomains []string) []string {
	verified := []string{}
	for _, entry := range customDomains {
		parts := strings.Split(entry, ":")
		if len(parts) == 1 {
			// Simple format: "domain.com" (assumed verified)
			verified = append(verified, parts[0])
		} else if len(parts) >= 2 && parts[1] == "verified" {
			// Format: "domain.com:verified"
			verified = append(verified, parts[0])
		} else if len(parts) >= 4 && parts[1] == "token" && parts[3] == "verified" {
			// Format: "domain.com:token:abc123:verified"
			verified = append(verified, parts[0])
		}
	}
	return verified
}

// ValidateCustomDomains validates custom domains before saving
// This checks for conflicts but doesn't require immediate verification
// Domains can be added as pending and verified later
func (s *Service) ValidateCustomDomains(ctx context.Context, deploymentID string, domains []string) error {
	for _, domain := range domains {
		// Normalize domain
		domain = strings.ToLower(strings.TrimSuffix(domain, "."))

		// Basic validation
		if domain == "" {
			continue // Skip empty domains
		}

		// Extract domain name (remove status suffix if present)
		domainName := extractDomainFromCustomDomainEntry(domain)

		// Check for conflicts with other deployments (security check)
		if err := s.checkDomainConflict(ctx, deploymentID, domainName); err != nil {
			return err
		}

		// If domain is already marked as verified, re-verify it to ensure it's still valid
		if strings.Contains(domain, ":verified") {
			if err := s.verifyDomainOwnershipInternal(ctx, deploymentID, domainName); err != nil {
				// If verification fails, don't block but log warning
				log.Printf("[ValidateCustomDomains] Warning: Previously verified domain %s failed verification: %v", domainName, err)
				// Continue - allow user to keep domain but mark as needing re-verification
			}
		}
		// If domain is pending or new, allow it without verification
		// User can verify it later via explicit verification endpoint
	}

	return nil
}

// lookupTXT performs a DNS TXT record lookup using the standard library
func lookupTXT(name string) ([]string, error) {
	txtRecords, err := net.LookupTXT(name)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed: %w", err)
	}
	return txtRecords, nil
}
