package superadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/pricing"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"
	"github.com/obiente/cloud/apps/shared/pkg/stripe"
	vpsorch "github.com/obiente/cloud/apps/vps-service/orchestrator"
	vpsservice "github.com/obiente/cloud/apps/vps-service/pkg/service"

	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	billingv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/billing/v1"
	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"
	superadminv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1/superadminv1connect"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
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

	// Get commit hashes and messages from environment variables (set at build time)
	apiCommit := os.Getenv("API_COMMIT")
	dashboardCommit := os.Getenv("DASHBOARD_COMMIT")
	apiCommitMessage := os.Getenv("API_COMMIT_MESSAGE")
	dashboardCommitMessage := os.Getenv("DASHBOARD_COMMIT_MESSAGE")

	resp := &superadminv1.GetOverviewResponse{
		Counts:         counts,
		Organizations:  organizations,
		PendingInvites: invites,
		Deployments:    deployments,
		Usages:         usages,
	}
	if apiCommit != "" {
		resp.ApiCommit = proto.String(apiCommit)
	}
	if dashboardCommit != "" {
		resp.DashboardCommit = proto.String(dashboardCommit)
	}
	if apiCommitMessage != "" {
		resp.ApiCommitMessage = proto.String(apiCommitMessage)
	}
	if dashboardCommitMessage != "" {
		resp.DashboardCommitMessage = proto.String(dashboardCommitMessage)
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
	
	// Handle SRV queries: _minecraft._tcp.gs-123.my.obiente.cloud
	// Also supports: _minecraft._udp (Bedrock), _rust._udp
	if recordType == "SRV" {
		if len(parts) < 4 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid SRV domain format"))
		}
		
		service := parts[0]  // _minecraft, _rust, etc.
		protocol := parts[1]  // _tcp, _udp
		gameServerID := parts[2] // gs-123
		
		if !strings.HasPrefix(gameServerID, "gs-") {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid game server ID format"))
		}
		
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

	// Handle A record queries: deploy-123.my.obiente.cloud or gs-123.my.obiente.cloud
	if len(parts) < 3 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid domain format"))
	}

	resourceID := parts[0]
	
	// Check if this is a game server (starts with gs-)
	if strings.HasPrefix(resourceID, "gs-") {
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
		
		// First, try to get IPs from the region in the query result
		if row.Region != nil && *row.Region != "" {
			region = *row.Region
			if regionIPs, ok := traefikIPMap[region]; ok && len(regionIPs) > 0 {
				ips = regionIPs
			}
		}

		// If no IPs from region, try to get IPs using the database function
		// This handles cases where the region wasn't in the JOIN or the region name doesn't match
		if len(ips) == 0 {
			if resolvedIPs, err := database.GetDeploymentTraefikIP(row.DeploymentID, traefikIPMap); err == nil && len(resolvedIPs) > 0 {
				ips = resolvedIPs
				// Update region if we got it from the function
				if region == "" {
					if deploymentRegion, err := database.GetDeploymentRegion(row.DeploymentID); err == nil {
						region = deploymentRegion
					}
				}
			} else {
				logger.Warn("[SuperAdmin] Failed to get Traefik IPs for deployment %s: %v", row.DeploymentID, err)
			}
		}

		// Final fallback: if still no IPs, try to use any available Traefik IPs
		// This handles cases where nodes don't have regions configured
		if len(ips) == 0 {
			// Try "default" region first
			if defaultIPs, ok := traefikIPMap["default"]; ok && len(defaultIPs) > 0 {
				ips = defaultIPs
				if region == "" {
					region = "default"
				}
			} else {
				// Use the first available region's IPs as fallback
				for reg, regIPs := range traefikIPMap {
					if len(regIPs) > 0 {
						ips = regIPs
						if region == "" {
							region = reg
						}
						logger.Debug("[SuperAdmin] Using fallback Traefik IPs from region %s for deployment %s", reg, row.DeploymentID)
						break
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
		// Format: gs-123.my.obiente.cloud
		aRecordDomain := fmt.Sprintf("%s.my.obiente.cloud", row.GameServerID)
		ipAddresses := []string{}
		if targetIP != "" {
			ipAddresses = []string{targetIP}
		}
		records = append(records, &superadminv1.DNSRecord{
			RecordType:     "A",
			GameServerId:   row.GameServerID,
			OrganizationId: row.OrganizationID,
			GameServerName: row.GameServerName,
			Domain:         aRecordDomain,
			IpAddresses:    ipAddresses, // IP addresses for A record (frontend displays this)
			Target:         targetIP,     // Also set target for consistency
			Port:           0,            // A records don't have ports
			Region:         region,
			Status:         fmt.Sprintf("%d", row.Status),
			LastResolved:   timestamppb.New(now),
		})

		// SRV records for games that support them
		// GameType enum values: MINECRAFT = 1, MINECRAFT_JAVA = 2, MINECRAFT_BEDROCK = 3, RUST = 6
		if row.GameType == 1 || row.GameType == 2 {
			// Minecraft Java Edition - TCP SRV record
			// Format: _minecraft._tcp.gs-123.my.obiente.cloud
			srvDomain := fmt.Sprintf("_minecraft._tcp.%s.my.obiente.cloud", row.GameServerID)
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
			// Format: _minecraft._udp.gs-123.my.obiente.cloud
			srvDomain := fmt.Sprintf("_minecraft._udp.%s.my.obiente.cloud", row.GameServerID)
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
			// Format: _rust._udp.gs-123.my.obiente.cloud
			srvDomain := fmt.Sprintf("_rust._udp.%s.my.obiente.cloud", row.GameServerID)
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

// ListDelegatedDNSRecords lists delegated DNS records with optional filters
func (s *Service) ListDelegatedDNSRecords(ctx context.Context, req *connect.Request[superadminv1.ListDelegatedDNSRecordsRequest]) (*connect.Response[superadminv1.ListDelegatedDNSRecordsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	
	// Non-superadmins can only see their own organization's records
	var organizationID string
	if !isSuperAdmin {
		// Get user's organization memberships
		var memberships []database.OrganizationMember
		if err := database.DB.Where("user_id = ? AND status = ?", user.Id, "active").Find(&memberships).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get user memberships: %w", err))
		}
		
		if len(memberships) == 0 {
			return connect.NewResponse(&superadminv1.ListDelegatedDNSRecordsResponse{
				Records: []*superadminv1.DelegatedDNSRecord{},
			}), nil
		}
		
		// If user specified an organization ID, verify they're a member
		if reqOrgID := req.Msg.GetOrganizationId(); reqOrgID != "" {
			hasAccess := false
			for _, m := range memberships {
				if m.OrganizationID == reqOrgID {
					hasAccess = true
					break
				}
			}
			if !hasAccess {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("access denied to organization"))
			}
			organizationID = reqOrgID
		} else {
			// Use first organization (or could return all)
			organizationID = memberships[0].OrganizationID
		}
	} else {
		// Superadmins can filter by any organization
		if reqOrgID := req.Msg.GetOrganizationId(); reqOrgID != "" {
			organizationID = reqOrgID
		}
	}

	// Get filters
	apiKeyID := req.Msg.GetApiKeyId()
	recordType := req.Msg.GetRecordType()

	// Query delegated DNS records
	dbRecords, err := database.ListDelegatedDNSRecords(organizationID, apiKeyID, recordType)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to list delegated DNS records: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list delegated DNS records: %w", err))
	}

	// Convert to proto records
	records := make([]*superadminv1.DelegatedDNSRecord, 0, len(dbRecords))
	for _, dbRecord := range dbRecords {
		// Parse records JSON
		var recordValues []string
		if err := json.Unmarshal([]byte(dbRecord.Records), &recordValues); err != nil {
			logger.Warn("[SuperAdmin] Failed to parse records JSON for %s: %v", dbRecord.Domain, err)
			continue
		}

		records = append(records, &superadminv1.DelegatedDNSRecord{
			Id:             dbRecord.ID,
			Domain:         dbRecord.Domain,
			RecordType:     dbRecord.RecordType,
			Records:        recordValues,
			SourceApi:      dbRecord.SourceAPI,
			ApiKeyId:       dbRecord.APIKeyID,
			OrganizationId: dbRecord.OrganizationID,
			Ttl:            dbRecord.TTL,
			ExpiresAt:      timestamppb.New(dbRecord.ExpiresAt),
			LastUpdated:    timestamppb.New(dbRecord.LastUpdated),
			CreatedAt:      timestamppb.New(dbRecord.CreatedAt),
		})
	}

	return connect.NewResponse(&superadminv1.ListDelegatedDNSRecordsResponse{
		Records: records,
	}), nil
}

// HasDelegatedDNS checks if the current user has delegated DNS
func (s *Service) HasDelegatedDNS(ctx context.Context, _ *connect.Request[superadminv1.HasDelegatedDNSRequest]) (*connect.Response[superadminv1.HasDelegatedDNSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	// Get user's organization memberships
	var memberships []database.OrganizationMember
	if err := database.DB.Where("user_id = ? AND status = ?", user.Id, "active").Find(&memberships).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get user memberships: %w", err))
	}

	// Check each organization for active DNS delegation API key
	for _, membership := range memberships {
		apiKey, err := database.GetActiveDNSDelegationAPIKeyForOrganization(membership.OrganizationID)
		if err == nil && apiKey != nil {
			return connect.NewResponse(&superadminv1.HasDelegatedDNSResponse{
				HasDelegatedDns: true,
				OrganizationId:   membership.OrganizationID,
				ApiKeyId:         apiKey.ID,
			}), nil
		}
	}

	return connect.NewResponse(&superadminv1.HasDelegatedDNSResponse{
		HasDelegatedDns: false,
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
	OwnerID         *string
	OwnerName       *string
}

func loadOrganizationOverviews() ([]*superadminv1.OrganizationOverview, error) {
	var rows []organizationOverviewRow
	if err := database.DB.Table("organizations o").
		Select(`o.id, o.name, o.slug, o.plan, o.status, o.domain, o.created_at,
			SUM(CASE WHEN m.status = 'active' THEN 1 ELSE 0 END) AS member_count,
			SUM(CASE WHEN m.status = 'invited' THEN 1 ELSE 0 END) AS invite_count,
			COUNT(d.id) AS deployment_count,
			owner_m.user_id AS owner_id,
			NULL AS owner_name`).
		Joins("LEFT JOIN organization_members m ON m.organization_id = o.id").
		Joins("LEFT JOIN deployments d ON d.organization_id = o.id").
		Joins("LEFT JOIN organization_members owner_m ON owner_m.organization_id = o.id AND owner_m.role = 'owner' AND owner_m.status = 'active'").
		Group("o.id, o.name, o.slug, o.plan, o.status, o.domain, o.created_at, owner_m.user_id").
		Order("o.created_at DESC").
		Limit(overviewLimit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load organizations: %w", err)
	}

	// Resolve owner names
	resolver := organizations.GetUserProfileResolver()
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
		if r.OwnerID != nil && *r.OwnerID != "" {
			item.OwnerId = r.OwnerID
			// Try to resolve owner name
			if resolver != nil && resolver.IsConfigured() {
				if profile, err := resolver.Resolve(context.Background(), *r.OwnerID); err == nil && profile != nil {
					ownerName := profile.Name
					if ownerName == "" {
						ownerName = profile.Email
					}
					if ownerName != "" {
						item.OwnerName = &ownerName
					}
				}
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func loadPendingInvites() ([]*superadminv1.SuperadminPendingInvite, error) {
	var rows []database.OrganizationMember
	if err := database.DB.Where("status = ?", "invited").Order("joined_at DESC").Limit(overviewLimit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load invites: %w", err)
	}

	items := make([]*superadminv1.SuperadminPendingInvite, 0, len(rows))
	for _, row := range rows {
		email := strings.TrimPrefix(row.UserID, "pending:")
		items = append(items, &superadminv1.SuperadminPendingInvite{
			Id:             row.ID,
			OrganizationId: row.OrganizationID,
			Email:          email,
			Role:           row.Role,
			InvitedAt:      toTimestamp(row.JoinedAt),
		})
	}
	return items, nil
}

type deploymentOverviewRow struct {
	database.Deployment
	OrganizationName *string
}

func loadDeploymentOverviews() ([]*superadminv1.DeploymentOverview, error) {
	var rows []deploymentOverviewRow
	if err := database.DB.Table("deployments d").
		Select("d.*, o.name AS organization_name").
		Joins("LEFT JOIN organizations o ON o.id = d.organization_id").
		Order("d.created_at DESC").
		Limit(overviewLimit).
		Scan(&rows).Error; err != nil {
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
		if row.OrganizationName != nil && *row.OrganizationName != "" {
			item.OrganizationName = row.OrganizationName
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

// GetAbuseDetection returns suspicious organizations and activities for abuse detection
func (s *Service) GetAbuseDetection(ctx context.Context, _ *connect.Request[superadminv1.GetAbuseDetectionRequest]) (*connect.Response[superadminv1.GetAbuseDetectionResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Delegate to the abuse detection module
	result, err := DetectAbuse(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to detect abuse: %w", err))
	}

	return connect.NewResponse(result), nil
}

// GetIncomeOverview returns billing and income analytics
func (s *Service) GetIncomeOverview(ctx context.Context, req *connect.Request[superadminv1.GetIncomeOverviewRequest]) (*connect.Response[superadminv1.GetIncomeOverviewResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Parse date range
	now := time.Now()
	startDate := now.AddDate(0, 0, -30) // Default to 30 days ago
	endDate := now

	if req.Msg.StartDate != nil && *req.Msg.StartDate != "" {
		if parsed, err := time.Parse("2006-01-02", *req.Msg.StartDate); err == nil {
			startDate = parsed
		}
	}
	if req.Msg.EndDate != nil && *req.Msg.EndDate != "" {
		if parsed, err := time.Parse("2006-01-02", *req.Msg.EndDate); err == nil {
			endDate = parsed
		}
	}

	// Get all credit transactions in period
	var transactions []database.CreditTransaction
	err = database.DB.Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Order("created_at DESC").
		Find(&transactions).Error
	if err != nil {
		logger.Error("[SuperAdmin] Failed to query transactions: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query transactions: %w", err))
	}

	// Calculate summary
	var totalRevenue float64
	var totalRefunds float64
	var successfulPayments int64
	var failedPayments int64
	var pendingPayments int64
	var totalTransactionCount int64
	var totalPaymentAmount float64
	var largestPayment float64

	// Map to track organization names
	orgNames := make(map[string]string)

	for _, t := range transactions {
		totalTransactionCount++
		amount := float64(t.AmountCents) / 100.0

		// Get organization name
		if _, ok := orgNames[t.OrganizationID]; !ok {
			var org database.Organization
			if err := database.DB.First(&org, "id = ?", t.OrganizationID).Error; err == nil {
				orgNames[t.OrganizationID] = org.Name
			} else {
				orgNames[t.OrganizationID] = t.OrganizationID[:8]
			}
		}

		if t.Type == "payment" {
			if t.AmountCents > 0 {
				totalRevenue += amount
				successfulPayments++
				totalPaymentAmount += amount
				if amount > largestPayment {
					largestPayment = amount
				}
			} else {
				totalRefunds += math.Abs(amount)
			}
		} else if t.Type == "refund" {
			totalRefunds += math.Abs(amount)
		}
	}

	netRevenue := totalRevenue - totalRefunds
	avgMonthlyRevenue := totalRevenue / float64(monthsBetween(startDate, endDate))
	mrr := avgMonthlyRevenue // Estimate MRR as average monthly revenue

	// Calculate estimated monthly income from all organizations based on usage
	estimatedMonthlyIncome := s.calculateEstimatedMonthlyIncome(ctx)

	successRate := float64(0)
	if successfulPayments+failedPayments > 0 {
		successRate = float64(successfulPayments) / float64(successfulPayments+failedPayments) * 100
	}
	avgPaymentAmount := float64(0)
	if successfulPayments > 0 {
		avgPaymentAmount = totalPaymentAmount / float64(successfulPayments)
	}

	summary := &superadminv1.IncomeSummary{
		TotalRevenue:          totalRevenue,
		MonthlyRecurringRevenue: mrr,
		AverageMonthlyRevenue: avgMonthlyRevenue,
		TotalTransactions:     totalTransactionCount,
		TotalRefunds:          totalRefunds,
		NetRevenue:           netRevenue,
		EstimatedMonthlyIncome: estimatedMonthlyIncome,
	}

	// Calculate monthly income breakdown
	monthlyIncomeMap := make(map[string]*monthlyIncomeData)
	for _, t := range transactions {
		if t.Type == "payment" && t.AmountCents > 0 {
			month := t.CreatedAt.Format("2006-01")
			if _, ok := monthlyIncomeMap[month]; !ok {
				monthlyIncomeMap[month] = &monthlyIncomeData{
					Revenue: 0,
					Count:   0,
					Refunds: 0,
				}
			}
			monthlyIncomeMap[month].Revenue += float64(t.AmountCents) / 100.0
			monthlyIncomeMap[month].Count++
		} else if t.Type == "refund" || (t.Type == "payment" && t.AmountCents < 0) {
			month := t.CreatedAt.Format("2006-01")
			if _, ok := monthlyIncomeMap[month]; !ok {
				monthlyIncomeMap[month] = &monthlyIncomeData{
					Revenue: 0,
					Count:   0,
					Refunds: 0,
				}
			}
			monthlyIncomeMap[month].Refunds += math.Abs(float64(t.AmountCents)) / 100.0
		}
	}

	var monthlyIncome []*superadminv1.MonthlyIncome
	for month, data := range monthlyIncomeMap {
		monthlyIncome = append(monthlyIncome, &superadminv1.MonthlyIncome{
			Month:           month,
			Revenue:         data.Revenue,
			TransactionCount: data.Count,
			Refunds:         data.Refunds,
		})
	}
	sort.Slice(monthlyIncome, func(i, j int) bool {
		return monthlyIncome[i].Month < monthlyIncome[j].Month
	})

	// Calculate top customers
	customerRevenue := make(map[string]*customerData)
	for _, t := range transactions {
		if t.Type == "payment" && t.AmountCents > 0 {
			if _, ok := customerRevenue[t.OrganizationID]; !ok {
				customerRevenue[t.OrganizationID] = &customerData{
					TotalRevenue:  0,
					Count:         0,
					FirstPayment:  t.CreatedAt,
					LastPayment:   t.CreatedAt,
				}
			}
			customerRevenue[t.OrganizationID].TotalRevenue += float64(t.AmountCents) / 100.0
			customerRevenue[t.OrganizationID].Count++
			if t.CreatedAt.Before(customerRevenue[t.OrganizationID].FirstPayment) {
				customerRevenue[t.OrganizationID].FirstPayment = t.CreatedAt
			}
			if t.CreatedAt.After(customerRevenue[t.OrganizationID].LastPayment) {
				customerRevenue[t.OrganizationID].LastPayment = t.CreatedAt
			}
		}
	}

	type topCustomerData struct {
		OrgID        string
		OrgName      string
		TotalRevenue float64
		Count        int64
		FirstPayment time.Time
		LastPayment  time.Time
	}

	var topCustomersData []topCustomerData
	for orgID, data := range customerRevenue {
		topCustomersData = append(topCustomersData, topCustomerData{
			OrgID:        orgID,
			OrgName:      orgNames[orgID],
			TotalRevenue: data.TotalRevenue,
			Count:        data.Count,
			FirstPayment: data.FirstPayment,
			LastPayment:  data.LastPayment,
		})
	}

	// Sort by revenue descending
	sort.Slice(topCustomersData, func(i, j int) bool {
		return topCustomersData[i].TotalRevenue > topCustomersData[j].TotalRevenue
	})

	// Take top 20
	topCustomersLimit := 20
	if len(topCustomersData) < topCustomersLimit {
		topCustomersLimit = len(topCustomersData)
	}

	var topCustomers []*superadminv1.TopCustomer
	for i := 0; i < topCustomersLimit; i++ {
		tc := topCustomersData[i]
		topCustomers = append(topCustomers, &superadminv1.TopCustomer{
			OrganizationId:  tc.OrgID,
			OrganizationName: tc.OrgName,
			TotalRevenue:    tc.TotalRevenue,
			TransactionCount: tc.Count,
			FirstPayment:    timestamppb.New(tc.FirstPayment),
			LastPayment:     timestamppb.New(tc.LastPayment),
		})
	}

	// Convert transactions to proto (limit to 1000 most recent)
	transactionLimit := 1000
	if len(transactions) > transactionLimit {
		transactions = transactions[:transactionLimit]
	}

	var billingTransactions []*superadminv1.BillingTransaction
	for _, t := range transactions {
		status := "succeeded"
		if t.AmountCents < 0 {
			status = "refunded"
		}

		// Extract Stripe IDs from note if available
		var stripeInvoiceID, stripePaymentIntentID *string
		if t.Note != nil {
			if strings.Contains(*t.Note, "invoice") {
				// Try to extract invoice ID from note
				parts := strings.Fields(*t.Note)
				for _, part := range parts {
					if strings.HasPrefix(part, "in_") {
						stripeInvoiceID = &part
						break
					}
				}
			}
			if strings.Contains(*t.Note, "pi_") {
				parts := strings.Fields(*t.Note)
				for _, part := range parts {
					if strings.HasPrefix(part, "pi_") {
						stripePaymentIntentID = &part
						break
					}
				}
			}
		}

		billingTransactions = append(billingTransactions, &superadminv1.BillingTransaction{
			Id:                    t.ID,
			OrganizationId:        t.OrganizationID,
			OrganizationName:      orgNames[t.OrganizationID],
			Type:                 t.Type,
			AmountCents:          float64(t.AmountCents),
			Currency:             "USD",
			Status:               status,
			StripeInvoiceId:      stripeInvoiceID,
			StripePaymentIntentId: stripePaymentIntentID,
			Note:                 t.Note,
			CreatedAt:            timestamppb.New(t.CreatedAt),
		})
	}

	paymentMetrics := &superadminv1.PaymentMetrics{
		SuccessRate:          successRate,
		SuccessfulPayments:   successfulPayments,
		FailedPayments:       failedPayments,
		PendingPayments:      pendingPayments,
		AveragePaymentAmount: avgPaymentAmount,
		LargestPayment:       largestPayment,
	}

	return connect.NewResponse(&superadminv1.GetIncomeOverviewResponse{
		Summary:        summary,
		MonthlyIncome:  monthlyIncome,
		TopCustomers:   topCustomers,
		Transactions:   billingTransactions,
		PaymentMetrics: paymentMetrics,
	}), nil
}

type monthlyIncomeData struct {
	Revenue float64
	Count   int64
	Refunds float64
}

type customerData struct {
	TotalRevenue float64
	Count        int64
	FirstPayment time.Time
	LastPayment  time.Time
}

func monthsBetween(start, end time.Time) int {
	if start.After(end) {
		return 1
	}
	months := int(end.Year()*12+int(end.Month())) - int(start.Year()*12+int(start.Month()))
	if months < 1 {
		return 1
	}
	return months
}

// calculateEstimatedMonthlyIncome calculates the total estimated monthly income
// from all organizations based on their current usage patterns
func (s *Service) calculateEstimatedMonthlyIncome(ctx context.Context) float64 {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)
	
	// Get all organizations
	var orgs []database.Organization
	if err := database.DB.Find(&orgs).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to query organizations for estimated income: %v", err)
		return 0
	}

	pricingModel := pricing.GetPricing()
	var totalEstimatedIncome int64

	// Aggregate cutoff: current hour (aggregates exist up to current hour)
	aggregateCutoff := now.Truncate(time.Hour)
	if aggregateCutoff.Before(monthStart) {
		aggregateCutoff = monthStart
	}

	// Get metrics DB for efficient querying
	metricsDB := database.GetMetricsDB()
	if metricsDB == nil {
		logger.Warn("[SuperAdmin] Metrics database not available for estimated income calculation")
		return 0
	}

	// Calculate elapsed ratio for projection
	elapsed := now.Sub(monthStart)
	monthDuration := monthEnd.Sub(monthStart)
	elapsedRatio := float64(elapsed) / float64(monthDuration)
	if elapsedRatio <= 0 {
		elapsedRatio = 1.0 // Avoid division by zero
	}

	// Aggregate usage across all organizations efficiently
	type orgUsage struct {
		OrganizationID    string
		CPUCoreSeconds    int64
		MemoryByteSeconds int64
		BandwidthRxBytes  int64
		BandwidthTxBytes  int64
		StorageBytes      int64
	}

	var usages []orgUsage
	
	// Get usage from hourly aggregates
	metricsDB.Table("deployment_usage_hourly duh").
		Select(`
			duh.organization_id,
			COALESCE(CAST(SUM((duh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
			COALESCE(CAST(SUM(duh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
			COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
			COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
		`).
		Where("duh.hour >= ? AND duh.hour < ?", monthStart, aggregateCutoff).
		Group("duh.organization_id").
		Scan(&usages)

	// Map organization IDs to usage
	usageMap := make(map[string]*orgUsage)
	for i := range usages {
		usageMap[usages[i].OrganizationID] = &usages[i]
	}

	// Add storage for each organization
	for _, org := range orgs {
		if _, ok := usageMap[org.ID]; !ok {
			usageMap[org.ID] = &orgUsage{
				OrganizationID: org.ID,
			}
		}
	}

	// Get storage for all organizations
	var storageResults []struct {
		OrganizationID string
		StorageBytes   int64
	}
	database.DB.Table("deployments d").
		Select("d.organization_id, COALESCE(SUM(d.storage_bytes), 0) as storage_bytes").
		Group("d.organization_id").
		Scan(&storageResults)

	for _, sr := range storageResults {
		if usage, ok := usageMap[sr.OrganizationID]; ok {
			usage.StorageBytes = sr.StorageBytes
		}
	}

	// Calculate estimated monthly cost for each organization
	for _, usage := range usageMap {
		// Project current usage to full month
		estimatedCPUCoreSeconds := int64(float64(usage.CPUCoreSeconds) / elapsedRatio)
		estimatedMemoryByteSeconds := int64(float64(usage.MemoryByteSeconds) / elapsedRatio)
		estimatedBandwidthBytes := usage.BandwidthRxBytes + usage.BandwidthTxBytes // Bandwidth is cumulative
		estimatedStorageBytes := usage.StorageBytes // Storage is already monthly

		// Calculate costs
		estCPUCost := pricingModel.CalculateCPUCost(estimatedCPUCoreSeconds)
		estMemoryCost := pricingModel.CalculateMemoryCost(estimatedMemoryByteSeconds)
		estBandwidthCost := pricingModel.CalculateBandwidthCost(estimatedBandwidthBytes)
		estStorageCost := pricingModel.CalculateStorageCost(estimatedStorageBytes)

		totalEstimatedIncome += estCPUCost + estMemoryCost + estBandwidthCost + estStorageCost
	}

	// Convert from cents to dollars
	return float64(totalEstimatedIncome) / 100.0
}

// ListAllInvoices lists all invoices across all organizations
func (s *Service) ListAllInvoices(ctx context.Context, req *connect.Request[superadminv1.ListAllInvoicesRequest]) (*connect.Response[superadminv1.ListAllInvoicesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	if s.stripeClient == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stripe is not configured"))
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	var status *string
	if req.Msg.Status != nil && *req.Msg.Status != "" {
		status = req.Msg.Status
	}

	var startDate, endDate *time.Time
	if req.Msg.StartDate != nil && *req.Msg.StartDate != "" {
		if parsed, err := time.Parse("2006-01-02", *req.Msg.StartDate); err == nil {
			startDate = &parsed
		}
	}
	if req.Msg.EndDate != nil && *req.Msg.EndDate != "" {
		if parsed, err := time.Parse("2006-01-02", *req.Msg.EndDate); err == nil {
			endDate = &parsed
		}
	}

	// Get all invoices from Stripe
	stripeInvoices, hasMore, err := s.stripeClient.ListAllInvoices(ctx, limit, status, startDate, endDate)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to list all invoices: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list invoices: %w", err))
	}

	// Map customer IDs to organization info
	customerToOrg := make(map[string]*customerOrgInfo)
	var orgIDFilter *string
	if req.Msg.OrganizationId != nil && *req.Msg.OrganizationId != "" {
		orgIDFilter = req.Msg.OrganizationId
	}

	// Get all billing accounts with Stripe customer IDs
	var billingAccounts []database.BillingAccount
	query := database.DB.Where("stripe_customer_id IS NOT NULL AND stripe_customer_id != ''")
	if orgIDFilter != nil {
		query = query.Where("organization_id = ?", *orgIDFilter)
	}
	if err := query.Find(&billingAccounts).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to query billing accounts: %v", err)
	} else {
		for _, ba := range billingAccounts {
			if ba.StripeCustomerID != nil {
				// Get organization name
				var org database.Organization
				if err := database.DB.First(&org, "id = ?", ba.OrganizationID).Error; err == nil {
					customerToOrg[*ba.StripeCustomerID] = &customerOrgInfo{
						OrganizationID: ba.OrganizationID,
						OrganizationName: org.Name,
						BillingEmail: ba.BillingEmail,
					}
				}
			}
		}
	}

	// Convert Stripe invoices to proto
	invoices := make([]*superadminv1.InvoiceWithOrganization, 0, len(stripeInvoices))
	for _, inv := range stripeInvoices {
		orgInfo := customerToOrg[inv.Customer.ID]
		if orgIDFilter != nil && orgInfo == nil {
			continue // Skip if org filter doesn't match
		}

		protoInvoice := &billingv1.Invoice{
			Id:         inv.ID,
			Number:     inv.Number,
			Status:     string(inv.Status),
			AmountDue:  inv.AmountDue,
			AmountPaid: inv.AmountPaid,
			Currency:   strings.ToUpper(string(inv.Currency)),
		}

		if inv.Created > 0 {
			protoInvoice.Date = timestamppb.New(time.Unix(inv.Created, 0))
		}

		if inv.DueDate > 0 {
			protoInvoice.DueDate = timestamppb.New(time.Unix(inv.DueDate, 0))
		}

		if inv.InvoicePDF != "" {
			protoInvoice.InvoicePdf = &inv.InvoicePDF
		}

		if inv.HostedInvoiceURL != "" {
			protoInvoice.HostedInvoiceUrl = &inv.HostedInvoiceURL
		}

		if inv.Description != "" {
			protoInvoice.Description = &inv.Description
		}

		customerEmail := ""
		if orgInfo != nil && orgInfo.BillingEmail != nil {
			customerEmail = *orgInfo.BillingEmail
		} else if inv.Customer.Email != "" {
			customerEmail = inv.Customer.Email
		}

		orgID := ""
		orgName := "Unknown"
		if orgInfo != nil {
			orgID = orgInfo.OrganizationID
			orgName = orgInfo.OrganizationName
		}

		invoices = append(invoices, &superadminv1.InvoiceWithOrganization{
			Invoice:          protoInvoice,
			OrganizationId:   orgID,
			OrganizationName: orgName,
			CustomerEmail:    customerEmail,
		})
	}

	return connect.NewResponse(&superadminv1.ListAllInvoicesResponse{
		Invoices:   invoices,
		HasMore:    hasMore,
		TotalCount: int64(len(invoices)),
	}), nil
}

// SendInvoiceReminder sends an invoice reminder email to the customer
func (s *Service) SendInvoiceReminder(ctx context.Context, req *connect.Request[superadminv1.SendInvoiceReminderRequest]) (*connect.Response[superadminv1.SendInvoiceReminderResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	if s.stripeClient == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stripe is not configured"))
	}

	invoiceID := strings.TrimSpace(req.Msg.GetInvoiceId())
	if invoiceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invoice_id is required"))
	}

	// Send invoice via Stripe
	_, err = s.stripeClient.SendInvoice(ctx, invoiceID)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to send invoice reminder: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to send invoice reminder: %w", err))
	}

	logger.Info("[SuperAdmin] Sent invoice reminder for invoice: %s", invoiceID)

	return connect.NewResponse(&superadminv1.SendInvoiceReminderResponse{
		Success: true,
		Message: "Invoice reminder sent successfully",
	}), nil
}

type customerOrgInfo struct {
	OrganizationID   string
	OrganizationName string
	BillingEmail     *string
}

// Plan Management Endpoints

func (s *Service) ListPlans(ctx context.Context, _ *connect.Request[superadminv1.ListPlansRequest]) (*connect.Response[superadminv1.ListPlansResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	var plans []database.OrganizationPlan
	if err := database.DB.Order("name ASC").Find(&plans).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list plans: %w", err))
	}

	protoPlans := make([]*superadminv1.Plan, len(plans))
	for i, plan := range plans {
		protoPlans[i] = &superadminv1.Plan{
			Id:                      plan.ID,
			Name:                    plan.Name,
			CpuCores:                int32(plan.CPUCores),
			MemoryBytes:             plan.MemoryBytes,
			DeploymentsMax:          int32(plan.DeploymentsMax),
			MaxVpsInstances:         int32(plan.MaxVpsInstances),
			BandwidthBytesMonth:     plan.BandwidthBytesMonth,
			StorageBytes:            plan.StorageBytes,
			MinimumPaymentCents:     plan.MinimumPaymentCents,
			MonthlyFreeCreditsCents: plan.MonthlyFreeCreditsCents,
			TrialDays:               int32(plan.TrialDays),
			Description:             plan.Description,
		}
	}

	return connect.NewResponse(&superadminv1.ListPlansResponse{
		Plans: protoPlans,
	}), nil
}

func (s *Service) CreatePlan(ctx context.Context, req *connect.Request[superadminv1.CreatePlanRequest]) (*connect.Response[superadminv1.CreatePlanResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	plan := &database.OrganizationPlan{
		ID:                      fmt.Sprintf("plan-%d", time.Now().UnixNano()),
		Name:                    req.Msg.GetName(),
		CPUCores:                int(req.Msg.GetCpuCores()),
		MemoryBytes:             req.Msg.GetMemoryBytes(),
		DeploymentsMax:          int(req.Msg.GetDeploymentsMax()),
		MaxVpsInstances:         int(req.Msg.GetMaxVpsInstances()),
		BandwidthBytesMonth:     req.Msg.GetBandwidthBytesMonth(),
		StorageBytes:            req.Msg.GetStorageBytes(),
		MinimumPaymentCents:     req.Msg.GetMinimumPaymentCents(),
		MonthlyFreeCreditsCents: req.Msg.GetMonthlyFreeCreditsCents(),
		TrialDays:               int(req.Msg.GetTrialDays()),
		Description:             req.Msg.GetDescription(),
	}

	if err := database.DB.Create(plan).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create plan: %w", err))
	}

	protoPlan := &superadminv1.Plan{
		Id:                      plan.ID,
		Name:                    plan.Name,
		CpuCores:                int32(plan.CPUCores),
		MemoryBytes:             plan.MemoryBytes,
		DeploymentsMax:          int32(plan.DeploymentsMax),
		MaxVpsInstances:         int32(plan.MaxVpsInstances),
		BandwidthBytesMonth:     plan.BandwidthBytesMonth,
		StorageBytes:            plan.StorageBytes,
		MinimumPaymentCents:     plan.MinimumPaymentCents,
		MonthlyFreeCreditsCents: plan.MonthlyFreeCreditsCents,
		Description:             plan.Description,
	}

	return connect.NewResponse(&superadminv1.CreatePlanResponse{
		Plan: protoPlan,
	}), nil
}

func (s *Service) UpdatePlan(ctx context.Context, req *connect.Request[superadminv1.UpdatePlanRequest]) (*connect.Response[superadminv1.UpdatePlanResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	var plan database.OrganizationPlan
	if err := database.DB.First(&plan, "id = ?", req.Msg.GetId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("plan not found"))
	}

	if req.Msg.Name != nil {
		plan.Name = *req.Msg.Name
	}
	if req.Msg.CpuCores != nil {
		plan.CPUCores = int(*req.Msg.CpuCores)
	}
	if req.Msg.MemoryBytes != nil {
		plan.MemoryBytes = *req.Msg.MemoryBytes
	}
	if req.Msg.DeploymentsMax != nil {
		plan.DeploymentsMax = int(*req.Msg.DeploymentsMax)
	}
	if req.Msg.MaxVpsInstances != nil {
		plan.MaxVpsInstances = int(*req.Msg.MaxVpsInstances)
	}
	if req.Msg.BandwidthBytesMonth != nil {
		plan.BandwidthBytesMonth = *req.Msg.BandwidthBytesMonth
	}
	if req.Msg.StorageBytes != nil {
		plan.StorageBytes = *req.Msg.StorageBytes
	}
	if req.Msg.MinimumPaymentCents != nil {
		plan.MinimumPaymentCents = *req.Msg.MinimumPaymentCents
	}
	if req.Msg.MonthlyFreeCreditsCents != nil {
		plan.MonthlyFreeCreditsCents = *req.Msg.MonthlyFreeCreditsCents
	}
	if req.Msg.TrialDays != nil {
		plan.TrialDays = int(*req.Msg.TrialDays)
	}
	if req.Msg.Description != nil {
		plan.Description = *req.Msg.Description
	}

	if err := database.DB.Save(&plan).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update plan: %w", err))
	}

	protoPlan := &superadminv1.Plan{
		Id:                      plan.ID,
		Name:                    plan.Name,
		CpuCores:                int32(plan.CPUCores),
		MemoryBytes:             plan.MemoryBytes,
		DeploymentsMax:          int32(plan.DeploymentsMax),
		MaxVpsInstances:         int32(plan.MaxVpsInstances),
		BandwidthBytesMonth:     plan.BandwidthBytesMonth,
		StorageBytes:            plan.StorageBytes,
		MinimumPaymentCents:     plan.MinimumPaymentCents,
		MonthlyFreeCreditsCents: plan.MonthlyFreeCreditsCents,
		TrialDays:               int32(plan.TrialDays),
		Description:             plan.Description,
	}

	return connect.NewResponse(&superadminv1.UpdatePlanResponse{
		Plan: protoPlan,
	}), nil
}

func (s *Service) DeletePlan(ctx context.Context, req *connect.Request[superadminv1.DeletePlanRequest]) (*connect.Response[superadminv1.DeletePlanResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Check if any organizations are using this plan
	var count int64
	if err := database.DB.Model(&database.OrgQuota{}).Where("plan_id = ?", req.Msg.GetId()).Count(&count).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check plan usage: %w", err))
	}
	if count > 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cannot delete plan: %d organizations are using it", count))
	}

	if err := database.DB.Delete(&database.OrganizationPlan{}, "id = ?", req.Msg.GetId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete plan: %w", err))
	}

	return connect.NewResponse(&superadminv1.DeletePlanResponse{
		Success: true,
	}), nil
}

func (s *Service) AssignPlanToOrganization(ctx context.Context, req *connect.Request[superadminv1.AssignPlanToOrganizationRequest]) (*connect.Response[superadminv1.AssignPlanToOrganizationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Verify plan exists
	var plan database.OrganizationPlan
	if err := database.DB.First(&plan, "id = ?", req.Msg.GetPlanId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("plan not found"))
	}

	// Verify organization exists
	var org database.Organization
	if err := database.DB.First(&org, "id = ?", req.Msg.GetOrganizationId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
	}

	// Get or create OrgQuota
	var quota database.OrgQuota
	if err := database.DB.First(&quota, "organization_id = ?", req.Msg.GetOrganizationId()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			quota = database.OrgQuota{
				OrganizationID: req.Msg.GetOrganizationId(),
				PlanID:        req.Msg.GetPlanId(),
			}
			if err := database.DB.Create(&quota).Error; err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create quota: %w", err))
			}
		} else {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get quota: %w", err))
		}
	} else {
		quota.PlanID = req.Msg.GetPlanId()
		if err := database.DB.Save(&quota).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update quota: %w", err))
		}
	}

	return connect.NewResponse(&superadminv1.AssignPlanToOrganizationResponse{
		Success: true,
		Message: fmt.Sprintf("Plan %s assigned to organization %s", plan.Name, org.Name),
	}), nil
}

// ListStripeWebhookEvents lists all Stripe webhook events with optional filters
func (s *Service) ListStripeWebhookEvents(ctx context.Context, req *connect.Request[superadminv1.ListStripeWebhookEventsRequest]) (*connect.Response[superadminv1.ListStripeWebhookEventsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	offset := int(req.Msg.GetOffset())
	if offset < 0 {
		offset = 0
	}

	// Build query with filters
	query := database.DB.Model(&database.StripeWebhookEvent{})

	if orgID := req.Msg.GetOrganizationId(); orgID != "" {
		query = query.Where("organization_id = ?", orgID)
	}

	if eventType := req.Msg.GetEventType(); eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	if customerID := req.Msg.GetCustomerId(); customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}

	if subscriptionID := req.Msg.GetSubscriptionId(); subscriptionID != "" {
		query = query.Where("subscription_id = ?", subscriptionID)
	}

	if invoiceID := req.Msg.GetInvoiceId(); invoiceID != "" {
		query = query.Where("invoice_id = ?", invoiceID)
	}

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to count webhook events: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count events: %w", err))
	}

	// Get events with pagination
	var events []database.StripeWebhookEvent
	if err := query.Order("processed_at DESC").Limit(limit).Offset(offset).Find(&events).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to query webhook events: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query events: %w", err))
	}

	// Get organization names for events that have organization_id
	orgIDs := make([]string, 0)
	orgIDSet := make(map[string]bool)
	for _, event := range events {
		if event.OrganizationID != nil && *event.OrganizationID != "" {
			if !orgIDSet[*event.OrganizationID] {
				orgIDs = append(orgIDs, *event.OrganizationID)
				orgIDSet[*event.OrganizationID] = true
			}
		}
	}

	orgNames := make(map[string]string)
	if len(orgIDs) > 0 {
		var orgs []database.Organization
		if err := database.DB.Where("id IN ?", orgIDs).Find(&orgs).Error; err == nil {
			for _, org := range orgs {
				orgNames[org.ID] = org.Name
			}
		}
	}

	// Convert to proto messages
	protoEvents := make([]*superadminv1.StripeWebhookEvent, 0, len(events))
	for _, event := range events {
		protoEvent := &superadminv1.StripeWebhookEvent{
			Id:         event.ID,
			EventType:  event.EventType,
			ProcessedAt: timestamppb.New(event.ProcessedAt),
			CreatedAt:  timestamppb.New(event.CreatedAt),
		}

		if event.OrganizationID != nil {
			protoEvent.OrganizationId = event.OrganizationID
			if orgName, ok := orgNames[*event.OrganizationID]; ok {
				protoEvent.OrganizationName = &orgName
			}
		}

		if event.CustomerID != nil {
			protoEvent.CustomerId = event.CustomerID
		}

		if event.SubscriptionID != nil {
			protoEvent.SubscriptionId = event.SubscriptionID
		}

		if event.InvoiceID != nil {
			protoEvent.InvoiceId = event.InvoiceID
		}

		if event.CheckoutSessionID != nil {
			protoEvent.CheckoutSessionId = event.CheckoutSessionID
		}

		protoEvents = append(protoEvents, protoEvent)
	}

	return connect.NewResponse(&superadminv1.ListStripeWebhookEventsResponse{
		Events:     protoEvents,
		TotalCount: totalCount,
	}), nil
}

func (s *Service) ListUsers(ctx context.Context, req *connect.Request[superadminv1.ListUsersRequest]) (*connect.Response[superadminv1.ListUsersResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Pagination
	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	// Get all unique user IDs from organization members
	// We'll need to query the database to get distinct user IDs
	var userIDs []string
	query := database.DB.Model(&database.OrganizationMember{}).
		Distinct("user_id").
		Where("user_id NOT LIKE ?", "pending:%")

	// Apply search filter if provided
	search := strings.TrimSpace(req.Msg.GetSearch())
	if search != "" {
		// For now, we can only search by user_id since we don't have user data in DB
		// The resolver will fetch full user details
		query = query.Where("user_id LIKE ?", "%"+search+"%")
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count users: %w", err))
	}

	// Get paginated user IDs
	if err := query.
		Order("user_id ASC").
		Limit(perPage).
		Offset(offset).
		Pluck("user_id", &userIDs).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list users: %w", err))
	}

	// Resolve user profiles using the user profile resolver
	resolver := organizations.GetUserProfileResolver()
	users := make([]*superadminv1.UserInfo, 0, len(userIDs))
	
	for _, userID := range userIDs {
		userInfo := &superadminv1.UserInfo{
			Id: userID,
		}

		// Try to resolve profile from Zitadel
		if resolver != nil && resolver.IsConfigured() {
			if profile, err := resolver.Resolve(ctx, userID); err == nil && profile != nil {
				userInfo.Id = profile.Id
				userInfo.Email = profile.Email
				userInfo.Name = profile.Name
				userInfo.PreferredUsername = profile.PreferredUsername
				userInfo.Locale = profile.Locale
				userInfo.EmailVerified = profile.EmailVerified
				if profile.AvatarUrl != "" {
					userInfo.AvatarUrl = &profile.AvatarUrl
				}
				if profile.UpdatedAt != nil {
					userInfo.UpdatedAt = profile.UpdatedAt
				}
			}
		}

		// Get user roles (check if superadmin)
		// Roles are determined by SUPERADMIN_EMAILS env var
		if userInfo.Email != "" {
			var roles []string
			// Check if user is superadmin by checking the superadmin emails map
			// We need to check the auth config's superadmin emails
			// For now, we'll check via HasRole which uses the same mechanism
			testUser := &authv1.User{Email: userInfo.Email, Roles: []string{}}
			if auth.HasRole(testUser, auth.RoleSuperAdmin) {
				roles = append(roles, auth.RoleSuperAdmin)
			}
			userInfo.Roles = roles
		}

		users = append(users, userInfo)
	}

	totalPages := (int(total) + perPage - 1) / perPage

	return connect.NewResponse(&superadminv1.ListUsersResponse{
		Users: users,
		Pagination: &commonv1.Pagination{
			Page:       int32(page),
			PerPage:    int32(perPage),
			Total:      int32(total),
			TotalPages: int32(totalPages),
		},
	}), nil
}

func (s *Service) GetUser(ctx context.Context, req *connect.Request[superadminv1.GetUserRequest]) (*connect.Response[superadminv1.GetUserResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	userID := strings.TrimSpace(req.Msg.GetUserId())
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	// Resolve user profile
	resolver := organizations.GetUserProfileResolver()
	userInfo := &superadminv1.UserInfo{
		Id: userID,
	}

	// Try to resolve profile from Zitadel
	if resolver != nil && resolver.IsConfigured() {
		if profile, err := resolver.Resolve(ctx, userID); err == nil && profile != nil {
			userInfo.Id = profile.Id
			userInfo.Email = profile.Email
			userInfo.Name = profile.Name
			userInfo.PreferredUsername = profile.PreferredUsername
			userInfo.Locale = profile.Locale
			userInfo.EmailVerified = profile.EmailVerified
			if profile.AvatarUrl != "" {
				userInfo.AvatarUrl = &profile.AvatarUrl
			}
			if profile.UpdatedAt != nil {
				userInfo.UpdatedAt = profile.UpdatedAt
			}
			if profile.CreatedAt != nil {
				userInfo.CreatedAt = profile.CreatedAt
			}
		}
	}

	// Get user roles
	// Roles are determined by SUPERADMIN_EMAILS env var
	if userInfo.Email != "" {
		var roles []string
		// Check if user is superadmin by checking the superadmin emails map
		testUser := &authv1.User{Email: userInfo.Email, Roles: []string{}}
		if auth.HasRole(testUser, auth.RoleSuperAdmin) {
			roles = append(roles, auth.RoleSuperAdmin)
		}
		userInfo.Roles = roles
	}

	// Get all organizations this user belongs to
	var members []database.OrganizationMember
	if err := database.DB.Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get user organizations: %w", err))
	}

	orgs := make([]*superadminv1.UserOrganization, 0, len(members))
	for _, member := range members {
		var org database.Organization
		if err := database.DB.First(&org, "id = ?", member.OrganizationID).Error; err != nil {
			logger.Warn("[SuperAdmin] Failed to load organization %s for user %s: %v", member.OrganizationID, userID, err)
			continue
		}

		orgs = append(orgs, &superadminv1.UserOrganization{
			OrganizationId:   org.ID,
			OrganizationName: org.Name,
			Role:            member.Role,
			Status:          member.Status,
			JoinedAt:        timestamppb.New(member.JoinedAt),
		})
	}

	return connect.NewResponse(&superadminv1.GetUserResponse{
		User:          userInfo,
		Organizations: orgs,
	}), nil
}

// ListAllVPS lists all VPS instances across all organizations (superadmin only)
func (s *Service) ListAllVPS(ctx context.Context, req *connect.Request[superadminv1.ListAllVPSRequest]) (*connect.Response[superadminv1.ListAllVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Parse pagination
	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	// Build query
	query := database.DB.Table("vps_instances v").
		Select(`
			v.*,
			o.name as organization_name
		`).
		Joins("LEFT JOIN organizations o ON o.id = v.organization_id").
		Where("v.deleted_at IS NULL")

	// Apply filters
	if orgID := req.Msg.GetOrganizationId(); orgID != "" {
		query = query.Where("v.organization_id = ?", orgID)
	}

	if req.Msg.Status != nil {
		status := req.Msg.GetStatus()
		if status != vpsv1.VPSStatus_VPS_STATUS_UNSPECIFIED {
			query = query.Where("v.status = ?", int32(status))
		}
	}

	// Apply search filter
	if search := strings.TrimSpace(req.Msg.GetSearch()); search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where(
			"v.name ILIKE ? OR v.id ILIKE ? OR v.organization_id ILIKE ? OR v.region ILIKE ? OR v.size ILIKE ? OR o.name ILIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm,
		)
	}

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to count VPS instances: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count VPS instances: %w", err))
	}

	// Apply pagination and ordering
	var vpsRows []struct {
		database.VPSInstance
		OrganizationName string `gorm:"column:organization_name"`
	}

	if err := query.Order("v.created_at DESC").Limit(perPage).Offset(offset).Find(&vpsRows).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to list VPS instances: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list VPS instances: %w", err))
	}

	// Convert to proto
	vpsOverviews := make([]*superadminv1.VPSOverview, 0, len(vpsRows))
	for _, row := range vpsRows {
		// Convert database model to proto
		vpsProto := convertVPSInstanceToProto(&row.VPSInstance)
		vpsOverviews = append(vpsOverviews, &superadminv1.VPSOverview{
			Vps:              vpsProto,
			OrganizationName: row.OrganizationName,
		})
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))

	return connect.NewResponse(&superadminv1.ListAllVPSResponse{
		VpsInstances: vpsOverviews,
		Pagination: &commonv1.Pagination{
			Page:       int32(page),
			PerPage:    int32(perPage),
			Total:      int32(totalCount),
			TotalPages: int32(totalPages),
		},
	}), nil
}

