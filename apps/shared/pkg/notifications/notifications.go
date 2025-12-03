package notifications

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
	notificationsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1/notificationsv1connect"
)

var (
	notificationsServiceURL string
	notificationsClient     notificationsv1connect.NotificationServiceClient
	internalServiceSecret   string
	retryConfig             retryConfiguration
)

type retryConfiguration struct {
	maxAttempts      int
	initialBackoff   time.Duration
	maxBackoff       time.Duration
}

func init() {
	notificationsServiceURL = os.Getenv("NOTIFICATIONS_SERVICE_URL")
	if notificationsServiceURL == "" {
		notificationsServiceURL = "http://notifications-service:3012"
	}

	internalServiceSecret = os.Getenv("INTERNAL_SERVICE_SECRET")
	if internalServiceSecret == "" {
		// Log warning but don't fail - will fail when trying to make calls
		logger.Warn("[Notifications] INTERNAL_SERVICE_SECRET not set - internal service calls will fail")
	}

	// Configure retry settings
	retryConfig = retryConfiguration{
		maxAttempts:    getEnvInt("NOTIFICATIONS_RETRY_MAX_ATTEMPTS", 3),
		initialBackoff: getEnvDuration("NOTIFICATIONS_RETRY_INITIAL_BACKOFF", 1*time.Second),
		maxBackoff:     getEnvDuration("NOTIFICATIONS_RETRY_MAX_BACKOFF", 10*time.Second),
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create Connect client with internal service auth interceptor
	var clientOpts []connect.ClientOption
	if internalServiceSecret != "" {
		clientOpts = append(clientOpts, connect.WithInterceptors(newInternalServiceAuthInterceptor(internalServiceSecret)))
	}

	notificationsClient = notificationsv1connect.NewNotificationServiceClient(
		httpClient,
		notificationsServiceURL,
		clientOpts...,
	)
}

// getEnvInt gets an integer environment variable or returns the default
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

// getEnvDuration gets a duration environment variable or returns the default
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

// isRetryableError checks if an error is retryable (transient network/service errors)
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	
	// Check for Connect RPC error codes that are retryable
	if connectErr, ok := err.(*connect.Error); ok {
		// Unavailable, DeadlineExceeded, and ResourceExhausted are retryable
		if connectErr.Code() == connect.CodeUnavailable ||
			connectErr.Code() == connect.CodeDeadlineExceeded ||
			connectErr.Code() == connect.CodeResourceExhausted {
			return true
		}
		// Permission denied, invalid argument, etc. are not retryable
		return false
	}

	// Check for network/connection errors in error message
	retryablePatterns := []string{
		"connection refused",
		"dial tcp",
		"no such host",
		"timeout",
		"deadline exceeded",
		"unavailable",
		"temporary failure",
		"network is unreachable",
		"connection reset",
		"EOF",
	}

	errStrLower := strings.ToLower(errStr)
	for _, pattern := range retryablePatterns {
		if strings.Contains(errStrLower, pattern) {
			return true
		}
	}

	return false
}

// internalServiceAuthInterceptor adds the internal service secret header to requests
type internalServiceAuthInterceptor struct {
	secret string
}

func newInternalServiceAuthInterceptor(secret string) connect.Interceptor {
	return &internalServiceAuthInterceptor{secret: secret}
}

func (i *internalServiceAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("x-internal-service-secret", i.secret)
		return next(ctx, req)
	}
}

func (i *internalServiceAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func (i *internalServiceAuthInterceptor) WrapUnaryClient(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("x-internal-service-secret", i.secret)
		return next(ctx, req)
	}
}

func (i *internalServiceAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		// Set header on the connection (streaming client connections support RequestHeader)
		conn.RequestHeader().Set("x-internal-service-secret", i.secret)
		return conn
	}
}

// CreateNotificationForUser creates a notification for a specific user by calling the notifications service
func CreateNotificationForUser(ctx context.Context, userID string, orgID *string, notificationType notificationsv1.NotificationType, severity notificationsv1.NotificationSeverity, title, message string, actionURL, actionLabel *string, metadata map[string]string) error {
	// Build request
	req := &notificationsv1.CreateNotificationRequest{
		UserId:   userID,
		Type:     notificationType,
		Severity: severity,
		Title:    title,
		Message:  message,
		Metadata: metadata,
	}

	if orgID != nil {
		req.OrganizationId = orgID
	}
	if actionURL != nil {
		req.ActionUrl = actionURL
	}
	if actionLabel != nil {
		req.ActionLabel = actionLabel
	}

	// Call notifications service with retry logic
	var lastErr error
	backoff := retryConfig.initialBackoff

	for attempt := 0; attempt < retryConfig.maxAttempts; attempt++ {
		_, err := notificationsClient.CreateNotification(ctx, connect.NewRequest(req))
		if err == nil {
			if attempt > 0 {
				logger.Info("[Notifications] Successfully created notification after %d retry attempts for user %s", attempt, userID)
			}
			logger.Info("[Notifications] Created notification via service for user %s, type %s, severity %s: %s", userID, notificationTypeToString(notificationType), notificationSeverityToString(severity), title)
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			logger.Warn("[Notifications] Non-retryable error creating notification for user %s: %v", userID, err)
			return fmt.Errorf("create notification via service: %w", err)
		}

		// Log retry attempt
		if attempt < retryConfig.maxAttempts-1 {
			logger.Warn("[Notifications] Failed to create notification for user %s (attempt %d/%d): %v. Retrying in %v...", userID, attempt+1, retryConfig.maxAttempts, err, backoff)
			
			// Use context with timeout for backoff to respect context cancellation
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
				// Exponential backoff
				backoff *= 2
				if backoff > retryConfig.maxBackoff {
					backoff = retryConfig.maxBackoff
				}
			}
		}
	}

	// All retries exhausted
	logger.Warn("[Notifications] Failed to create notification via service for user %s after %d attempts: %v", userID, retryConfig.maxAttempts, lastErr)
	return fmt.Errorf("create notification via service (failed after %d attempts): %w", retryConfig.maxAttempts, lastErr)
}

