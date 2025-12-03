package superadmin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/notifications"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"

	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DetectAbuse performs comprehensive abuse detection and returns all suspicious organizations and activities
func DetectAbuse(ctx context.Context) (*superadminv1.GetAbuseDetectionResponse, error) {
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)

	// Find suspicious organizations
	suspiciousOrgs, err := detectSuspiciousOrganizations(twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect suspicious organizations: %v", err)
		suspiciousOrgs = []*superadminv1.SuspiciousOrganization{}
	}

	// Find suspicious activities
	suspiciousActivities, err := detectSuspiciousActivities(ctx, twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect suspicious activities: %v", err)
		suspiciousActivities = []*superadminv1.SuspiciousActivity{}
	}

	// Send notifications to superadmins if abuse is detected
	if len(suspiciousOrgs) > 0 || len(suspiciousActivities) > 0 {
		go notifySuperadminsOfAbuse(ctx, suspiciousOrgs, suspiciousActivities)
	}

	// Calculate metrics
	highRiskCount := int64(0)
	for _, org := range suspiciousOrgs {
		if org.RiskScore > 70 {
			highRiskCount++
		}
	}

	// Count activity types for metrics
	rapidCreationsCount := int64(0)
	failedPaymentsCount := int64(0)
	sshBruteForceCount := int64(0)
	apiAbuseCount := int64(0)
	failedAuthCount := int64(0)
	multipleAccountsCount := int64(0)
	usageSpikesCount := int64(0)
	dnsAbuseCount := int64(0)
	gameServerAbuseCount := int64(0)

	for _, activity := range suspiciousActivities {
		switch activity.ActivityType {
		case "rapid_creation":
			rapidCreationsCount++
		case "failed_payments":
			failedPaymentsCount++
		case "ssh_brute_force":
			sshBruteForceCount++
		case "api_abuse":
			apiAbuseCount++
		case "failed_authentication":
			failedAuthCount++
		case "multiple_accounts":
			multipleAccountsCount++
		case "usage_spike":
			usageSpikesCount++
		case "dns_delegation_abuse":
			dnsAbuseCount++
		case "game_server_abuse":
			gameServerAbuseCount++
		}
	}

	metrics := &superadminv1.AbuseMetrics{
		TotalSuspiciousOrgs:         int64(len(suspiciousOrgs)),
		HighRiskOrgs:                highRiskCount,
		RapidCreations_24H:          rapidCreationsCount,
		FailedPaymentAttempts_24H:   failedPaymentsCount,
		UnusualUsageSpikes_24H:      usageSpikesCount,
	}

	return &superadminv1.GetAbuseDetectionResponse{
		SuspiciousOrganizations: suspiciousOrgs,
		SuspiciousActivities:    suspiciousActivities,
		Metrics:                 metrics,
	}, nil
}

// detectSuspiciousOrganizations finds organizations with suspicious activity patterns
// Uses audit logs as primary source to prevent bypass by deleting records
func detectSuspiciousOrganizations(twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousOrganization, error) {
	var suspiciousOrgs []*superadminv1.SuspiciousOrganization

	type orgActivity struct {
		OrganizationID string
		Name           string
		CreatedAt      time.Time
		Created24h     int64
		Failed24h      int64
		CreditsSpent   int64
		LastActivity   time.Time
	}

	var activities []orgActivity

	// PRIMARY: Use audit logs to count resource creations (can't be bypassed)
	// FALLBACK: Use direct table queries if audit logs unavailable
	if database.MetricsDB != nil {
		// Query audit logs for resource creations
		var auditCreations []struct {
			OrganizationID string
			CreatedCount   int64
		}
		err := database.MetricsDB.Raw(`
			SELECT 
				organization_id,
				COUNT(*) as created_count
			FROM audit_logs
			WHERE created_at >= ?
				AND action IN ('CreateVPS', 'CreateDeployment', 'CreateGameServer')
				AND response_status = 200
				AND organization_id IS NOT NULL
			GROUP BY organization_id
			HAVING COUNT(*) > 10
		`, twentyFourHoursAgo).
			Scan(&auditCreations).Error

		if err == nil && len(auditCreations) > 0 {
			// Get organization details and combine with other metrics
			for _, ac := range auditCreations {
				var org database.Organization
				if err := database.DB.First(&org, "id = ?", ac.OrganizationID).Error; err != nil {
					continue
				}

				// Get failed deployments from audit logs (can't be bypassed)
				var failedCount int64
				if database.MetricsDB != nil {
					database.MetricsDB.Raw(`
						SELECT COUNT(*) as count
						FROM audit_logs
						WHERE organization_id = ?
							AND created_at >= ?
							AND action = 'CreateDeployment'
							AND (response_status != 200 OR error_message IS NOT NULL)
					`, ac.OrganizationID, twentyFourHoursAgo).
						Scan(&failedCount)
				}

				// Get credits from direct table (credit transactions are immutable)
				var creditsSpent int64
				var lastActivity time.Time

				database.DB.Table("credit_transactions").
					Select("COALESCE(SUM(ABS(amount_cents)), 0) as credits").
					Where("organization_id = ? AND created_at >= ?", ac.OrganizationID, twentyFourHoursAgo).
					Scan(&creditsSpent)

				database.DB.Table("credit_transactions").
					Select("MAX(created_at) as last_activity").
					Where("organization_id = ? AND created_at >= ?", ac.OrganizationID, twentyFourHoursAgo).
					Scan(&lastActivity)

				if lastActivity.IsZero() {
					lastActivity = org.CreatedAt
				}

				activities = append(activities, orgActivity{
					OrganizationID: ac.OrganizationID,
					Name:           org.Name,
					CreatedAt:      org.CreatedAt,
					Created24h:     ac.CreatedCount,
					Failed24h:      failedCount,
					CreditsSpent:   creditsSpent,
					LastActivity:   lastActivity,
				})
			}
		} else if err != nil {
			logger.Warn("[SuperAdmin] Failed to query audit logs for suspicious organizations, falling back to direct queries: %v", err)
		}
	}

	// FALLBACK: Use direct table queries if audit logs unavailable or failed
	if len(activities) == 0 {
		err := database.DB.Table("organizations o").
			Select(`
				o.id as organization_id,
				o.name,
				o.created_at,
				COALESCE(COUNT(DISTINCT d.id), 0) as created_24h,
				COALESCE(SUM(CASE WHEN d.status = 5 THEN 1 ELSE 0 END), 0) as failed_24h,
				COALESCE(SUM(ABS(ct.amount_cents)), 0) as credits_spent,
				COALESCE(MAX(GREATEST(d.created_at, d.last_deployed_at, ct.created_at)), o.created_at) as last_activity
			`).
			Joins("LEFT JOIN deployments d ON d.organization_id = o.id AND d.created_at >= ?", twentyFourHoursAgo).
			Joins("LEFT JOIN credit_transactions ct ON ct.organization_id = o.id AND ct.created_at >= ?", twentyFourHoursAgo).
			Group("o.id, o.name, o.created_at").
			Having("COUNT(DISTINCT d.id) > 10 OR COALESCE(SUM(CASE WHEN d.status = 5 THEN 1 ELSE 0 END), 0) > 5").
			Find(&activities).Error

		if err != nil {
			return nil, fmt.Errorf("query suspicious organizations: %w", err)
		}
	}

	for _, act := range activities {
		riskScore := int64(0)
		reasons := []string{}

		if act.Created24h > 10 {
			riskScore += 30
			reasons = append(reasons, fmt.Sprintf("Created %d resources in 24h", act.Created24h))
		}
		if act.Failed24h > 5 {
			riskScore += 40
			reasons = append(reasons, fmt.Sprintf("%d failed deployments in 24h", act.Failed24h))
		}
		if act.Created24h > 20 {
			riskScore += 30
		}
		if act.CreditsSpent == 0 && act.Created24h > 5 {
			riskScore += 20 // High activity but no payment
		}

		if riskScore > 0 {
			suspiciousOrgs = append(suspiciousOrgs, &superadminv1.SuspiciousOrganization{
				OrganizationId:      act.OrganizationID,
				OrganizationName:    act.Name,
				Reason:             strings.Join(reasons, "; "),
				RiskScore:          riskScore,
				CreatedCount_24H:    act.Created24h,
				FailedDeployments_24H: act.Failed24h,
				TotalCreditsSpent:  act.CreditsSpent,
				CreatedAt:          timestamppb.New(act.CreatedAt),
				LastActivity:       timestamppb.New(act.LastActivity),
			})
		}
	}

	return suspiciousOrgs, nil
}

