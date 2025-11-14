package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// AuditLogInterceptor creates a Connect interceptor for audit logging
func AuditLogInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			startTime := time.Now()
			procedure := req.Spec().Procedure


			// Skip audit logging for certain procedures
			if shouldSkipAuditLog(procedure) {
				return next(ctx, req)
			}

			// Extract IP address and user agent (before calling next)
			ipAddress := getClientIP(req)
			
			userAgent := req.Header().Get("User-Agent")
			if userAgent == "" {
				userAgent = "unknown"
			}

			// Parse service and action from procedure
			service, action := parseProcedure(procedure)

			// Extract resource information from request (before calling next)
			// Use panic recovery for resource extraction to prevent panics from breaking the request
			var resourceType, resourceID, orgID *string
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("[Audit] Panic extracting resource info for %s: %v", procedure, r)
					}
				}()
				resourceType, resourceID, orgID = extractResourceInfo(req, procedure)
			}()

			// Sanitize request data (remove sensitive fields)
			// Use panic recovery for sanitization to prevent panics from breaking the request
			var requestData string
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("[Audit] Panic sanitizing request data for %s: %v", procedure, r)
						requestData = "{}"
					}
				}()
				requestData = sanitizeRequestData(req)
			}()

			// Execute the request (auth interceptor will set user in context)
			// Note: In Connect, with connect.WithInterceptors(auditInterceptor, authInterceptor),
			// authInterceptor wraps the handler (innermost, runs first), then auditInterceptor
			// wraps authInterceptor (outermost, runs second). This means when auditInterceptor
			// calls next(), it runs authInterceptor which sets the user in context and calls the handler.
			// The context chain should allow us to access the user via context.Value().
			var userID string = "system"
			
			resp, err := next(ctx, req)

			// Try to extract user from context - the auth interceptor should have set it
			// context.Value() searches up the chain, so if the inner interceptor set it,
			// we should be able to access it. If that fails, try response headers.
			user, userErr := auth.GetUserFromContext(ctx)
			if user != nil && user.Id != "" {
				userID = user.Id
				logger.Debug("[Audit] Extracted user ID from context: %s", userID)
			} else {
				// Fallback: Try to extract from response headers (if auth interceptor exposes them)
				// Use panic recovery since resp might be a typed nil that passes != nil check
				if resp != nil {
					func() {
						defer func() {
							if r := recover(); r != nil {
								// resp is a typed nil - ignore header extraction
							}
						}()
						// Defensively check that Header() doesn't return nil
						if headers := resp.Header(); headers != nil {
							if userIDHeader := headers.Get("X-User-ID"); userIDHeader != "" {
								userID = userIDHeader
								logger.Debug("[Audit] Extracted user ID from response header: %s", userID)
							}
						}
					}()
				}
				// If we still don't have a user ID and there was an error getting it, log it
				if userID == "system" && userErr != nil {
					logger.Debug("[Audit] Could not extract user from context for %s: %v", procedure, userErr)
				}
			}

			// Calculate duration
			duration := time.Since(startTime)

			// Determine response status
			responseStatus := int32(0)
			var errorMessage *string
			if err != nil {
				if connectErr, ok := err.(*connect.Error); ok {
					responseStatus = int32(connectErr.Code())
					msg := connectErr.Message()
					errorMessage = &msg
				} else {
					responseStatus = 500
					msg := err.Error()
					errorMessage = &msg
				}
			} else {
				responseStatus = 200 // Success
			}

			// Create audit log entry asynchronously (don't block the request)
			// Use background context with timeout to avoid cancellation when request completes
			go func() {
				// Recover from any panics in the goroutine
				defer func() {
					if r := recover(); r != nil {
						logger.Error("[Audit] Panic in audit log goroutine for %s/%s: %v", service, action, r)
					}
				}()
				
				// Use background context with timeout instead of request context to avoid cancellation
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				
				// Log what we're saving for debugging
				orgIDStr := "nil"
				if orgID != nil {
					orgIDStr = *orgID
				}
				logger.Debug("[Audit] Saving log: service=%s, action=%s, userID=%s, orgID=%s, resourceType=%v, resourceID=%v",
					service, action, userID, orgIDStr, resourceType, resourceID)

				if err := createAuditLog(ctx, auditLogData{
					UserID:         userID,
					OrganizationID: orgID,
					Action:         action,
					Service:        service,
					ResourceType:   resourceType,
					ResourceID:     resourceID,
					IPAddress:      ipAddress,
					UserAgent:      userAgent,
					RequestData:    requestData,
					ResponseStatus: responseStatus,
					ErrorMessage:   errorMessage,
					DurationMs:     duration.Milliseconds(),
				}); err != nil {
					logger.Error("[Audit] Failed to create audit log for %s/%s: %v", service, action, err)
				}
			}()

			return resp, err
		}
	}
}