// convertVPSInstanceToProto converts a database.VPSInstance to vpsv1.VPSInstance
func convertVPSInstanceToProto(vps *database.VPSInstance) *vpsv1.VPSInstance {
	protoVPS := &vpsv1.VPSInstance{
		Id:             vps.ID,
		Name:           vps.Name,
		Description:    vps.Description,
		Status:         vpsv1.VPSStatus(vps.Status),
		Region:         vps.Region,
		Image:          vpsv1.VPSImage(vps.Image),
		ImageId:        vps.ImageID,
		Size:           vps.Size,
		CpuCores:       vps.CPUCores,
		MemoryBytes:    vps.MemoryBytes,
		DiskBytes:      vps.DiskBytes,
		InstanceId:     vps.InstanceID,
		NodeId:         vps.NodeID,
		SshKeyId:       vps.SSHKeyID,
		CreatedAt:      timestamppb.New(vps.CreatedAt),
		UpdatedAt:      timestamppb.New(vps.UpdatedAt),
		OrganizationId: vps.OrganizationID,
		CreatedBy:      vps.CreatedBy,
	}

	if vps.LastStartedAt != nil {
		protoVPS.LastStartedAt = timestamppb.New(*vps.LastStartedAt)
	}
	if vps.DeletedAt != nil {
		protoVPS.DeletedAt = timestamppb.New(*vps.DeletedAt)
	}

	// Unmarshal JSON fields
	if vps.IPv4Addresses != "" {
		var ipv4s []string
		if err := json.Unmarshal([]byte(vps.IPv4Addresses), &ipv4s); err == nil {
			protoVPS.Ipv4Addresses = ipv4s
		}
	}
	if vps.IPv6Addresses != "" {
		var ipv6s []string
		if err := json.Unmarshal([]byte(vps.IPv6Addresses), &ipv6s); err == nil {
			protoVPS.Ipv6Addresses = ipv6s
		}
	}
	if vps.Metadata != "" {
		var metadata map[string]string
		if err := json.Unmarshal([]byte(vps.Metadata), &metadata); err == nil {
			protoVPS.Metadata = metadata
		}
	}

	return protoVPS
}