// detectSuspiciousActivities finds various types of suspicious activities
func detectSuspiciousActivities(ctx context.Context, twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var allActivities []*superadminv1.SuspiciousActivity

	// Check for rapid resource creation
	rapidCreations, err := detectRapidResourceCreation(twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect rapid resource creation: %v", err)
	} else {
		allActivities = append(allActivities, rapidCreations...)
	}

	// Check for failed payment attempts
	failedPayments, err := detectFailedPayments(twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect failed payments: %v", err)
	} else {
		allActivities = append(allActivities, failedPayments...)
	}

	// Check for SSH brute force attempts
	sshBruteForce, err := detectSSHBruteForce(ctx, twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect SSH brute force: %v", err)
	} else {
		allActivities = append(allActivities, sshBruteForce...)
	}

	// Check for API abuse
	apiAbuse, err := detectAPIAbuse(ctx, twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect API abuse: %v", err)
	} else {
		allActivities = append(allActivities, apiAbuse...)
	}

	// Check for failed authentication attempts
	failedAuth, err := detectFailedAuthentication(ctx, twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect failed authentication: %v", err)
	} else {
		allActivities = append(allActivities, failedAuth...)
	}

	// Check for multiple account creation
	multipleAccounts, err := detectMultipleAccountCreation(ctx, twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect multiple account creation: %v", err)
	} else {
		allActivities = append(allActivities, multipleAccounts...)
	}

	// Check for DNS delegation API abuse
	dnsAbuse, err := detectDNSDelegationAbuse(twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect DNS delegation abuse: %v", err)
	} else {
		allActivities = append(allActivities, dnsAbuse...)
	}

	// Check for unusual usage spikes
	usageSpikes, err := detectUsageSpikes(ctx, twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect usage spikes: %v", err)
	} else {
		allActivities = append(allActivities, usageSpikes...)
	}

	// Check for game server abuse
	gameServerAbuse, err := detectGameServerAbuse(twentyFourHoursAgo)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to detect game server abuse: %v", err)
	} else {
		allActivities = append(allActivities, gameServerAbuse...)
	}

	return allActivities, nil
}

