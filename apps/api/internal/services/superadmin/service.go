package superadmin

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	superadminv1 "api/gen/proto/obiente/cloud/superadmin/v1"
	superadminv1connect "api/gen/proto/obiente/cloud/superadmin/v1/superadminv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/pricing"
	"api/internal/stripe"

	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const overviewLimit = 50
const cacheTTL = 60 * time.Second

type Service struct {
	superadminv1connect.UnimplementedSuperadminServiceHandler
	stripeClient *stripe.Client
}

func NewService() superadminv1connect.SuperadminServiceHandler {
	stripeClient, _ := stripe.NewClient() // Stripe is optional for superadmin
	return &Service{
		stripeClient: stripeClient,
	}
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

	// Support A and SRV record types
	if recordType != "A" && recordType != "SRV" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("only A and SRV record types are supported"))
	}

	// Extract resource ID from domain
	if !strings.HasSuffix(strings.ToLower(domain), ".my.obiente.cloud") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("domain must be a *.my.obiente.cloud domain"))
	}

	parts := strings.Split(strings.ToLower(domain), ".")
	
	// Handle SRV queries: _minecraft._tcp.gameserver-123.my.obiente.cloud
	// Also supports: _minecraft._udp (Bedrock), _rust._udp
	if recordType == "SRV" {
		if len(parts) < 4 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid SRV domain format"))
		}
		
		service := parts[0]  // _minecraft, _rust, etc.
		protocol := parts[1]  // _tcp, _udp
		resourceID := parts[2] // gameserver-123
		
		if !strings.HasPrefix(resourceID, "gameserver-") {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid game server ID format"))
		}
		
		gameServerID := resourceID
		
		// Get game server type to validate SRV service matches
		gameType, err := database.GetGameServerType(gameServerID)
		if err != nil {
			return connect.NewResponse(&superadminv1.QueryDNSResponse{
				Domain:     domain,
				RecordType: recordType,
				Error:      err.Error(),
				Ttl:        int64(cacheTTL.Seconds()),
			}), nil
		}
		
		// Validate SRV service/protocol matches game type
		// GameType enum values: MINECRAFT = 1, MINECRAFT_JAVA = 2, MINECRAFT_BEDROCK = 3, RUST = 6
		isValid := false
		if service == "_minecraft" {
			if protocol == "_tcp" && (gameType == 1 || gameType == 2) {
				// Minecraft Java Edition uses TCP
				isValid = true
			} else if protocol == "_udp" && (gameType == 1 || gameType == 3) {
				// Minecraft Bedrock Edition uses UDP
				isValid = true
			}
		} else if service == "_rust" && protocol == "_udp" && gameType == 6 {
			// Rust uses UDP
			isValid = true
		}
		
		if !isValid {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported SRV service/protocol for this game type"))
		}
		
		// Get game server location
		_, port, err := database.GetGameServerLocation(gameServerID)
		if err != nil {
			return connect.NewResponse(&superadminv1.QueryDNSResponse{
				Domain:     domain,
				RecordType: recordType,
				Error:      err.Error(),
				Ttl:        int64(cacheTTL.Seconds()),
			}), nil
		}
		
		// Return SRV record format: priority weight port target
		// Target should be the A record hostname
		targetHostname := gameServerID + ".my.obiente.cloud"
		srvRecord := fmt.Sprintf("0 0 %d %s", port, targetHostname)
		return connect.NewResponse(&superadminv1.QueryDNSResponse{
			Domain:     domain,
			RecordType: recordType,
			Records:    []string{srvRecord},
			Ttl:        int64(cacheTTL.Seconds()),
		}), nil
	}

	// Handle A record queries: deploy-123.my.obiente.cloud or gameserver-123.my.obiente.cloud
	if len(parts) < 3 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid domain format"))
	}

	resourceID := parts[0]
	
	// Check if this is a game server
	if strings.HasPrefix(resourceID, "gameserver-") {
		// Get game server IP
		nodeIP, err := database.GetGameServerIP(resourceID)
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
			Records:    []string{nodeIP},
			Ttl:        int64(cacheTTL.Seconds()),
		}), nil
	}
	
	// Otherwise, treat as deployment
	deploymentID := resourceID

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

