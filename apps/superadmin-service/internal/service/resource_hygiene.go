package superadmin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultDormantMinInactiveDays = 30

type dormantResourceQueryRow struct {
	UserID                 string     `gorm:"column:user_id"`
	OrganizationID         string     `gorm:"column:organization_id"`
	VPSCount               int64      `gorm:"column:vps_count"`
	DatabaseCount          int64      `gorm:"column:database_count"`
	DeploymentCount        int64      `gorm:"column:deployment_count"`
	GameServerCount        int64      `gorm:"column:game_server_count"`
	VPSDiskBytes           int64      `gorm:"column:vps_disk_bytes"`
	DatabaseDiskBytes      int64      `gorm:"column:database_disk_bytes"`
	DeploymentStorageBytes int64      `gorm:"column:deployment_storage_bytes"`
	GameServerStorageBytes int64      `gorm:"column:game_server_storage_bytes"`
	LastResourceCreatedAt  *time.Time `gorm:"column:last_resource_created_at"`
	LastResourceUpdatedAt  *time.Time `gorm:"column:last_resource_updated_at"`
}

type dormantResourceLastActivityRow struct {
	UserID         string    `gorm:"column:user_id"`
	LastActivityAt time.Time `gorm:"column:last_activity_at"`
}

type dormantResourceOrganizationAggregate struct {
	OrganizationID   string
	OrganizationName string
	VPSCount         int32
	DatabaseCount    int32
	DeploymentCount  int32
	GameServerCount  int32
	ReservedBytes    int64
}

type dormantResourceOwnerAggregate struct {
	UserID                string
	VPSCount              int32
	DatabaseCount         int32
	DeploymentCount       int32
	GameServerCount       int32
	TotalReservedBytes    int64
	LastResourceCreatedAt *time.Time
	LastResourceUpdatedAt *time.Time
	Organizations         map[string]*dormantResourceOrganizationAggregate
}

