package gameservers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

const (
	gameServerRouteDomainSuffix = "my.obiente.cloud"

	gameServerDomainStatusPending  = "pending"
	gameServerDomainStatusVerified = "verified"
	gameServerDomainStatusFailed   = "failed"
	gameServerDomainStatusExpired  = "expired"
)

var gameServerDNSFallbackResolvers = []string{"1.1.1.1:53", "8.8.8.8:53"}

func (s *Service) GetGameServerHTTPRoutes(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerHTTPRoutesRequest]) (*connect.Response[gameserversv1.GetGameServerHTTPRoutesResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	orgID := req.Msg.GetOrganizationId()

	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	if err := s.checkGameServerPermission(ctx, gameServerID, "read"); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}
	if orgID != "" && gameServer.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("game server does not belong to organization"))
	}

	routes, err := database.GetGameServerHTTPRoutes(gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load routes: %w", err))
	}

	protoRoutes := make([]*gameserversv1.GameServerHTTPRoute, 0, len(routes))
	for _, route := range routes {
		protoRoutes = append(protoRoutes, dbGameServerRouteToProto(route))
	}

	return connect.NewResponse(&gameserversv1.GetGameServerHTTPRoutesResponse{Routes: protoRoutes}), nil
}

func (s *Service) UpsertGameServerHTTPRoute(ctx context.Context, req *connect.Request[gameserversv1.UpsertGameServerHTTPRouteRequest]) (*connect.Response[gameserversv1.UpsertGameServerHTTPRouteResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	orgID := req.Msg.GetOrganizationId()

	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	if err := s.checkGameServerPermission(ctx, gameServerID, "update"); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}
	if orgID != "" && gameServer.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("game server does not belong to organization"))
	}

	domain := normalizeRouteDomain(req.Msg.GetDomain())
	if domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("domain is required"))
	}

	availableDomains, err := getAvailableDomainsForGameServer(gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to resolve available domains: %w", err))
	}
	if _, ok := availableDomains[domain]; !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("domain %s is not available for this game server. You can only use the default domain or verified custom domains", domain))
	}

	if err := checkGameServerDomainConflict(gameServerID, domain); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	targetPort := int(req.Msg.GetTargetPort())
	if targetPort < 1 || targetPort > 65535 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("target_port must be between 1 and 65535"))
	}

	protocol := strings.ToLower(strings.TrimSpace(req.Msg.GetProtocol()))
	if protocol == "" {
		protocol = "http"
	}
	if protocol != "http" && protocol != "https" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("protocol must be either http or https"))
	}

	sslEnabled := req.Msg.GetSslEnabled()
	if protocol == "http" {
		sslEnabled = false
	} else if protocol == "https" {
		sslEnabled = true
	}

	sslCertResolver := strings.TrimSpace(req.Msg.GetSslCertResolver())
	if protocol == "http" {
		sslCertResolver = ""
	} else if sslCertResolver == "" {
		sslCertResolver = "letsencrypt"
	}

	routeID := strings.TrimSpace(req.Msg.GetRouteId())
	if routeID == "" {
		routeID = fmt.Sprintf("gs-route-%s-%s-%d", gameServerID, sanitizeRouteIDPart(domain), targetPort)
	}

	route := &database.GameServerHTTPRoute{
		ID:              routeID,
		GameServerID:    gameServerID,
		Domain:          domain,
		PathPrefix:      req.Msg.GetPathPrefix(),
		TargetPort:      targetPort,
		Protocol:        protocol,
		SSLEnabled:      sslEnabled,
		SSLCertResolver: sslCertResolver,
		UpdatedAt:       time.Now(),
	}

	if err := database.UpsertGameServerHTTPRoute(route); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save route: %w", err))
	}

	manager, err := s.getGameServerManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server manager: %w", err))
	}
	if err := manager.ApplyGameServerHTTPRoutes(ctx, gameServerID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("route saved but failed to apply to container: %w", err))
	}

	return connect.NewResponse(&gameserversv1.UpsertGameServerHTTPRouteResponse{
		Route: dbGameServerRouteToProto(*route),
	}), nil
}