// ListDNSRecords lists all DNS records for deployments and game servers
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

	recordTypeFilter := strings.ToUpper(strings.TrimSpace(req.Msg.GetRecordType()))
	records := make([]*superadminv1.DNSRecord, 0)
	now := time.Now()

	// Fetch deployment A records if filter allows
	if recordTypeFilter == "" || recordTypeFilter == "A" {
		deploymentRecords, err := s.listDeploymentDNSRecords(req, traefikIPMap, now)
		if err != nil {
			return nil, err
		}
		records = append(records, deploymentRecords...)
	}

	// Fetch game server SRV records if filter allows
	if recordTypeFilter == "" || recordTypeFilter == "SRV" {
		gameServerRecords, err := s.listGameServerDNSRecords(req, now)
		if err != nil {
			return nil, err
		}
		records = append(records, gameServerRecords...)
	}

	return connect.NewResponse(&superadminv1.ListDNSRecordsResponse{
		Records: records,
	}), nil
}

// listDeploymentDNSRecords lists DNS A records for deployments
func (s *Service) listDeploymentDNSRecords(req *connect.Request[superadminv1.ListDNSRecordsRequest], traefikIPMap map[string][]string, now time.Time) ([]*superadminv1.DNSRecord, error) {
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
		Status         int32
		NodeID         *string
		Region         *string
	}

	var rows []dnsRecordRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query DNS records: %w", err))
	}

	records := make([]*superadminv1.DNSRecord, 0, len(rows))

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
				if region == "" {
					if deploymentRegion, err := database.GetDeploymentRegion(row.DeploymentID); err == nil {
						region = deploymentRegion
					}
				}
			}
		}

		records = append(records, &superadminv1.DNSRecord{
			RecordType:     "A",
			DeploymentId:   row.DeploymentID,
			OrganizationId: row.OrganizationID,
			DeploymentName: row.DeploymentName,
			Domain:         domain,
			IpAddresses:    ips,
			Region:         region,
			Status:         fmt.Sprintf("%d", row.Status),
			LastResolved:   timestamppb.New(now),
		})
	}

	return records, nil
}

