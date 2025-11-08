package organizations

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	organizationsv1 "api/gen/proto/obiente/cloud/organizations/v1"
	"api/internal/database"
	"api/internal/email"

	"connectrpc.com/connect"
)

const (
	quotaWarningThreshold  = 0.80 // 80% usage triggers warning
	quotaCriticalThreshold = 0.95 // 95% usage triggers critical warning
)

// CheckAndNotifyQuotaWarnings checks resource usage and sends email notifications if approaching limits
func (s *Service) CheckAndNotifyQuotaWarnings(ctx context.Context, orgID string) error {
	// Get organization
	var org database.Organization
	if err := database.DB.First(&org, "id = ?", orgID).Error; err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	// Get quota and plan limits
	var quota database.OrgQuota
	hasQuota := database.DB.Where("organization_id = ?", orgID).First(&quota).Error == nil

	var plan database.OrganizationPlan
	hasPlan := false
	if hasQuota && quota.PlanID != "" {
		hasPlan = database.DB.First(&plan, "id = ?", quota.PlanID).Error == nil
	}

	if !hasPlan {
		return nil // No plan, no limits to check
	}

	// Get effective limits: use overrides if set, but cap them to plan limits
	// Plan limits are the final boundary - org overrides cannot exceed them
	effDeployMax := plan.DeploymentsMax
	if quota.DeploymentsMaxOverride != nil {
		overrideDeployMax := *quota.DeploymentsMaxOverride
		if overrideDeployMax > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if plan.DeploymentsMax > 0 && overrideDeployMax > plan.DeploymentsMax {
				effDeployMax = plan.DeploymentsMax
			} else {
				effDeployMax = overrideDeployMax
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	effMem := plan.MemoryBytes
	if quota.MemoryBytesOverride != nil {
		overrideMem := *quota.MemoryBytesOverride
		if overrideMem > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if plan.MemoryBytes > 0 && overrideMem > plan.MemoryBytes {
				effMem = plan.MemoryBytes
			} else {
				effMem = overrideMem
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	// CPU limits are checked in deployment creation, not here for monthly usage
	// We'll focus on deployments, memory, bandwidth, and storage for monthly warnings

	effBandwidth := plan.BandwidthBytesMonth
	if quota.BandwidthBytesMonthOverride != nil {
		overrideBandwidth := *quota.BandwidthBytesMonthOverride
		if overrideBandwidth > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if plan.BandwidthBytesMonth > 0 && overrideBandwidth > plan.BandwidthBytesMonth {
				effBandwidth = plan.BandwidthBytesMonth
			} else {
				effBandwidth = overrideBandwidth
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	effStorage := plan.StorageBytes
	if quota.StorageBytesOverride != nil {
		overrideStorage := *quota.StorageBytesOverride
		if overrideStorage > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if plan.StorageBytes > 0 && overrideStorage > plan.StorageBytes {
				effStorage = plan.StorageBytes
			} else {
				effStorage = overrideStorage
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	// Get current usage
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Get usage data using GetUsage
	monthStr := now.Format("2006-01")
	usageReq := &organizationsv1.GetUsageRequest{
		OrganizationId: orgID,
		Month:          &monthStr,
	}
	usageResp, err := s.GetUsage(ctx, connect.NewRequest(usageReq))
	if err != nil {
		log.Printf("[Quota Warnings] Failed to get usage for org %s: %v", orgID, err)
		return nil // Don't fail if we can't get usage
	}

	usage := usageResp.Msg.Current
	if usage == nil {
		return nil // No usage data
	}

	// Check each resource and send warnings if needed
	warnings := []string{}
	criticalWarnings := []string{}

	// Check deployments
	if effDeployMax > 0 {
		currentDeployments := int(usage.DeploymentsActivePeak)
		usagePercent := float64(currentDeployments) / float64(effDeployMax)
		if usagePercent >= quotaCriticalThreshold {
			criticalWarnings = append(criticalWarnings, fmt.Sprintf("Deployments: %d/%d (%.1f%%)", currentDeployments, effDeployMax, usagePercent*100))
		} else if usagePercent >= quotaWarningThreshold {
			warnings = append(warnings, fmt.Sprintf("Deployments: %d/%d (%.1f%%)", currentDeployments, effDeployMax, usagePercent*100))
		}
	}

	// Check memory (convert byte-seconds to bytes for comparison)
	if effMem > 0 {
		// Get average memory usage (byte-seconds / seconds in month so far)
		secondsElapsed := int64(now.Sub(monthStart).Seconds())
		if secondsElapsed > 0 {
			avgMemoryBytes := usage.MemoryByteSeconds / secondsElapsed
			usagePercent := float64(avgMemoryBytes) / float64(effMem)
			if usagePercent >= quotaCriticalThreshold {
				criticalWarnings = append(criticalWarnings, fmt.Sprintf("Memory: %.1f GB / %.1f GB (%.1f%%)",
					float64(avgMemoryBytes)/(1024*1024*1024), float64(effMem)/(1024*1024*1024), usagePercent*100))
			} else if usagePercent >= quotaWarningThreshold {
				warnings = append(warnings, fmt.Sprintf("Memory: %.1f GB / %.1f GB (%.1f%%)",
					float64(avgMemoryBytes)/(1024*1024*1024), float64(effMem)/(1024*1024*1024), usagePercent*100))
			}
		}
	}

	// Check bandwidth
	if effBandwidth > 0 {
		bandwidthUsed := usage.BandwidthRxBytes + usage.BandwidthTxBytes
		usagePercent := float64(bandwidthUsed) / float64(effBandwidth)
		if usagePercent >= quotaCriticalThreshold {
			criticalWarnings = append(criticalWarnings, fmt.Sprintf("Bandwidth: %.1f GB / %.1f GB (%.1f%%)",
				float64(bandwidthUsed)/(1024*1024*1024), float64(effBandwidth)/(1024*1024*1024), usagePercent*100))
		} else if usagePercent >= quotaWarningThreshold {
			warnings = append(warnings, fmt.Sprintf("Bandwidth: %.1f GB / %.1f GB (%.1f%%)",
				float64(bandwidthUsed)/(1024*1024*1024), float64(effBandwidth)/(1024*1024*1024), usagePercent*100))
		}
	}

	// Check storage
	if effStorage > 0 {
		usagePercent := float64(usage.StorageBytes) / float64(effStorage)
		if usagePercent >= quotaCriticalThreshold {
			criticalWarnings = append(criticalWarnings, fmt.Sprintf("Storage: %.1f GB / %.1f GB (%.1f%%)",
				float64(usage.StorageBytes)/(1024*1024*1024), float64(effStorage)/(1024*1024*1024), usagePercent*100))
		} else if usagePercent >= quotaWarningThreshold {
			warnings = append(warnings, fmt.Sprintf("Storage: %.1f GB / %.1f GB (%.1f%%)",
				float64(usage.StorageBytes)/(1024*1024*1024), float64(effStorage)/(1024*1024*1024), usagePercent*100))
		}
	}

	// Send email if there are warnings
	if len(criticalWarnings) > 0 || len(warnings) > 0 {
		return s.sendQuotaWarningEmail(ctx, orgID, org.Name, criticalWarnings, warnings, plan.Name)
	}

	return nil
}

func (s *Service) sendQuotaWarningEmail(ctx context.Context, orgID, orgName string, criticalWarnings, warnings []string, planName string) error {
	if !s.mailer.Enabled() {
		log.Printf("[Quota Warnings] Email disabled, skipping quota warning for org %s", orgID)
		return nil
	}

	// Get organization owners/admins for email
	var members []database.OrganizationMember
	if err := database.DB.Where("organization_id = ? AND role IN (?, ?) AND status = ?", orgID, "owner", "admin", "active").Find(&members).Error; err != nil {
		return fmt.Errorf("get members: %w", err)
	}

	if len(members) == 0 {
		return nil // No one to email
	}

	// Get user emails using the user profile resolver
	var emails []string
	resolver := getUserProfileResolver()
	for _, member := range members {
		if member.UserID == "" || strings.HasPrefix(member.UserID, "pending:") {
			continue // Skip pending invites
		}

		// Resolve user profile to get email
		userProfile, err := resolver.Resolve(ctx, member.UserID)
		if err != nil {
			log.Printf("[Quota Warnings] Failed to resolve user profile for %s: %v", member.UserID, err)
			continue
		}

		if userProfile != nil && userProfile.Email != "" {
			emails = append(emails, userProfile.Email)
		}
	}

	// If no emails found, log and return (don't fail)
	if len(emails) == 0 {
		log.Printf("[Quota Warnings] No email addresses found for org %s members, skipping email notification", orgID)
		return nil
	}

	// Determine severity and subject
	severity := "warning"
	subject := "Resource Usage Warning - Approaching Plan Limits"
	if len(criticalWarnings) > 0 {
		severity = "critical"
		subject = "URGENT: Resource Usage Critical - Near Plan Limits"
	}

	// Build email content
	introLines := []string{
		fmt.Sprintf("Your organization '%s' is approaching resource limits on the %s plan.", orgName, planName),
	}

	if len(criticalWarnings) > 0 {
		introLines = append(introLines, "The following resources are critically high:")
	} else {
		introLines = append(introLines, "The following resources are approaching their limits:")
	}

	highlights := []email.Highlight{}
	allWarnings := append(criticalWarnings, warnings...)
	for i, warning := range allWarnings {
		if i < 3 { // Show first 3 in highlights
			highlights = append(highlights, email.Highlight{
				Label: "Resource",
				Value: warning,
			})
		}
	}

	sections := []email.Section{
		{
			Title: "What you can do",
			Lines: []string{
				"Review your resource usage in the dashboard",
				"Consider upgrading your plan for higher limits",
				"Stop unused deployments to free up resources",
				"Contact support if you need assistance",
			},
		},
	}

	consoleURL := fmt.Sprintf("%s/billing?organizationId=%s", s.consoleURL, orgID)
	if s.consoleURL == "" {
		consoleURL = fmt.Sprintf("https://obiente.cloud/billing?organizationId=%s", orgID)
	}

	tmpl := email.TemplateData{
		Subject:     subject,
		PreviewText: fmt.Sprintf("Your %s plan resources are %s", planName, severity),
		Greeting:    fmt.Sprintf("Hi %s,", orgName),
		Heading:     "Resource Usage Alert",
		IntroLines:  introLines,
		Highlights:  highlights,
		Sections:    sections,
		CTA: &email.CTA{
			Label: "View Usage & Plans",
			URL:   consoleURL,
		},
		Category:     email.CategoryNotification,
		SupportEmail: s.supportEmail,
	}

	msg := &email.Message{
		To:       emails,
		Subject:  subject,
		Template: &tmpl,
		Category: email.CategoryNotification,
		Metadata: map[string]string{
			"organization_id": orgID,
			"severity":        severity,
			"plan_name":       planName,
		},
	}

	if err := s.mailer.Send(ctx, msg); err != nil {
		log.Printf("[Quota Warnings] Failed to send email for org %s: %v", orgID, err)
		return fmt.Errorf("send email: %w", err)
	}

	log.Printf("[Quota Warnings] Sent %s quota warning email to %d recipients for org %s", severity, len(emails), orgID)
	return nil
}