func (s *Service) DeleteGameServerHTTPRoute(ctx context.Context, req *connect.Request[gameserversv1.DeleteGameServerHTTPRouteRequest]) (*connect.Response[gameserversv1.DeleteGameServerHTTPRouteResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	orgID := req.Msg.GetOrganizationId()
	routeID := req.Msg.GetRouteId()

	if gameServerID == "" || routeID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id and route_id are required"))
	}

	if err := s.checkGameServerPermission(ctx, gameServerID, "update"); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}
	if orgID != "" && gameServer.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("game server does not belong to organization"))
	}

	route, err := database.GetGameServerHTTPRouteByID(routeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("route %s not found", routeID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load route: %w", err))
	}
	if route.GameServerID != gameServerID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("route does not belong to game server"))
	}

	if err := database.DeleteGameServerHTTPRoute(routeID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete route: %w", err))
	}

	manager, err := s.getGameServerManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server manager: %w", err))
	}
	if err := manager.ApplyGameServerHTTPRoutes(ctx, gameServerID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("route deleted but failed to apply to container: %w", err))
	}

	return connect.NewResponse(&gameserversv1.DeleteGameServerHTTPRouteResponse{Success: true}), nil
}

func (s *Service) GetGameServerDomainVerificationToken(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerDomainVerificationTokenRequest]) (*connect.Response[gameserversv1.GetGameServerDomainVerificationTokenResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	orgID := req.Msg.GetOrganizationId()
	domain := normalizeRouteDomain(req.Msg.GetDomain())

	if gameServerID == "" || domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id and domain are required"))
	}
	if domain == normalizeDefaultGameServerDomain(gameServerID) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("default domain does not require verification"))
	}

	if err := s.checkGameServerPermission(ctx, gameServerID, "update"); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}
	if orgID != "" && gameServer.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("game server does not belong to organization"))
	}

	verification, err := getOrCreateGameServerDomainVerification(gameServerID, domain)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&gameserversv1.GetGameServerDomainVerificationTokenResponse{
		Domain:         domain,
		Token:          verification.Token,
		TxtRecordName:  fmt.Sprintf("_obiente-verification.%s", domain),
		TxtRecordValue: fmt.Sprintf("obiente-verification=%s", verification.Token),
		Status:         verification.Status,
	}), nil
}

func (s *Service) VerifyGameServerDomain(ctx context.Context, req *connect.Request[gameserversv1.VerifyGameServerDomainRequest]) (*connect.Response[gameserversv1.VerifyGameServerDomainResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	orgID := req.Msg.GetOrganizationId()
	domain := normalizeRouteDomain(req.Msg.GetDomain())

	if gameServerID == "" || domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id and domain are required"))
	}
	if domain == normalizeDefaultGameServerDomain(gameServerID) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("default domain does not require verification"))
	}

	if err := s.checkGameServerPermission(ctx, gameServerID, "update"); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}
	if orgID != "" && gameServer.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("game server does not belong to organization"))
	}

	if err := verifyGameServerDomainOwnership(gameServerID, domain); err != nil {
		errMsg := err.Error()
		return connect.NewResponse(&gameserversv1.VerifyGameServerDomainResponse{
			Domain:   domain,
			Verified: false,
			Status:   gameServerDomainStatusFailed,
			Message:  &errMsg,
		}), nil
	}

	verification, err := database.GetGameServerDomainVerification(gameServerID, domain)
	status := gameServerDomainStatusVerified
	if err == nil {
		status = verification.Status
	}

	return connect.NewResponse(&gameserversv1.VerifyGameServerDomainResponse{
		Domain:   domain,
		Verified: true,
		Status:   status,
	}), nil
}

func dbGameServerRouteToProto(route database.GameServerHTTPRoute) *gameserversv1.GameServerHTTPRoute {
	protoRoute := &gameserversv1.GameServerHTTPRoute{
		Id:           route.ID,
		GameServerId: route.GameServerID,
		Domain:       route.Domain,
		PathPrefix:   route.PathPrefix,
		TargetPort:   int32(route.TargetPort),
		Protocol:     route.Protocol,
		SslEnabled:   route.SSLEnabled,
	}
	if strings.TrimSpace(route.SSLCertResolver) != "" {
		resolver := route.SSLCertResolver
		protoRoute.SslCertResolver = &resolver
	}
	return protoRoute
}

func normalizeRouteDomain(domain string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
}