// CreateNotificationForOrganization creates notifications for all active members of an organization by calling the notifications service
func CreateNotificationForOrganization(ctx context.Context, orgID string, notificationType notificationsv1.NotificationType, severity notificationsv1.NotificationSeverity, title, message string, actionURL, actionLabel *string, metadata map[string]string, roles []string) error {
	// Build request
	req := &notificationsv1.CreateOrganizationNotificationRequest{
		OrganizationId: orgID,
		Type:           notificationType,
		Severity:       severity,
		Title:          title,
		Message:        message,
		Metadata:       metadata,
		Roles:          roles,
	}

	if actionURL != nil {
		req.ActionUrl = actionURL
	}
	if actionLabel != nil {
		req.ActionLabel = actionLabel
	}

	// Call notifications service with retry logic
	var lastErr error
	backoff := retryConfig.initialBackoff

	for attempt := 0; attempt < retryConfig.maxAttempts; attempt++ {
		_, err := notificationsClient.CreateOrganizationNotification(ctx, connect.NewRequest(req))
		if err == nil {
			if attempt > 0 {
				logger.Info("[Notifications] Successfully created organization notification after %d retry attempts for org %s", attempt, orgID)
			}
			logger.Info("[Notifications] Created organization notification via service for org %s, type %s, severity %s: %s", orgID, notificationTypeToString(notificationType), notificationSeverityToString(severity), title)
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			logger.Warn("[Notifications] Non-retryable error creating organization notification for org %s: %v", orgID, err)
			return fmt.Errorf("create organization notification via service: %w", err)
		}

		// Log retry attempt
		if attempt < retryConfig.maxAttempts-1 {
			logger.Warn("[Notifications] Failed to create organization notification for org %s (attempt %d/%d): %v. Retrying in %v...", orgID, attempt+1, retryConfig.maxAttempts, err, backoff)
			
			// Use context with timeout for backoff to respect context cancellation
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
				// Exponential backoff
				backoff *= 2
				if backoff > retryConfig.maxBackoff {
					backoff = retryConfig.maxBackoff
				}
			}
		}
	}

	// All retries exhausted
	logger.Warn("[Notifications] Failed to create organization notification via service for org %s after %d attempts: %v", orgID, retryConfig.maxAttempts, lastErr)
	return fmt.Errorf("create organization notification via service (failed after %d attempts): %w", retryConfig.maxAttempts, lastErr)
}

// CreateNotificationForUserByEmail creates a notification for a user by their email
// This is useful for invites where we only have the email
// Note: For pending invites, we'll create the notification when the user accepts the invite
func CreateNotificationForUserByEmail(ctx context.Context, email string, orgID *string, notificationType notificationsv1.NotificationType, severity notificationsv1.NotificationSeverity, title, message string, actionURL, actionLabel *string, metadata map[string]string) error {
	// For invites, we need to find the user via Zitadel or wait until they accept
	// For now, we'll skip creating notifications for users that don't exist yet
	// The notification will be created when they accept the invite (see AcceptInvite)
	return nil
}

// Helper functions

func notificationTypeToString(t notificationsv1.NotificationType) string {
	switch t {
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_INFO:
		return "INFO"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_SUCCESS:
		return "SUCCESS"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_WARNING:
		return "WARNING"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_ERROR:
		return "ERROR"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_DEPLOYMENT:
		return "DEPLOYMENT"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_BILLING:
		return "BILLING"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_QUOTA:
		return "QUOTA"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_INVITE:
		return "INVITE"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM:
		return "SYSTEM"
	default:
		return "INFO"
	}
}

func notificationSeverityToString(s notificationsv1.NotificationSeverity) string {
	switch s {
	case notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW:
		return "LOW"
	case notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM:
		return "MEDIUM"
	case notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH:
		return "HIGH"
	case notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL:
		return "CRITICAL"
	default:
		return "MEDIUM"
	}
}