// detectRapidResourceCreation finds organizations creating resources too quickly
// Checks multiple time windows with progressively stricter thresholds for shorter periods
func detectRapidResourceCreation(twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity
	now := time.Now()

	// Define time windows and thresholds (shorter windows = stricter thresholds)
	// Aggressive thresholds for all resource types
	// Check in order from most severe to least severe
	timeWindows := []struct {
		duration  time.Duration
		threshold int64
		label     string
		severity  int64
	}{
		{10 * time.Minute, 3, "10 minutes", 95},  // Very aggressive: 3+ in 10 min
		{30 * time.Minute, 5, "30 minutes", 85},  // Aggressive: 5+ in 30 min
		{1 * time.Hour, 8, "1 hour", 75},         // Moderate: 8+ in 1 hour
		{6 * time.Hour, 12, "6 hours", 65},       // Standard: 12+ in 6 hours
		{24 * time.Hour, 15, "24 hours", 55},     // Daily: 15+ in 24 hours
	}

	// Track organizations we've already flagged to avoid duplicates (keep most severe)
	flaggedOrgs := make(map[string]bool)

	for _, window := range timeWindows {
		windowStart := now.Add(-window.duration)

		// Check combined resources (deployments + VPS + game servers) for overall rapid creation
		// PRIMARY: Use audit logs which persist even after hard deletes - prevents bypass
		// FALLBACK: Use direct table queries only if audit logs are unavailable
		var combinedCreations []struct {
			OrganizationID string
			Count          int64
		}
		
		// PRIMARY METHOD: Query audit logs (can't be bypassed by deleting records)
		if database.MetricsDB != nil {
			// Main query
			// For CreateVPS, the audit middleware stores resource_type='organization' and resource_id=<org_id>
			// because "create" matches the organization action pattern before the VPS pattern is checked
			// So we need to extract organization_id from resource_id when resource_type='organization' AND service='VPSService'
			// NOTE: We count BOTH successful (200) and failed attempts, as rapid failed attempts are also suspicious
			err := database.MetricsDB.Raw(`
				WITH creation_actions AS (
					SELECT 
						al.organization_id,
						al.resource_id,
						al.resource_type,
						al.service,
						al.action,
						al.created_at
					FROM audit_logs al
					WHERE al.created_at >= ?
						-- Count both successful and failed attempts (rapid failures are also suspicious)
						AND (
							(al.action IN ('CreateVPS', 'CreateDeployment', 'CreateGameServer'))
							OR (al.service = 'VPSService' AND al.action ILIKE '%create%')
							OR (al.service = 'DeploymentService' AND al.action ILIKE '%create%')
							OR (al.service = 'GameServerService' AND al.action ILIKE '%create%')
						)
				),
				actions_with_org AS (
					SELECT 
						COALESCE(
							-- 1. Direct organization_id column (preferred)
							ca.organization_id,
							-- 2. For CreateVPS actions, resource_type='organization' and resource_id=<org_id>
							-- This happens because the audit middleware matches "create" as an org action before VPS
							CASE 
								WHEN ca.resource_type = 'organization' AND ca.resource_id IS NOT NULL AND ca.resource_id != '' THEN
									ca.resource_id
								-- 3. For other cases where resource_type might be 'vps' but organization_id is NULL
								-- Try to extract from request_data JSON (as fallback, but may fail if JSON is invalid)
								ELSE NULL
							END
						) as organization_id
					FROM creation_actions ca
				)
				SELECT 
					organization_id,
					COUNT(*) as count
				FROM actions_with_org
				WHERE organization_id IS NOT NULL 
					AND organization_id != ''
					AND organization_id != 'unknown'
				GROUP BY organization_id
				HAVING COUNT(*) > ?
			`, windowStart, window.threshold).
				Scan(&combinedCreations).Error
			
			if err != nil {
				logger.Warn("[SuperAdmin] Failed to query audit logs for rapid creations, falling back to direct table queries: %v", err)
				combinedCreations = []struct {
					OrganizationID string
					Count          int64
				}{}
			}
		} else {
			logger.Warn("[SuperAdmin] MetricsDB not available, cannot use audit logs for abuse detection")
		}
		
		// FALLBACK: Only use direct table queries if audit logs are unavailable or failed
		// This is less reliable as it can be bypassed by deleting records
		if len(combinedCreations) == 0 && database.MetricsDB == nil {
			err := database.DB.Raw(`
				SELECT organization_id, COUNT(*) as count
				FROM (
					SELECT organization_id, created_at FROM deployments WHERE created_at >= ?
					UNION ALL
					SELECT organization_id, created_at FROM vps_instances WHERE created_at >= ? AND deleted_at IS NULL
					UNION ALL
					SELECT organization_id, created_at FROM game_servers WHERE created_at >= ? AND deleted_at IS NULL
				) combined
				GROUP BY organization_id
				HAVING COUNT(*) > ?
			`, windowStart, windowStart, windowStart, window.threshold).
				Scan(&combinedCreations).Error
			
			if err != nil {
				logger.Error("[SuperAdmin] Failed to query rapid combined creations: %v", err)
				continue
			}
		}

		for _, cc := range combinedCreations {
			// Only flag if not already flagged (we check most severe windows first)
			if !flaggedOrgs[cc.OrganizationID] {
				var org database.Organization
				if err := database.DB.First(&org, "id = ?", cc.OrganizationID).Error; err == nil {
					activities = append(activities, &superadminv1.SuspiciousActivity{
						Id:              fmt.Sprintf("rapid-%s-%d", cc.OrganizationID, window.duration),
						OrganizationId:   cc.OrganizationID,
						OrganizationName: org.Name,
						ActivityType:    "rapid_creation",
						Description:     fmt.Sprintf("Created %d resources (deployments + VPS + game servers) in %s", cc.Count, window.label),
						Severity:        window.severity,
						OccurredAt:      timestamppb.New(time.Now()),
					})
					flaggedOrgs[cc.OrganizationID] = true
				}
			}
		}
	}

	return activities, nil
}

// detectFailedPayments finds organizations with failed payment attempts
func detectFailedPayments(twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	// NOTE: Currently, failed payments are not stored in credit_transactions.
	// The webhook handler only logs payment_intent.payment_failed events.
	// This detection method is incomplete and should be enhanced by either:
	// 1. Storing failed payment events in a dedicated table with organization_id
	// 2. Querying StripeWebhookEvent for payment_intent.payment_failed and linking via billing accounts
	// For now, we check for organizations with payment attempts but no successful payments in 24h
	var failedPayments []struct {
		OrganizationID string
		Count          int64
	}
	// This query won't find actual failed payments since they're not stored,
	// but we can detect suspicious patterns (many payment attempts with no success)
	err := database.DB.Table("credit_transactions").
		Select("organization_id, COUNT(*) as count").
		Where("created_at >= ? AND type = ? AND amount_cents < 0", twentyFourHoursAgo, "payment").
		Group("organization_id").
		Having("COUNT(*) > 2").
		Scan(&failedPayments).Error
	if err != nil {
		return nil, fmt.Errorf("query failed payments: %w", err)
	}

	for _, fp := range failedPayments {
		var org database.Organization
		if err := database.DB.First(&org, "id = ?", fp.OrganizationID).Error; err == nil {
			activities = append(activities, &superadminv1.SuspiciousActivity{
				Id:              fmt.Sprintf("payment-%s", fp.OrganizationID),
				OrganizationId:   fp.OrganizationID,
				OrganizationName: org.Name,
				ActivityType:    "failed_payments",
				Description:     fmt.Sprintf("%d failed payment attempts in 24 hours", fp.Count),
				Severity:        70,
				OccurredAt:      timestamppb.New(time.Now()),
			})
		}
	}

	return activities, nil
}