func normalizeDefaultGameServerDomain(gameServerID string) string {
	normalizedID := strings.ToLower(strings.TrimSpace(gameServerID))
	if strings.HasPrefix(normalizedID, "gs-") {
		return normalizedID + "." + gameServerRouteDomainSuffix
	}
	return "gs-" + normalizedID + "." + gameServerRouteDomainSuffix
}

func sanitizeRouteIDPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, ".", "-")
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "default"
	}
	return value
}

func getAvailableDomainsForGameServer(gameServerID string) (map[string]struct{}, error) {
	availableDomains := map[string]struct{}{
		normalizeDefaultGameServerDomain(gameServerID): {},
	}

	verifiedDomains, err := database.GetVerifiedGameServerDomains(gameServerID)
	if err != nil {
		return nil, err
	}

	for _, verified := range verifiedDomains {
		domain := normalizeRouteDomain(verified.Domain)
		if domain != "" {
			availableDomains[domain] = struct{}{}
		}
	}

	return availableDomains, nil
}

func checkGameServerDomainConflict(gameServerID string, domain string) error {
	if domain == normalizeDefaultGameServerDomain(gameServerID) {
		return nil
	}

	// Deployment default/custom domains
	var deployments []database.Deployment
	if err := database.DB.Find(&deployments).Error; err != nil {
		return fmt.Errorf("failed to query deployments for domain conflict: %w", err)
	}

	for _, deployment := range deployments {
		if normalizeRouteDomain(deployment.Domain) == domain {
			return fmt.Errorf("domain %s is already in use by deployment %s", domain, deployment.ID)
		}

		if deployment.CustomDomains == "" {
			continue
		}

		var customDomains []string
		if err := json.Unmarshal([]byte(deployment.CustomDomains), &customDomains); err != nil {
			continue
		}

		for _, entry := range customDomains {
			entryDomain := normalizeRouteDomain(extractDomainFromCustomDomainEntry(entry))
			if entryDomain == domain {
				return fmt.Errorf("domain %s is already in use by deployment %s", domain, deployment.ID)
			}
		}
	}

	// Existing game server route domain
	existingRoute, err := database.GetGameServerHTTPRouteByDomain(domain)
	switch {
	case err == nil && existingRoute.GameServerID != gameServerID:
		return fmt.Errorf("domain %s is already in use by game server %s", domain, existingRoute.GameServerID)
	case errors.Is(err, gorm.ErrRecordNotFound):
		// No existing route for this domain.
	case err != nil:
		return fmt.Errorf("failed to validate existing game server routes: %w", err)
	}

	// Existing game server domain verification claims
	var existingVerification database.GameServerDomainVerification
	err = database.DB.Where("domain = ? AND game_server_id <> ? AND status IN ?", domain, gameServerID, []string{gameServerDomainStatusPending, gameServerDomainStatusVerified}).First(&existingVerification).Error
	switch {
	case err == nil:
		return fmt.Errorf("domain %s is already claimed by game server %s", domain, existingVerification.GameServerID)
	case errors.Is(err, gorm.ErrRecordNotFound):
		// No existing verification claim for this domain.
	default:
		return fmt.Errorf("failed to validate existing game server domain claims: %w", err)
	}

	// Prevent using other game servers' default domains
	var otherGameServers []database.GameServer
	if err := database.DB.Select("id").Where("id <> ?", gameServerID).Find(&otherGameServers).Error; err != nil {
		return fmt.Errorf("failed to query game servers for domain conflict: %w", err)
	}
	for _, gameServer := range otherGameServers {
		if normalizeDefaultGameServerDomain(gameServer.ID) == domain {
			return fmt.Errorf("domain %s is reserved by game server %s", domain, gameServer.ID)
		}
	}

	return nil
}