// listGameServerDNSRecords lists DNS SRV records for game servers
func (s *Service) listGameServerDNSRecords(req *connect.Request[superadminv1.ListDNSRecordsRequest], now time.Time) ([]*superadminv1.DNSRecord, error) {
	// Build query for game servers
	query := database.DB.Table("game_servers gs").
		Select(`
			gs.id as game_server_id,
			gs.organization_id,
			gs.name as game_server_name,
			gs.status,
			gs.game_type,
			gsl.node_id,
			gsl.port,
			gsl.node_ip,
			nm.region
		`).
		Joins("LEFT JOIN game_server_locations gsl ON gsl.game_server_id = gs.id AND gsl.status = 'running'").
		Joins("LEFT JOIN node_metadata nm ON nm.id = gsl.node_id")

	// Apply filters
	if orgID := req.Msg.GetOrganizationId(); orgID != "" {
		query = query.Where("gs.organization_id = ?", orgID)
	}

	// Only get game servers that have locations (running)
	query = query.Where("gsl.id IS NOT NULL")

	type gameServerRecordRow struct {
		GameServerID   string
		OrganizationID string
		GameServerName string
		Status         int32
		GameType       int32
		NodeID         *string
		Port           int32
		NodeIP         string
		Region         *string
	}

	var rows []gameServerRecordRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query game server DNS records: %w", err))
	}

	records := make([]*superadminv1.DNSRecord, 0, len(rows))

	for _, row := range rows {
		// Get game server location (IP and port) from database
		// This ensures we always have the correct values even if row.NodeIP is empty
		targetIP, port, err := database.GetGameServerLocation(row.GameServerID)
		if err != nil {
			// If we can't get location, skip this record or use defaults
			logger.Warn("[SuperAdmin] Failed to get location for game server %s: %v", row.GameServerID, err)
			continue
		}

		// Use the port from database, fallback to row.Port if database returns 0
		if port == 0 {
			port = row.Port
		}

		region := ""
		if row.Region != nil {
			region = *row.Region
		}

		// A record for all game servers
		// Format: gameserver-123.my.obiente.cloud
		aRecordDomain := fmt.Sprintf("gameserver-%s.my.obiente.cloud", row.GameServerID)
		records = append(records, &superadminv1.DNSRecord{
			RecordType:     "A",
			GameServerId:   row.GameServerID,
			OrganizationId: row.OrganizationID,
			GameServerName: row.GameServerName,
			Domain:         aRecordDomain,
			Target:         targetIP, // IP address for A record
			Port:           0, // A records don't have ports
			Region:         region,
			Status:         fmt.Sprintf("%d", row.Status),
			LastResolved:   timestamppb.New(now),
		})

		// SRV records for games that support them
		// GameType enum values: MINECRAFT = 1, MINECRAFT_JAVA = 2, MINECRAFT_BEDROCK = 3, RUST = 6
		if row.GameType == 1 || row.GameType == 2 {
			// Minecraft Java Edition - TCP SRV record
			// Format: _minecraft._tcp.gameserver-123.my.obiente.cloud
			srvDomain := fmt.Sprintf("_minecraft._tcp.gameserver-%s.my.obiente.cloud", row.GameServerID)
			records = append(records, &superadminv1.DNSRecord{
				RecordType:     "SRV",
				GameServerId:   row.GameServerID,
				OrganizationId: row.OrganizationID,
				GameServerName: row.GameServerName,
				Domain:         srvDomain,
				Target:         aRecordDomain, // A record domain for SRV target
				Port:           port,
				Region:         region,
				Status:         fmt.Sprintf("%d", row.Status),
				LastResolved:   timestamppb.New(now),
			})
		}
		
		if row.GameType == 1 || row.GameType == 3 {
			// Minecraft Bedrock Edition - UDP SRV record
			// Format: _minecraft._udp.gameserver-123.my.obiente.cloud
			srvDomain := fmt.Sprintf("_minecraft._udp.gameserver-%s.my.obiente.cloud", row.GameServerID)
			records = append(records, &superadminv1.DNSRecord{
				RecordType:     "SRV",
				GameServerId:   row.GameServerID,
				OrganizationId: row.OrganizationID,
				GameServerName: row.GameServerName,
				Domain:         srvDomain,
				Target:         aRecordDomain, // A record domain for SRV target
				Port:           port,
				Region:         region,
				Status:         fmt.Sprintf("%d", row.Status),
				LastResolved:   timestamppb.New(now),
			})
		}
		
		if row.GameType == 6 {
			// Rust - UDP SRV record
			// Format: _rust._udp.gameserver-123.my.obiente.cloud
			srvDomain := fmt.Sprintf("_rust._udp.gameserver-%s.my.obiente.cloud", row.GameServerID)
			records = append(records, &superadminv1.DNSRecord{
				RecordType:     "SRV",
				GameServerId:   row.GameServerID,
				OrganizationId: row.OrganizationID,
				GameServerName: row.GameServerName,
				Domain:         srvDomain,
				Target:         aRecordDomain, // A record domain for SRV target
				Port:           port,
				Region:         region,
				Status:         fmt.Sprintf("%d", row.Status),
				LastResolved:   timestamppb.New(now),
			})
		}
	}

	return records, nil
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
		dnsIPsSplit := strings.Split(dnsIPsEnv, ",")
		for _, ip := range dnsIPsSplit {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			// Validate IP address
			if net.ParseIP(ip) != nil {
				dnsIPs = append(dnsIPs, ip)
			} else {
				logger.Warn("[SuperAdmin] Invalid DNS IP address in DNS_IPS: %q (skipping)", ip)
			}
		}
		if len(dnsIPs) == 0 && dnsIPsEnv != "" {
			logger.Warn("[SuperAdmin] DNS_IPS is set but contains no valid IP addresses")
		}
	} else {
		logger.Debug("[SuperAdmin] DNS_IPS environment variable is not set")
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

// CreateDNSDelegationAPIKey creates a new API key for DNS delegation
// Superadmins can create keys for any organization
// Regular users can create keys for their organization if they have an active subscription
func (s *Service) CreateDNSDelegationAPIKey(ctx context.Context, req *connect.Request[superadminv1.CreateDNSDelegationAPIKeyRequest]) (*connect.Response[superadminv1.CreateDNSDelegationAPIKeyResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)

	description := strings.TrimSpace(req.Msg.GetDescription())
	if description == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("description is required"))
	}

	sourceAPI := ""
	if req.Msg.SourceApi != nil {
		sourceAPI = strings.TrimSpace(*req.Msg.SourceApi)
	}

	var organizationID string
	var stripeSubscriptionID *string

	// Get organization ID from request if provided
	if orgID := req.Msg.GetOrganizationId(); orgID != "" {
		organizationID = strings.TrimSpace(orgID)
		logger.Debug("[SuperAdmin] Organization ID from request: %s", organizationID)
		
		// Verify user is a member of this organization (for non-superadmins)
		if !isSuperAdmin {
			var member database.OrganizationMember
			if err := database.DB.Where("organization_id = ? AND user_id = ? AND status = ?", organizationID, user.Id, "active").First(&member).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you are not a member of this organization"))
				}
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization membership: %w", err))
			}
		}
	} else {
		logger.Debug("[SuperAdmin] No organization ID in request")
	}

	// For non-superadmins, require organization membership and active subscription
	if !isSuperAdmin {
		// If organizationId wasn't provided, get it from user's memberships
		if organizationID == "" {
			var member database.OrganizationMember
			if err := database.DB.Where("user_id = ? AND status = ?", user.Id, "active").
				Order("joined_at DESC").
				First(&member).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a member of an organization"))
				}
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization membership: %w", err))
			}
			organizationID = member.OrganizationID
		}

		// Check if organization has an active subscription
		// First try database (API key method), then check Stripe directly
		hasSubscription, subscriptionID, err := database.HasActiveDNSDelegationSubscription(organizationID)
		if err != nil {
			logger.Error("[SuperAdmin] Failed to check subscription: %v", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check subscription: %w", err))
		}
		
		// If not found in database, check Stripe directly (webhook might not have processed yet)
		if !hasSubscription || subscriptionID == "" {
			// Get billing account to find Stripe customer ID
			var billingAccount database.BillingAccount
			if err := database.DB.Where("organization_id = ?", organizationID).First(&billingAccount).Error; err == nil {
				if billingAccount.StripeCustomerID != nil && *billingAccount.StripeCustomerID != "" && s.stripeClient != nil {
					// Check Stripe for active DNS delegation subscription
					sub, err := s.stripeClient.FindDNSDelegationSubscription(ctx, *billingAccount.StripeCustomerID)
					if err == nil && sub != nil {
						hasSubscription = true
						subscriptionID = sub.ID
					}
				}
			}
		}
		
		if !hasSubscription {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("your organization must have an active DNS delegation subscription. Subscribe at $2/month to enable DNS delegation"))
		}

		// Check if organization already has an active API key
		existingKey, err := database.GetActiveDNSDelegationAPIKeyForOrganization(organizationID)
		if err == nil && existingKey != nil {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("your organization already has an active API key. Revoke the existing key first to create a new one"))
		}

		stripeSubscriptionID = &subscriptionID
	} else if organizationID != "" {
		// For superadmins, if organizationId is provided, try to get subscription ID for linking
		hasSubscription, subscriptionID, err := database.HasActiveDNSDelegationSubscription(organizationID)
		if err == nil && hasSubscription && subscriptionID != "" {
			stripeSubscriptionID = &subscriptionID
		} else {
			// If not found in database, check Stripe directly (webhook might not have processed yet)
			var billingAccount database.BillingAccount
			if err := database.DB.Where("organization_id = ?", organizationID).First(&billingAccount).Error; err == nil {
				if billingAccount.StripeCustomerID != nil && *billingAccount.StripeCustomerID != "" && s.stripeClient != nil {
					// Check Stripe for active DNS delegation subscription
					sub, err := s.stripeClient.FindDNSDelegationSubscription(ctx, *billingAccount.StripeCustomerID)
					if err == nil && sub != nil {
						stripeSubscriptionID = &sub.ID
						logger.Debug("[SuperAdmin] Found subscription in Stripe for superadmin: %s", sub.ID)
					}
				}
			}
		}
	}

	apiKey, err := database.CreateDNSDelegationAPIKey(description, sourceAPI, organizationID, stripeSubscriptionID)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to create DNS delegation API key: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create API key: %w", err))
	}

	logger.Info("[SuperAdmin] Created DNS delegation API key for: %s (source: %s, org: %s, subscription: %v)", description, sourceAPI, organizationID, stripeSubscriptionID != nil && *stripeSubscriptionID != "")

	return connect.NewResponse(&superadminv1.CreateDNSDelegationAPIKeyResponse{
		ApiKey:     apiKey,
		Message:    "API key created successfully. Save this key securely - it will not be shown again.",
		Description: description,
	}), nil
}