// ListVPSSizes lists all VPS sizes in the catalog (superadmin only)
func (s *Service) ListVPSSizes(ctx context.Context, req *connect.Request[superadminv1.ListVPSSizesRequest]) (*connect.Response[superadminv1.ListVPSSizesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	region := req.Msg.GetRegion()
	includeUnavailable := req.Msg.GetIncludeUnavailable()

	sizes, err := database.ListAllVPSSizeCatalog(region, includeUnavailable)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to list VPS sizes: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list VPS sizes: %w", err))
	}

	protoSizes := make([]*commonv1.VPSSize, 0, len(sizes))
	for _, size := range sizes {
		sizeProto := &commonv1.VPSSize{
			Id:                  size.ID,
			Name:                size.Name,
			CpuCores:            size.CPUCores,
			MemoryBytes:         size.MemoryBytes,
			DiskBytes:           size.DiskBytes,
			BandwidthBytesMonth: size.BandwidthBytesMonth,
			MinimumPaymentCents: size.MinimumPaymentCents,
			Available:           size.Available,
			Region:              size.Region,
			CreatedAt:           timestamppb.New(size.CreatedAt),
			UpdatedAt:           timestamppb.New(size.UpdatedAt),
		}
		if size.Description != "" {
			sizeProto.Description = &size.Description
		}
		protoSizes = append(protoSizes, sizeProto)
	}

	return connect.NewResponse(&superadminv1.ListVPSSizesResponse{
		Sizes: protoSizes,
	}), nil
}