// detectSSHBruteForce finds IP addresses with multiple failed SSH connection attempts
// Uses aggressive thresholds for shorter time windows
func detectSSHBruteForce(ctx context.Context, twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	if database.MetricsDB == nil {
		// MetricsDB not available, skip SSH brute force detection
		return activities, nil
	}

	now := time.Now()
	timeWindows := []struct {
		duration  time.Duration
		threshold int64
		label     string
		severity  int64
	}{
		{10 * time.Minute, 3, "10 minutes", 95},  // Very aggressive: 3+ failures in 10 min
		{30 * time.Minute, 5, "30 minutes", 85},  // Aggressive: 5+ failures in 30 min
		{1 * time.Hour, 8, "1 hour", 75},         // Moderate: 8+ failures in 1 hour
		{24 * time.Hour, 10, "24 hours", 65},     // Standard: 10+ failures in 24 hours
	}

	flaggedIPs := make(map[string]bool)

	for _, window := range timeWindows {
		windowStart := now.Add(-window.duration)
		
		var bruteForceAttempts []struct {
			IPAddress string
			Count     int64
			UserID    string
			OrgID     *string
		}

		err := database.MetricsDB.WithContext(ctx).Table("audit_logs").
			Select("ip_address, COUNT(*) as count, MAX(user_id) as user_id, MAX(organization_id) as org_id").
			Where("action = ? AND response_status != 200 AND created_at >= ?", "SSHConnect", windowStart).
			Group("ip_address").
			Having("COUNT(*) > ?", window.threshold).
			Scan(&bruteForceAttempts).Error
		if err != nil {
			logger.Warn("[SuperAdmin] Failed to query SSH brute force for window %s: %v", window.label, err)
			continue
		}

		for _, bf := range bruteForceAttempts {
			// Only flag if not already flagged (we check most severe windows first)
			if !flaggedIPs[bf.IPAddress] {
				orgID := "unknown"
				if bf.OrgID != nil && *bf.OrgID != "" {
					orgID = *bf.OrgID
				}

				activities = append(activities, &superadminv1.SuspiciousActivity{
					Id:            fmt.Sprintf("ssh-bf-%s-%d", bf.IPAddress, window.duration),
					OrganizationId: orgID,
					ActivityType:  "ssh_brute_force",
					Description:   fmt.Sprintf("IP %s: %d failed SSH connection attempts in %s", bf.IPAddress, bf.Count, window.label),
					Severity:      window.severity,
					OccurredAt:    timestamppb.New(time.Now()),
				})
				flaggedIPs[bf.IPAddress] = true
			}
		}
	}

	return activities, nil
}

// detectAPIAbuse finds users/organizations with excessive API calls
// Uses aggressive thresholds for shorter time windows
func detectAPIAbuse(ctx context.Context, twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	if database.MetricsDB == nil {
		// MetricsDB not available, skip API abuse detection
		return activities, nil
	}

	now := time.Now()
	timeWindows := []struct {
		duration  time.Duration
		threshold int64
		label     string
		severity  int64
	}{
		{10 * time.Minute, 100, "10 minutes", 90},   // Very aggressive: 100+ calls in 10 min
		{30 * time.Minute, 300, "30 minutes", 80},  // Aggressive: 300+ calls in 30 min
		{1 * time.Hour, 500, "1 hour", 70},          // Moderate: 500+ calls in 1 hour
		{24 * time.Hour, 2000, "24 hours", 60},       // Standard: 2000+ calls in 24 hours
	}

	flaggedUsers := make(map[string]bool)
	flaggedIPs := make(map[string]bool)

	for _, window := range timeWindows {
		windowStart := now.Add(-window.duration)
		
		// Check for excessive API calls per user
		var userAbuse []struct {
			UserID         string
			OrganizationID *string
			Count          int64
			IPAddress      string
		}

		err := database.MetricsDB.WithContext(ctx).Table("audit_logs").
			Select("user_id, MAX(organization_id) as organization_id, COUNT(*) as count, MAX(ip_address) as ip_address").
			Where("created_at >= ?", windowStart).
			Group("user_id").
			Having("COUNT(*) > ?", window.threshold).
			Scan(&userAbuse).Error
		if err != nil {
			logger.Warn("[SuperAdmin] Failed to query API abuse by user for window %s: %v", window.label, err)
			continue
		}

		for _, abuse := range userAbuse {
			// Only flag if not already flagged (we check most severe windows first)
			if !flaggedUsers[abuse.UserID] {
				orgID := "unknown"
				if abuse.OrganizationID != nil && *abuse.OrganizationID != "" {
					orgID = *abuse.OrganizationID
				}

				activities = append(activities, &superadminv1.SuspiciousActivity{
					Id:            fmt.Sprintf("api-user-%s-%d", abuse.UserID, window.duration),
					OrganizationId: orgID,
					ActivityType:  "api_abuse",
					Description:   fmt.Sprintf("User %s: %d API calls in %s (IP: %s)", abuse.UserID, abuse.Count, window.label, abuse.IPAddress),
					Severity:      window.severity,
					OccurredAt:    timestamppb.New(time.Now()),
				})
				flaggedUsers[abuse.UserID] = true
			}
		}

		// Check for excessive API calls per IP address
		var ipAbuse []struct {
			IPAddress string
			Count     int64
		}

		err = database.MetricsDB.WithContext(ctx).Table("audit_logs").
			Select("ip_address, COUNT(*) as count").
			Where("created_at >= ?", windowStart).
			Group("ip_address").
			Having("COUNT(*) > ?", window.threshold*2). // IP threshold is 2x user threshold
			Scan(&ipAbuse).Error
		if err != nil {
			logger.Warn("[SuperAdmin] Failed to query API abuse by IP for window %s: %v", window.label, err)
			continue
		}

		for _, abuse := range ipAbuse {
			// Only flag if not already flagged
			if !flaggedIPs[abuse.IPAddress] {
				activities = append(activities, &superadminv1.SuspiciousActivity{
					Id:            fmt.Sprintf("api-ip-%s-%d", abuse.IPAddress, window.duration),
					OrganizationId: "unknown",
					ActivityType:  "api_abuse",
					Description:   fmt.Sprintf("IP %s: %d API calls in %s", abuse.IPAddress, abuse.Count, window.label),
					Severity:      window.severity,
					OccurredAt:    timestamppb.New(time.Now()),
				})
				flaggedIPs[abuse.IPAddress] = true
			}
		}
	}

	return activities, nil
}

