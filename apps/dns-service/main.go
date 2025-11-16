package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"api/internal/database"
	"api/internal/metrics"

	"github.com/miekg/dns"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	cacheTTL = 60 * time.Second // Cache DNS responses for 60 seconds
)

type DNSServer struct {
	db           *gorm.DB
	traefikIPMap map[string][]string
	redisCache   *database.RedisCache
}

func NewDNSServer() (*DNSServer, error) {
	s := &DNSServer{}

	// Parse Traefik IPs from environment
	traefikIPsEnv := os.Getenv("TRAEFIK_IPS")
	if traefikIPsEnv == "" {
		return nil, fmt.Errorf("TRAEFIK_IPS environment variable is required")
	}

	var err error
	s.traefikIPMap, err = database.ParseTraefikIPsFromEnv(traefikIPsEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRAEFIK_IPS: %w", err)
	}

	// DNS_IPS is optional - used for documentation/deployment purposes
	// It doesn't affect server operation but can be used to configure nameserver records
	dnsIPsEnv := os.Getenv("DNS_IPS")
	if dnsIPsEnv != "" {
		dnsIPs := strings.Split(dnsIPsEnv, ",")
		validIPs := make([]string, 0)
		invalidIPs := make([]string, 0)

		for _, ip := range dnsIPs {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			// Validate IP address
			if net.ParseIP(ip) != nil {
				validIPs = append(validIPs, ip)
			} else {
				invalidIPs = append(invalidIPs, ip)
			}
		}

		if len(validIPs) > 0 {
			log.Printf("[DNS] Configured DNS server IPs: %v", validIPs)
		}

		if len(invalidIPs) > 0 {
			log.Printf("[DNS] WARNING: Invalid DNS server IPs (must be IP addresses, not hostnames): %v", invalidIPs)
			log.Printf("[DNS] WARNING: DNS_IPS should contain IP addresses (e.g., '127.0.0.1' or '10.0.9.10'), not container names or hostnames")
			log.Printf("[DNS] WARNING: If DNS_IPS is not set or invalid, containers may not be able to resolve DNS queries")
		}
	} else {
		log.Printf("[DNS] WARNING: DNS_IPS environment variable is not set")
		log.Printf("[DNS] WARNING: This may cause issues if containers need to resolve DNS queries")
		log.Printf("[DNS] WARNING: Set DNS_IPS to a comma-separated list of IP addresses (e.g., '127.0.0.1' or '10.0.9.10')")
	}

	// Connect to database
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		return nil, fmt.Errorf("database environment variables (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME) are required")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize the global database.DB variable so database functions can use it
	database.DB = s.db

	// Initialize Redis cache
	if err := database.InitRedis(); err != nil {
		log.Printf("[DNS] Warning: Redis initialization failed: %v (will run without cache)", err)
	} else {
		log.Printf("[DNS] Redis cache initialized")
	}
	s.redisCache = database.RedisClient

	return s, nil
}

func (s *DNSServer) getCached(ctx context.Context, deploymentID string) ([]string, bool) {
	if s.redisCache == nil {
		return nil, false
	}

	cacheKey := fmt.Sprintf("dns:deployment:%s", deploymentID)
	cachedData, err := s.redisCache.Get(ctx, cacheKey)
	if err != nil || cachedData == "" {
		return nil, false
	}

	var ips []string
	if err := json.Unmarshal([]byte(cachedData), &ips); err != nil {
		return nil, false
	}

	return ips, true
}

func (s *DNSServer) setCache(ctx context.Context, deploymentID string, ips []string) {
	if s.redisCache == nil {
		return
	}

	cacheKey := fmt.Sprintf("dns:deployment:%s", deploymentID)
	s.redisCache.Set(ctx, cacheKey, ips, cacheTTL)
}