// CreateVPSSize creates a new VPS size in the catalog (superadmin only)
func (s *Service) CreateVPSSize(ctx context.Context, req *connect.Request[superadminv1.CreateVPSSizeRequest]) (*connect.Response[superadminv1.CreateVPSSizeResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Validate required fields
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	if req.Msg.GetCpuCores() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cpu_cores must be greater than 0"))
	}
	if req.Msg.GetMemoryBytes() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("memory_bytes must be greater than 0"))
	}
	if req.Msg.GetDiskBytes() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("disk_bytes must be greater than 0"))
	}

	// Check if size already exists
	var existing database.VPSSizeCatalog
	if err := database.DB.Where("id = ?", req.Msg.GetId()).First(&existing).Error; err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("VPS size with id %s already exists", req.Msg.GetId()))
	} else if err != gorm.ErrRecordNotFound {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing size: %w", err))
	}

	// Create new size
	size := &database.VPSSizeCatalog{
		ID:                  req.Msg.GetId(),
		Name:                req.Msg.GetName(),
		Description:         req.Msg.GetDescription(),
		CPUCores:            req.Msg.GetCpuCores(),
		MemoryBytes:         req.Msg.GetMemoryBytes(),
		DiskBytes:           req.Msg.GetDiskBytes(),
		BandwidthBytesMonth: req.Msg.GetBandwidthBytesMonth(),
		MinimumPaymentCents: req.Msg.GetMinimumPaymentCents(),
		Available:           req.Msg.GetAvailable(),
		Region:              req.Msg.GetRegion(),
	}

	if err := database.CreateVPSSizeCatalog(size); err != nil {
		logger.Error("[SuperAdmin] Failed to create VPS size: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS size: %w", err))
	}

	sizeProto := &commonv1.VPSSize{
		Id:                  size.ID,
		Name:                size.Name,
		CpuCores:            size.CPUCores,
		MemoryBytes:         size.MemoryBytes,
		DiskBytes:           size.DiskBytes,
		BandwidthBytesMonth: size.BandwidthBytesMonth,
			MinimumPaymentCents: size.MinimumPaymentCents,
		Available:           size.Available,
		Region:              size.Region,
		CreatedAt:           timestamppb.New(size.CreatedAt),
		UpdatedAt:           timestamppb.New(size.UpdatedAt),
	}
	if size.Description != "" {
		sizeProto.Description = &size.Description
	}
	return connect.NewResponse(&superadminv1.CreateVPSSizeResponse{
		Size: sizeProto,
	}), nil
}

