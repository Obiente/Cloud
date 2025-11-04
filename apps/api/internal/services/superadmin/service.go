package superadmin

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	superadminv1 "api/gen/proto/obiente/cloud/superadmin/v1"
	superadminv1connect "api/gen/proto/obiente/cloud/superadmin/v1/superadminv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/pricing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const overviewLimit = 50
const cacheTTL = 60 * time.Second

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

	logger.Debug("[SuperAdmin] Loading overview data...")

	counts, err := aggregateCounts()
	if err != nil {
		logger.Error("[SuperAdmin] Failed to aggregate counts: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load counts: %w", err))
	}

	organizations, err := loadOrganizationOverviews()
	if err != nil {
		logger.Error("[SuperAdmin] Failed to load organizations: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load organizations: %w", err))
	}

	invites, err := loadPendingInvites()
	if err != nil {
		logger.Error("[SuperAdmin] Failed to load invites: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load invites: %w", err))
	}

	deployments, err := loadDeploymentOverviews()
	if err != nil {
		logger.Error("[SuperAdmin] Failed to load deployments: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load deployments: %w", err))
	}

	usages, err := loadCurrentUsage()
	if err != nil {
		logger.Error("[SuperAdmin] Failed to load usage: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load usage: %w", err))
	}

	resp := &superadminv1.GetOverviewResponse{
		Counts:         counts,
		Organizations:  organizations,
		PendingInvites: invites,
		Deployments:    deployments,
		Usages:         usages,
	}

	logger.Debug("[SuperAdmin] Overview loaded successfully")
	return connect.NewResponse(resp), nil
}

// QueryDNS queries DNS for a specific domain
func (s *Service) QueryDNS(ctx context.Context, req *connect.Request[superadminv1.QueryDNSRequest]) (*connect.Response[superadminv1.QueryDNSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	domain := strings.TrimSpace(req.Msg.GetDomain())
	if domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("domain is required"))
	}

	recordType := strings.ToUpper(strings.TrimSpace(req.Msg.GetRecordType()))
	if recordType == "" {
		recordType = "A"
	}

	// Only support A records for now (as that's what the DNS server handles)
	if recordType != "A" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("only A record type is supported"))
	}

	// Extract deployment ID from domain (e.g., deploy-123.my.obiente.cloud -> deploy-123)
	if !strings.HasSuffix(strings.ToLower(domain), ".my.obiente.cloud") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("domain must be a *.my.obiente.cloud domain"))
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 3 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid domain format"))
	}

	deploymentID := parts[0]

	// Get Traefik IPs from environment
	traefikIPsEnv := os.Getenv("TRAEFIK_IPS")
	logger.Debug("[SuperAdmin] QueryDNS - TRAEFIK_IPS env value: %q", traefikIPsEnv)
	if traefikIPsEnv == "" {
		logger.Error("[SuperAdmin] QueryDNS - TRAEFIK_IPS environment variable is empty")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("TRAEFIK_IPS not configured"))
	}

	traefikIPMap, err := database.ParseTraefikIPsFromEnv(traefikIPsEnv)
	if err != nil {
		logger.Error("[SuperAdmin] QueryDNS - Failed to parse TRAEFIK_IPS: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to parse TRAEFIK_IPS: %w", err))
	}
	logger.Debug("[SuperAdmin] QueryDNS - Parsed TRAEFIK_IPS: %+v", traefikIPMap)

	// Query database for deployment location
	ips, err := database.GetDeploymentTraefikIP(deploymentID, traefikIPMap)
	if err != nil {
		return connect.NewResponse(&superadminv1.QueryDNSResponse{
			Domain:     domain,
			RecordType: recordType,
			Error:      err.Error(),
			Ttl:        int64(cacheTTL.Seconds()),
		}), nil
	}

	return connect.NewResponse(&superadminv1.QueryDNSResponse{
		Domain:     domain,
		RecordType: recordType,
		Records:    ips,
		Ttl:        int64(cacheTTL.Seconds()),
	}), nil
}

