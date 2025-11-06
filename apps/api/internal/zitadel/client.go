package zitadel

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client handles Zitadel API v2 interactions
//
// IMPORTANT: Organization Membership vs Project-Level Roles
//
// In Zitadel, there's a critical distinction:
//   - Project-level roles (e.g., "Org Manager" in a project): Permissions within a specific project
//   - Organization membership: Explicit membership in the organization itself
//
// Session creation requires ACTUAL ORGANIZATION MEMBERSHIP, not just project-level permissions.
// Even if you have "Org Manager" role in a project, you must be added as a member of the organization.
//
// Required Zitadel Service Account Organization Permissions:
//   - Org User Manager: Required for searching users by email and managing user sessions
//     This permission allows:
//   - Searching users via POST /v2/users (User Service v2)
//   - Creating sessions via /v2/sessions
//
// Alternative permissions that may work:
// - Org Owner: Full organization access (includes user management)
// - Org Admin Impersonator: Can impersonate users (may include session creation)
//
// To configure in Zitadel Console:
// 1. Go to Organizations → Select your organization (NOT Projects)
// 2. Go to Members → Find your Service Account (or add it if not present)
// 3. Click "Grant" or "Edit" to assign organization-level permissions
// 4. Grant "Org User Manager" permission (or "Org Owner" for full access) at the ORGANIZATION level
// 5. Go to Projects → Select your project
// 6. Generate Personal Access Token (scopes are automatically set based on the project)
// 7. IMPORTANT: Regenerate the token AFTER adding the member - tokens capture permissions at creation time
//
// NOTE: If you get "membership not found" error even though you have project-level roles,
//       you need to add the user/service account as an organization member explicitly.
// NOTE: Personal Access Tokens capture permissions/memberships at creation time, so you must
//       regenerate the token after adding organization membership or roles.
type Client struct {
	baseURL         string
	clientID        string
	managementToken string
	organizationID  string // REQUIRED: Organization ID for API requests (service users don't have default org)
	httpClient      *http.Client
}

// NewClient creates a new Zitadel API v2 client
// Note: Service users don't have a default organization, so ZITADEL_ORGANIZATION_ID is required
func NewClient() *Client {
	zitadelURL := os.Getenv("ZITADEL_URL")
	if zitadelURL == "" {
		zitadelURL = "https://auth.obiente.cloud"
	}
	zitadelURL = strings.TrimSuffix(zitadelURL, "/")

	clientID := os.Getenv("ZITADEL_CLIENT_ID")
	managementToken := strings.TrimSpace(os.Getenv("ZITADEL_MANAGEMENT_TOKEN"))
	organizationID := strings.TrimSpace(os.Getenv("ZITADEL_ORGANIZATION_ID"))

	// Log configuration (without sensitive data)
	if managementToken != "" {
		tokenPreview := managementToken
		if len(tokenPreview) > 10 {
			tokenPreview = tokenPreview[:10] + "..."
		}
		fmt.Printf("[Zitadel Client] Initialized with:\n")
		fmt.Printf("  ZITADEL_URL: %s\n", zitadelURL)
		fmt.Printf("  ZITADEL_CLIENT_ID: %s\n", clientID)
		fmt.Printf("  ZITADEL_MANAGEMENT_TOKEN: %s (configured)\n", tokenPreview)
		if organizationID != "" {
			fmt.Printf("  ZITADEL_ORGANIZATION_ID: %s\n", organizationID)
		} else {
			fmt.Printf("  ZITADEL_ORGANIZATION_ID: (not set - REQUIRED for service users)\n")
			fmt.Printf("  WARNING: Service users don't have a default organization. ZITADEL_ORGANIZATION_ID must be set.\n")
		}
	} else {
		fmt.Printf("[Zitadel Client] WARNING: ZITADEL_MANAGEMENT_TOKEN not configured\n")
	}

	return &Client{
		baseURL:         zitadelURL,
		clientID:        clientID,
		managementToken: managementToken,
		organizationID:  organizationID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// LoginResponse represents the response from a login operation
type LoginResponse struct {
	Success      bool
	AccessToken  string
	RefreshToken string
	ExpiresIn    int32
	Message      string
}

// Login authenticates a user with email and password using Zitadel Session API v2
// Implements the recommended Session API flow per Zitadel documentation
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password
//
// Note: Zitadel does NOT support ROPC grant (deprecated in OAuth 2.1).
// The Session API flow is the recommended approach for custom login UIs.
func (c *Client) Login(email, password string) (*LoginResponse, error) {
	// Use Session API flow (Zitadel's recommended approach)
	// This requires a service account with Org User Manager permission
	if c.managementToken == "" {
		return &LoginResponse{
			Success: false,
			Message: "Management token required for Session API authentication. Configure ZITADEL_MANAGEMENT_TOKEN.",
		}, fmt.Errorf("management token not configured")
	}

	result, err := c.authenticateWithSessionAPI(email, password)
	if err != nil {
		// Ensure we return a LoginResponse with the error message even if authentication fails
		if result == nil {
			result = &LoginResponse{
				Success: false,
				Message: err.Error(),
			}
		} else if result.Message == "" {
			result.Message = err.Error()
		}
		return result, err
	}
	return result, nil
}

// decodeTokenClaims decodes JWT token claims to inspect scopes and organization context
func (c *Client) decodeTokenClaims() (map[string]interface{}, error) {
	parts := strings.Split(c.managementToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("token is not a JWT (expected 3 parts, got %d)", len(parts))
	}
	
	// Decode the payload (second part)
	payload := parts[1]
	// Add padding if needed
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}
	
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}
	
	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}
	
	return claims, nil
}

