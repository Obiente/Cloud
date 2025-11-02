package superadmin

import (
	"context"
	"fmt"
	"strings"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	superadminv1 "api/gen/proto/obiente/cloud/superadmin/v1"
	superadminv1connect "api/gen/proto/obiente/cloud/superadmin/v1/superadminv1connect"
	"api/internal/auth"
	"api/internal/database"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const overviewLimit = 50

type Service struct {
	superadminv1connect.UnimplementedSuperadminServiceHandler
}

func NewService() superadminv1connect.SuperadminServiceHandler {
	return &Service{}
}

func (s *Service) GetOverview(ctx context.Context, _ *connect.Request[superadminv1.GetOverviewRequest]) (*connect.Response[superadminv1.GetOverviewResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	if database.DB == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialised"))
	}

	counts, err := aggregateCounts()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	organizations, err := loadOrganizationOverviews()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	invites, err := loadPendingInvites()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	deployments, err := loadDeploymentOverviews()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	usages, err := loadCurrentUsage()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &superadminv1.GetOverviewResponse{
		Counts:         counts,
		Organizations:  organizations,
		PendingInvites: invites,
		Deployments:    deployments,
		Usages:         usages,
	}

	return connect.NewResponse(resp), nil
}

func aggregateCounts() (*superadminv1.OverviewCounts, error) {
	var totalOrgs int64
	if err := database.DB.Model(&database.Organization{}).Count(&totalOrgs).Error; err != nil {
		return nil, fmt.Errorf("count organizations: %w", err)
	}

	var activeMembers int64
	if err := database.DB.Model(&database.OrganizationMember{}).Where("status = ?", "active").Count(&activeMembers).Error; err != nil {
		return nil, fmt.Errorf("count members: %w", err)
	}

	var pendingInvites int64
	if err := database.DB.Model(&database.OrganizationMember{}).Where("status = ?", "invited").Count(&pendingInvites).Error; err != nil {
		return nil, fmt.Errorf("count invites: %w", err)
	}

	var totalDeployments int64
	if err := database.DB.Model(&database.Deployment{}).Count(&totalDeployments).Error; err != nil {
		return nil, fmt.Errorf("count deployments: %w", err)
	}

	return &superadminv1.OverviewCounts{
		TotalOrganizations: totalOrgs,
		ActiveMembers:      activeMembers,
		PendingInvites:     pendingInvites,
		TotalDeployments:   totalDeployments,
	}, nil
}

type organizationOverviewRow struct {
	ID              string
	Name            string
	Slug            string
	Plan            string
	Status          string
	Domain          *string
	CreatedAt       time.Time
	MemberCount     int64
	InviteCount     int64
	DeploymentCount int64
}

func loadOrganizationOverviews() ([]*superadminv1.OrganizationOverview, error) {
	var rows []organizationOverviewRow
	if err := database.DB.Table("organizations o").
		Select(`o.id, o.name, o.slug, o.plan, o.status, o.domain, o.created_at,
			SUM(CASE WHEN m.status = 'active' THEN 1 ELSE 0 END) AS member_count,
			SUM(CASE WHEN m.status = 'invited' THEN 1 ELSE 0 END) AS invite_count,
			COUNT(d.id) AS deployment_count`).
		Joins("LEFT JOIN organization_members m ON m.organization_id = o.id").
		Joins("LEFT JOIN deployments d ON d.organization_id = o.id").
		Group("o.id, o.name, o.slug, o.plan, o.status, o.domain, o.created_at").
		Order("o.created_at DESC").
		Limit(overviewLimit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load organizations: %w", err)
	}

	items := make([]*superadminv1.OrganizationOverview, 0, len(rows))
	for _, r := range rows {
		item := &superadminv1.OrganizationOverview{
			Id:              r.ID,
			Name:            r.Name,
			Slug:            r.Slug,
			Plan:            r.Plan,
			Status:          r.Status,
			CreatedAt:       toTimestamp(r.CreatedAt),
			MemberCount:     r.MemberCount,
			InviteCount:     r.InviteCount,
			DeploymentCount: r.DeploymentCount,
		}
		if r.Domain != nil {
			item.Domain = r.Domain
		}
		items = append(items, item)
	}
	return items, nil
}

func loadPendingInvites() ([]*superadminv1.PendingInvite, error) {
	var rows []database.OrganizationMember
	if err := database.DB.Where("status = ?", "invited").Order("joined_at DESC").Limit(overviewLimit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load invites: %w", err)
	}

	items := make([]*superadminv1.PendingInvite, 0, len(rows))
	for _, row := range rows {
		email := strings.TrimPrefix(row.UserID, "pending:")
		items = append(items, &superadminv1.PendingInvite{
			Id:             row.ID,
			OrganizationId: row.OrganizationID,
			Email:          email,
			Role:           row.Role,
			InvitedAt:      toTimestamp(row.JoinedAt),
		})
	}
	return items, nil
}

func loadDeploymentOverviews() ([]*superadminv1.DeploymentOverview, error) {
	var rows []database.Deployment
	if err := database.DB.Order("created_at DESC").Limit(overviewLimit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load deployments: %w", err)
	}

	items := make([]*superadminv1.DeploymentOverview, 0, len(rows))
	for _, row := range rows {
		item := &superadminv1.DeploymentOverview{
			Id:             row.ID,
			OrganizationId: row.OrganizationID,
			Name:           row.Name,
			Environment:    deploymentsv1.Environment(row.Environment),
			Status:         deploymentsv1.DeploymentStatus(row.Status),
			CreatedAt:      toTimestamp(row.CreatedAt),
			LastDeployedAt: toTimestamp(row.LastDeployedAt),
		}
		if row.Domain != "" {
			domain := row.Domain
			item.Domain = &domain
		}
		items = append(items, item)
	}
	return items, nil
}

type usageRow struct {
	OrganizationID        string
	OrganizationName      string
	Month                 string
	CPUCoreSeconds        int64
	MemoryByteSeconds     int64
	BandwidthRxBytes      int64
	BandwidthTxBytes      int64
	StorageBytes          int64
	DeploymentsActivePeak int32
}

func loadCurrentUsage() ([]*superadminv1.OrganizationUsage, error) {
	now := time.Now().UTC()
	month := now.Format("2006-01")
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)
	
	// Calculate usage from deployment_usage_hourly (single source of truth)
	var rows []usageRow
	if err := database.DB.Table("deployment_usage_hourly duh").
		Select(`
			duh.organization_id,
			COALESCE(o.name, '') AS organization_name,
			? AS month,
			COALESCE(SUM((duh.avg_cpu_usage / 100.0) * 3600), 0) AS cpu_core_seconds,
			COALESCE(SUM(duh.avg_memory_usage * 3600), 0) AS memory_byte_seconds,
			COALESCE(SUM(duh.bandwidth_rx_bytes), 0) AS bandwidth_rx_bytes,
			COALESCE(SUM(duh.bandwidth_tx_bytes), 0) AS bandwidth_tx_bytes,
			COALESCE((SELECT SUM(storage_bytes) FROM deployments WHERE organization_id = duh.organization_id), 0) AS storage_bytes,
			0 AS deployments_active_peak
		`, month).
		Joins("LEFT JOIN organizations o ON o.id = duh.organization_id").
		Where("duh.hour >= ? AND duh.hour <= ?", monthStart, monthEnd).
		Group("duh.organization_id, o.name").
		Order("cpu_core_seconds DESC").
		Limit(overviewLimit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load usage: %w", err)
	}

	items := make([]*superadminv1.OrganizationUsage, 0, len(rows))
	for _, row := range rows {
		items = append(items, &superadminv1.OrganizationUsage{
			OrganizationId:        row.OrganizationID,
			OrganizationName:      row.OrganizationName,
			Month:                 row.Month,
			CpuCoreSeconds:        row.CPUCoreSeconds,
			MemoryByteSeconds:     row.MemoryByteSeconds,
			BandwidthRxBytes:      row.BandwidthRxBytes,
			BandwidthTxBytes:      row.BandwidthTxBytes,
			StorageBytes:          row.StorageBytes,
			DeploymentsActivePeak: row.DeploymentsActivePeak,
		})
	}
	return items, nil
}

func toTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
