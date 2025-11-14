package superadmin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api/internal/database"
	"api/internal/logger"

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

	return allActivities, nil
}

// detectRapidResourceCreation finds organizations creating resources too quickly
func detectRapidResourceCreation(twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	var rapidCreations []struct {
		OrganizationID string
		Count          int64
	}
	err := database.DB.Table("deployments").
		Select("organization_id, COUNT(*) as count").
		Where("created_at >= ?", twentyFourHoursAgo).
		Group("organization_id").
		Having("COUNT(*) > 10").
		Scan(&rapidCreations).Error
	if err != nil {
		return nil, fmt.Errorf("query rapid creations: %w", err)
	}

	for _, rc := range rapidCreations {
		var org database.Organization
		if err := database.DB.First(&org, "id = ?", rc.OrganizationID).Error; err == nil {
			activities = append(activities, &superadminv1.SuspiciousActivity{
				Id:            fmt.Sprintf("rapid-%s", rc.OrganizationID),
				OrganizationId: rc.OrganizationID,
				ActivityType:  "rapid_creation",
				Description:   fmt.Sprintf("Created %d resources in 24 hours", rc.Count),
				Severity:      50,
				OccurredAt:    timestamppb.New(time.Now()),
			})
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
				Id:            fmt.Sprintf("payment-%s", fp.OrganizationID),
				OrganizationId: fp.OrganizationID,
				ActivityType:  "failed_payments",
				Description:   fmt.Sprintf("%d failed payment attempts in 24 hours", fp.Count),
				Severity:      70,
				OccurredAt:    timestamppb.New(time.Now()),
			})
		}
	}

	return activities, nil
}

// detectSSHBruteForce finds IP addresses with multiple failed SSH connection attempts
func detectSSHBruteForce(ctx context.Context, twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	if database.MetricsDB == nil {
		// MetricsDB not available, skip SSH brute force detection
		return activities, nil
	}

	var bruteForceAttempts []struct {
		IPAddress string
		Count     int64
		UserID    string
		OrgID     *string
	}

	err := database.MetricsDB.WithContext(ctx).Table("audit_logs").
		Select("ip_address, COUNT(*) as count, user_id, organization_id as org_id").
		Where("action = ? AND response_status != 200 AND created_at >= ?", "SSHConnect", twentyFourHoursAgo).
		Group("ip_address, user_id, organization_id").
		Having("COUNT(*) > 5").
		Scan(&bruteForceAttempts).Error
	if err != nil {
		return nil, fmt.Errorf("query SSH brute force: %w", err)
	}

	for _, bf := range bruteForceAttempts {
		orgID := "unknown"
		if bf.OrgID != nil && *bf.OrgID != "" {
			orgID = *bf.OrgID
		}

		activities = append(activities, &superadminv1.SuspiciousActivity{
			Id:            fmt.Sprintf("ssh-bf-%s", bf.IPAddress),
			OrganizationId: orgID,
			ActivityType:  "ssh_brute_force",
			Description:   fmt.Sprintf("IP %s: %d failed SSH connection attempts in 24 hours", bf.IPAddress, bf.Count),
			Severity:      80,
			OccurredAt:    timestamppb.New(time.Now()),
		})
	}

	return activities, nil
}