func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	ctx := context.Background()

	for _, q := range r.Question {
		domain := strings.ToLower(q.Name)
		// Normalize domain - remove trailing dot if present for comparison
		domainNormalized := strings.TrimSuffix(domain, ".")
		if !strings.HasSuffix(domainNormalized, ".my.obiente.cloud") {
			// Not our domain, return NXDOMAIN
			log.Printf("[DNS] Query for non-my.obiente.cloud domain: %s", domain)
			msg.SetRcode(r, dns.RcodeNameError)
			w.WriteMsg(msg)
			return
		}

		// Handle SRV record queries for game servers
		// Format: _minecraft._tcp.gs-123.my.obiente.cloud
		// Format: _minecraft._udp.gs-123.my.obiente.cloud (Bedrock)
		// Format: _rust._udp.gs-123.my.obiente.cloud
		if q.Qtype == dns.TypeSRV {
			if s.handleSRVQuery(msg, domain, q) {
				w.WriteMsg(msg)
				return
			}
			// If SRV handling didn't find a match, continue to check other types
		}

		// Handle A record queries for deployments and game servers
		// Format: deploy-123.my.obiente.cloud (deployments)
		// Format: gs-123.my.obiente.cloud (game servers)
		if q.Qtype == dns.TypeA {
			if s.handleAQuery(ctx, msg, domain, q) {
				w.WriteMsg(msg)
				return
			}
		}

		// If no answer was generated, return NXDOMAIN
		if len(msg.Answer) == 0 {
			msg.SetRcode(r, dns.RcodeNameError)
		}
	}

	w.WriteMsg(msg)
}