// ListDNSRecords lists all DNS records for deployments
func (s *Service) ListDNSRecords(ctx context.Context, req *connect.Request[superadminv1.ListDNSRecordsRequest]) (*connect.Response[superadminv1.ListDNSRecordsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Get Traefik IPs from environment
	traefikIPsEnv := os.Getenv("TRAEFIK_IPS")
	if traefikIPsEnv == "" {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("TRAEFIK_IPS not configured"))
	}

	traefikIPMap, err := database.ParseTraefikIPsFromEnv(traefikIPsEnv)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to parse TRAEFIK_IPS: %w", err))
	}

	// Build query
	query := database.DB.Table("deployments d").
		Select(`
			d.id as deployment_id,
			d.organization_id,
			d.name as deployment_name,
			d.status,
			dl.node_id,
			nm.region
		`).
		Joins("LEFT JOIN deployment_locations dl ON dl.deployment_id = d.id AND dl.status = 'running'").
		Joins("LEFT JOIN node_metadata nm ON nm.id = dl.node_id")

	// Apply filters
	if deploymentID := req.Msg.GetDeploymentId(); deploymentID != "" {
		query = query.Where("d.id = ?", deploymentID)
	}
	if orgID := req.Msg.GetOrganizationId(); orgID != "" {
		query = query.Where("d.organization_id = ?", orgID)
	}

	// Only get deployments that have locations (running)
	query = query.Where("dl.id IS NOT NULL")

	type dnsRecordRow struct {
		DeploymentID   string
		OrganizationID string
		DeploymentName string
		Status         int32  // Status is an integer in the database
		NodeID         *string
		Region         *string
	}

	var rows []dnsRecordRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query DNS records: %w", err))
	}

	records := make([]*superadminv1.DNSRecord, 0, len(rows))
	now := time.Now()

	for _, row := range rows {
		// Build domain
		domain := fmt.Sprintf("%s.my.obiente.cloud", row.DeploymentID)

		// Get IPs for this deployment
		var ips []string
		var region string
		if row.Region != nil && *row.Region != "" {
			region = *row.Region
			if regionIPs, ok := traefikIPMap[region]; ok {
				ips = regionIPs
			}
		}

		// Fallback: try to get IPs using the database function
		if len(ips) == 0 {
			if resolvedIPs, err := database.GetDeploymentTraefikIP(row.DeploymentID, traefikIPMap); err == nil {
				ips = resolvedIPs
				// If we got IPs from the fallback, also try to get region
				if region == "" {
					if deploymentRegion, err := database.GetDeploymentRegion(row.DeploymentID); err == nil {
						region = deploymentRegion
					}
				}
			}
		}

		// Send status as integer - frontend will convert it
		records = append(records, &superadminv1.DNSRecord{
			DeploymentId:   row.DeploymentID,
			OrganizationId: row.OrganizationID,
			DeploymentName: row.DeploymentName,
			Domain:         domain,
			IpAddresses:    ips,
			Region:         region,
			Status:         fmt.Sprintf("%d", row.Status), // Send as string representation of integer
			LastResolved:   timestamppb.New(now),
		})
	}

	return connect.NewResponse(&superadminv1.ListDNSRecordsResponse{
		Records: records,
	}), nil
}

// GetDNSConfig returns DNS server configuration
func (s *Service) GetDNSConfig(ctx context.Context, _ *connect.Request[superadminv1.GetDNSConfigRequest]) (*connect.Response[superadminv1.GetDNSConfigResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Get Traefik IPs from environment
	traefikIPsEnv := os.Getenv("TRAEFIK_IPS")
	logger.Debug("[SuperAdmin] TRAEFIK_IPS env value: %q", traefikIPsEnv)
	traefikIPMap, err := database.ParseTraefikIPsFromEnv(traefikIPsEnv)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to parse TRAEFIK_IPS: %v", err)
		// Don't fail, just return empty map
		traefikIPMap = make(map[string][]string)
	} else {
		logger.Debug("[SuperAdmin] Parsed TRAEFIK_IPS: %+v", traefikIPMap)
	}

	// Collect all unique IPs
	allIPs := make(map[string]struct{})
	traefikIPsByRegion := make(map[string]*superadminv1.TraefikIPs)
	for region, ips := range traefikIPMap {
		for _, ip := range ips {
			allIPs[ip] = struct{}{}
		}
		traefikIPsByRegion[region] = &superadminv1.TraefikIPs{
			Region: region,
			Ips:    ips,
		}
	}

	// Convert to slice
	traefikIPsList := make([]string, 0, len(allIPs))
	for ip := range allIPs {
		traefikIPsList = append(traefikIPsList, ip)
	}

	// Get DNS server IPs
	dnsIPsEnv := os.Getenv("DNS_IPS")
	var dnsIPs []string
	if dnsIPsEnv != "" {
		dnsIPs = strings.Split(dnsIPsEnv, ",")
		for i := range dnsIPs {
			dnsIPs[i] = strings.TrimSpace(dnsIPs[i])
		}
	}

	// Get DNS port
	dnsPort := os.Getenv("DNS_PORT")
	if dnsPort == "" {
		dnsPort = "53"
	}

	config := &superadminv1.DNSConfig{
		TraefikIps:          traefikIPsList,
		TraefikIpsByRegion:  traefikIPsByRegion,
		DnsServerIps:       dnsIPs,
		DnsPort:             dnsPort,
		CacheTtlSeconds:    int64(cacheTTL.Seconds()),
	}

	return connect.NewResponse(&superadminv1.GetDNSConfigResponse{
		Config: config,
	}), nil
}