type auditLogData struct {
	UserID         string
	OrganizationID *string
	Action         string
	Service        string
	ResourceType   *string
	ResourceID     *string
	IPAddress      string
	UserAgent      string
	RequestData    string
	ResponseStatus int32
	ErrorMessage   *string
	DurationMs     int64
}

func createAuditLog(ctx context.Context, data auditLogData) error {
	// Use MetricsDB (TimescaleDB) for audit logs - no fallback to main DB
	// This ensures audit logs are always stored in TimescaleDB for optimal performance
	if database.MetricsDB == nil {
		return fmt.Errorf("metrics database (TimescaleDB) not initialized - audit logs require TimescaleDB")
	}

	db := database.MetricsDB

	auditLog := database.AuditLog{
		ID:             uuid.New().String(),
		UserID:         data.UserID,
		OrganizationID: data.OrganizationID,
		Action:         data.Action,
		Service:        data.Service,
		ResourceType:   data.ResourceType,
		ResourceID:     data.ResourceID,
		IPAddress:      data.IPAddress,
		UserAgent:      data.UserAgent,
		RequestData:    data.RequestData,
		ResponseStatus: data.ResponseStatus,
		ErrorMessage:   data.ErrorMessage,
		DurationMs:     data.DurationMs,
		CreatedAt:      time.Now(),
	}

	if err := db.WithContext(ctx).Create(&auditLog).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// shouldSkipAuditLog determines if a procedure should be skipped for audit logging
// We only log actual actions (mutations), not read-only operations
func shouldSkipAuditLog(procedure string) bool {
	// Skip public endpoints
	skipProcedures := []string{
		"/obiente.cloud.auth.v1.AuthService/Login",
		"/obiente.cloud.auth.v1.AuthService/GetPublicConfig",
		"/obiente.cloud.superadmin.v1.SuperadminService/GetPricing",
	}

	for _, skip := range skipProcedures {
		if procedure == skip {
			return true
		}
	}

	// Extract action name from procedure
	parts := strings.Split(procedure, "/")
	if len(parts) < 2 {
		return false // Unknown format, log it to be safe
	}
	action := parts[len(parts)-1]

	// Skip read-only operations (List, Get, Stream, Watch operations)
	skipPrefixes := []string{
		"List",     // ListDeployments, ListBuilds, ListOrganizations, etc.
		"Get",      // GetDeployment, GetBuild, GetOrganization, etc.
		"Stream",   // StreamBuildLogs, StreamDeploymentLogs, etc.
		"Watch",    // WatchDeployment, etc.
		"Query",    // QueryDeployments, etc.
		"Search",   // SearchDeployments, etc.
		"Validate", // ValidateDeploymentCompose, etc. (read-only validation)
		"Check",    // CheckDomain, CheckStatus, etc.
		"Has",      // HasDelegatedDNS, HasPermission, etc. (read-only checks)
		"Ping",     // Health checks
		"Health",   // Health checks
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(action, prefix) {
			return true
		}
	}

	// Log all mutations (Create, Update, Delete, Start, Stop, Deploy, Invite, Leave, etc.)
	// These will be captured for audit logging
	return false
}

// parseProcedure extracts service and action from procedure path
func parseProcedure(procedure string) (service, action string) {
	// Procedure format: /package.Service/Method
	parts := strings.Split(strings.TrimPrefix(procedure, "/"), "/")
	if len(parts) != 2 {
		return "unknown", "unknown"
	}

	serviceParts := strings.Split(parts[0], ".")
	if len(serviceParts) > 0 {
		service = serviceParts[len(serviceParts)-1]
	} else {
		service = parts[0]
	}

	action = parts[1]
	return service, action
}

// extractResourceInfo extracts resource type, ID, and organization ID from request
func extractResourceInfo(req connect.AnyRequest, procedure string) (resourceType *string, resourceID *string, orgID *string) {
	// Try to extract from request message
	msg := req.Any()
	if msg == nil {
		return nil, nil, nil
	}

	// Use reflection to extract common fields
	if protoMsg, ok := msg.(proto.Message); ok {
		// Convert to JSON to extract fields
		jsonBytes, err := protojson.Marshal(protoMsg)
		if err == nil {
			var jsonData map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &jsonData); err == nil {
				// Parse procedure once
				_, action := parseProcedure(procedure)

				// Try to extract organization_id from request
			if orgIDVal, ok := jsonData["organizationId"].(string); ok && orgIDVal != "" {
				orgID = &orgIDVal
			} else if orgIDVal, ok := jsonData["organization_id"].(string); ok && orgIDVal != "" {
				orgID = &orgIDVal
			}

				// Determine resource type and ID based on procedure
				resourceType, resourceID = inferResourceFromAction(action, jsonData)

				// If we have a deployment ID but no org ID, try to look it up from the deployment
				if orgID == nil && resourceType != nil && *resourceType == "deployment" && resourceID != nil && *resourceID != "" {
					if lookedUpOrgID := lookupDeploymentOrgID(*resourceID); lookedUpOrgID != nil {
						orgID = lookedUpOrgID
					}
				}

				// If we have a VPS ID but no org ID, try to look it up from the VPS
				if orgID == nil && resourceType != nil && *resourceType == "vps" && resourceID != nil && *resourceID != "" {
					if lookedUpOrgID := lookupVPSOrgID(*resourceID); lookedUpOrgID != nil {
						orgID = lookedUpOrgID
					}
				}
			}
		}
	}

	return resourceType, resourceID, orgID
}