// handleSRVQuery handles SRV record queries for game servers
// Supports:
// - Minecraft Java: _minecraft._tcp.gs-123.my.obiente.cloud
// - Minecraft Bedrock: _minecraft._udp.gs-123.my.obiente.cloud
// - Rust: _rust._udp.gs-123.my.obiente.cloud
func (s *DNSServer) handleSRVQuery(msg *dns.Msg, domain string, q dns.Question) bool {
	// Parse SRV query format: _service._protocol.gs-123.my.obiente.cloud.
	// Normalize domain - remove trailing dot if present
	domainNormalized := strings.TrimSuffix(domain, ".")
	parts := strings.Split(domainNormalized, ".")
	// Need at least: _service._protocol.gs-123.my.obiente.cloud = 5 parts
	if len(parts) < 5 {
		log.Printf("[DNS] Invalid SRV domain format (too few parts): %s (parts: %v)", domain, parts)
		return false
	}

	service := parts[0]  // _minecraft, _rust, etc.
	protocol := parts[1] // _tcp, _udp
	gameServerID := parts[2]

	if service == "" || protocol == "" || gameServerID == "" {
		log.Printf("[DNS] Empty service/protocol/gameserver ID in SRV domain: %s", domain)
		return false
	}

	// Extract game server ID (gameserver-123 or gs-123)
	// Normalize: if it's gameserver-gs-123, extract gs-123; if it's gameserver-123, convert to gs-123
	if strings.HasPrefix(gameServerID, "gameserver-") {
		// Extract actual game server ID
		actualID := strings.TrimPrefix(gameServerID, "gameserver-")
		if strings.HasPrefix(actualID, "gs-") {
			gameServerID = actualID
		} else {
			// Legacy format: gameserver-123 -> gs-123
			gameServerID = "gs-" + actualID
		}
	} else if !strings.HasPrefix(gameServerID, "gs-") {
		// Not a valid game server ID format
		return false
	}

	// Get game server type to validate SRV service matches
	gameType, err := database.GetGameServerType(gameServerID)
	if err != nil {
		log.Printf("[DNS] Failed to get game type for %s: %v", gameServerID, err)
		return false
	}

	// Validate SRV service/protocol matches game type
	// GameType enum values:
	// MINECRAFT = 1, MINECRAFT_JAVA = 2, MINECRAFT_BEDROCK = 3, RUST = 6
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
		return false
	}

	// Check local database FIRST - our own records always get priority
	// Get game server location (IP and port) from local database
	nodeIP, port, err := database.GetGameServerLocation(gameServerID)
	if err == nil && nodeIP != "" {
		// Successfully got from local database - use it
		// For SRV records, use the A record hostname as target
		// Format: gs-123.my.obiente.cloud
		targetHostname := gameServerID + ".my.obiente.cloud"

		srv := &dns.SRV{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeSRV,
				Class:  dns.ClassINET,
				Ttl:    uint32(cacheTTL.Seconds()),
			},
			Priority: 0,
			Weight:   0,
			Port:     uint16(port),
			Target:   targetHostname,
		}
		msg.Answer = append(msg.Answer, srv)

		// Also add an A record for the target hostname
		// This is required for SRV records - the target must resolve to an IP
		ip := net.ParseIP(nodeIP)
		if ip == nil {
			// Try to resolve hostname
			resolvedIPs, resolveErr := net.LookupIP(nodeIP)
			if resolveErr == nil && len(resolvedIPs) > 0 {
				for _, resolvedIP := range resolvedIPs {
					if resolvedIP.To4() != nil {
						ip = resolvedIP
						break
					}
				}
				if ip == nil && len(resolvedIPs) > 0 {
					ip = resolvedIPs[0]
				}
			}
		}
		if ip != nil {
			a := &dns.A{
				Hdr: dns.RR_Header{
					Name:   targetHostname + ".",
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    uint32(cacheTTL.Seconds()),
				},
				A: ip,
			}
			msg.Extra = append(msg.Extra, a)
			log.Printf("[DNS] Resolved SRV for game server %s via local database", gameServerID)
			return true
		}
	}

	// Fallback to delegated DNS records if local database lookup failed
	// domainNormalized is already defined above
	delegatedRecord, delegationErr := database.GetDelegatedDNSRecord(domainNormalized, "SRV")
	if delegationErr == nil && delegatedRecord != nil {
		// Parse JSON records (SRV format: "priority weight port target")
		var srvRecords []string
		if err := json.Unmarshal([]byte(delegatedRecord.Records), &srvRecords); err == nil && len(srvRecords) > 0 {
			// Parse SRV record: "priority weight port target"
			parts := strings.Fields(srvRecords[0])
			if len(parts) >= 4 {
				var portInt int32
				if _, err := fmt.Sscanf(parts[2], "%d", &portInt); err == nil {
					port := portInt
					// Get A record for the target hostname
					targetDomain := parts[3]
					// Normalize target domain by removing trailing dot for database lookup
					targetDomainNormalized := strings.TrimSuffix(targetDomain, ".")
					aRecord, aErr := database.GetDelegatedDNSRecord(targetDomainNormalized, "A")
					var nodeIP string
					var ttl uint32 = uint32(delegatedRecord.TTL)
					if aErr == nil && aRecord != nil {
						var recordIPs []string
						if err := json.Unmarshal([]byte(aRecord.Records), &recordIPs); err == nil && len(recordIPs) > 0 {
							nodeIP = recordIPs[0]
						}
					}
					// If delegated A record not found, try local database
					if nodeIP == "" {
						localIP, localErr := database.GetGameServerIP(gameServerID)
						if localErr == nil && localIP != "" {
							nodeIP = localIP
							ttl = uint32(cacheTTL.Seconds()) // Use cache TTL for local records
						}
					}

					// If we have an IP (from delegated or local), return the SRV record
					if nodeIP != "" {
						ip := net.ParseIP(nodeIP)
						if ip == nil {
							// Try to resolve hostname
							resolvedIPs, resolveErr := net.LookupIP(nodeIP)
							if resolveErr == nil && len(resolvedIPs) > 0 {
								for _, resolvedIP := range resolvedIPs {
									if resolvedIP.To4() != nil {
										ip = resolvedIP
										break
									}
								}
								if ip == nil && len(resolvedIPs) > 0 {
									ip = resolvedIPs[0]
								}
							}
						}
						if ip != nil {
							targetHostname := gameServerID + ".my.obiente.cloud"
							srv := &dns.SRV{
								Hdr: dns.RR_Header{
									Name:   q.Name,
									Rrtype: dns.TypeSRV,
									Class:  dns.ClassINET,
									Ttl:    uint32(delegatedRecord.TTL),
								},
								Priority: 0,
								Weight:   0,
								Port:     uint16(port),
								Target:   targetHostname,
							}
							msg.Answer = append(msg.Answer, srv)
							// Also add A record for the target
							a := &dns.A{
								Hdr: dns.RR_Header{
									Name:   targetHostname + ".",
									Rrtype: dns.TypeA,
									Class:  dns.ClassINET,
									Ttl:    ttl,
								},
								A: ip,
							}
							msg.Extra = append(msg.Extra, a)
							log.Printf("[DNS] Resolved SRV for game server %s via delegated record (fallback, A record from %s)", gameServerID, func() string {
								if aErr == nil && aRecord != nil {
									return "delegated"
								}
								return "local"
							}())
							return true
						}
					}
				}
			}
		}
	}

	// Both local and delegated lookups failed
	log.Printf("[DNS] Failed to resolve game server %s locally or via delegated records", gameServerID)
	return false
}

