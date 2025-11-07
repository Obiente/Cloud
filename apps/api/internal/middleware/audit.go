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

			// Add panic recovery for the entire audit logging process
			defer func() {
				if r := recover(); r != nil {
					logger.Error("[Audit] Panic in audit logging for %s: %v", procedure, r)
				}
			}()

			// Extract IP address and user agent (before calling next)
			ipAddress := getClientIP(req)
			userAgent := req.Header().Get("User-Agent")
			if userAgent == "" {
				userAgent = "unknown"
			}

			// Parse service and action from procedure
			service, action := parseProcedure(procedure)

			// Extract resource information from request (before calling next)
			resourceType, resourceID, orgID := extractResourceInfo(req, procedure)

			// Sanitize request data (remove sensitive fields)
			requestData := sanitizeRequestData(req)

			// Execute the request (auth interceptor will set user in context)
			// Note: In Connect, with connect.WithInterceptors(auditInterceptor, authInterceptor),
			// authInterceptor wraps the handler (innermost, runs first), then auditInterceptor
			// wraps authInterceptor (outermost, runs second). This means when auditInterceptor
			// calls next(), it runs authInterceptor which sets the user in context and calls the handler.
			// However, context.WithValue creates a NEW context, so the original ctx in the outer
			// interceptor won't have the user. We need to extract the user from the context chain.
			// Actually, context.Value() searches up the chain, so if the inner interceptor set it,
			// we should be able to access it. But to be safe, we'll also try to extract from the
			// response headers if available.
			var userID string = "system"
			
			resp, err := next(ctx, req)

			// Try to extract user from context - the auth interceptor should have set it
			// Note: In Connect, when we call next(ctx, req), the auth interceptor (inner) receives
			// the context, sets the user via context.WithValue, and passes the new context to the handler.
			// However, context.WithValue creates a NEW context, so the original ctx in the outer
			// interceptor doesn't have the user. But context.Value() searches up the chain, so we
			// should be able to access it. If that doesn't work, we'll try response headers.
			user, _ := auth.GetUserFromContext(ctx)
			if user != nil {
				userID = user.Id
			} else {
				// Fallback: Try to extract from response headers (if auth interceptor exposes them)
				if resp != nil {
					if userIDHeader := resp.Header().Get("X-User-ID"); userIDHeader != "" {
						userID = userIDHeader
					}
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
func getClientIP(req connect.AnyRequest) string {
	// Try to get from X-Forwarded-For header
	forwarded := req.Header().Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Try X-Real-IP header
	realIP := req.Header().Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fallback to remote address (if available in context)
	// Note: Connect doesn't expose the underlying HTTP request directly,
	// so we can't get the remote address easily. This is a limitation.
	return "unknown"
}