func (s *Service) ListDormantResourceOwners(ctx context.Context, req *connect.Request[superadminv1.ListDormantResourceOwnersRequest]) (*connect.Response[superadminv1.ListDormantResourceOwnersResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.users.read") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}
	if database.DB == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialised"))
	}

	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 {
		perPage = 25
	}
	if perPage > 100 {
		perPage = 100
	}
	minInactiveDays := int(req.Msg.GetMinInactiveDays())
	if minInactiveDays < 1 {
		minInactiveDays = defaultDormantMinInactiveDays
	}
	searchTerm := strings.TrimSpace(req.Msg.GetSearch())

	orgRows, err := loadDormantResourceRows(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load dormant resource rows: %w", err))
	}

	owners := aggregateDormantResourceRows(orgRows)
	if len(owners) == 0 {
		return connect.NewResponse(&superadminv1.ListDormantResourceOwnersResponse{
			Owners: []*superadminv1.DormantResourceOwner{},
			Pagination: &commonv1.Pagination{
				Page:       int32(page),
				PerPage:    int32(perPage),
				Total:      0,
				TotalPages: 0,
			},
			Summary: &superadminv1.DormantResourceSummary{},
		}), nil
	}

	userIDs := make([]string, 0, len(owners))
	orgIDs := make(map[string]struct{})
	for userID, owner := range owners {
		userIDs = append(userIDs, userID)
		for orgID := range owner.Organizations {
			orgIDs[orgID] = struct{}{}
		}
	}

	orgNames, err := loadOrganizationNames(ctx, keysFromSet(orgIDs))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load organization names: %w", err))
	}
	lastActivities := loadDormantResourceLastActivities(ctx, userIDs)

	resolver := organizations.GetUserProfileResolver()
	profiles := make(map[string]*authv1.User, len(userIDs))
	if resolver != nil && resolver.IsConfigured() {
		for _, userID := range userIDs {
			profile, err := resolver.Resolve(ctx, userID)
			if err != nil {
				logger.Debug("[SuperAdmin] Failed to resolve profile for dormant resource owner %s: %v", userID, err)
				continue
			}
			profiles[userID] = profile
		}
	}

	now := time.Now()
	filtered := make([]*superadminv1.DormantResourceOwner, 0, len(owners))
	for _, owner := range owners {
		profile := profiles[owner.UserID]
		userInfo := buildDormantResourceUserInfo(owner.UserID, profile)
		lastActivityAt, lastActivitySource := chooseDormantOwnerLastActivity(profile, lastActivities[owner.UserID], owner.LastResourceUpdatedAt, owner.LastResourceCreatedAt)
		inactiveDays := calculateInactiveDays(now, lastActivityAt)

		if inactiveDays < int32(minInactiveDays) {
			continue
		}

		orgs := make([]*superadminv1.DormantResourceOrganization, 0, len(owner.Organizations))
		for _, org := range owner.Organizations {
			orgName := orgNames[org.OrganizationID]
			if orgName == "" {
				orgName = org.OrganizationID
			}
			org.OrganizationName = orgName
			orgs = append(orgs, &superadminv1.DormantResourceOrganization{
				OrganizationId:   org.OrganizationID,
				OrganizationName: orgName,
				VpsCount:         org.VPSCount,
				DatabaseCount:    org.DatabaseCount,
				DeploymentCount:  org.DeploymentCount,
				GameServerCount:  org.GameServerCount,
				ReservedBytes:    org.ReservedBytes,
			})
		}
		sort.Slice(orgs, func(i, j int) bool {
			if orgs[i].ReservedBytes == orgs[j].ReservedBytes {
				return orgs[i].OrganizationName < orgs[j].OrganizationName
			}
			return orgs[i].ReservedBytes > orgs[j].ReservedBytes
		})

		resourceOwner := &superadminv1.DormantResourceOwner{
			User:               userInfo,
			LastActivitySource: lastActivitySource,
			InactiveDays:       inactiveDays,
			OrganizationCount:  int32(len(orgs)),
			VpsCount:           owner.VPSCount,
			DatabaseCount:      owner.DatabaseCount,
			DeploymentCount:    owner.DeploymentCount,
			GameServerCount:    owner.GameServerCount,
			TotalReservedBytes: owner.TotalReservedBytes,
			Organizations:      orgs,
		}
		if lastActivityAt != nil {
			resourceOwner.LastActivityAt = timestamppb.New(*lastActivityAt)
		}
		if owner.LastResourceCreatedAt != nil {
			resourceOwner.LastResourceCreatedAt = timestamppb.New(*owner.LastResourceCreatedAt)
		}
		if owner.LastResourceUpdatedAt != nil {
			resourceOwner.LastResourceUpdatedAt = timestamppb.New(*owner.LastResourceUpdatedAt)
		}

		if !matchesDormantResourceSearch(resourceOwner, searchTerm) {
			continue
		}

		filtered = append(filtered, resourceOwner)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].InactiveDays == filtered[j].InactiveDays {
			if filtered[i].TotalReservedBytes == filtered[j].TotalReservedBytes {
				return filtered[i].User.GetId() < filtered[j].User.GetId()
			}
			return filtered[i].TotalReservedBytes > filtered[j].TotalReservedBytes
		}
		return filtered[i].InactiveDays > filtered[j].InactiveDays
	})

	summary := buildDormantResourceSummary(filtered)
	total := len(filtered)
	totalPages := 0
	if total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}
	paged := filtered[start:end]

	return connect.NewResponse(&superadminv1.ListDormantResourceOwnersResponse{
		Owners: paged,
		Pagination: &commonv1.Pagination{
			Page:       int32(page),
			PerPage:    int32(perPage),
			Total:      int32(total),
			TotalPages: int32(totalPages),
		},
		Summary: summary,
	}), nil
}