// lookupDeploymentOrgID looks up the organization ID for a deployment from the database
func lookupDeploymentOrgID(deploymentID string) *string {
	if database.DB == nil {
		return nil
	}

	var orgID string
	if err := database.DB.Table("deployments").
		Select("organization_id").
		Where("id = ?", deploymentID).
		Pluck("organization_id", &orgID).Error; err != nil {
		return nil
	}

	if orgID == "" {
		return nil
	}

	return &orgID
}

// lookupVPSOrgID looks up the organization ID for a VPS from the database
func lookupVPSOrgID(vpsID string) *string {
	if database.DB == nil {
		return nil
	}

	var orgID string
	if err := database.DB.Table("vps_instances").
		Select("organization_id").
		Where("id = ? AND deleted_at IS NULL", vpsID).
		Pluck("organization_id", &orgID).Error; err != nil {
		return nil
	}

	if orgID == "" {
		return nil
	}

	return &orgID
}

// lookupSSHKeyVPSID looks up the VPS ID for an SSH key from the database
// Returns nil if the key is organization-wide or not found
func lookupSSHKeyVPSID(keyID string) *string {
	if database.DB == nil {
		return nil
	}

	// Use a struct to properly handle NULL values
	var result struct {
		VPSID *string `gorm:"column:vps_id"`
	}
	if err := database.DB.Table("ssh_keys").
		Select("vps_id").
		Where("id = ?", keyID).
		First(&result).Error; err != nil {
		// Key not found or error
		return nil
	}

	// If vpsID is nil or empty, the key is organization-wide
	if result.VPSID == nil || *result.VPSID == "" {
		return nil
	}

	return result.VPSID
}