func getOrCreateGameServerDomainVerification(gameServerID string, domain string) (*database.GameServerDomainVerification, error) {
	if err := checkGameServerDomainConflict(gameServerID, domain); err != nil {
		return nil, err
	}

	existing, err := database.GetGameServerDomainVerification(gameServerID, domain)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	verification := &database.GameServerDomainVerification{
		ID:           fmt.Sprintf("gsv-%s-%d", gameServerID, time.Now().UnixNano()),
		GameServerID: gameServerID,
		Domain:       domain,
		Token:        generateGameServerDeterministicToken(gameServerID, domain),
		Status:       gameServerDomainStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := database.UpsertGameServerDomainVerification(verification); err != nil {
		return nil, fmt.Errorf("failed to store verification token: %w", err)
	}

	return verification, nil
}

func verifyGameServerDomainOwnership(gameServerID string, domain string) error {
	if err := checkGameServerDomainConflict(gameServerID, domain); err != nil {
		return err
	}

	verification, err := getOrCreateGameServerDomainVerification(gameServerID, domain)
	if err != nil {
		return fmt.Errorf("failed to get verification record: %w", err)
	}

	if verification.Status == gameServerDomainStatusVerified {
		return nil
	}

	if time.Since(verification.CreatedAt) > 7*24*time.Hour {
		_ = updateGameServerDomainVerificationStatus(verification, gameServerDomainStatusExpired)
		return fmt.Errorf("verification expired. Please request a new verification token")
	}

	txtRecordName := fmt.Sprintf("_obiente-verification.%s", domain)
	verificationToken := fmt.Sprintf("obiente-verification=%s", verification.Token)

	txtRecords, err := lookupGameServerTXT(txtRecordName)
	if err != nil {
		log.Printf("[VerifyGameServerDomain] DNS lookup failed for %s: %v", txtRecordName, err)
		return fmt.Errorf("DNS lookup failed: %w. Please ensure the TXT record is configured correctly", err)
	}

	found := false
	for _, record := range txtRecords {
		if strings.Contains(record, verificationToken) {
			found = true
			break
		}
	}

	if !found {
		_ = updateGameServerDomainVerificationStatus(verification, gameServerDomainStatusFailed)
		return fmt.Errorf("verification failed: TXT record not found or token mismatch. Please add TXT record: %s = %s", txtRecordName, verificationToken)
	}

	if err := updateGameServerDomainVerificationStatus(verification, gameServerDomainStatusVerified); err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}

	return nil
}

func updateGameServerDomainVerificationStatus(verification *database.GameServerDomainVerification, status string) error {
	verification.Status = status
	verification.UpdatedAt = time.Now()
	if status == gameServerDomainStatusVerified {
		now := time.Now()
		verification.VerifiedAt = &now
	}
	return database.UpsertGameServerDomainVerification(verification)
}

func generateGameServerDeterministicToken(gameServerID string, domain string) string {
	gameServerID = strings.ToLower(strings.TrimSpace(gameServerID))
	domain = normalizeRouteDomain(domain)

	secret := os.Getenv("SECRET")
	if secret == "" {
		log.Printf("[generateGameServerDeterministicToken] Warning: SECRET environment variable not set, using fallback")
		secret = "fallback-secret-not-configured"
	}

	input := fmt.Sprintf("%s:%s", gameServerID, domain)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	hash := mac.Sum(nil)

	return hex.EncodeToString(hash[:16])
}

func lookupGameServerTXT(name string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if records, err := net.DefaultResolver.LookupTXT(ctx, name); err == nil {
		return records, nil
	} else {
		log.Printf("[lookupGameServerTXT] default resolver failed: %v", err)
	}

	resolvers := append(gameServerFallbackResolversFromEnv(), gameServerDNSFallbackResolvers...)
	var lastErr error

	for _, addr := range resolvers {
		resolver := customGameServerResolver(addr)
		if resolver == nil {
			continue
		}

		records, err := resolver.LookupTXT(ctx, name)
		if err == nil {
			return records, nil
		}
		lastErr = err
		log.Printf("[lookupGameServerTXT] resolver %s failed: %v", addr, err)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no resolvers available")
	}

	return nil, fmt.Errorf("DNS lookup failed (all resolvers): %w", lastErr)
}

func gameServerFallbackResolversFromEnv() []string {
	env := strings.TrimSpace(os.Getenv("OBIENTE_DNS_RESOLVERS"))
	if env == "" {
		return nil
	}

	parts := strings.Split(env, ",")
	resolvers := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.Contains(p, ":") {
			p = fmt.Sprintf("%s:53", p)
		}
		resolvers = append(resolvers, p)
	}
	return resolvers
}

func customGameServerResolver(addr string) *net.Resolver {
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
	}
}

func extractDomainFromCustomDomainEntry(entry string) string {
	parts := strings.Split(entry, ":")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