func loadDormantResourceRows(ctx context.Context) ([]dormantResourceQueryRow, error) {
	rows := make([]dormantResourceQueryRow, 0, 64)

	var vpsRows []dormantResourceQueryRow
	if err := database.DB.WithContext(ctx).
		Model(&database.VPSInstance{}).
		Select(`
			created_by AS user_id,
			organization_id,
			COUNT(*) AS vps_count,
			COALESCE(SUM(disk_bytes), 0) AS vps_disk_bytes,
			MAX(created_at) AS last_resource_created_at,
			MAX(updated_at) AS last_resource_updated_at
		`).
		Where("deleted_at IS NULL AND created_by <> ''").
		Group("created_by, organization_id").
		Scan(&vpsRows).Error; err != nil {
		return nil, fmt.Errorf("query vps resource audit rows: %w", err)
	}
	rows = append(rows, vpsRows...)

	var databaseRows []dormantResourceQueryRow
	if err := database.DB.WithContext(ctx).
		Model(&database.DatabaseInstance{}).
		Select(`
			created_by AS user_id,
			organization_id,
			COUNT(*) AS database_count,
			COALESCE(SUM(disk_bytes), 0) AS database_disk_bytes,
			MAX(created_at) AS last_resource_created_at,
			MAX(updated_at) AS last_resource_updated_at
		`).
		Where("deleted_at IS NULL AND created_by <> ''").
		Group("created_by, organization_id").
		Scan(&databaseRows).Error; err != nil {
		return nil, fmt.Errorf("query database resource audit rows: %w", err)
	}
	rows = append(rows, databaseRows...)

	var deploymentRows []dormantResourceQueryRow
	if err := database.DB.WithContext(ctx).
		Model(&database.Deployment{}).
		Select(`
			created_by AS user_id,
			organization_id,
			COUNT(*) AS deployment_count,
			COALESCE(SUM(storage_bytes), 0) AS deployment_storage_bytes,
			MAX(created_at) AS last_resource_created_at,
			MAX(COALESCE(last_deployed_at, created_at)) AS last_resource_updated_at
		`).
		Where("deleted_at IS NULL AND created_by <> ''").
		Group("created_by, organization_id").
		Scan(&deploymentRows).Error; err != nil {
		return nil, fmt.Errorf("query deployment resource audit rows: %w", err)
	}
	rows = append(rows, deploymentRows...)

	var gameServerRows []dormantResourceQueryRow
	if err := database.DB.WithContext(ctx).
		Model(&database.GameServer{}).
		Select(`
			created_by AS user_id,
			organization_id,
			COUNT(*) AS game_server_count,
			COALESCE(SUM(storage_bytes), 0) AS game_server_storage_bytes,
			MAX(created_at) AS last_resource_created_at,
			MAX(updated_at) AS last_resource_updated_at
		`).
		Where("deleted_at IS NULL AND created_by <> ''").
		Group("created_by, organization_id").
		Scan(&gameServerRows).Error; err != nil {
		return nil, fmt.Errorf("query game server resource audit rows: %w", err)
	}
	rows = append(rows, gameServerRows...)

	return rows, nil
}

func aggregateDormantResourceRows(rows []dormantResourceQueryRow) map[string]*dormantResourceOwnerAggregate {
	owners := make(map[string]*dormantResourceOwnerAggregate, len(rows))
	for _, row := range rows {
		if row.UserID == "" {
			continue
		}
		owner := owners[row.UserID]
		if owner == nil {
			owner = &dormantResourceOwnerAggregate{
				UserID:        row.UserID,
				Organizations: map[string]*dormantResourceOrganizationAggregate{},
			}
			owners[row.UserID] = owner
		}

		org := owner.Organizations[row.OrganizationID]
		if org == nil {
			org = &dormantResourceOrganizationAggregate{
				OrganizationID: row.OrganizationID,
			}
			owner.Organizations[row.OrganizationID] = org
		}

		vpsCount := int32(row.VPSCount)
		databaseCount := int32(row.DatabaseCount)
		deploymentCount := int32(row.DeploymentCount)
		gameServerCount := int32(row.GameServerCount)
		reservedBytes := row.VPSDiskBytes + row.DatabaseDiskBytes + row.DeploymentStorageBytes + row.GameServerStorageBytes

		owner.VPSCount += vpsCount
		owner.DatabaseCount += databaseCount
		owner.DeploymentCount += deploymentCount
		owner.GameServerCount += gameServerCount
		owner.TotalReservedBytes += reservedBytes

		org.VPSCount += vpsCount
		org.DatabaseCount += databaseCount
		org.DeploymentCount += deploymentCount
		org.GameServerCount += gameServerCount
		org.ReservedBytes += reservedBytes

		owner.LastResourceCreatedAt = maxTimePtr(owner.LastResourceCreatedAt, row.LastResourceCreatedAt)
		owner.LastResourceUpdatedAt = maxTimePtr(owner.LastResourceUpdatedAt, row.LastResourceUpdatedAt)
	}

	return owners
}

func loadOrganizationNames(ctx context.Context, organizationIDs []string) (map[string]string, error) {
	if len(organizationIDs) == 0 {
		return map[string]string{}, nil
	}

	var organizations []database.Organization
	if err := database.DB.WithContext(ctx).
		Select("id, name").
		Where("id IN ?", organizationIDs).
		Find(&organizations).Error; err != nil {
		return nil, err
	}

	names := make(map[string]string, len(organizations))
	for _, organization := range organizations {
		names[organization.ID] = organization.Name
	}
	return names, nil
}