// GetPricing returns current pricing information - public endpoint, no authentication required
func (s *Service) GetPricing(ctx context.Context, _ *connect.Request[superadminv1.GetPricingRequest]) (*connect.Response[superadminv1.GetPricingResponse], error) {
	pricingModel := pricing.GetPricing()
	
	return connect.NewResponse(&superadminv1.GetPricingResponse{
		CpuCostPerCoreSecond:     pricingModel.CPUCostPerCoreSecond,
		MemoryCostPerByteSecond:  pricingModel.MemoryCostPerByteSecond,
		BandwidthCostPerByte:      pricingModel.BandwidthCostPerByte,
		StorageCostPerByteMonth:  pricingModel.StorageCostPerByteMonth,
		PricingInfo:               pricingModel.GetPricingInfo(),
	}), nil
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
	// Check if metrics database is available
	if database.MetricsDB == nil {
		logger.Debug("[SuperAdmin] Metrics database not available, returning empty usage list")
		return []*superadminv1.OrganizationUsage{}, nil
	}

	now := time.Now().UTC()
	month := now.Format("2006-01")
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)
	
	// Calculate usage from deployment_usage_hourly (single source of truth)
	var rows []usageRow
	
	// Check if table exists first
	var tableExists bool
	if err := database.MetricsDB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'deployment_usage_hourly'
		)
	`).Scan(&tableExists).Error; err != nil {
		logger.Warn("[SuperAdmin] Failed to check if deployment_usage_hourly table exists: %v", err)
		return []*superadminv1.OrganizationUsage{}, nil
	}
	
	if !tableExists {
		logger.Debug("[SuperAdmin] deployment_usage_hourly table does not exist, returning empty usage list")
		return []*superadminv1.OrganizationUsage{}, nil
	}
	
	// Query usage from metrics database (without organization join since it's in main DB)
	if err := database.MetricsDB.Table("deployment_usage_hourly duh").
		Select(`
			duh.organization_id,
			'' AS organization_name,
			? AS month,
			COALESCE(CAST(SUM((duh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) AS cpu_core_seconds,
			COALESCE(SUM(duh.avg_memory_usage * 3600), 0) AS memory_byte_seconds,
			COALESCE(SUM(duh.bandwidth_rx_bytes), 0) AS bandwidth_rx_bytes,
			COALESCE(SUM(duh.bandwidth_tx_bytes), 0) AS bandwidth_tx_bytes,
			0 AS storage_bytes,
			0 AS deployments_active_peak
		`, month).
		Where("duh.hour >= ? AND duh.hour <= ?", monthStart, monthEnd).
		Group("duh.organization_id").
		Order("cpu_core_seconds DESC").
		Limit(overviewLimit).
		Scan(&rows).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to query deployment_usage_hourly: %v", err)
		return nil, fmt.Errorf("load usage: %w", err)
	}

	// Fetch organization names from main database
	orgIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		orgIDs = append(orgIDs, row.OrganizationID)
	}
	
	orgNames := make(map[string]string)
	if len(orgIDs) > 0 {
		type orgRow struct {
			ID   string
			Name string
		}
		var orgRows []orgRow
		if err := database.DB.Model(&database.Organization{}).
			Select("id, name").
			Where("id IN ?", orgIDs).
			Scan(&orgRows).Error; err == nil {
			for _, org := range orgRows {
				orgNames[org.ID] = org.Name
			}
		}
	}

	// Fetch storage bytes from main database
	storageByOrg := make(map[string]int64)
	if len(orgIDs) > 0 {
		type storageRow struct {
			OrganizationID string
			StorageBytes   int64
		}
		var storageRows []storageRow
		if err := database.DB.Model(&database.Deployment{}).
			Select("organization_id, COALESCE(SUM(storage_bytes), 0) AS storage_bytes").
			Where("organization_id IN ?", orgIDs).
			Group("organization_id").
			Scan(&storageRows).Error; err == nil {
			for _, s := range storageRows {
				storageByOrg[s.OrganizationID] = s.StorageBytes
			}
		}
	}

	items := make([]*superadminv1.OrganizationUsage, 0, len(rows))
	for _, row := range rows {
		orgName := orgNames[row.OrganizationID]
		if orgName == "" {
			orgName = row.OrganizationName
		}
		storageBytes := storageByOrg[row.OrganizationID]
		
		items = append(items, &superadminv1.OrganizationUsage{
			OrganizationId:        row.OrganizationID,
			OrganizationName:      orgName,
			Month:                 row.Month,
			CpuCoreSeconds:        row.CPUCoreSeconds,
			MemoryByteSeconds:     row.MemoryByteSeconds,
			BandwidthRxBytes:      row.BandwidthRxBytes,
			BandwidthTxBytes:      row.BandwidthTxBytes,
			StorageBytes:          storageBytes,
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