// UpdateVPSSize updates an existing VPS size in the catalog (superadmin only)
func (s *Service) UpdateVPSSize(ctx context.Context, req *connect.Request[superadminv1.UpdateVPSSizeRequest]) (*connect.Response[superadminv1.UpdateVPSSizeResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}

	// Check if size exists
	var existing database.VPSSizeCatalog
	if err := database.DB.Where("id = ?", req.Msg.GetId()).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS size with id %s not found", req.Msg.GetId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find VPS size: %w", err))
	}

	// Build update map
	updates := make(map[string]interface{})
	if req.Msg.Name != nil {
		updates["name"] = req.Msg.GetName()
	}
	if req.Msg.Description != nil {
		updates["description"] = req.Msg.GetDescription()
	}
	if req.Msg.CpuCores != nil {
		if req.Msg.GetCpuCores() <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cpu_cores must be greater than 0"))
		}
		updates["cpu_cores"] = req.Msg.GetCpuCores()
	}
	if req.Msg.MemoryBytes != nil {
		if req.Msg.GetMemoryBytes() <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("memory_bytes must be greater than 0"))
		}
		updates["memory_bytes"] = req.Msg.GetMemoryBytes()
	}
	if req.Msg.DiskBytes != nil {
		if req.Msg.GetDiskBytes() <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("disk_bytes must be greater than 0"))
		}
		updates["disk_bytes"] = req.Msg.GetDiskBytes()
	}
	if req.Msg.BandwidthBytesMonth != nil {
		updates["bandwidth_bytes_month"] = req.Msg.GetBandwidthBytesMonth()
	}
	if req.Msg.MinimumPaymentCents != nil {
		updates["minimum_payment_cents"] = req.Msg.GetMinimumPaymentCents()
	}
	if req.Msg.Available != nil {
		updates["available"] = req.Msg.GetAvailable()
	}
	if req.Msg.Region != nil {
		updates["region"] = req.Msg.GetRegion()
	}

	if len(updates) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("no fields to update"))
	}

	// Update size
	if err := database.UpdateVPSSizeCatalog(req.Msg.GetId(), updates); err != nil {
		logger.Error("[SuperAdmin] Failed to update VPS size: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update VPS size: %w", err))
	}

	// Fetch updated size
	var updated database.VPSSizeCatalog
	if err := database.DB.Where("id = ?", req.Msg.GetId()).First(&updated).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch updated size: %w", err))
	}

	sizeProto := &commonv1.VPSSize{
		Id:                  updated.ID,
		Name:                updated.Name,
		CpuCores:            updated.CPUCores,
		MemoryBytes:         updated.MemoryBytes,
		DiskBytes:           updated.DiskBytes,
		BandwidthBytesMonth: updated.BandwidthBytesMonth,
			MinimumPaymentCents: updated.MinimumPaymentCents,
		Available:           updated.Available,
		Region:              updated.Region,
		CreatedAt:           timestamppb.New(updated.CreatedAt),
		UpdatedAt:           timestamppb.New(updated.UpdatedAt),
	}
	if updated.Description != "" {
		sizeProto.Description = &updated.Description
	}
	return connect.NewResponse(&superadminv1.UpdateVPSSizeResponse{
		Size: sizeProto,
	}), nil
}