// RevokeDNSDelegationAPIKey revokes a DNS delegation API key (superadmin only)
func (s *Service) RevokeDNSDelegationAPIKey(ctx context.Context, req *connect.Request[superadminv1.RevokeDNSDelegationAPIKeyRequest]) (*connect.Response[superadminv1.RevokeDNSDelegationAPIKeyResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	apiKey := strings.TrimSpace(req.Msg.GetApiKey())
	if apiKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("api_key is required"))
	}

	keyHash := database.HashAPIKeyForDelegation(apiKey)
	if err := database.RevokeDNSDelegationAPIKey(keyHash); err != nil {
		logger.Error("[SuperAdmin] Failed to revoke DNS delegation API key: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke API key: %w", err))
	}

	logger.Info("[SuperAdmin] Revoked DNS delegation API key")

	return connect.NewResponse(&superadminv1.RevokeDNSDelegationAPIKeyResponse{
		Success: true,
		Message: "API key revoked successfully",
	}), nil
}

// RevokeDNSDelegationAPIKeyForOrganization revokes the DNS delegation API key for an organization
func (s *Service) RevokeDNSDelegationAPIKeyForOrganization(ctx context.Context, req *connect.Request[superadminv1.RevokeDNSDelegationAPIKeyForOrganizationRequest]) (*connect.Response[superadminv1.RevokeDNSDelegationAPIKeyForOrganizationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Check if user is superadmin or member of the organization
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ? AND status = ?", orgID, user.Id, "active").First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can revoke API keys
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	// Get active API key for organization
	apiKey, err := database.GetActiveDNSDelegationAPIKeyForOrganization(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no active API key found for this organization"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get API key: %w", err))
	}

	// Revoke the API key
	if err := database.RevokeDNSDelegationAPIKey(apiKey.KeyHash); err != nil {
		logger.Error("[SuperAdmin] Failed to revoke DNS delegation API key for organization: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke API key: %w", err))
	}

	logger.Info("[SuperAdmin] Revoked DNS delegation API key for organization %s", orgID)

	return connect.NewResponse(&superadminv1.RevokeDNSDelegationAPIKeyForOrganizationResponse{
		Success: true,
		Message: "API key revoked successfully",
	}), nil
}