// detectFailedAuthentication finds IP addresses with multiple failed login attempts
func detectFailedAuthentication(ctx context.Context, twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	if database.MetricsDB == nil {
		// MetricsDB not available, skip failed auth detection
		return activities, nil
	}

	var failedAuths []struct {
		IPAddress string
		Count     int64
		UserID    string
	}

	err := database.MetricsDB.WithContext(ctx).Table("audit_logs").
		Select("ip_address, COUNT(*) as count, MAX(user_id) as user_id").
		Where("action = ? AND (response_status != 200 OR error_message IS NOT NULL) AND created_at >= ?", "Login", twentyFourHoursAgo).
		Group("ip_address").
		Having("COUNT(*) > 5").
		Scan(&failedAuths).Error
	if err != nil {
		return nil, fmt.Errorf("query failed authentication: %w", err)
	}

	for _, fa := range failedAuths {
		activities = append(activities, &superadminv1.SuspiciousActivity{
			Id:            fmt.Sprintf("auth-fail-%s", fa.IPAddress),
			OrganizationId: "unknown",
			ActivityType:  "failed_authentication",
			Description:   fmt.Sprintf("IP %s: %d failed login attempts in 24 hours", fa.IPAddress, fa.Count),
			Severity:      75,
			OccurredAt:    timestamppb.New(time.Now()),
		})
	}

	return activities, nil
}

// detectMultipleAccountCreation finds users creating multiple organizations in short time
func detectMultipleAccountCreation(ctx context.Context, twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	if database.MetricsDB == nil {
		// MetricsDB not available, skip multiple account detection
		return activities, nil
	}

	// Find users who created multiple organizations in the last 24 hours
	var multipleOrgs []struct {
		UserID         string
		IPAddress      string
		Count          int64
		OrganizationIDs string
	}

	err := database.MetricsDB.WithContext(ctx).Table("audit_logs al").
		Select(`
			al.user_id,
			MAX(al.ip_address) as ip_address,
			COUNT(DISTINCT al.resource_id) as count,
			STRING_AGG(DISTINCT al.resource_id, ', ') as organization_ids
		`).
		Where("al.action = ? AND al.resource_type = ? AND al.created_at >= ?", "CreateOrganization", "organization", twentyFourHoursAgo).
		Group("al.user_id").
		Having("COUNT(DISTINCT al.resource_id) > 2").
		Scan(&multipleOrgs).Error
	if err != nil {
		// If STRING_AGG is not available (older PostgreSQL), try without it
		err = database.MetricsDB.WithContext(ctx).Table("audit_logs al").
			Select(`
				al.user_id,
				MAX(al.ip_address) as ip_address,
				COUNT(DISTINCT al.resource_id) as count
			`).
			Where("al.action = ? AND al.resource_type = ? AND al.created_at >= ?", "CreateOrganization", "organization", twentyFourHoursAgo).
			Group("al.user_id").
			Having("COUNT(DISTINCT al.resource_id) > 2").
			Scan(&multipleOrgs).Error
		if err != nil {
			return nil, fmt.Errorf("query multiple account creation: %w", err)
		}
	}

	for _, mo := range multipleOrgs {
		desc := fmt.Sprintf("User %s created %d organizations in 24 hours (IP: %s)", mo.UserID, mo.Count, mo.IPAddress)
		if mo.OrganizationIDs != "" {
			desc += fmt.Sprintf(" - Orgs: %s", mo.OrganizationIDs)
		}

		activities = append(activities, &superadminv1.SuspiciousActivity{
			Id:            fmt.Sprintf("multi-org-%s", mo.UserID),
			OrganizationId: "unknown",
			ActivityType:  "multiple_accounts",
			Description:   desc,
			Severity:      65,
			OccurredAt:    timestamppb.New(time.Now()),
		})
	}

	return activities, nil
}

// detectDNSDelegationAbuse finds excessive DNS delegation API usage
// Uses audit logs as primary source to prevent bypass by deleting records
func detectDNSDelegationAbuse(twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	// PRIMARY: Use audit logs (can't be bypassed)
	var dnsKeyAbuse []struct {
		OrganizationID string
		Count          int64
	}

	if database.MetricsDB != nil {
		err := database.MetricsDB.Raw(`
			SELECT 
				organization_id,
				COUNT(*) as count
			FROM audit_logs
			WHERE created_at >= ?
				AND action = 'CreateDNSDelegationAPIKey'
				AND response_status = 200
				AND organization_id IS NOT NULL
			GROUP BY organization_id
			HAVING COUNT(*) > 5
		`, twentyFourHoursAgo).
			Scan(&dnsKeyAbuse).Error

		if err != nil {
			logger.Warn("[SuperAdmin] Failed to query audit logs for DNS abuse, falling back to direct query: %v", err)
			dnsKeyAbuse = []struct {
				OrganizationID string
				Count          int64
			}{}
		}
	}

	// FALLBACK: Use direct table query if audit logs unavailable
	if len(dnsKeyAbuse) == 0 && database.MetricsDB == nil {
		err := database.DB.Table("dns_delegation_api_keys").
			Select("organization_id, COUNT(*) as count").
			Where("created_at >= ?", twentyFourHoursAgo).
			Group("organization_id").
			Having("COUNT(*) > 5").
			Scan(&dnsKeyAbuse).Error
		if err != nil {
			return nil, fmt.Errorf("query DNS delegation abuse: %w", err)
		}
	}

	for _, dns := range dnsKeyAbuse {
		if dns.OrganizationID != "" {
			var org database.Organization
			if err := database.DB.First(&org, "id = ?", dns.OrganizationID).Error; err == nil {
				activities = append(activities, &superadminv1.SuspiciousActivity{
					Id:              fmt.Sprintf("dns-%s", dns.OrganizationID),
					OrganizationId:   dns.OrganizationID,
					OrganizationName: org.Name,
					ActivityType:    "dns_delegation_abuse",
					Description:     fmt.Sprintf("Created %d DNS delegation API keys in 24 hours", dns.Count),
					Severity:        55,
					OccurredAt:      timestamppb.New(time.Now()),
				})
			}
		}
	}

	return activities, nil
}