// handleAQuery handles A record queries for deployments and game servers
// Format: deploy-123.my.obiente.cloud -> deploy-123 (deployments)
// Format: gs-123.my.obiente.cloud -> gs-123 (game servers)
func (s *DNSServer) handleAQuery(ctx context.Context, msg *dns.Msg, domain string, q dns.Question) bool {
	// Normalize domain - remove trailing dot if present
	domainNormalized := strings.TrimSuffix(domain, ".")
	parts := strings.Split(domainNormalized, ".")
	// Need at least: resourceID.my.obiente.cloud = 4 parts
	if len(parts) < 4 {
		log.Printf("[DNS] Invalid domain format (too few parts): %s (parts: %v)", domain, parts)
		return false
	}

	// Extract resource ID (first part)
	resourceID := parts[0]
	if resourceID == "" {
		log.Printf("[DNS] Empty resource ID in domain: %s", domain)
		return false
	}

	// Check if this is a game server (gs-123)
	if strings.HasPrefix(resourceID, "gs-") {
		return s.handleGameServerAQuery(msg, domain, q, resourceID)
	}

	// Otherwise, treat as deployment (deploy-123)
	return s.handleDeploymentAQuery(ctx, msg, domain, q, resourceID)
}

// handleDeploymentAQuery handles A record queries for deployments
func (s *DNSServer) handleDeploymentAQuery(ctx context.Context, msg *dns.Msg, domain string, q dns.Question, deploymentID string) bool {
	// Check for delegated DNS records FIRST - if a self-hosted instance is pushing records,
	// those should take precedence over local database entries
	// Normalize domain by removing trailing dot for database lookup
	domainNormalized := strings.TrimSuffix(domain, ".")
	log.Printf("[DNS] Looking up delegated record for deployment %s: original domain=%q, normalized domain=%q", deploymentID, domain, domainNormalized)
	delegatedRecord, delegationErr := database.GetDelegatedDNSRecord(domainNormalized, "A")
	if delegationErr == nil && delegatedRecord != nil {
		// Parse JSON records
		var recordIPs []string
		if err := json.Unmarshal([]byte(delegatedRecord.Records), &recordIPs); err == nil && len(recordIPs) > 0 {
			// Use delegated records - cache and return them
			s.setCache(ctx, deploymentID, recordIPs)
			for _, ip := range recordIPs {
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    uint32(delegatedRecord.TTL),
					},
					A: net.ParseIP(ip),
				}
				if rr.A == nil {
					log.Printf("[DNS] Failed to parse IP from delegated record: %s", ip)
					continue
				}
				msg.Answer = append(msg.Answer, rr)
			}
			if len(msg.Answer) > 0 {
				log.Printf("[DNS] Resolved deployment %s via delegated record: %v", deploymentID, recordIPs)
				return true
			}
		} else {
			log.Printf("[DNS] Failed to parse delegated DNS record for %s: %v", deploymentID, err)
		}
	}

	// Check cache for local records
	if ips, ok := s.getCached(ctx, deploymentID); ok {
		for _, ip := range ips {
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    uint32(cacheTTL.Seconds()),
				},
				A: net.ParseIP(ip),
			}
			if rr.A == nil {
				log.Printf("[DNS] Failed to parse IP: %s", ip)
				continue
			}
			msg.Answer = append(msg.Answer, rr)
		}
		if len(msg.Answer) > 0 {
			return true
		}
	}

	// Query database for deployment location using shared database functions
	ips, err := database.GetDeploymentTraefikIP(deploymentID, s.traefikIPMap)
	if err != nil {
		log.Printf("[DNS] Failed to resolve deployment %s locally: %v", deploymentID, err)
		return false
	}

	// Cache the result
	s.setCache(ctx, deploymentID, ips)

	// Return A records for all Traefik IPs (for load balancing)
	for _, ip := range ips {
		rr := &dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(cacheTTL.Seconds()),
			},
			A: net.ParseIP(ip),
		}
		if rr.A == nil {
			log.Printf("[DNS] Failed to parse IP: %s", ip)
			continue
		}
		msg.Answer = append(msg.Answer, rr)
	}

	return len(msg.Answer) > 0
}