// ListDNSDelegationAPIKeys lists DNS delegation API keys (superadmin only)
func (s *Service) ListDNSDelegationAPIKeys(ctx context.Context, req *connect.Request[superadminv1.ListDNSDelegationAPIKeysRequest]) (*connect.Response[superadminv1.ListDNSDelegationAPIKeysResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	organizationID := ""
	if req.Msg.OrganizationId != nil && *req.Msg.OrganizationId != "" {
		organizationID = strings.TrimSpace(*req.Msg.OrganizationId)
	}

	keys, err := database.ListDNSDelegationAPIKeys(organizationID)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to list DNS delegation API keys: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list API keys: %w", err))
	}

	apiKeys := make([]*superadminv1.DNSDelegationAPIKeyInfo, 0, len(keys))
	for _, key := range keys {
		info := &superadminv1.DNSDelegationAPIKeyInfo{
			Id:                  key.ID,
			Description:         key.Description,
			SourceApi:           key.SourceAPI,
			OrganizationId:      key.OrganizationID,
			IsActive:            key.IsActive,
			CreatedAt:           timestamppb.New(key.CreatedAt),
		}
		
		if key.RevokedAt != nil {
			info.RevokedAt = timestamppb.New(*key.RevokedAt)
		}
		
		if key.StripeSubscriptionID != nil {
			info.StripeSubscriptionId = *key.StripeSubscriptionID
		}
		
		apiKeys = append(apiKeys, info)
	}

	return connect.NewResponse(&superadminv1.ListDNSDelegationAPIKeysResponse{
		ApiKeys: apiKeys,
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
	
	// Query usage from metrics database - combine deployment and game server usage
	// Check if game_server_usage_hourly table exists
	var gameServerTableExists bool
	database.MetricsDB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'game_server_usage_hourly'
		)
	`).Scan(&gameServerTableExists)
	
	// Query deployment usage
	var deploymentRows []usageRow
	if err := database.MetricsDB.Table("deployment_usage_hourly duh").
		Select(`
			duh.organization_id,
			'' AS organization_name,
			? AS month,
			COALESCE(CAST(SUM((duh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) AS cpu_core_seconds,
			COALESCE(CAST(SUM(duh.avg_memory_usage * 3600) AS BIGINT), 0) AS memory_byte_seconds,
			COALESCE(SUM(duh.bandwidth_rx_bytes), 0) AS bandwidth_rx_bytes,
			COALESCE(SUM(duh.bandwidth_tx_bytes), 0) AS bandwidth_tx_bytes,
			0 AS storage_bytes,
			0 AS deployments_active_peak
		`, month).
		Where("duh.hour >= ? AND duh.hour <= ?", monthStart, monthEnd).
		Group("duh.organization_id").
		Scan(&deploymentRows).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to query deployment_usage_hourly: %v", err)
		return nil, fmt.Errorf("load usage: %w", err)
	}
	
	// Query game server usage if table exists
	var gameServerRows []usageRow
	if gameServerTableExists {
		if err := database.MetricsDB.Table("game_server_usage_hourly gsuh").
			Select(`
				gsuh.organization_id,
				'' AS organization_name,
				? AS month,
				COALESCE(CAST(SUM((gsuh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) AS cpu_core_seconds,
				COALESCE(CAST(SUM(gsuh.avg_memory_usage * 3600) AS BIGINT), 0) AS memory_byte_seconds,
				COALESCE(SUM(gsuh.bandwidth_rx_bytes), 0) AS bandwidth_rx_bytes,
				COALESCE(SUM(gsuh.bandwidth_tx_bytes), 0) AS bandwidth_tx_bytes,
				0 AS storage_bytes,
				0 AS deployments_active_peak
			`, month).
			Where("gsuh.hour >= ? AND gsuh.hour <= ?", monthStart, monthEnd).
			Group("gsuh.organization_id").
			Scan(&gameServerRows).Error; err != nil {
			logger.Warn("[SuperAdmin] Failed to query game_server_usage_hourly: %v", err)
			// Continue with deployment data only
		}
	}
	
	// Combine deployment and game server usage by organization
	usageMap := make(map[string]*usageRow)
	for _, row := range deploymentRows {
		usageMap[row.OrganizationID] = &row
	}
	
	// Add game server usage to existing organizations or create new entries
	for _, row := range gameServerRows {
		if existing, ok := usageMap[row.OrganizationID]; ok {
			// Add game server usage to existing deployment usage
			existing.CPUCoreSeconds += row.CPUCoreSeconds
			existing.MemoryByteSeconds += row.MemoryByteSeconds
			existing.BandwidthRxBytes += row.BandwidthRxBytes
			existing.BandwidthTxBytes += row.BandwidthTxBytes
		} else {
			// Create new entry for organization that only has game servers
			usageMap[row.OrganizationID] = &row
		}
	}
	
	// Convert map to slice
	var rows []usageRow
	for _, row := range usageMap {
		rows = append(rows, *row)
	}
	
	// Sort by CPU core seconds descending
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CPUCoreSeconds > rows[j].CPUCoreSeconds
	})
	
	// Limit results
	if len(rows) > overviewLimit {
		rows = rows[:overviewLimit]
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