// detectUsageSpikes finds deployments with unusual resource usage spikes
func detectUsageSpikes(ctx context.Context, twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	if database.MetricsDB == nil {
		// MetricsDB not available, skip usage spike detection
		return activities, nil
	}

	// Get average usage for each deployment over the last 7 days (excluding last 24h)
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	oneDayAgo := time.Now().Add(-24 * time.Hour)

	// Calculate baseline (average usage in the 6 days before the last 24h)
	type baselineUsage struct {
		DeploymentID string
		AvgCPU       float64
		AvgMemory    float64
		AvgNetwork   float64
	}

	var baselines []baselineUsage
	err := database.MetricsDB.WithContext(ctx).Table("deployment_usage_hourly duh").
		Select(`
			duh.deployment_id,
			AVG(duh.avg_cpu_usage) as avg_cpu,
			AVG(duh.avg_memory_usage) as avg_memory,
			AVG(duh.bandwidth_rx_bytes + duh.bandwidth_tx_bytes) as avg_network
		`).
		Where("duh.hour >= ? AND duh.hour < ?", sevenDaysAgo, oneDayAgo).
		Group("duh.deployment_id").
		Scan(&baselines).Error
	if err != nil {
		// Table might not exist or have no data
		return activities, nil
	}

	// Get recent usage (last 24h)
	// Note: deployment_usage_hourly has organization_id denormalized, so we don't need to join
	type recentUsage struct {
		DeploymentID   string
		MaxCPU         float64
		MaxMemory      float64
		MaxNetwork     float64
		OrganizationID string
	}

	var recent []recentUsage
	err = database.MetricsDB.WithContext(ctx).Table("deployment_usage_hourly duh").
		Select(`
			duh.deployment_id,
			MAX(duh.avg_cpu_usage) as max_cpu,
			MAX(duh.avg_memory_usage) as max_memory,
			MAX(duh.bandwidth_rx_bytes + duh.bandwidth_tx_bytes) as max_network,
			MAX(duh.organization_id) as organization_id
		`).
		Where("duh.hour >= ?", twentyFourHoursAgo).
		Group("duh.deployment_id").
		Scan(&recent).Error
	if err != nil {
		// Table might not exist or have no data
		return activities, nil
	}

	// Create baseline map
	baselineMap := make(map[string]baselineUsage)
	for _, b := range baselines {
		baselineMap[b.DeploymentID] = b
	}

	// Check for spikes (recent usage > 3x baseline)
	for _, r := range recent {
		baseline, exists := baselineMap[r.DeploymentID]
		if !exists {
			continue // No baseline to compare against
		}

		hasSpike := false
		spikeDesc := []string{}

		// Check CPU spike (avoid division by zero)
		if baseline.AvgCPU > 0 && r.MaxCPU > baseline.AvgCPU*3 {
			hasSpike = true
			spikeDesc = append(spikeDesc, fmt.Sprintf("CPU: %.2f (baseline: %.2f)", r.MaxCPU, baseline.AvgCPU))
		}

		// Check memory spike
		if baseline.AvgMemory > 0 && r.MaxMemory > baseline.AvgMemory*3 {
			hasSpike = true
			spikeDesc = append(spikeDesc, fmt.Sprintf("Memory: %.2f MB (baseline: %.2f MB)", r.MaxMemory, baseline.AvgMemory))
		}

		// Check network spike
		if baseline.AvgNetwork > 0 && r.MaxNetwork > baseline.AvgNetwork*3 {
			hasSpike = true
			spikeDesc = append(spikeDesc, fmt.Sprintf("Network: %.2f bytes (baseline: %.2f bytes)", r.MaxNetwork, baseline.AvgNetwork))
		}

		if hasSpike {
			orgID := "unknown"
			orgName := "Unknown"
			if r.OrganizationID != "" {
				orgID = r.OrganizationID
				// Fetch organization name
				var org database.Organization
				if err := database.DB.First(&org, "id = ?", r.OrganizationID).Error; err == nil {
					orgName = org.Name
				}
			}

			activities = append(activities, &superadminv1.SuspiciousActivity{
				Id:              fmt.Sprintf("usage-spike-%s", r.DeploymentID),
				OrganizationId:   orgID,
				OrganizationName: orgName,
				ActivityType:    "usage_spike",
				Description:     fmt.Sprintf("Deployment %s: Unusual usage spike - %s", r.DeploymentID, strings.Join(spikeDesc, ", ")),
				Severity:        50,
				OccurredAt:      timestamppb.New(time.Now()),
			})
		}
	}

	return activities, nil
}