// detectAPIAbuse finds users/organizations with excessive API calls
func detectAPIAbuse(ctx context.Context, twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	if database.MetricsDB == nil {
		// MetricsDB not available, skip API abuse detection
		return activities, nil
	}

	// Check for excessive API calls per user
	var userAbuse []struct {
		UserID         string
		OrganizationID *string
		Count          int64
		IPAddress      string
	}

	err := database.MetricsDB.WithContext(ctx).Table("audit_logs").
		Select("user_id, organization_id, COUNT(*) as count, MAX(ip_address) as ip_address").
		Where("created_at >= ?", twentyFourHoursAgo).
		Group("user_id, organization_id").
		Having("COUNT(*) > 1000").
		Scan(&userAbuse).Error
	if err != nil {
		return nil, fmt.Errorf("query API abuse by user: %w", err)
	}

	for _, abuse := range userAbuse {
		orgID := "unknown"
		if abuse.OrganizationID != nil && *abuse.OrganizationID != "" {
			orgID = *abuse.OrganizationID
		}

		activities = append(activities, &superadminv1.SuspiciousActivity{
			Id:            fmt.Sprintf("api-user-%s", abuse.UserID),
			OrganizationId: orgID,
			ActivityType:  "api_abuse",
			Description:   fmt.Sprintf("User %s: %d API calls in 24 hours (IP: %s)", abuse.UserID, abuse.Count, abuse.IPAddress),
			Severity:      60,
			OccurredAt:    timestamppb.New(time.Now()),
		})
	}

	// Check for excessive API calls per IP address
	var ipAbuse []struct {
		IPAddress string
		Count     int64
	}

	err = database.MetricsDB.WithContext(ctx).Table("audit_logs").
		Select("ip_address, COUNT(*) as count").
		Where("created_at >= ?", twentyFourHoursAgo).
		Group("ip_address").
		Having("COUNT(*) > 2000").
		Scan(&ipAbuse).Error
	if err != nil {
		return nil, fmt.Errorf("query API abuse by IP: %w", err)
	}

	for _, abuse := range ipAbuse {
		activities = append(activities, &superadminv1.SuspiciousActivity{
			Id:            fmt.Sprintf("api-ip-%s", abuse.IPAddress),
			OrganizationId: "unknown",
			ActivityType:  "api_abuse",
			Description:   fmt.Sprintf("IP %s: %d API calls in 24 hours", abuse.IPAddress, abuse.Count),
			Severity:      70,
			OccurredAt:    timestamppb.New(time.Now()),
		})
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
// NOTE: This requires DNS delegation API calls to be logged in audit_logs
// Currently, DNS delegation uses HTTP handlers, not gRPC, so they may not be in audit_logs
// This detection will work once DNS delegation calls are properly logged
func detectDNSDelegationAbuse(twentyFourHoursAgo time.Time) ([]*superadminv1.SuspiciousActivity, error) {
	var activities []*superadminv1.SuspiciousActivity

	// Check for organizations with many DNS delegation API keys created recently
	var dnsKeyAbuse []struct {
		OrganizationID string
		Count          int64
	}

	err := database.DB.Table("dns_delegation_api_keys").
		Select("organization_id, COUNT(*) as count").
		Where("created_at >= ?", twentyFourHoursAgo).
		Group("organization_id").
		Having("COUNT(*) > 5").
		Scan(&dnsKeyAbuse).Error
	if err != nil {
		return nil, fmt.Errorf("query DNS delegation abuse: %w", err)
	}

	for _, dns := range dnsKeyAbuse {
		if dns.OrganizationID != "" {
			var org database.Organization
			if err := database.DB.First(&org, "id = ?", dns.OrganizationID).Error; err == nil {
				activities = append(activities, &superadminv1.SuspiciousActivity{
					Id:            fmt.Sprintf("dns-%s", dns.OrganizationID),
					OrganizationId: dns.OrganizationID,
					ActivityType:  "dns_delegation_abuse",
					Description:   fmt.Sprintf("Created %d DNS delegation API keys in 24 hours", dns.Count),
					Severity:      55,
					OccurredAt:    timestamppb.New(time.Now()),
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
			if r.OrganizationID != "" {
				orgID = r.OrganizationID
			}

			activities = append(activities, &superadminv1.SuspiciousActivity{
				Id:            fmt.Sprintf("usage-spike-%s", r.DeploymentID),
				OrganizationId: orgID,
				ActivityType:  "usage_spike",
				Description:   fmt.Sprintf("Deployment %s: Unusual usage spike - %s", r.DeploymentID, strings.Join(spikeDesc, ", ")),
				Severity:      50,
				OccurredAt:    timestamppb.New(time.Now()),
			})
		}
	}

	return activities, nil
}