// DeleteVPSSize deletes a VPS size from the catalog (superadmin only)
func (s *Service) DeleteVPSSize(ctx context.Context, req *connect.Request[superadminv1.DeleteVPSSizeRequest]) (*connect.Response[superadminv1.DeleteVPSSizeResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}

	// Check if size exists
	var existing database.VPSSizeCatalog
	if err := database.DB.Where("id = ?", req.Msg.GetId()).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS size with id %s not found", req.Msg.GetId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find VPS size: %w", err))
	}

	// Check if any VPS instances are using this size
	var count int64
	if err := database.DB.Table("vps_instances").Where("size = ? AND deleted_at IS NULL", req.Msg.GetId()).Count(&count).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check VPS instances: %w", err))
	}
	if count > 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cannot delete VPS size: %d VPS instances are using this size", count))
	}

	// Delete size
	if err := database.DeleteVPSSizeCatalog(req.Msg.GetId()); err != nil {
		logger.Error("[SuperAdmin] Failed to delete VPS size: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete VPS size: %w", err))
	}

	return connect.NewResponse(&superadminv1.DeleteVPSSizeResponse{
		Success: true,
	}), nil
}

// SuperadminGetVPS gets a VPS instance by ID (superadmin - bypasses organization checks)
func (s *Service) SuperadminGetVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminGetVPSRequest]) (*connect.Response[superadminv1.SuperadminGetVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ?", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Get organization name
	var org database.Organization
	orgName := "Unknown"
	if err := database.DB.Where("id = ?", vps.OrganizationID).First(&org).Error; err == nil {
		orgName = org.Name
	}

	// Convert to proto
	protoVPS := convertVPSInstanceToProto(&vps)

	return connect.NewResponse(&superadminv1.SuperadminGetVPSResponse{
		Vps:              protoVPS,
		OrganizationName: orgName,
	}), nil
}

// SuperadminResizeVPS resizes a VPS instance to a new size (superadmin only)
func (s *Service) SuperadminResizeVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminResizeVPSRequest]) (*connect.Response[superadminv1.SuperadminResizeVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	newSize := req.Msg.GetNewSize()
	growDisk := req.Msg.GetGrowDisk()
	applyCloudInit := req.Msg.GetApplyCloudinit()

	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}
	if newSize == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("new_size is required"))
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID (not provisioned yet)"))
	}

	// Get new size from catalog
	newSizeCatalog, err := database.GetVPSSizeCatalog(newSize, vps.Region)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid VPS size: %w", err))
	}

	// Check if resize is needed
	if vps.Size == newSize && vps.CPUCores == newSizeCatalog.CPUCores && vps.MemoryBytes == newSizeCatalog.MemoryBytes && vps.DiskBytes == newSizeCatalog.DiskBytes {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("VPS is already at the requested size"))
	}

	// Get Proxmox configuration
	proxmoxConfig, err := vpsorch.GetProxmoxConfig()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox config: %w", err))
	}

	// Create Proxmox client
	proxmoxClient, err := vpsorch.NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create Proxmox client: %w", err))
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid VM ID: %s", *vps.InstanceID))
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find Proxmox node: %w", err))
	}
	nodeName := nodes[0]

	// Stop VM if running (required for resize)
	status, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VM status: %w", err))
	}

	wasRunning := status == "running"
	if wasRunning {
		logger.Info("[SuperAdmin] Stopping VM %d for resize", vmIDInt)
		if err := proxmoxClient.StopVM(ctx, nodeName, vmIDInt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop VM: %w", err))
		}
		// Wait for VM to stop
		for i := 0; i < 30; i++ {
			time.Sleep(2 * time.Second)
			status, _ := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
			if status == "stopped" {
				break
			}
		}
	}

	// Update CPU and memory
	vmConfig := make(map[string]interface{})
	if vps.CPUCores != newSizeCatalog.CPUCores {
		vmConfig["cores"] = newSizeCatalog.CPUCores
		logger.Info("[SuperAdmin] Resizing CPU from %d to %d cores", vps.CPUCores, newSizeCatalog.CPUCores)
	}
	if vps.MemoryBytes != newSizeCatalog.MemoryBytes {
		vmConfig["memory"] = int(newSizeCatalog.MemoryBytes / (1024 * 1024)) // Convert to MB
		logger.Info("[SuperAdmin] Resizing memory from %d to %d bytes", vps.MemoryBytes, newSizeCatalog.MemoryBytes)
	}

	if len(vmConfig) > 0 {
		// Update VM config via Proxmox API
		if err := proxmoxClient.UpdateVMConfig(ctx, nodeName, vmIDInt, vmConfig); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update VM config: %w", err))
		}
		logger.Info("[SuperAdmin] Successfully updated VM config: %v", vmConfig)
	}

	// Resize disk if needed and requested
	if growDisk && vps.DiskBytes != newSizeCatalog.DiskBytes {
		// Find disk key
		vmConfigAfter, err := proxmoxClient.GetVMConfig(ctx, nodeName, vmIDInt)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VM config: %w", err))
		}

		diskKeys := []string{"scsi0", "virtio0", "sata0", "ide0"}
		var diskKey string
		for _, key := range diskKeys {
			if disk, ok := vmConfigAfter[key].(string); ok && disk != "" {
				diskKey = key
				break
			}
		}

		if diskKey == "" {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("could not find disk to resize"))
		}

		newDiskSizeGB := newSizeCatalog.DiskBytes / (1024 * 1024 * 1024)
		logger.Info("[SuperAdmin] Resizing disk %s from %dGB to %dGB", diskKey, vps.DiskBytes/(1024*1024*1024), newDiskSizeGB)

		// Resize disk via Proxmox API
		if err := proxmoxClient.ResizeDisk(ctx, nodeName, vmIDInt, diskKey, newDiskSizeGB); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to resize disk: %w", err))
		}
		logger.Info("[SuperAdmin] Successfully resized disk %s to %dGB", diskKey, newDiskSizeGB)

		// If cloud-init should be applied, update it to grow the filesystem
		if applyCloudInit {
			// Use VPS config service to load existing config
			// Create a minimal ConfigService instance (VPSManager not needed for these operations)
			configService := vpsservice.NewConfigService(nil)
			cloudInitConfig, err := configService.LoadCloudInitConfig(ctx, &vps)
			if err != nil {
				logger.Warn("[SuperAdmin] Failed to load cloud-init config: %v, using default", err)
				cloudInitConfig = &vpsorch.CloudInitConfig{
					PackageUpdate:    boolPtr(true),
					PackageUpgrade:   boolPtr(false),
					SSHInstallServer: boolPtr(true),
					SSHAllowPW:       boolPtr(true),
					Runcmd:           []string{},
				}
			}

			// Add growpart and resize2fs commands to runcmd if not already present
			growPartCmd := "growpart /dev/sda 1 || growpart /dev/vda 1 || true"
			resizeFsCmd := "resize2fs /dev/sda1 || resize2fs /dev/vda1 || true"
			
			// Check if commands already exist
			hasGrowPart := false
			hasResizeFs := false
			for _, cmd := range cloudInitConfig.Runcmd {
				if strings.Contains(cmd, "growpart") {
					hasGrowPart = true
				}
				if strings.Contains(cmd, "resize2fs") {
					hasResizeFs = true
				}
			}

			if !hasGrowPart {
				cloudInitConfig.Runcmd = append([]string{growPartCmd}, cloudInitConfig.Runcmd...)
			}
			if !hasResizeFs {
				cloudInitConfig.Runcmd = append([]string{resizeFsCmd}, cloudInitConfig.Runcmd...)
			}

			// Save cloud-init config using ConfigService
			if err := configService.SaveCloudInitConfig(ctx, &vps, cloudInitConfig); err != nil {
				logger.Warn("[SuperAdmin] Failed to update cloud-init config: %v", err)
			} else {
				logger.Info("[SuperAdmin] Updated cloud-init config with disk growth commands")
			}
		}
	}

	// Update database
	vps.Size = newSize
	vps.CPUCores = newSizeCatalog.CPUCores
	vps.MemoryBytes = newSizeCatalog.MemoryBytes
	if growDisk {
		vps.DiskBytes = newSizeCatalog.DiskBytes
	}
	vps.UpdatedAt = time.Now()

	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update VPS in database: %w", err))
	}

	// Restart VM if it was running
	if wasRunning {
		logger.Info("[SuperAdmin] Starting VM %d after resize", vmIDInt)
		if err := proxmoxClient.StartVM(ctx, nodeName, vmIDInt); err != nil {
			logger.Warn("[SuperAdmin] Failed to start VM after resize: %v", err)
		}
	}

	message := fmt.Sprintf("VPS resized successfully. CPU: %d cores, Memory: %s, Disk: %s", 
		newSizeCatalog.CPUCores,
		formatBytes(newSizeCatalog.MemoryBytes),
		formatBytes(newSizeCatalog.DiskBytes))
	if growDisk && applyCloudInit {
		message += " Disk will be grown on next boot via cloud-init."
	}

	return connect.NewResponse(&superadminv1.SuperadminResizeVPSResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: message,
	}), nil
}