// debugTokenInfo logs detailed information about the token
func (c *Client) debugTokenInfo() {
	fmt.Printf("[Zitadel] === Token Debug Information ===\n")
	fmt.Printf("[Zitadel] Token preview: %s...\n", c.managementToken[:min(20, len(c.managementToken))])
	
	claims, err := c.decodeTokenClaims()
	if err != nil {
		fmt.Printf("[Zitadel] WARNING: Could not decode token claims: %v\n", err)
		fmt.Printf("[Zitadel] Token might not be a JWT, or might be encrypted\n")
		return
	}
	
	fmt.Printf("[Zitadel] Token claims decoded successfully\n")
	
	// Log important claims
	if sub, ok := claims["sub"].(string); ok {
		fmt.Printf("[Zitadel] Token subject (sub): %s\n", sub)
	}
	if iss, ok := claims["iss"].(string); ok {
		fmt.Printf("[Zitadel] Token issuer (iss): %s\n", iss)
	}
	if aud, ok := claims["aud"]; ok {
		fmt.Printf("[Zitadel] Token audience (aud): %v\n", aud)
	}
	if scope, ok := claims["scope"].(string); ok {
		fmt.Printf("[Zitadel] Token scopes: %s\n", scope)
		scopes := strings.Split(scope, " ")
		fmt.Printf("[Zitadel] Individual scopes: %v\n", scopes)
	}
	if orgID, ok := claims["org_id"].(string); ok {
		fmt.Printf("[Zitadel] Token organization ID (org_id): %s\n", orgID)
	}
	if orgID, ok := claims["organization_id"].(string); ok {
		fmt.Printf("[Zitadel] Token organization ID (organization_id): %s\n", orgID)
	}
	if orgID, ok := claims["orgId"].(string); ok {
		fmt.Printf("[Zitadel] Token organization ID (orgId): %s\n", orgID)
	}
	if roles, ok := claims["roles"].([]interface{}); ok {
		fmt.Printf("[Zitadel] Token roles: %v\n", roles)
	}
	if roles, ok := claims["urn:zitadel:iam:org:project:roles"].([]interface{}); ok {
		fmt.Printf("[Zitadel] Token org project roles: %v\n", roles)
	}
	
	// Log all claims for debugging
	fmt.Printf("[Zitadel] All token claims:\n")
	for key, value := range claims {
		fmt.Printf("[Zitadel]   %s: %v\n", key, value)
	}
	fmt.Printf("[Zitadel] === End Token Debug ===\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getTokenOrganizationContext attempts to determine what organization the token is scoped to
// by making a test API call without organization headers
func (c *Client) getTokenOrganizationContext() (string, error) {
	// First try to get org from token claims
	claims, err := c.decodeTokenClaims()
	if err == nil {
		// Try various possible claim names for organization ID
		if orgID, ok := claims["org_id"].(string); ok && orgID != "" {
			return orgID, nil
		}
		if orgID, ok := claims["organization_id"].(string); ok && orgID != "" {
			return orgID, nil
		}
		if orgID, ok := claims["orgId"].(string); ok && orgID != "" {
			return orgID, nil
		}
	}
	
	// Fallback: Try to query organizations endpoint to see what org the token defaults to
	// This is a best-effort check - not all Zitadel instances expose this
	orgsURL := fmt.Sprintf("%s/v2/organizations/_search", c.baseURL)
	
	searchBody := map[string]interface{}{
		"queries": []map[string]interface{}{},
		"limit":   1,
	}
	
	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequest("POST", orgsURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	// Don't set org header - use token's default
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		var result struct {
			Result []struct {
				ID string `json:"id"`
			} `json:"result"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && len(result.Result) > 0 {
			return result.Result[0].ID, nil
		}
	}
	
	return "", fmt.Errorf("could not determine token organization context")
}

// getServiceAccountInfo attempts to identify the service account and verify its actual organization memberships
// Note: Being able to query organizations doesn't mean you're a member - membership requires explicit addition
func (c *Client) getServiceAccountInfo() (userID string, verifiedMemberships []string) {
	// Try multiple endpoints to identify the PAT owner
	// Personal Access Tokens might work with different endpoints than regular OAuth tokens
	
	// Endpoint 1: Try Management API user info endpoint
	endpoints := []string{
		fmt.Sprintf("%s/management/v1/users/me", c.baseURL),
		fmt.Sprintf("%s/auth/v1/users/me", c.baseURL),
		fmt.Sprintf("%s/oidc/v1/userinfo", c.baseURL),
	}
	
	for _, userInfoURL := range endpoints {
		req, err := http.NewRequest("GET", userInfoURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+c.managementToken)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("[Zitadel] Successfully queried user info from %s\n", userInfoURL)
			
			// Try to parse as Management API response (nested user object)
			var userInfoWrapper struct {
				User struct {
					UserID string `json:"id"`
					Details struct {
						ResourceOwner string `json:"resourceOwner"`
					} `json:"details"`
				} `json:"user"`
			}
			if err := json.Unmarshal(body, &userInfoWrapper); err == nil && userInfoWrapper.User.UserID != "" {
				fmt.Printf("[Zitadel] Service account identified: User ID: %s, Default Org: %s\n", userInfoWrapper.User.UserID, userInfoWrapper.User.Details.ResourceOwner)
				userID = userInfoWrapper.User.UserID
				if userInfoWrapper.User.Details.ResourceOwner != "" {
					verifiedMemberships = append(verifiedMemberships, userInfoWrapper.User.Details.ResourceOwner)
				}
				return userID, verifiedMemberships
			}
			
			// Try to parse as flat Management API response
			var userInfoFlat struct {
				UserID string `json:"id"`
				Details struct {
					ResourceOwner string `json:"resourceOwner"`
				} `json:"details"`
			}
			if err := json.Unmarshal(body, &userInfoFlat); err == nil && userInfoFlat.UserID != "" {
				fmt.Printf("[Zitadel] Service account identified: User ID: %s, Default Org: %s\n", userInfoFlat.UserID, userInfoFlat.Details.ResourceOwner)
				userID = userInfoFlat.UserID
				if userInfoFlat.Details.ResourceOwner != "" {
					verifiedMemberships = append(verifiedMemberships, userInfoFlat.Details.ResourceOwner)
				}
				return userID, verifiedMemberships
			}
			
			// Try to parse as OIDC userinfo response
			var oidcInfo struct {
				Sub string `json:"sub"`
			}
			if err := json.Unmarshal(body, &oidcInfo); err == nil && oidcInfo.Sub != "" {
				fmt.Printf("[Zitadel] Service account identified via OIDC: Subject: %s\n", oidcInfo.Sub)
				userID = oidcInfo.Sub
				return userID, verifiedMemberships
			}
			
			// Log the raw response for debugging
			fmt.Printf("[Zitadel] User info response (could not parse): %s\n", string(body))
		}
	}
	
	fmt.Printf("[Zitadel] Could not identify service account via standard endpoints\n")
	fmt.Printf("[Zitadel] To find the PAT owner:\n")
	fmt.Printf("[Zitadel] 1. Go to Zitadel Console → Projects → Your Project → Personal Access Tokens\n")
	fmt.Printf("[Zitadel] 2. Find the token starting with 'LL1-4CuGS-...'\n")
	fmt.Printf("[Zitadel] 3. Check which user/service account created it\n")
	
	// Try to query organizations the service account can see
	// NOTE: This doesn't mean membership - it just means the PAT has project-level permissions to query orgs
	orgsURL := fmt.Sprintf("%s/v2/organizations/_search", c.baseURL)
	searchBody := map[string]interface{}{
		"queries": []map[string]interface{}{},
		"limit":   100,
	}
	
	bodyBytes, err := json.Marshal(searchBody)
	if err == nil {
		req2, err := http.NewRequest("POST", orgsURL, strings.NewReader(string(bodyBytes)))
		if err == nil {
			req2.Header.Set("Authorization", "Bearer "+c.managementToken)
			req2.Header.Set("Content-Type", "application/json")
			resp, err := c.httpClient.Do(req2)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var result struct {
						Result []struct {
							ID string `json:"id"`
						} `json:"result"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
						fmt.Printf("[Zitadel] Service account can query %d organization(s) (this doesn't imply membership):\n", len(result.Result))
						for _, org := range result.Result {
							fmt.Printf("[Zitadel]   - Organization ID: %s\n", org.ID)
						}
						// Don't add these to verifiedMemberships - we need to verify actual membership separately
					}
				}
			}
		}
	}
	
	return userID, verifiedMemberships
}

