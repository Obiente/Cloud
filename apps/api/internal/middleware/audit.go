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

			// Extract user info from context
			user, _ := auth.GetUserFromContext(ctx)
			userID := "system"
			if user != nil {
				userID = user.Id
			}

			// Extract IP address and user agent
			ipAddress := getClientIP(req)
			userAgent := req.Header().Get("User-Agent")
			if userAgent == "" {
				userAgent = "unknown"
			}

			// Parse service and action from procedure
			service, action := parseProcedure(procedure)

			// Extract resource information from request
			resourceType, resourceID, orgID := extractResourceInfo(req, procedure)

			// Sanitize request data (remove sensitive fields)
			requestData := sanitizeRequestData(req)

			// Execute the request
			resp, err := next(ctx, req)

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
				// Use background context with timeout instead of request context to avoid cancellation
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				// Log what we're saving for debugging
				orgIDStr := "nil"
				if orgID != nil {
					orgIDStr = *orgID
				}
				logger.Debug("[Audit] Saving log: service=%s, action=%s, orgID=%s, resourceType=%v, resourceID=%v",
					service, action, orgIDStr, resourceType, resourceID)

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
				} else {
					logger.Debug("[Audit] Successfully logged %s/%s (orgID=%s)", service, action, orgIDStr)
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
			if err := json.Unmarshal(jsonBytes, jsonData); err == nil {
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
						logger.Debug("[Audit] Looked up orgID=%s from deployment %s", *lookedUpOrgID, *resourceID)
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
	if strings.Contains(actionLower, "organization") {
		rt := "organization"
		resourceType = &rt

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
		return
	}

	// Support ticket-related actions
	if strings.Contains(actionLower, "ticket") {
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