func loadDormantResourceLastActivities(ctx context.Context, userIDs []string) map[string]*time.Time {
	activities := make(map[string]*time.Time, len(userIDs))
	if database.MetricsDB == nil || len(userIDs) == 0 {
		return activities
	}

	var rows []dormantResourceLastActivityRow
	if err := database.MetricsDB.WithContext(ctx).
		Model(&database.AuditLog{}).
		Select("user_id, MAX(created_at) AS last_activity_at").
		Where("user_id IN ?", userIDs).
		Group("user_id").
		Scan(&rows).Error; err != nil {
		logger.Warn("[SuperAdmin] Failed to load last audit activity for dormant resource owners: %v", err)
		return activities
	}

	for _, row := range rows {
		lastActivity := row.LastActivityAt
		activities[row.UserID] = &lastActivity
	}
	return activities
}

func buildDormantResourceUserInfo(userID string, profile *authv1.User) *superadminv1.UserInfo {
	userInfo := &superadminv1.UserInfo{
		Id: userID,
	}
	if profile == nil {
		return userInfo
	}

	userInfo.Id = profile.Id
	userInfo.Email = profile.Email
	userInfo.Name = profile.Name
	userInfo.PreferredUsername = profile.PreferredUsername
	userInfo.Locale = profile.Locale
	userInfo.EmailVerified = profile.EmailVerified
	if profile.AvatarUrl != "" {
		userInfo.AvatarUrl = &profile.AvatarUrl
	}
	if profile.CreatedAt != nil {
		userInfo.CreatedAt = profile.CreatedAt
	}
	if profile.UpdatedAt != nil {
		userInfo.UpdatedAt = profile.UpdatedAt
	}

	if userInfo.Email != "" {
		testUser := &authv1.User{Email: userInfo.Email}
		if auth.HasRole(testUser, auth.RoleSuperAdmin) {
			userInfo.Roles = []string{auth.RoleSuperAdmin}
		}
	}
	return userInfo
}

func chooseDormantOwnerLastActivity(profile *authv1.User, auditActivityAt, lastResourceUpdatedAt, lastResourceCreatedAt *time.Time) (*time.Time, string) {
	if auditActivityAt != nil {
		return auditActivityAt, "audit_log"
	}
	if profile != nil && profile.UpdatedAt != nil {
		updatedAt := profile.UpdatedAt.AsTime()
		if !updatedAt.IsZero() {
			return &updatedAt, "profile_updated"
		}
	}
	if profile != nil && profile.CreatedAt != nil {
		createdAt := profile.CreatedAt.AsTime()
		if !createdAt.IsZero() {
			return &createdAt, "profile_created"
		}
	}
	if lastResourceUpdatedAt != nil {
		return lastResourceUpdatedAt, "resource_updated"
	}
	if lastResourceCreatedAt != nil {
		return lastResourceCreatedAt, "resource_created"
	}
	return nil, "unknown"
}

func calculateInactiveDays(now time.Time, lastActivityAt *time.Time) int32 {
	if lastActivityAt == nil {
		return 0
	}
	inactiveFor := now.Sub(*lastActivityAt)
	if inactiveFor <= 0 {
		return 0
	}
	return int32(inactiveFor.Hours() / 24)
}

func matchesDormantResourceSearch(owner *superadminv1.DormantResourceOwner, searchTerm string) bool {
	searchTerm = strings.TrimSpace(strings.ToLower(searchTerm))
	if searchTerm == "" {
		return true
	}

	parts := []string{
		owner.GetUser().GetId(),
		owner.GetUser().GetEmail(),
		owner.GetUser().GetName(),
		owner.GetUser().GetPreferredUsername(),
	}
	for _, organization := range owner.GetOrganizations() {
		parts = append(parts, organization.GetOrganizationId(), organization.GetOrganizationName())
	}

	return strings.Contains(strings.ToLower(strings.Join(parts, " ")), searchTerm)
}

func buildDormantResourceSummary(owners []*superadminv1.DormantResourceOwner) *superadminv1.DormantResourceSummary {
	summary := &superadminv1.DormantResourceSummary{
		DormantUsers: int32(len(owners)),
	}
	for _, owner := range owners {
		if owner.GetVpsCount() > 0 {
			summary.UsersWithVps++
		}
		if owner.GetDatabaseCount() > 0 {
			summary.UsersWithDatabases++
		}
		if owner.GetDeploymentCount() > 0 {
			summary.UsersWithDeployments++
		}
		if owner.GetGameServerCount() > 0 {
			summary.UsersWithGameServers++
		}
		summary.TotalReservedBytes += owner.GetTotalReservedBytes()
	}
	return summary
}

func maxTimePtr(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.After(*current) {
		copy := *candidate
		return &copy
	}
	return current
}

func keysFromSet(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		if value == "" {
			continue
		}
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
}