// verifyActualMembership checks if the service account is actually a member of the organization
// by trying to query the organization's members list and looking for the service account
func (c *Client) verifyActualMembership(serviceAccountID, orgID string) bool {
	if serviceAccountID == "" {
		fmt.Printf("[Zitadel] Cannot verify membership: Service account ID is unknown\n")
		return false
	}
	
	// Try to query members and see if the service account is in the list
	membersURL := fmt.Sprintf("%s/v2/organizations/%s/members/_search", c.baseURL, orgID)
	searchBody := map[string]interface{}{
		"queries": []map[string]interface{}{},
		"limit":   1000, // Get enough to find our service account
	}
	
	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		fmt.Printf("[Zitadel] Failed to marshal members search request: %v\n", err)
		return false
	}
	
	req, err := http.NewRequest("POST", membersURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		fmt.Printf("[Zitadel] Failed to create members search request: %v\n", err)
		return false
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-zitadel-orgid", orgID)
	
	fmt.Printf("[Zitadel] Querying organization members to verify service account (User ID: %s) membership...\n", serviceAccountID)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("[Zitadel] Members query request failed: %v\n", err)
		return false
	}
	defer resp.Body.Close()
	
	fmt.Printf("[Zitadel] Members query response status: %s\n", resp.Status)
	
	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var result struct {
			Result []struct {
				UserID string   `json:"userId"`
				Roles  []string `json:"roles"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &result); err == nil {
			fmt.Printf("[Zitadel] Found %d members in organization\n", len(result.Result))
			for _, member := range result.Result {
				if member.UserID == serviceAccountID {
					fmt.Printf("[Zitadel] ✓ VERIFIED: Service account (User ID: %s) IS a member of organization %s\n", serviceAccountID, orgID)
					fmt.Printf("[Zitadel]   Roles assigned: %v\n", member.Roles)
					
					// Check for expected roles
					hasOrgOwner := false
					hasOrgUserManager := false
					hasImpersonator := false
					for _, role := range member.Roles {
						if strings.Contains(strings.ToLower(role), "owner") {
							hasOrgOwner = true
						}
						if strings.Contains(strings.ToLower(role), "user") && strings.Contains(strings.ToLower(role), "manager") {
							hasOrgUserManager = true
						}
						if strings.Contains(strings.ToLower(role), "impersonator") || strings.Contains(strings.ToLower(role), "impersonate") {
							hasImpersonator = true
						}
					}
					
					if hasOrgOwner {
						fmt.Printf("[Zitadel]   ✓ Has Org Owner role\n")
					}
					if hasOrgUserManager {
						fmt.Printf("[Zitadel]   ✓ Has Org User Manager role\n")
					}
					if hasImpersonator {
						fmt.Printf("[Zitadel]   ✓ Has Impersonator role\n")
					}
					
					return true
				}
			}
			fmt.Printf("[Zitadel] ✗ VERIFIED: Service account (User ID: %s) is NOT in the members list of organization %s\n", serviceAccountID, orgID)
			fmt.Printf("[Zitadel]   Found %d other members, but service account is not among them\n", len(result.Result))
			return false
		} else {
			fmt.Printf("[Zitadel] Failed to parse members response: %v\n", err)
			fmt.Printf("[Zitadel] Response body: %s\n", string(body))
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Zitadel] Members query failed with status %d: %s\n", resp.StatusCode, string(body))
		if resp.StatusCode == http.StatusNotFound {
			fmt.Printf("[Zitadel] 404 Not Found - This usually means the service account cannot query members\n")
			fmt.Printf("[Zitadel] This could indicate the service account is not a member, OR the token needs to be regenerated\n")
		}
	}
	
	return false
}

// verifyServiceAccountPermissions attempts to verify what permissions the service account has
func (c *Client) verifyServiceAccountPermissions(orgID string) {
	// First, try to identify the service account
	fmt.Printf("[Zitadel] Attempting to identify service account...\n")
	serviceAccountID, verifiedMemberships := c.getServiceAccountInfo()
	
	if serviceAccountID != "" {
		fmt.Printf("[Zitadel] Service account User ID: %s\n", serviceAccountID)
	} else {
		fmt.Printf("[Zitadel] WARNING: Could not identify service account User ID\n")
		fmt.Printf("[Zitadel] The Personal Access Token might not allow querying user info\n")
	}
	
	// IMPORTANT: Check if user has project-level permissions but not org membership
	// In Zitadel, project-level roles (like "Org Manager" in a project) do NOT grant organization membership
	// Session creation requires actual organization membership, not just project permissions
	fmt.Printf("[Zitadel] Verifying actual membership in organization %s...\n", orgID)
	fmt.Printf("[Zitadel] NOTE: Project-level roles/permissions do NOT grant organization membership\n")
	fmt.Printf("[Zitadel] Even if you're an 'Org Manager' in a project, you must be added as an organization member\n")
	
	isMember := c.verifyActualMembership(serviceAccountID, orgID)
	
	if !isMember {
		fmt.Printf("[Zitadel] ✗ CRITICAL: Service account is NOT a member of organization %s\n", orgID)
		fmt.Printf("[Zitadel] \n")
		fmt.Printf("[Zitadel] IMPORTANT: Even if you've added roles (Org Owner, End User Impersonator, etc.),\n")
		fmt.Printf("[Zitadel] you MUST regenerate the Personal Access Token for the changes to take effect!\n")
		fmt.Printf("[Zitadel] \n")
		fmt.Printf("[Zitadel] COMMON ISSUE: You might have 'Org Manager' role in a PROJECT, but not be a MEMBER of the ORGANIZATION\n")
		fmt.Printf("[Zitadel] In Zitadel, these are different:\n")
		fmt.Printf("[Zitadel]   - Project-level roles: Permissions within a specific project\n")
		fmt.Printf("[Zitadel]   - Organization membership: Explicit membership in the organization (required for session creation)\n")
		fmt.Printf("[Zitadel] \n")
		if serviceAccountID != "" {
			fmt.Printf("[Zitadel] ACTION REQUIRED:\n")
			fmt.Printf("[Zitadel] 1. Verify in Zitadel Console → Organizations → %s → Members\n", orgID)
			fmt.Printf("[Zitadel] 2. Confirm service account (User ID: %s) is listed as a member\n", serviceAccountID)
			fmt.Printf("[Zitadel] 3. Verify roles are assigned at ORGANIZATION level (not project level)\n")
			fmt.Printf("[Zitadel] 4. CRITICAL: Regenerate the Personal Access Token after adding the member!\n")
			fmt.Printf("[Zitadel]    - Go to Projects → Your Project → Personal Access Tokens\n")
			fmt.Printf("[Zitadel]    - Delete the old token\n")
			fmt.Printf("[Zitadel]    - Create a new Personal Access Token\n")
			fmt.Printf("[Zitadel]    - Update ZITADEL_MANAGEMENT_TOKEN environment variable\n")
			fmt.Printf("[Zitadel]    - Restart your application/service\n")
		} else {
			fmt.Printf("[Zitadel] ACTION REQUIRED:\n")
			fmt.Printf("[Zitadel] 1. Identify which user/service account owns the Personal Access Token\n")
			fmt.Printf("[Zitadel] 2. Verify in Zitadel Console → Organizations → %s → Members\n", orgID)
			fmt.Printf("[Zitadel] 3. Confirm the user/service account is listed as a member\n")
			fmt.Printf("[Zitadel] 4. Verify roles are assigned at ORGANIZATION level\n")
			fmt.Printf("[Zitadel] 5. CRITICAL: Regenerate the Personal Access Token after adding the member!\n")
			fmt.Printf("[Zitadel]    - Go to Projects → Your Project → Personal Access Tokens\n")
			fmt.Printf("[Zitadel]    - Delete the old token\n")
			fmt.Printf("[Zitadel]    - Create a new Personal Access Token\n")
			fmt.Printf("[Zitadel]    - Update ZITADEL_MANAGEMENT_TOKEN environment variable\n")
			fmt.Printf("[Zitadel]    - Restart your application/service\n")
		}
	} else {
		fmt.Printf("[Zitadel] ✓ Service account membership verified - roles should be active\n")
		fmt.Printf("[Zitadel] If you still get 'membership not found' errors, try regenerating the Personal Access Token\n")
	}
	
	if len(verifiedMemberships) > 0 {
		fmt.Printf("[Zitadel] Service account's verified organization memberships: %v\n", verifiedMemberships)
	}
	
	// Additional diagnostic: Try to query members list (this will fail if not a member)
	// Note: verifyActualMembership already did this check, but we log it here for diagnostics
	if !isMember {
		fmt.Printf("[Zitadel] Attempting to query organization members list for diagnostics...\n")
		membersURL := fmt.Sprintf("%s/v2/organizations/%s/members/_search", c.baseURL, orgID)
		searchBody := map[string]interface{}{
			"queries": []map[string]interface{}{},
			"limit":   10,
		}
		
		bodyBytes, err := json.Marshal(searchBody)
		if err == nil {
			req, err := http.NewRequest("POST", membersURL, strings.NewReader(string(bodyBytes)))
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+c.managementToken)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("x-zitadel-orgid", orgID)
				resp, err := c.httpClient.Do(req)
				if err == nil {
					defer resp.Body.Close()
					if resp.StatusCode == http.StatusNotFound {
						fmt.Printf("[Zitadel] Members query returned 404 - confirms service account is not a member\n")
					} else if resp.StatusCode != http.StatusOK {
						body, _ := io.ReadAll(resp.Body)
						fmt.Printf("[Zitadel] Members query diagnostic failed with status %d: %s\n", resp.StatusCode, string(body))
					}
				}
			}
		}
	}
}

// getOAuthTokensForUser uses OAuth2 Resource Owner Password Credentials grant
func (c *Client) getOAuthTokensForUser(email, password string) (*LoginResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth/v2/token", c.baseURL)

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", email)
	data.Set("password", password)
	data.Set("client_id", c.clientID)
	data.Set("scope", "openid profile email offline_access")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed: %s, body: %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}

	return &LoginResponse{
		Success:      true,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    int32(tokenResp.ExpiresIn),
	}, nil
}

// authenticateWithSessionAPI implements the Session API flow for username/password authentication
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password
// Requires: Org User Manager (or Org Owner) organization permission
func (c *Client) authenticateWithSessionAPI(email, password string) (*LoginResponse, error) {
	fmt.Printf("[Zitadel] Starting authentication flow for email: %s\n", email)
	
	// Step 1: Find the user first to determine their organization
	// The user might be in a different organization than the service account
	fmt.Printf("[Zitadel] Step 1: Searching for user by email...\n")
	userID, userOrgID, err := c.findUserByEmailWithOrg(email)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: User lookup failed: %v\n", err)
		return nil, fmt.Errorf("user lookup: %w", err)
	}
	
	fmt.Printf("[Zitadel] Step 1: SUCCESS - Found user ID: %s in organization: %s\n", userID, userOrgID)
	fmt.Printf("[Zitadel] Service account's configured organization ID: %s\n", c.organizationID)
	fmt.Printf("[Zitadel] User's organization ID: %s\n", userOrgID)
	if c.organizationID != "" && c.organizationID != userOrgID {
		fmt.Printf("[Zitadel] WARNING: Service account org (%s) differs from user org (%s)\n", c.organizationID, userOrgID)
	}
	
	// Debug token information
	c.debugTokenInfo()
	
	// Try to verify token's organization context
	fmt.Printf("[Zitadel] Verifying token's organization context...\n")
	tokenOrgID, err := c.getTokenOrganizationContext()
	if err != nil {
		fmt.Printf("[Zitadel] WARNING: Could not determine token's organization context: %v\n", err)
	} else {
		fmt.Printf("[Zitadel] Token appears to be scoped to organization: %s\n", tokenOrgID)
		if tokenOrgID != "" && tokenOrgID != userOrgID {
			fmt.Printf("[Zitadel] ERROR: Token is scoped to organization %s, but user is in organization %s\n", tokenOrgID, userOrgID)
			fmt.Printf("[Zitadel] The Personal Access Token must be generated from a project within organization %s\n", userOrgID)
		}
	}
	
	// Try to verify service account membership and permissions
	fmt.Printf("[Zitadel] Verifying service account permissions in organization %s...\n", userOrgID)
	c.verifyServiceAccountPermissions(userOrgID)

	// Step 2: Create a session
	// Use the user's organization ID (where the user actually exists)
	// The service account must have permissions in the user's organization
	fmt.Printf("[Zitadel] Step 2: Creating session for user in their organization: %s...\n", userOrgID)
	sessionID, err := c.createSessionForUser(email, password, userOrgID)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Session creation failed: %v\n", err)
		// Return a LoginResponse with the error message so it can be properly propagated
		return &LoginResponse{
			Success: false,
			Message: err.Error(),
		}, fmt.Errorf("session creation: %w", err)
	}
	fmt.Printf("[Zitadel] Step 2: SUCCESS - Created session ID: %s\n", sessionID)

	// Step 3: Complete the session and exchange for OAuth tokens
	// Use the user's organization ID (same as session creation)
	fmt.Printf("[Zitadel] Step 3: Completing session and exchanging for OAuth tokens (org: %s)...\n", userOrgID)
	tokens, err := c.completeSessionAndGetTokens(sessionID, userOrgID)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Token exchange failed: %v\n", err)
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	fmt.Printf("[Zitadel] Step 3: SUCCESS - Authentication complete\n")
	return tokens, nil
}

// findUserByEmail searches for a user by email using Zitadel API v2
// Requires: Org User Manager (or Org Owner) organization permission
// Endpoint: POST /v2/users (see https://zitadel.com/docs/apis/resources/user_service_v2/user-service-list-users)
func (c *Client) findUserByEmail(email string) (string, error) {
	// Zitadel User Service v2 uses POST /v2/users endpoint for searching
	searchURL := fmt.Sprintf("%s/v2/users", c.baseURL)

	// Request body structure for Zitadel User Service v2 search
	// See: https://zitadel.com/docs/apis/resources/user_service_v2/user-service-list-users
	searchBody := map[string]interface{}{
		"queries": []map[string]interface{}{
			{
				"emailQuery": map[string]interface{}{
					"emailAddress": email,
					"method":       "TEXT_QUERY_METHOD_CONTAINS_IGNORE_CASE",
				},
			},
		},
		"limit": 1,
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return "", fmt.Errorf("marshal search: %w", err)
	}

	req, err := http.NewRequest("POST", searchURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	// Add organization context header if specified
	if c.organizationID != "" {
		req.Header.Set("x-zitadel-orgid", c.organizationID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Provide more helpful error message for 404
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("search endpoint not found (404): %s. Verify the endpoint path and ensure your service account has Org User Manager permission. Response: %s", searchURL, string(body))
		}
		return "", fmt.Errorf("search failed: %s, body: %s", resp.Status, string(body))
	}

	var searchResult struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(searchResult.Result) == 0 {
		return "", fmt.Errorf("user not found")
	}

	return searchResult.Result[0].ID, nil
}

// findUserByEmailWithOrg searches for a user by email and returns both user ID and organization ID
// This searches without organization filter first to find the user across all accessible organizations
// Requires: Org User Manager (or Org Owner) organization permission
// Endpoint: POST /v2/users (see https://zitadel.com/docs/apis/resources/user_service_v2/user-service-list-users)
func (c *Client) findUserByEmailWithOrg(email string) (userID string, orgID string, err error) {
	// Zitadel User Service v2 uses POST /v2/users endpoint for searching
	// Try searching without organization filter first (service account can search across orgs it has access to)
	searchURL := fmt.Sprintf("%s/v2/users", c.baseURL)

	// Request body structure for Zitadel User Service v2 search
	// See: https://zitadel.com/docs/apis/resources/user_service_v2/user-service-list-users
	searchBody := map[string]interface{}{
		"queries": []map[string]interface{}{
			{
				"emailQuery": map[string]interface{}{
					"emailAddress": email,
					"method":       "TEXT_QUERY_METHOD_CONTAINS_IGNORE_CASE",
				},
			},
		},
		"limit": 1,
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return "", "", fmt.Errorf("marshal search: %w", err)
	}

	req, err := http.NewRequest("POST", searchURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	// Don't set organization header - search across all organizations the service account has access to
	// If that fails, fall back to searching in the specified organization
	fmt.Printf("[Zitadel] Searching for user '%s' across all accessible organizations (no org filter)\n", email)
	fmt.Printf("[Zitadel] Request URL: %s\n", searchURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: User search request failed: %v\n", err)
		return "", "", fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[Zitadel] User search response status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Zitadel] User search failed with status %d, body: %s\n", resp.StatusCode, string(body))
		// If search without org fails, try with the service account's organization
		if c.organizationID != "" {
			fmt.Printf("[Zitadel] Falling back to search in service account's organization: %s\n", c.organizationID)
			return c.findUserByEmailInOrg(email, c.organizationID)
		}
		if resp.StatusCode == http.StatusNotFound {
			return "", "", fmt.Errorf("search endpoint not found (404): %s. Verify the endpoint path and ensure your service account has Org User Manager permission. Response: %s", searchURL, string(body))
		}
		return "", "", fmt.Errorf("search failed: %s, body: %s", resp.Status, string(body))
	}

	// Read the response body first to log it
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read response: %w", err)
	}
	fmt.Printf("[Zitadel] User search response body: %s\n", string(responseBody))

	var searchResult struct {
		Result []struct {
			UserID string `json:"userId"`
			Details struct {
				ResourceOwner string `json:"resourceOwner"`
			} `json:"details"`
		} `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &searchResult); err != nil {
		fmt.Printf("[Zitadel] ERROR: Failed to decode search response: %v\n", err)
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	fmt.Printf("[Zitadel] Parsed search result: %d users found\n", len(searchResult.Result))
	if len(searchResult.Result) > 0 {
		fmt.Printf("[Zitadel] First result: UserID=%s, ResourceOwner=%s\n", searchResult.Result[0].UserID, searchResult.Result[0].Details.ResourceOwner)
	}

	if len(searchResult.Result) == 0 {
		fmt.Printf("[Zitadel] No users found in global search\n")
		// Try searching in the service account's organization as fallback
		if c.organizationID != "" {
			fmt.Printf("[Zitadel] Falling back to search in service account's organization: %s\n", c.organizationID)
			return c.findUserByEmailInOrg(email, c.organizationID)
		}
		return "", "", fmt.Errorf("user not found")
	}

	userID = searchResult.Result[0].UserID
	orgID = searchResult.Result[0].Details.ResourceOwner
	
	if userID == "" {
		fmt.Printf("[Zitadel] ERROR: User ID is empty in search result!\n")
		return "", "", fmt.Errorf("user ID is empty in search result")
	}
	
	if orgID == "" {
		fmt.Printf("[Zitadel] WARNING: Organization ID not in search response, using service account's organization: %s\n", c.organizationID)
		// If organization ID not in response, use service account's organization
		orgID = c.organizationID
	}

	fmt.Printf("[Zitadel] User search successful - User ID: %s, Organization ID: %s\n", userID, orgID)
	return userID, orgID, nil
}

// findUserByEmailInOrg searches for a user by email within a specific organization
func (c *Client) findUserByEmailInOrg(email, orgID string) (userID string, foundOrgID string, err error) {
	searchURL := fmt.Sprintf("%s/v2/users", c.baseURL)

	searchBody := map[string]interface{}{
		"queries": []map[string]interface{}{
			{
				"emailQuery": map[string]interface{}{
					"emailAddress": email,
					"method":       "TEXT_QUERY_METHOD_CONTAINS_IGNORE_CASE",
				},
			},
		},
		"limit": 1,
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return "", "", fmt.Errorf("marshal search: %w", err)
	}

	req, err := http.NewRequest("POST", searchURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-zitadel-orgid", orgID)
	fmt.Printf("[Zitadel] Searching for user '%s' in organization %s\n", email, orgID)
	fmt.Printf("[Zitadel] Request URL: %s\n", searchURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: User search request failed: %v\n", err)
		return "", "", fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[Zitadel] User search (org-specific) response status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Zitadel] User search failed with status %d, body: %s\n", resp.StatusCode, string(body))
		return "", "", fmt.Errorf("search failed: %s, body: %s", resp.Status, string(body))
	}

	// Read the response body first to log it
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read response: %w", err)
	}
	fmt.Printf("[Zitadel] User search (org-specific) response body: %s\n", string(responseBody))

	var searchResult struct {
		Result []struct {
			UserID string `json:"userId"`
			Details struct {
				ResourceOwner string `json:"resourceOwner"`
			} `json:"details"`
		} `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &searchResult); err != nil {
		fmt.Printf("[Zitadel] ERROR: Failed to decode search response: %v\n", err)
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	fmt.Printf("[Zitadel] Parsed search result: %d users found\n", len(searchResult.Result))
	if len(searchResult.Result) > 0 {
		fmt.Printf("[Zitadel] First result: UserID=%s, ResourceOwner=%s\n", searchResult.Result[0].UserID, searchResult.Result[0].Details.ResourceOwner)
	}

	if len(searchResult.Result) == 0 {
		fmt.Printf("[Zitadel] No users found in organization %s\n", orgID)
		return "", "", fmt.Errorf("user not found in organization %s", orgID)
	}

	userID = searchResult.Result[0].UserID
	foundOrgID = searchResult.Result[0].Details.ResourceOwner
	
	if userID == "" {
		fmt.Printf("[Zitadel] ERROR: User ID is empty in search result!\n")
		return "", "", fmt.Errorf("user ID is empty in search result")
	}
	
	if foundOrgID == "" {
		fmt.Printf("[Zitadel] Organization ID not in response, using requested org: %s\n", orgID)
		foundOrgID = orgID
	}

	fmt.Printf("[Zitadel] User search (org-specific) successful - User ID: %s, Organization ID: %s\n", userID, foundOrgID)
	return userID, foundOrgID, nil
}

// createSessionForUser creates a session for a user
// See: https://zitadel.com/docs/apis/resources/session_service_v2/session-service-create-session
// Requires: Org User Manager (or Org Owner) organization permission
// Tries multiple strategies: without org header (token's default), then with specified org ID
func (c *Client) createSessionForUser(email, password, orgID string) (string, error) {
	// First, verify the token can access the organization by trying a simple org query
	fmt.Printf("[Zitadel] Pre-flight check: Verifying token can access organization %s...\n", orgID)
	if err := c.verifyTokenCanAccessOrg(orgID); err != nil {
		fmt.Printf("[Zitadel] WARNING: Token access verification failed: %v\n", err)
		fmt.Printf("[Zitadel] This might indicate the token doesn't have the right permissions\n")
	} else {
		fmt.Printf("[Zitadel] ✓ Token can access organization %s\n", orgID)
	}
	
	// Session API uses /v2/sessions endpoint (same as User API, not /management/v2)
	sessionURL := fmt.Sprintf("%s/v2/sessions", c.baseURL)
	
	// Try creating session with organization context in the body instead of header
	// Some Zitadel APIs require org context in the request body
	sessionBody := map[string]interface{}{}
	if orgID != "" {
		sessionBody["organizationId"] = orgID
		fmt.Printf("[Zitadel] Creating session with organizationId in request body: %s\n", orgID)
	} else {
		fmt.Printf("[Zitadel] Creating session without organizationId in request body\n")
	}
	
	bodyBytes, err := json.Marshal(sessionBody)
	if err != nil {
		return "", fmt.Errorf("marshal session: %w", err)
	}

	// Strategy 1: Try creating session WITHOUT organizationId in body, but WITH header
	// Some Zitadel APIs require org context only in header, not body
	fmt.Printf("[Zitadel] Strategy 1: Creating session with org header only (no orgId in body)...\n")
	emptyBodyBytes, _ := json.Marshal(map[string]interface{}{})
	sessionID, err := c.tryCreateSession(sessionURL, emptyBodyBytes, orgID)
	if err == nil {
		// Session created, now update it with credentials
		return c.updateSessionWithCredentials(sessionID, email, password, orgID)
	}
	fmt.Printf("[Zitadel] Strategy 1 failed: %v\n", err)
	
	// Strategy 2: Try creating session with organizationId in body but NO header
	// Some Zitadel APIs prefer body over header for organization context
	fmt.Printf("[Zitadel] Strategy 2: Creating session with organizationId in body (no header)...\n")
	sessionID, err = c.tryCreateSession(sessionURL, bodyBytes, "")
	if err == nil {
		// Session created, now update it with credentials
		return c.updateSessionWithCredentials(sessionID, email, password, orgID)
	}
	fmt.Printf("[Zitadel] Strategy 2 failed: %v\n", err)
	
	// Strategy 3: Try with organization header AND body
	if orgID != "" {
		fmt.Printf("[Zitadel] Strategy 3: Creating session with organization header AND body: %s\n", orgID)
		sessionID, err2 := c.tryCreateSession(sessionURL, bodyBytes, orgID)
		if err2 == nil {
			// Session created, now update it with credentials using the same org ID
			return c.updateSessionWithCredentials(sessionID, email, password, orgID)
		}
		fmt.Printf("[Zitadel] Strategy 3 failed: %v\n", err2)
		// Return the more specific error from the org-specific attempt
		return "", err2
	}
	
	return "", err
}

// verifyTokenCanAccessOrg performs a simple check to see if the token can access the organization
func (c *Client) verifyTokenCanAccessOrg(orgID string) error {
	// Try to get organization details - this is a simple read operation
	orgURL := fmt.Sprintf("%s/v2/organizations/%s", c.baseURL, orgID)
	
	req, err := http.NewRequest("GET", orgURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("x-zitadel-orgid", orgID)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		fmt.Printf("[Zitadel] ✓ Token can successfully query organization %s\n", orgID)
		return nil
	}
	
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("cannot access organization (status %d): %s", resp.StatusCode, string(body))
}

// updateSessionWithCredentials updates a session with user credentials
func (c *Client) updateSessionWithCredentials(sessionID, email, password, orgID string) (string, error) {
	fmt.Printf("[Zitadel] Updating session %s with user credentials...\n", sessionID)
	// Session API uses /v2/sessions endpoint (same as User API, not /management/v2)
	updateURL := fmt.Sprintf("%s/v2/sessions/%s", c.baseURL, sessionID)

	updateBody := map[string]interface{}{
		"checks": map[string]interface{}{
			"user": map[string]interface{}{
				"loginName": email,
				"password": map[string]interface{}{
					"password": password,
				},
			},
		},
		"challenges": []string{"PASSWORD"},
	}

	updateBytes, err := json.Marshal(updateBody)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Failed to marshal session update body: %v\n", err)
		return "", fmt.Errorf("marshal update: %w", err)
	}

	updateReq, err := http.NewRequest("PUT", updateURL, strings.NewReader(string(updateBytes)))
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Failed to create session update request: %v\n", err)
		return "", fmt.Errorf("create update request: %w", err)
	}
	updateReq.Header.Set("Authorization", "Bearer "+c.managementToken)
	updateReq.Header.Set("Content-Type", "application/json")
	// Use the same organization ID that was used for session creation
	if orgID != "" {
		updateReq.Header.Set("x-zitadel-orgid", orgID)
		fmt.Printf("[Zitadel] Session update request: PUT %s (with x-zitadel-orgid=%s)\n", updateURL, orgID)
	} else {
		fmt.Printf("[Zitadel] Session update request: PUT %s (no org header, using token's default context)\n", updateURL)
	}

	updateResp, err := c.httpClient.Do(updateReq)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Session update request failed: %v\n", err)
		return "", fmt.Errorf("session update request: %w", err)
	}
	defer updateResp.Body.Close()

	fmt.Printf("[Zitadel] Session update response status: %s\n", updateResp.Status)

	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		fmt.Printf("[Zitadel] ERROR: Session update failed with status %d, body: %s\n", updateResp.StatusCode, string(body))
		return "", fmt.Errorf("session update failed: %s, body: %s", updateResp.Status, string(body))
	}

	fmt.Printf("[Zitadel] Session updated successfully\n")
	return sessionID, nil
}

// tryCreateSession attempts to create a session with optional organization header
func (c *Client) tryCreateSession(sessionURL string, bodyBytes []byte, orgID string) (string, error) {
	req, err := http.NewRequest("POST", sessionURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	
	if orgID != "" {
		req.Header.Set("x-zitadel-orgid", orgID)
		fmt.Printf("[Zitadel] Session creation request: POST %s (with x-zitadel-orgid=%s)\n", sessionURL, orgID)
	} else {
		fmt.Printf("[Zitadel] Session creation request: POST %s (no org header, using token's default context)\n", sessionURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Session creation request failed: %v\n", err)
		return "", fmt.Errorf("session request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[Zitadel] Session creation response status: %s\n", resp.Status)
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Zitadel] Error response body: %s\n", string(body))
		if resp.StatusCode == http.StatusNotFound {
			orgContext := "token's default organization"
			if orgID != "" {
				orgContext = fmt.Sprintf("organization ID: %s", orgID)
			}
			errorMsg := fmt.Sprintf("session creation failed (404): %s. The service account used for ZITADEL_MANAGEMENT_TOKEN is not a member of the %s, or does not have 'Org User Manager' or 'Org Owner' permission.", resp.Status, orgContext)
			errorMsg += "\n\nCRITICAL: If you just added roles (Org Owner, End User Impersonator, etc.),"
			errorMsg += "\nyou MUST regenerate the Personal Access Token for the changes to take effect!"
			errorMsg += "\n\nTroubleshooting steps:"
			errorMsg += "\n1. Verify the service account is a member of the organization in Zitadel Console"
			errorMsg += "\n   (Organizations → " + orgID + " → Members)"
			errorMsg += "\n2. Ensure roles are assigned at ORGANIZATION level (not project level)"
			errorMsg += "\n3. CRITICAL: Regenerate the Personal Access Token after adding the member:"
			errorMsg += "\n   - Go to Projects → Your Project → Personal Access Tokens"
			errorMsg += "\n   - Delete the old token"
			errorMsg += "\n   - Create a new Personal Access Token"
			errorMsg += "\n   - Update ZITADEL_MANAGEMENT_TOKEN environment variable"
			errorMsg += "\n   - Restart your application/service"
			if orgID != "" {
				errorMsg += fmt.Sprintf("\n4. Verify the organization ID %s matches the organization where the service account is a member", orgID)
			} else {
				errorMsg += "\n4. The token might be scoped to a different organization. Try setting ZITADEL_ORGANIZATION_ID to the correct organization ID"
			}
			errorMsg += fmt.Sprintf("\n\nResponse: %s", string(body))
			return "", fmt.Errorf("%s", errorMsg)
		}
		return "", fmt.Errorf("session creation failed: %s, body: %s", resp.Status, string(body))
	}

	var sessionResult struct {
		SessionID    string `json:"sessionId"`
		SessionToken string `json:"sessionToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sessionResult); err != nil {
		return "", fmt.Errorf("decode session: %w", err)
	}

	if sessionResult.SessionID == "" {
		return "", fmt.Errorf("no session ID in response")
	}

	sessionID := sessionResult.SessionID
	fmt.Printf("[Zitadel] Session created successfully - Session ID: %s\n", sessionID)
	return sessionID, nil
}