// detectGameServerAbuse finds organizations with suspicious game server activity
// Uses audit logs to prevent bypass by deleting records
func detectGameServerAbuse(twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	// PRIMARY: Use audit logs (can't be bypassed)
	var rapidGameServers []struct {
		OrganizationID string
		Count          int64
	}

	if database.MetricsDB != nil {
		err := database.MetricsDB.Raw(`
			SELECT 
				organization_id,
				COUNT(*) as count
			FROM audit_logs
			WHERE created_at >= ?
				AND action = 'CreateGameServer'
				AND response_status = 200
				AND organization_id IS NOT NULL
			GROUP BY organization_id
			HAVING COUNT(*) > 5
		`, twentyFourHoursAgo).
			Scan(&rapidGameServers).Error

		if err != nil {
			logger.Warn("[SuperAdmin] Failed to query audit logs for game server abuse, falling back to direct query: %v", err)
			rapidGameServers = []struct {
				OrganizationID string
				Count          int64
			}{}
		}
	}

	// FALLBACK: Use direct table query if audit logs unavailable
	if len(rapidGameServers) == 0 && database.MetricsDB == nil {
		err := database.DB.Table("game_servers").
			Select("organization_id, COUNT(*) as count").
			Where("created_at >= ? AND deleted_at IS NULL", twentyFourHoursAgo).
			Group("organization_id").
			Having("COUNT(*) > 5").
			Scan(&rapidGameServers).Error
		if err != nil {
			return nil, fmt.Errorf("query rapid game server creation: %w", err)
		}
	}

	for _, gs := range rapidGameServers {
		var org database.Organization
		if err := database.DB.First(&org, "id = ?", gs.OrganizationID).Error; err == nil {
			activities = append(activities, &superadminv1.SuspiciousActivity{
				Id:              fmt.Sprintf("game-server-%s", gs.OrganizationID),
				OrganizationId:   gs.OrganizationID,
				OrganizationName: org.Name,
				ActivityType:    "game_server_abuse",
				Description:     fmt.Sprintf("Created %d game servers in 24 hours", gs.Count),
				Severity:        60,
				OccurredAt:      timestamppb.New(time.Now()),
			})
		}
	}

	return activities, nil
}

// notifySuperadminsOfAbuse sends notifications to all superadmins when abuse is detected
// Only sends notifications if the abuse detection results have changed since the last notification
// Uses a background context to avoid cancellation issues
func notifySuperadminsOfAbuse(ctx context.Context, suspiciousOrgs []*superadminv1.SuspiciousOrganization, suspiciousActivities []*superadminv1.SuspiciousActivity) {
	// Use background context with timeout to avoid cancellation
	bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Generate a fingerprint of the current abuse detection results
	abuseFingerprint := generateAbuseFingerprint(suspiciousOrgs, suspiciousActivities)
	
	// Check if we've already sent a notification for this exact abuse pattern
	// Look for recent notifications with the same fingerprint in metadata
	// Use JSONB query to check metadata field
	var existingNotification database.Notification
	err := database.DB.Where("type = ? AND metadata::jsonb->>'abuse_fingerprint' = ?", 
		"SYSTEM", 
		abuseFingerprint).
		Order("created_at DESC").
		First(&existingNotification).Error
	
	// If we found a recent notification (within last 24 hours) with the same fingerprint, skip
	if err == nil {
		// Check if notification is recent (within 24 hours)
		if time.Since(existingNotification.CreatedAt) < 24*time.Hour {
			logger.Info("[SuperAdmin] Abuse detection results unchanged, skipping duplicate notification (fingerprint: %s)", abuseFingerprint)
			return
		}
	}
	
	// Get superadmin emails from environment
	superAdminEmails := getSuperAdminEmails()
	logger.Info("[SuperAdmin] Found %d superadmin email(s) configured", len(superAdminEmails))
	if len(superAdminEmails) == 0 {
		logger.Warn("[SuperAdmin] No superadmin emails configured, skipping abuse notifications")
		return
	}

	// Resolve emails to user IDs by checking all organization members
	resolver := organizations.GetUserProfileResolver()
	if resolver == nil || !resolver.IsConfigured() {
		logger.Warn("[SuperAdmin] User profile resolver not configured, skipping abuse notifications")
		return
	}

	// Find superadmin user IDs by email
	// First, try to find user IDs from organization members and match by email
	var userIDs []string
	if err := database.DB.Model(&database.OrganizationMember{}).
		Distinct("user_id").
		Where("user_id NOT LIKE ? AND user_id != ''", "pending:%").
		Pluck("user_id", &userIDs).Error; err != nil {
		logger.Warn("[SuperAdmin] Failed to get user IDs for abuse notifications: %v", err)
		return
	}
	logger.Info("[SuperAdmin] Found %d user ID(s) from organization members to check for superadmin status", len(userIDs))
	
	// Find superadmin user IDs by resolving profiles and checking emails
	superAdminUserIDs := make(map[string]bool)
	for _, userID := range userIDs {
		profile, err := resolver.Resolve(bgCtx, userID)
		if err != nil {
			logger.Debug("[SuperAdmin] Failed to resolve profile for user %s: %v", userID, err)
			continue
		}
		if profile == nil {
			logger.Debug("[SuperAdmin] Profile is nil for user %s", userID)
			continue
		}
		email := profile.Email
		if email == "" {
			logger.Debug("[SuperAdmin] User %s has no email in profile", userID)
			continue
		}
		lowerEmail := strings.ToLower(email)
		logger.Debug("[SuperAdmin] Checking user %s with email %s against superadmin list", userID, lowerEmail)
		if _, isSuperAdmin := superAdminEmails[lowerEmail]; isSuperAdmin {
			superAdminUserIDs[userID] = true
			logger.Info("[SuperAdmin] Found superadmin user: %s (email: %s)", userID, lowerEmail)
		}
	}
	
	// If we didn't find all superadmins, log a warning
	// Superadmins need to be in at least one organization to receive notifications
	if len(superAdminUserIDs) < len(superAdminEmails) {
		logger.Warn("[SuperAdmin] Only found %d/%d superadmins from organization members", len(superAdminUserIDs), len(superAdminEmails))
		for email := range superAdminEmails {
			found := false
			for userID := range superAdminUserIDs {
				if profile, err := resolver.Resolve(bgCtx, userID); err == nil && profile != nil {
					if strings.ToLower(profile.Email) == email {
						found = true
						break
					}
				}
			}
			if !found {
				logger.Warn("[SuperAdmin] Superadmin email %s not found. User must be a member of at least one organization to receive notifications.", email)
			}
		}
	}

	// Count high-severity activities
	highSeverityCount := 0
	for _, activity := range suspiciousActivities {
		if activity.Severity >= 70 {
			highSeverityCount++
		}
	}

	// Build notification message
	title := "Abuse Detection Alert"
	message := fmt.Sprintf("Detected %d suspicious organization(s) and %d suspicious activity/activities", len(suspiciousOrgs), len(suspiciousActivities))
	if highSeverityCount > 0 {
		message += fmt.Sprintf(" (%d high-severity)", highSeverityCount)
	}

	// Build detailed message
	details := []string{}
	if len(suspiciousOrgs) > 0 {
		details = append(details, fmt.Sprintf("• %d suspicious organization(s)", len(suspiciousOrgs)))
	}
	
	// Group activities by type
	activityCounts := make(map[string]int)
	for _, activity := range suspiciousActivities {
		activityCounts[activity.ActivityType]++
	}
	for activityType, count := range activityCounts {
		details = append(details, fmt.Sprintf("• %d %s", count, formatActivityType(activityType)))
	}

	if len(details) > 0 {
		message += "\n\n" + strings.Join(details, "\n")
	}

	// Get dashboard URL for action link
	dashboardURL := os.Getenv("DASHBOARD_URL")
	if dashboardURL == "" {
		dashboardURL = "https://cloud.obiente.com"
	}
	actionURL := dashboardURL + "/superadmin/abuse"
	actionLabel := "View Abuse Detection"

	// Determine severity based on findings
	// Default to HIGH since abuse detection is always important and SYSTEM notifications require HIGH minimum severity for emails
	severity := notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH
	if highSeverityCount > 10 || len(suspiciousOrgs) > 20 {
		severity = notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL
	}
	
	logger.Info("[SuperAdmin] Resolved %d superadmin user ID(s) from %d total users", len(superAdminUserIDs), len(userIDs))
	if len(superAdminUserIDs) == 0 {
		logger.Warn("[SuperAdmin] No superadmin user IDs found, cannot send abuse notifications")
		return
	}

	// Send notification to each superadmin
	// Use bgCtx to avoid cancellation issues
	notificationCount := 0
	for userID := range superAdminUserIDs {
		logger.Info("[SuperAdmin] Attempting to send abuse notification to superadmin user: %s", userID)
		if err := notifications.CreateNotificationForUser(
			bgCtx,
			userID,
			nil, // No specific organization
			notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM,
			severity,
			title,
			message,
			&actionURL,
			&actionLabel,
			map[string]string{
				"abuse_type":        "detection",
				"abuse_fingerprint":  abuseFingerprint,
				"suspicious_orgs":    fmt.Sprintf("%d", len(suspiciousOrgs)),
				"suspicious_acts":    fmt.Sprintf("%d", len(suspiciousActivities)),
				"high_severity":      fmt.Sprintf("%d", highSeverityCount),
			},
		); err != nil {
			logger.Error("[SuperAdmin] Failed to send abuse notification to user %s: %v", userID, err)
		} else {
			notificationCount++
			logger.Info("[SuperAdmin] Successfully sent abuse detection notification to superadmin user: %s", userID)
		}
	}
	
	if notificationCount == 0 {
		logger.Error("[SuperAdmin] No abuse notifications were successfully sent to any superadmin")
	} else {
		logger.Info("[SuperAdmin] Successfully sent %d abuse detection notification(s) to superadmins", notificationCount)
	}
}