// SuperadminSuspendVPS suspends a VPS instance (superadmin only)
func (s *Service) SuperadminSuspendVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminSuspendVPSRequest]) (*connect.Response[superadminv1.SuperadminSuspendVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID (not provisioned yet)"))
	}

	// Update status to SUSPENDED
	vps.Status = int32(vpsv1.VPSStatus_SUSPENDED)
	vps.UpdatedAt = time.Now()

	// Store suspension reason in metadata if provided
	if reason := req.Msg.GetReason(); reason != "" {
		var metadata map[string]string
		if vps.Metadata != "" {
			json.Unmarshal([]byte(vps.Metadata), &metadata)
		}
		if metadata == nil {
			metadata = make(map[string]string)
		}
		metadata["suspended_reason"] = reason
		metadata["suspended_at"] = time.Now().Format(time.RFC3339)
		metadataJSON, _ := json.Marshal(metadata)
		vps.Metadata = string(metadataJSON)
	}

	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to suspend VPS: %w", err))
	}

	message := "VPS suspended successfully"
	if reason := req.Msg.GetReason(); reason != "" {
		message += fmt.Sprintf(" (reason: %s)", reason)
	}

	return connect.NewResponse(&superadminv1.SuperadminSuspendVPSResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: message,
	}), nil
}

// SuperadminUnsuspendVPS unsuspends a VPS instance (superadmin only)
func (s *Service) SuperadminUnsuspendVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminUnsuspendVPSRequest]) (*connect.Response[superadminv1.SuperadminUnsuspendVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Update status to STOPPED (was suspended)
	vps.Status = int32(vpsv1.VPSStatus_STOPPED)
	vps.UpdatedAt = time.Now()

	// Remove suspension metadata
	if vps.Metadata != "" {
		var metadata map[string]string
		if err := json.Unmarshal([]byte(vps.Metadata), &metadata); err == nil {
			delete(metadata, "suspended_reason")
			delete(metadata, "suspended_at")
			metadataJSON, _ := json.Marshal(metadata)
			vps.Metadata = string(metadataJSON)
		}
	}

	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unsuspend VPS: %w", err))
	}

	return connect.NewResponse(&superadminv1.SuperadminUnsuspendVPSResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: "VPS unsuspended successfully",
	}), nil
}

// SuperadminUpdateVPSCloudInit updates the cloud-init configuration for a VPS (superadmin only)
func (s *Service) SuperadminUpdateVPSCloudInit(ctx context.Context, req *connect.Request[superadminv1.SuperadminUpdateVPSCloudInitRequest]) (*connect.Response[superadminv1.SuperadminUpdateVPSCloudInitResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID (not provisioned yet)"))
	}

	// Convert proto cloud-init to orchestrator format
	cloudInitProto := req.Msg.GetCloudInit()
	if cloudInitProto == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cloud_init is required"))
	}

	// Convert proto to orchestrator format
	cloudInitConfig := protoToCloudInitConfigForSuperadmin(cloudInitProto)

	// Use VPS config service to save cloud-init config
	// Create a minimal ConfigService instance (VPSManager not needed for these operations)
	configService := vpsservice.NewConfigService(nil)
	if err := configService.SaveCloudInitConfig(ctx, &vps, cloudInitConfig); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update cloud-init config: %w", err))
	}
	
	logger.Info("[SuperAdmin] Successfully updated cloud-init config for VPS %s", vpsID)

	message := "Cloud-init configuration updated. Changes will take effect on the next reboot or when cloud-init is re-run."
	if req.Msg.GetGrowDiskIfNeeded() {
		message += " Disk growth commands have been added if needed."
	}

	return connect.NewResponse(&superadminv1.SuperadminUpdateVPSCloudInitResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: message,
	}), nil
}

// SuperadminForceStopVPS force stops a VPS instance (superadmin only)
func (s *Service) SuperadminForceStopVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminForceStopVPSRequest]) (*connect.Response[superadminv1.SuperadminForceStopVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID (not provisioned yet)"))
	}

	// Get VPS manager and force stop
	vpsManager, err := vpsorch.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}

	// Force stop (use force=true for forceful stop)
	if err := vpsManager.StopVPS(ctx, vpsID, true); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to force stop VPS: %w", err))
	}

	// Refresh VPS status
	if err := database.DB.Where("id = ?", vpsID).First(&vps).Error; err == nil {
		vps.Status = int32(vpsv1.VPSStatus_STOPPED)
		vps.UpdatedAt = time.Now()
		database.DB.Save(&vps)
	}

	return connect.NewResponse(&superadminv1.SuperadminForceStopVPSResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: "VPS force stopped successfully",
	}), nil
}