// handleGameServerAQuery handles A record queries for game servers
func (s *DNSServer) handleGameServerAQuery(msg *dns.Msg, domain string, q dns.Question, gameServerID string) bool {
	log.Printf("[DNS] Handling A query for game server %s (domain: %s)", gameServerID, domain)

	// Check local database FIRST - our own records always get priority
	// Get game server IP from local database
	nodeIP, err := database.GetGameServerIP(gameServerID)
	if err != nil {
		log.Printf("[DNS] Failed to resolve game server %s locally: %v", gameServerID, err)
		// Check if game server exists but is not running
		var gameServer struct {
			ID     string
			Status int32
		}
		if dbErr := database.DB.Table("game_servers").Select("id, status").Where("id = ?", gameServerID).First(&gameServer).Error; dbErr == nil {
			log.Printf("[DNS] Game server %s exists with status %d but has no running location", gameServerID, gameServer.Status)
		}
		// Fall through to delegated DNS records as fallback
		nodeIP = ""
	}

	// Parse IP address (or resolve hostname if fallback returned hostname)
	ip := net.ParseIP(nodeIP)
	if ip == nil {
		// If nodeIP is not a valid IP, it might be a hostname from fallback
		// Try to resolve it
		log.Printf("[DNS] nodeIP is not a valid IP address, attempting to resolve as hostname: %s", nodeIP)
		resolvedIPs, err := net.LookupIP(nodeIP)
		if err != nil || len(resolvedIPs) == 0 {
			log.Printf("[DNS] Failed to resolve hostname %s for game server %s: %v", nodeIP, gameServerID, err)
			// Fall through to delegated DNS records as fallback
			ip = nil
		} else {
			// Use the first resolved IP (prefer IPv4)
			for _, resolvedIP := range resolvedIPs {
				if resolvedIP.To4() != nil {
					ip = resolvedIP
					break
				}
			}
			if ip == nil {
				// No IPv4 found, use first IP (IPv6)
				ip = resolvedIPs[0]
			}
			log.Printf("[DNS] Resolved hostname %s to IP %s for game server %s", nodeIP, ip.String(), gameServerID)
		}
	}

	// If we successfully got an IP from local database, use it
	if ip != nil {
		rr := &dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(cacheTTL.Seconds()),
			},
			A: ip,
		}
		msg.Answer = append(msg.Answer, rr)
		log.Printf("[DNS] Resolved game server %s via local database: %s", gameServerID, ip.String())
		return true
	}

	// Fallback to delegated DNS records if local database lookup failed
	domainNormalized := strings.TrimSuffix(domain, ".")
	delegatedRecord, delegationErr := database.GetDelegatedDNSRecord(domainNormalized, "A")
	if delegationErr == nil && delegatedRecord != nil {
		// Parse JSON records
		var recordIPs []string
		if err := json.Unmarshal([]byte(delegatedRecord.Records), &recordIPs); err == nil && len(recordIPs) > 0 {
			nodeIP := recordIPs[0]
			ip := net.ParseIP(nodeIP)
			if ip != nil {
				// Use delegated record
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    uint32(delegatedRecord.TTL),
					},
					A: ip,
				}
				msg.Answer = append(msg.Answer, rr)
				log.Printf("[DNS] Resolved game server %s via delegated record (fallback): %s", gameServerID, nodeIP)
				return true
			}
		}
	}

	// Both local and delegated lookups failed
	return false
}