// getSuperAdminEmails returns the set of superadmin emails from environment
func getSuperAdminEmails() map[string]struct{} {
	superAdmins := make(map[string]struct{})
	envValue := os.Getenv("SUPERADMIN_EMAILS")
	
	for _, raw := range strings.Split(envValue, ",") {
		email := strings.TrimSpace(raw)
		email = strings.Trim(email, "\"'")
		email = strings.ToLower(email)
		if email != "" {
			superAdmins[email] = struct{}{}
		}
	}
	
	return superAdmins
}

// formatActivityType formats activity type for display
func formatActivityType(activityType string) string {
	typeMap := map[string]string{
		"rapid_creation":         "rapid resource creation(s)",
		"failed_payments":       "failed payment attempt(s)",
		"ssh_brute_force":       "SSH brute force attempt(s)",
		"api_abuse":             "API abuse case(s)",
		"failed_authentication": "failed authentication attempt(s)",
		"multiple_accounts":     "multiple account creation(s)",
		"dns_delegation_abuse":  "DNS delegation abuse case(s)",
		"usage_spike":           "usage spike(s)",
		"game_server_abuse":     "game server abuse case(s)",
	}
	if formatted, ok := typeMap[activityType]; ok {
		return formatted
	}
	return activityType
}

// generateAbuseFingerprint creates a unique hash of the abuse detection results
// This is used to detect if the abuse pattern has changed since the last notification
func generateAbuseFingerprint(suspiciousOrgs []*superadminv1.SuspiciousOrganization, suspiciousActivities []*superadminv1.SuspiciousActivity) string {
	// Collect all organization IDs and activity IDs in sorted order for consistent hashing
	orgIDs := make([]string, 0, len(suspiciousOrgs))
	for _, org := range suspiciousOrgs {
		orgIDs = append(orgIDs, org.OrganizationId)
	}
	sort.Strings(orgIDs)
	
	activityIDs := make([]string, 0, len(suspiciousActivities))
	for _, activity := range suspiciousActivities {
		activityIDs = append(activityIDs, activity.Id)
	}
	sort.Strings(activityIDs)
	
	// Create a JSON representation for hashing
	fingerprintData := map[string]interface{}{
		"orgs":     orgIDs,
		"activities": activityIDs,
	}
	
	jsonData, err := json.Marshal(fingerprintData)
	if err != nil {
		// Fallback to simple concatenation if JSON marshaling fails
		return fmt.Sprintf("%d-%d", len(orgIDs), len(activityIDs))
	}
	
	// Generate SHA256 hash
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