// inferResourceFromAction infers resource type and ID from action name and request data
func inferResourceFromAction(action string, jsonData map[string]interface{}) (resourceType *string, resourceID *string) {
	actionLower := strings.ToLower(action)

	// Build-related actions (ListBuilds, GetBuild, etc.) - these are related to deployments
	if strings.Contains(actionLower, "build") {
		rt := "deployment"
		resourceType = &rt

		// Try to extract deployment ID from build-related requests
		if id, ok := jsonData["deploymentId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["deployment_id"].(string); ok && id != "" {
			resourceID = &id
		}
		return
	}

	// Deployment-related actions
	if strings.Contains(actionLower, "deployment") {
		rt := "deployment"
		resourceType = &rt

		// Try to extract deployment ID
		if id, ok := jsonData["deploymentId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["deployment_id"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["id"].(string); ok && id != "" {
			resourceID = &id
		}
		return
	}

	// SSH key actions - check if they're VPS-specific by looking for vpsId in request
	// Action names: AddSSHKey, RemoveSSHKey
	if strings.Contains(actionLower, "ssh") && strings.Contains(actionLower, "key") && !strings.Contains(actionLower, "vps") {
		// For AddSSHKey, check if there's a vpsId in the request
		if strings.Contains(actionLower, "add") {
			// Check both camelCase and snake_case field names
			var vpsID string
			var hasVPSID bool
			if vpsIDVal, ok := jsonData["vpsId"].(string); ok && vpsIDVal != "" {
				vpsID = vpsIDVal
				hasVPSID = true
			} else if vpsIDVal, ok := jsonData["vps_id"].(string); ok && vpsIDVal != "" {
				vpsID = vpsIDVal
				hasVPSID = true
			}
			
			if hasVPSID {
				rt := "vps"
				resourceType = &rt
				resourceID = &vpsID
				logger.Debug("[Audit] SSH key action %s is VPS-specific (vpsId: %s)", action, vpsID)
				return
			}
		}
		
		// For RemoveSSHKey, we need to look up the key in the database to see if it's VPS-specific
		if strings.Contains(actionLower, "remove") {
			// Extract key_id from request
			var keyID string
			if keyIDVal, ok := jsonData["keyId"].(string); ok && keyIDVal != "" {
				keyID = keyIDVal
			} else if keyIDVal, ok := jsonData["key_id"].(string); ok && keyIDVal != "" {
				keyID = keyIDVal
			}
			
			if keyID != "" {
				// Look up the key in database to see if it's VPS-specific
				// This MUST happen before the key is deleted in the handler
				if vpsID := lookupSSHKeyVPSID(keyID); vpsID != nil && *vpsID != "" {
					rt := "vps"
					resourceType = &rt
					resourceID = vpsID
					return
				}
			}
		}
		
		// Otherwise, it's an organization-wide SSH key
		rt := "organization"
		resourceType = &rt
		// Extract organization ID as resource ID
		if id, ok := jsonData["organizationId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["organization_id"].(string); ok && id != "" {
			resourceID = &id
		}
		logger.Debug("[Audit] SSH key action %s is organization-wide (orgId: %v)", action, resourceID)
		return
	}

	// Organization-related actions
	// Check for organization in action name OR organization member actions
	organizationActions := []string{
		"invite",      // InviteMember
		"remove",      // RemoveMember
		"decline",     // DeclineInvite
		"accept",      // AcceptInvite
		"leave",       // LeaveOrganization
		"create",      // CreateOrganization
		"update",      // UpdateOrganization
		"delete",      // DeleteOrganization
		"member",      // Any member-related action
		"organization", // Direct organization actions
	}
	
	isOrgAction := strings.Contains(actionLower, "organization")
	if !isOrgAction {
		for _, orgAction := range organizationActions {
			if strings.Contains(actionLower, orgAction) {
				// Check if request has organizationId - if so, it's an org action
				if orgIDVal, ok := jsonData["organizationId"].(string); ok && orgIDVal != "" {
					isOrgAction = true
					break
				} else if orgIDVal, ok := jsonData["organization_id"].(string); ok && orgIDVal != "" {
					isOrgAction = true
					break
				}
			}
		}
	}

	if isOrgAction {
		rt := "organization"
		resourceType = &rt

		// Extract organization ID as resource ID
		if id, ok := jsonData["organizationId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["organization_id"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["id"].(string); ok && id != "" {
			resourceID = &id
		}
		return
	}

	// Game server-related actions
	if strings.Contains(actionLower, "gameserver") {
		rt := "game_server"
		resourceType = &rt

		if id, ok := jsonData["gameServerId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["game_server_id"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["id"].(string); ok && id != "" {
			resourceID = &id
		}
		return
	}

	// VPS-related actions
	// Simplified: If action contains "vps" OR vpsId is present in request, it's a VPS action
	isVPSAction := strings.Contains(actionLower, "vps")
	
	// Also check if vpsId is present in the request (catches VPSConfigService actions without "vps" in name)
	if !isVPSAction {
		if vpsIDVal, ok := jsonData["vpsId"].(string); ok && vpsIDVal != "" {
			isVPSAction = true
		} else if vpsIDVal, ok := jsonData["vps_id"].(string); ok && vpsIDVal != "" {
			isVPSAction = true
		}
	}
	
	if isVPSAction {
		rt := "vps"
		resourceType = &rt

		// Try vpsId first (most common)
		if id, ok := jsonData["vpsId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["vps_id"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["id"].(string); ok && id != "" {
			resourceID = &id
		}
		return
	}

	// Billing-related actions
	if strings.Contains(actionLower, "billing") {
		rt := "billing"
		resourceType = &rt

		// Try to extract billing account ID or organization ID
		if id, ok := jsonData["billingAccountId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["billing_account_id"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["organizationId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["organization_id"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["id"].(string); ok && id != "" {
			resourceID = &id
		}
		return
	}

	// Support ticket-related actions
	if strings.Contains(actionLower, "ticket") || strings.Contains(actionLower, "comment") {
		rt := "support_ticket"
		resourceType = &rt

		if id, ok := jsonData["ticketId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["ticket_id"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["id"].(string); ok && id != "" {
			resourceID = &id
		}
		return
	}

	// Admin/Role-related actions
	if strings.Contains(actionLower, "role") || strings.Contains(actionLower, "binding") {
		rt := "role"
		resourceType = &rt

		if id, ok := jsonData["roleId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["role_id"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["bindingId"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["binding_id"].(string); ok && id != "" {
			resourceID = &id
		} else if id, ok := jsonData["id"].(string); ok && id != "" {
			resourceID = &id
		}
		return
	}

	// If we have an organizationId but no resource type yet, infer from context
	// This catches actions that might be organization-scoped but don't have explicit resource types
	if orgID, ok := jsonData["organizationId"].(string); ok && orgID != "" {
		// If action contains "create", "update", "delete" and we have orgId, it's likely an org action
		if strings.Contains(actionLower, "create") || strings.Contains(actionLower, "update") || strings.Contains(actionLower, "delete") {
			rt := "organization"
			resourceType = &rt
			resourceID = &orgID
			return
		}
	}

	return nil, nil
}

// sanitizeRequestData sanitizes request data by removing sensitive fields
func sanitizeRequestData(req connect.AnyRequest) string {
	msg := req.Any()
	if msg == nil {
		return "{}"
	}

	if protoMsg, ok := msg.(proto.Message); ok {
		// Convert to JSON
		jsonBytes, err := protojson.Marshal(protoMsg)
		if err != nil {
			return "{}"
		}

		var jsonData map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
			return "{}"
		}

		// Remove sensitive fields
		sensitiveFields := []string{
			"password",
			"token",
			"secret",
			"api_key",
			"apiKey",
			"access_token",
			"accessToken",
			"refresh_token",
			"refreshToken",
			"authorization",
		}

		sanitizeMap(jsonData, sensitiveFields)

		// Convert back to JSON
		sanitizedBytes, err := json.Marshal(jsonData)
		if err != nil {
			return "{}"
		}

		return string(sanitizedBytes)
	}

	return "{}"
}

// sanitizeMap recursively removes sensitive fields from a map
func sanitizeMap(data map[string]interface{}, sensitiveFields []string) {
	for key, value := range data {
		keyLower := strings.ToLower(key)
		for _, sensitive := range sensitiveFields {
			if strings.Contains(keyLower, strings.ToLower(sensitive)) {
				data[key] = "[REDACTED]"
				continue
			}
		}

		// Recursively sanitize nested maps
		if nestedMap, ok := value.(map[string]interface{}); ok {
			sanitizeMap(nestedMap, sensitiveFields)
		}
	}
}

// getClientIP extracts the client IP address from the request
// It checks multiple headers in order of preference to get the real client IP
// Traefik is configured with forwardedHeaders middleware to properly forward the real client IP
func getClientIP(req connect.AnyRequest) string {
	// Try CF-Connecting-IP (Cloudflare)
	if cfIP := req.Header().Get("CF-Connecting-IP"); cfIP != "" {
		return strings.TrimSpace(cfIP)
	}

	// Try True-Client-IP (used by some proxies)
	if trueClientIP := req.Header().Get("True-Client-IP"); trueClientIP != "" {
		return strings.TrimSpace(trueClientIP)
	}

	// Try X-Forwarded-For header (Traefik sets this with forwardedHeaders middleware)
	// Format: "client-ip, proxy1-ip, proxy2-ip, ..."
	// The first IP is the original client IP
	forwarded := req.Header().Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Try X-Real-IP header (nginx and some proxies)
	if realIP := req.Header().Get("X-Real-IP"); realIP != "" {
		ip := strings.TrimSpace(realIP)
		if ip != "" {
			return ip
		}
	}

	// Try X-Client-IP (some proxies)
	if clientIP := req.Header().Get("X-Client-IP"); clientIP != "" {
		ip := strings.TrimSpace(clientIP)
		if ip != "" {
			return ip
		}
	}

	// Fallback: return "unknown" since Connect doesn't expose RemoteAddr directly
	return "unknown"
}