// handlePushDNSRecord handles DNS record push requests from remote APIs
func handlePushDNSRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check API key authentication
	apiKey := r.Header.Get("Authorization")
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	// Validate API key and get key info
	apiKeyInfo, err := database.GetDNSDelegationAPIKeyByHash(apiKey)
	if err != nil {
		log.Printf("[DNS Delegation] API key validation error: %v", err)
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		Domain     string   `json:"domain"`
		RecordType string   `json:"record_type"` // "A" or "SRV"
		Records    []string `json:"records"`     // Array of record values
		TTL        int64    `json:"ttl"`         // TTL in seconds (default: 300)
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	domain := strings.TrimSpace(req.Domain)
	recordType := strings.ToUpper(strings.TrimSpace(req.RecordType))
	if recordType == "" {
		recordType = "A"
	}

	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	if len(req.Records) == 0 {
		http.Error(w, "At least one record is required", http.StatusBadRequest)
		return
	}

	// Validate domain format
	if !strings.HasSuffix(strings.ToLower(domain), ".my.obiente.cloud") {
		http.Error(w, "Domain must be a *.my.obiente.cloud domain", http.StatusBadRequest)
		return
	}

	// Support A and SRV record types
	if recordType != "A" && recordType != "SRV" {
		http.Error(w, "Only A and SRV record types are supported", http.StatusBadRequest)
		return
	}

	// Set default TTL if not provided
	ttl := req.TTL
	if ttl == 0 {
		ttl = 300 // Default: 5 minutes
	}

	// Get source API URL from request (for tracking and chain prevention)
	sourceAPI := r.Header.Get("X-Source-API")
	if sourceAPI == "" {
		sourceAPI = r.RemoteAddr // Fallback to client IP
	}

	// Prevent delegation chains: if the source API is itself using delegation, reject
	var existingKey database.DNSDelegationAPIKey
	result := database.DB.Where("source_api = ? AND is_active = ? AND revoked_at IS NULL", sourceAPI, true).First(&existingKey)
	if result.Error == nil {
		log.Printf("[DNS Delegation] Rejected delegation chain: source API %s is itself using delegation", sourceAPI)
		metrics.RecordDNSDelegationPushError(apiKeyInfo.OrganizationID, apiKeyInfo.ID, "delegation_chain_prevented")
		http.Error(w, "Delegation chains are not allowed. Servers using DNS delegation cannot accept delegation requests from other servers.", http.StatusForbidden)
		return
	}

	// Convert records to JSON
	recordsJSON, err := json.Marshal(req.Records)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal records: %v", err), http.StatusInternalServerError)
		return
	}

	// Upsert the delegated DNS record with API key tracking
	if err := database.UpsertDelegatedDNSRecordWithAPIKey(domain, recordType, string(recordsJSON), sourceAPI, apiKeyInfo.ID, apiKeyInfo.OrganizationID, ttl); err != nil {
		metrics.RecordDNSDelegationPushError(apiKeyInfo.OrganizationID, apiKeyInfo.ID, "upsert_failed")
		http.Error(w, fmt.Sprintf("Failed to store DNS record: %v", err), http.StatusInternalServerError)
		return
	}

	// Record metrics
	metrics.RecordDNSDelegationPush(apiKeyInfo.OrganizationID, apiKeyInfo.ID, recordType)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"domain":  domain,
		"type":    recordType,
	})
}