// SuperadminForceDeleteVPS force deletes a VPS instance (superadmin only)
func (s *Service) SuperadminForceDeleteVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminForceDeleteVPSRequest]) (*connect.Response[superadminv1.SuperadminForceDeleteVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	hardDelete := req.Msg.GetHardDelete()

	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ?", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Get VPS manager
	vpsManager, err := vpsorch.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}

	// Delete VPS (hard delete handled by setting deleted_at or actual deletion)
	if err := vpsManager.DeleteVPS(ctx, vpsID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete VPS: %w", err))
	}
	
	// If hard delete, also remove from database permanently
	if hardDelete {
		if err := database.DB.Unscoped().Delete(&vps).Error; err != nil {
			logger.Warn("[SuperAdmin] Failed to hard delete VPS from database: %v", err)
		}
	}

	message := "VPS deleted successfully"
	if hardDelete {
		message = "VPS permanently deleted (hard delete)"
	}

	return connect.NewResponse(&superadminv1.SuperadminForceDeleteVPSResponse{
		Success: true,
		Message: message,
	}), nil
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// protoToCloudInitConfigForSuperadmin converts proto CloudInitConfig to orchestrator format
func protoToCloudInitConfigForSuperadmin(proto *vpsv1.CloudInitConfig) *vpsorch.CloudInitConfig {
	if proto == nil {
		return nil
	}

	config := &vpsorch.CloudInitConfig{
		Users:      make([]vpsorch.CloudInitUser, 0, len(proto.Users)),
		Packages:   proto.Packages,
		Runcmd:     proto.Runcmd,
		WriteFiles: make([]vpsorch.CloudInitWriteFile, 0, len(proto.WriteFiles)),
	}

	// Convert users
	for _, userProto := range proto.Users {
		user := vpsorch.CloudInitUser{
			Name:              userProto.GetName(),
			SSHAuthorizedKeys: userProto.SshAuthorizedKeys,
			Groups:            userProto.Groups,
		}

		if userProto.Password != nil {
			pass := userProto.GetPassword()
			user.Password = &pass
		}
		if userProto.Sudo != nil {
			sudo := userProto.GetSudo()
			user.Sudo = &sudo
		}
		if userProto.SudoNopasswd != nil {
			sudoNopasswd := userProto.GetSudoNopasswd()
			user.SudoNopasswd = &sudoNopasswd
		}
		if userProto.Shell != nil {
			shell := userProto.GetShell()
			user.Shell = &shell
		}
		if userProto.LockPasswd != nil {
			lockPasswd := userProto.GetLockPasswd()
			user.LockPasswd = &lockPasswd
		}
		if userProto.Gecos != nil {
			gecos := userProto.GetGecos()
			user.Gecos = &gecos
		}

		config.Users = append(config.Users, user)
	}

	// Convert system configuration
	if proto.Hostname != nil {
		hostname := proto.GetHostname()
		config.Hostname = &hostname
	}
	if proto.Timezone != nil {
		timezone := proto.GetTimezone()
		config.Timezone = &timezone
	}
	if proto.Locale != nil {
		locale := proto.GetLocale()
		config.Locale = &locale
	}

	// Convert package management
	if proto.PackageUpdate != nil {
		packageUpdate := proto.GetPackageUpdate()
		config.PackageUpdate = &packageUpdate
	}
	if proto.PackageUpgrade != nil {
		packageUpgrade := proto.GetPackageUpgrade()
		config.PackageUpgrade = &packageUpgrade
	}

	// Convert SSH configuration
	if proto.SshInstallServer != nil {
		sshInstallServer := proto.GetSshInstallServer()
		config.SSHInstallServer = &sshInstallServer
	}
	if proto.SshAllowPw != nil {
		sshAllowPW := proto.GetSshAllowPw()
		config.SSHAllowPW = &sshAllowPW
	}

	// Convert write files
	for _, fileProto := range proto.WriteFiles {
		file := vpsorch.CloudInitWriteFile{
			Path:    fileProto.GetPath(),
			Content: fileProto.GetContent(),
		}

		if fileProto.Owner != nil {
			owner := fileProto.GetOwner()
			file.Owner = &owner
		}
		if fileProto.Permissions != nil {
			permissions := fileProto.GetPermissions()
			file.Permissions = &permissions
		}
		if fileProto.Append != nil {
			append := fileProto.GetAppend()
			file.Append = &append
		}
		if fileProto.Defer != nil {
			deferVal := fileProto.GetDefer()
			file.Defer = &deferVal
		}

		config.WriteFiles = append(config.WriteFiles, file)
	}

	return config
}

// ListNodes lists all nodes in the cluster
func (s *Service) ListNodes(ctx context.Context, req *connect.Request[superadminv1.ListNodesRequest]) (*connect.Response[superadminv1.ListNodesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	var nodes []database.NodeMetadata
	query := database.DB

	// Apply filters
	if req.Msg.Role != nil {
		query = query.Where("role = ?", req.Msg.GetRole())
	}
	if req.Msg.Availability != nil {
		query = query.Where("availability = ?", req.Msg.GetAvailability())
	}
	if req.Msg.Status != nil {
		query = query.Where("status = ?", req.Msg.GetStatus())
	}
	if req.Msg.Region != nil {
		query = query.Where("region = ?", req.Msg.GetRegion())
	}

	if err := query.Find(&nodes).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to list nodes: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list nodes: %w", err))
	}

	nodeInfos := make([]*superadminv1.NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		nodeInfo := convertNodeToProto(&node)
		nodeInfos = append(nodeInfos, nodeInfo)
	}

	return connect.NewResponse(&superadminv1.ListNodesResponse{
		Nodes: nodeInfos,
	}), nil
}

// GetNode gets a specific node by ID
func (s *Service) GetNode(ctx context.Context, req *connect.Request[superadminv1.GetNodeRequest]) (*connect.Response[superadminv1.GetNodeResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	var node database.NodeMetadata
	if err := database.DB.Where("id = ?", req.Msg.GetNodeId()).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("node not found"))
		}
		logger.Error("[SuperAdmin] Failed to get node: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get node: %w", err))
	}

	nodeInfo := convertNodeToProto(&node)
	return connect.NewResponse(&superadminv1.GetNodeResponse{
		Node: nodeInfo,
	}), nil
}

// UpdateNodeConfig updates node-specific configuration
func (s *Service) UpdateNodeConfig(ctx context.Context, req *connect.Request[superadminv1.UpdateNodeConfigRequest]) (*connect.Response[superadminv1.UpdateNodeConfigResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	var node database.NodeMetadata
	if err := database.DB.Where("id = ?", req.Msg.GetNodeId()).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("node not found"))
		}
		logger.Error("[SuperAdmin] Failed to get node: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get node: %w", err))
	}

	// Check if this is a compose deployment node (node ID starts with "local-")
	// Subdomain configuration is only allowed for Swarm stack services
	isComposeDeployment := strings.HasPrefix(node.ID, "local-")
	
	// Reject subdomain-related config updates for compose deployments
	if isComposeDeployment {
		if req.Msg.Subdomain != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("node subdomain configuration is not available for compose deployments. Use NODE_SUBDOMAIN environment variable instead"))
		}
		if req.Msg.UseNodeSpecificDomains != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("node-specific domains configuration is not available for compose deployments. Use environment variables instead"))
		}
		if req.Msg.ServiceDomainPattern != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("service domain pattern configuration is not available for compose deployments. Use environment variables instead"))
		}
	}

	// Parse existing labels
	var labels map[string]interface{}
	if node.Labels != "" {
		if err := json.Unmarshal([]byte(node.Labels), &labels); err != nil {
			logger.Warn("[SuperAdmin] Failed to parse existing labels for node %s: %v", node.ID, err)
			labels = make(map[string]interface{})
		}
	} else {
		labels = make(map[string]interface{})
	}

	// Update configuration in labels (only for Swarm nodes)
	if req.Msg.Subdomain != nil {
		labels["obiente.subdomain"] = req.Msg.GetSubdomain()
		labels["subdomain"] = req.Msg.GetSubdomain() // Also set for backwards compatibility
	}
	if req.Msg.UseNodeSpecificDomains != nil {
		labels["obiente.use_node_specific_domains"] = req.Msg.GetUseNodeSpecificDomains()
	}
	if req.Msg.ServiceDomainPattern != nil {
		labels["obiente.service_domain_pattern"] = req.Msg.GetServiceDomainPattern()
	}
	if req.Msg.Region != nil {
		node.Region = req.Msg.GetRegion()
	}
	if req.Msg.MaxDeployments != nil {
		node.MaxDeployments = int(req.Msg.GetMaxDeployments())
	}

	// Merge custom labels
	if req.Msg.CustomLabels != nil {
		for k, v := range req.Msg.CustomLabels {
			labels[k] = v
		}
	}

	// Serialize labels back to JSON
	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to marshal labels: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal labels: %w", err))
	}
	node.Labels = string(labelsJSON)

	// Save node
	if err := database.DB.Save(&node).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to update node: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update node: %w", err))
	}

	logger.Info("[SuperAdmin] Updated node configuration for node %s (%s)", node.ID, node.Hostname)
	nodeInfo := convertNodeToProto(&node)
	return connect.NewResponse(&superadminv1.UpdateNodeConfigResponse{
		Node:    nodeInfo,
		Message: "Node configuration updated successfully",
	}), nil
}

// convertNodeToProto converts a database NodeMetadata to a proto NodeInfo
func convertNodeToProto(node *database.NodeMetadata) *superadminv1.NodeInfo {
	nodeInfo := &superadminv1.NodeInfo{
		Id:              node.ID,
		Hostname:        node.Hostname,
		Ip:              node.IP,
		Role:            node.Role,
		Availability:    node.Availability,
		Status:          node.Status,
		TotalCpu:        int32(node.TotalCPU),
		TotalMemory:     node.TotalMemory,
		UsedCpu:         node.UsedCPU,
		UsedMemory:      node.UsedMemory,
		DeploymentCount: int32(node.DeploymentCount),
		MaxDeployments:  int32(node.MaxDeployments),
		LastHeartbeat:   timestamppb.New(node.LastHeartbeat),
		CreatedAt:       timestamppb.New(node.CreatedAt),
		UpdatedAt:       timestamppb.New(node.UpdatedAt),
	}

	if node.Region != "" {
		nodeInfo.Region = proto.String(node.Region)
	}

	// Parse node configuration from labels
	config := &superadminv1.NodeConfig{
		CustomLabels: make(map[string]string),
	}

	if node.Labels != "" {
		var labels map[string]interface{}
		if err := json.Unmarshal([]byte(node.Labels), &labels); err == nil {
			// Extract node configuration
			if subdomain, ok := labels["obiente.subdomain"].(string); ok && subdomain != "" {
				config.Subdomain = proto.String(subdomain)
			} else if subdomain, ok := labels["subdomain"].(string); ok && subdomain != "" {
				config.Subdomain = proto.String(subdomain)
			}

			if useNodeSpecific, ok := labels["obiente.use_node_specific_domains"].(bool); ok {
				config.UseNodeSpecificDomains = proto.Bool(useNodeSpecific)
			}

			if pattern, ok := labels["obiente.service_domain_pattern"].(string); ok && pattern != "" {
				config.ServiceDomainPattern = proto.String(pattern)
			}

			// Extract all custom labels (excluding obiente.* keys)
			for k, v := range labels {
				if !strings.HasPrefix(k, "obiente.") && k != "subdomain" {
					if strVal, ok := v.(string); ok {
						config.CustomLabels[k] = strVal
					}
				}
			}
		}
	}

	nodeInfo.Config = config
	return nodeInfo
}