// completeSessionAndGetTokens completes the session and exchanges it for OAuth tokens
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password
func (c *Client) completeSessionAndGetTokens(sessionID string, userOrgID string) (*LoginResponse, error) {
	// Step 1: Set intent to authenticate and get session token
	fmt.Printf("[Zitadel] Creating OIDC intent for session %s...\n", sessionID)
	// Session API uses /v2/sessions endpoint (same as User API, not /management/v2)
	intentURL := fmt.Sprintf("%s/v2/sessions/%s/intents/oidc", c.baseURL, sessionID)
	
	intentBody := map[string]interface{}{
		"clientId": c.clientID,
		"scope":    []string{"openid", "profile", "email", "offline_access"},
	}

	bodyBytes, err := json.Marshal(intentBody)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Failed to marshal intent body: %v\n", err)
		return nil, fmt.Errorf("marshal intent: %w", err)
	}

	req, err := http.NewRequest("POST", intentURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Failed to create intent request: %v\n", err)
		return nil, fmt.Errorf("create intent request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	// Use the user's organization ID for the intent as well
	if userOrgID != "" {
		req.Header.Set("x-zitadel-orgid", userOrgID)
		fmt.Printf("[Zitadel] Intent request: POST %s\n", intentURL)
		fmt.Printf("[Zitadel] Request headers: Authorization=Bearer <token>, x-zitadel-orgid=%s\n", userOrgID)
	} else {
		fmt.Printf("[Zitadel] WARNING: No organization ID for intent creation\n")
		fmt.Printf("[Zitadel] Intent request: POST %s (no org header)\n", intentURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Intent request failed: %v\n", err)
		return nil, fmt.Errorf("intent request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[Zitadel] Intent creation response status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Zitadel] ERROR: Intent creation failed with status %d, body: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("intent creation failed: %s, body: %s", resp.Status, string(body))
	}

	var intentResult struct {
		AuthRequestID string `json:"authRequestId"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&intentResult); err != nil {
		fmt.Printf("[Zitadel] ERROR: Failed to decode intent response: %v\n", err)
		return nil, fmt.Errorf("decode intent: %w", err)
	}

	if intentResult.AuthRequestID == "" {
		fmt.Printf("[Zitadel] ERROR: No auth request ID in intent response\n")
		return nil, fmt.Errorf("no auth request ID in response")
	}

	fmt.Printf("[Zitadel] Intent created successfully - Auth Request ID: %s\n", intentResult.AuthRequestID)

	// Step 2: Exchange auth request for tokens
	fmt.Printf("[Zitadel] Exchanging auth request ID for OAuth tokens...\n")
	return c.exchangeAuthRequestForTokens(intentResult.AuthRequestID)
}

// exchangeAuthRequestForTokens exchanges an auth request ID for OAuth tokens
func (c *Client) exchangeAuthRequestForTokens(authRequestID string) (*LoginResponse, error) {
	// Use the OAuth token endpoint with the auth request ID
	tokenURL := fmt.Sprintf("%s/oauth/v2/token", c.baseURL)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", authRequestID)
	data.Set("client_id", c.clientID)
	data.Set("redirect_uri", "urn:ietf:wg:oauth:2.0:oob") // Out-of-band redirect for service accounts

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Failed to create token request: %v\n", err)
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	fmt.Printf("[Zitadel] Token exchange request: POST %s\n", tokenURL)
	fmt.Printf("[Zitadel] Auth Request ID: %s\n", authRequestID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("[Zitadel] ERROR: Token exchange request failed: %v\n", err)
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[Zitadel] Token exchange response status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Zitadel] ERROR: Token exchange failed with status %d, body: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token exchange failed: %s, body: %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		fmt.Printf("[Zitadel] ERROR: Failed to decode token response: %v\n", err)
		return nil, fmt.Errorf("decode token: %w", err)
	}

	if tokenResp.AccessToken == "" {
		fmt.Printf("[Zitadel] ERROR: No access token in response\n")
		return nil, fmt.Errorf("no access token in response")
	}

	fmt.Printf("[Zitadel] Token exchange successful - Access token received (expires in %d seconds)\n", tokenResp.ExpiresIn)

	return &LoginResponse{
		Success:      true,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    int32(tokenResp.ExpiresIn),
	}, nil
}