// handlePushDNSRecords handles batch DNS record push requests
func handlePushDNSRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check API key authentication
	apiKey := r.Header.Get("Authorization")
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	// Validate API key and get key info
	apiKeyInfo, err := database.GetDNSDelegationAPIKeyByHash(apiKey)
	if err != nil {
		log.Printf("[DNS Delegation] API key validation error: %v", err)
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		Records []struct {
			Domain     string   `json:"domain"`
			RecordType string   `json:"record_type"`
			Records    []string `json:"records"`
			TTL        int64    `json:"ttl"`
		} `json:"records"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Records) == 0 {
		http.Error(w, "At least one record is required", http.StatusBadRequest)
		return
	}

	// Get source API URL from request (for tracking and chain prevention)
	sourceAPI := r.Header.Get("X-Source-API")
	if sourceAPI == "" {
		sourceAPI = r.RemoteAddr
	}

	// Prevent delegation chains: if the source API is itself using delegation, reject
	var existingKey database.DNSDelegationAPIKey
	result := database.DB.Where("source_api = ? AND is_active = ? AND revoked_at IS NULL", sourceAPI, true).First(&existingKey)
	if result.Error == nil {
		log.Printf("[DNS Delegation] Rejected delegation chain: source API %s is itself using delegation", sourceAPI)
		metrics.RecordDNSDelegationPushError(apiKeyInfo.OrganizationID, apiKeyInfo.ID, "delegation_chain_prevented")
		http.Error(w, "Delegation chains are not allowed. Servers using DNS delegation cannot accept delegation requests from other servers.", http.StatusForbidden)
		return
	}

	successCount := 0
	errors := make([]string, 0)

	for _, recordReq := range req.Records {
		domain := strings.TrimSpace(recordReq.Domain)
		recordType := strings.ToUpper(strings.TrimSpace(recordReq.RecordType))
		if recordType == "" {
			recordType = "A"
		}

		if domain == "" || len(recordReq.Records) == 0 {
			errors = append(errors, fmt.Sprintf("Invalid record: domain=%s, records=%d", domain, len(recordReq.Records)))
			continue
		}

		if !strings.HasSuffix(strings.ToLower(domain), ".my.obiente.cloud") {
			errors = append(errors, fmt.Sprintf("Invalid domain format: %s", domain))
			continue
		}

		if recordType != "A" && recordType != "SRV" {
			errors = append(errors, fmt.Sprintf("Unsupported record type: %s", recordType))
			continue
		}

		ttl := recordReq.TTL
		if ttl == 0 {
			ttl = 300
		}

		recordsJSON, err := json.Marshal(recordReq.Records)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to marshal records for %s: %v", domain, err))
			continue
		}

		if err := database.UpsertDelegatedDNSRecordWithAPIKey(domain, recordType, string(recordsJSON), sourceAPI, apiKeyInfo.ID, apiKeyInfo.OrganizationID, ttl); err != nil {
			metrics.RecordDNSDelegationPushError(apiKeyInfo.OrganizationID, apiKeyInfo.ID, "upsert_failed")
			errors = append(errors, fmt.Sprintf("Failed to store %s: %v", domain, err))
			continue
		}

		// Record metrics
		metrics.RecordDNSDelegationPush(apiKeyInfo.OrganizationID, apiKeyInfo.ID, recordType)
		successCount++
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      successCount,
		"errors":       errors,
		"total":        len(req.Records),
		"success_count": successCount,
	})
}

func main() {
	// Start HTTP server for DNS delegation push endpoints
	// This must always run, even when DNS service is disabled, to receive delegation pushes
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/dns/push", handlePushDNSRecord)
	httpMux.HandleFunc("/dns/push/batch", handlePushDNSRecords)
	
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8053" // Default HTTP port for DNS service
	}
	
	// Start HTTP server in a goroutine - this must always run
	go func() {
		log.Printf("[DNS] Starting HTTP server for DNS delegation on port %s", httpPort)
		if err := http.ListenAndServe(":"+httpPort, httpMux); err != nil {
			log.Fatalf("[DNS] Failed to start HTTP server: %v", err)
		}
	}()

	// Initialize database connection - required for HTTP delegation endpoints
	// This must be done even when DNS server is disabled
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatalf("Database environment variables (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME) are required")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize the global database.DB variable so database functions can use it
	database.DB = db
	log.Printf("[DNS] Database connection established")

	// Initialize Redis cache (optional, but useful for caching)
	if err := database.InitRedis(); err != nil {
		log.Printf("[DNS] Warning: Redis initialization failed: %v (will run without cache)", err)
	} else {
		log.Printf("[DNS] Redis cache initialized")
	}

	// Check if DNS service is enabled
	// Set ENABLE_DNS=false to disable the DNS server (but keep HTTP server for delegation)
	enableDNS := os.Getenv("ENABLE_DNS")
	if enableDNS == "false" || enableDNS == "0" {
		log.Printf("[DNS] DNS server is disabled (ENABLE_DNS=%s). HTTP delegation endpoints remain active.", enableDNS)
		log.Printf("[DNS] This is expected when using DNS delegation to an external service.")
		log.Printf("[DNS] HTTP server on port %s will continue running to receive delegation pushes.", httpPort)
		// Keep the process alive to serve HTTP endpoints
		select {} // Block forever to keep the process running
	}

	// Create DNS server (only when DNS is enabled)
	server, err := NewDNSServer()
	if err != nil {
		log.Fatalf("Failed to create DNS server: %v", err)
	}

	log.Printf("[DNS] Starting DNS server for my.obiente.cloud zone")
	log.Printf("[DNS] Traefik IPs configured for regions: %v", server.traefikIPMap)

	// Get DNS port from environment (default to 53)
	dnsPort := os.Getenv("DNS_PORT")
	if dnsPort == "" {
		dnsPort = "53"
	}

	log.Printf("[DNS] Using port %s for DNS server", dnsPort)

	// Start cleanup goroutine for expired delegated DNS records
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
		defer ticker.Stop()
		for range ticker.C {
			if err := database.CleanupExpiredDelegatedRecords(); err != nil {
				log.Printf("[DNS] Failed to cleanup expired delegated records: %v", err)
			}
		}
	}()

	// Register DNS handler for my.obiente.cloud zone
	dns.HandleFunc("my.obiente.cloud.", server.handleDNSRequest)

	// Start UDP server
	// Explicitly bind to 0.0.0.0 to listen on all interfaces (not just loopback)
	go func() {
		udpServer := &dns.Server{
			Addr:    "0.0.0.0:" + dnsPort,
			Net:     "udp",
			Handler: dns.DefaultServeMux,
		}
		log.Printf("[DNS] Starting DNS server on UDP port %s (0.0.0.0:%s)", dnsPort, dnsPort)
		if err := udpServer.ListenAndServe(); err != nil {
			log.Fatalf("[DNS] Failed to start UDP DNS server: %v", err)
		}
	}()

	// Start TCP server
	// Explicitly bind to 0.0.0.0 to listen on all interfaces (not just loopback)
	tcpServer := &dns.Server{
		Addr:    "0.0.0.0:" + dnsPort,
		Net:     "tcp",
		Handler: dns.DefaultServeMux,
	}
	log.Printf("[DNS] Starting DNS server on TCP port %s (0.0.0.0:%s)", dnsPort, dnsPort)
	if err := tcpServer.ListenAndServe(); err != nil {
		log.Fatalf("[DNS] Failed to start TCP DNS server: %v", err)
	}
}
