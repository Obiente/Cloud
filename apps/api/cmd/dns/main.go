package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"api/internal/database"

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
		// Format: _minecraft._tcp.gameserver-123.my.obiente.cloud
		// Format: _minecraft._udp.gameserver-123.my.obiente.cloud (Bedrock)
		// Format: _rust._udp.gameserver-123.my.obiente.cloud
		if q.Qtype == dns.TypeSRV {
			if s.handleSRVQuery(msg, domain, q) {
				w.WriteMsg(msg)
				return
			}
			// If SRV handling didn't find a match, continue to check other types
		}

		// Handle A record queries for deployments and game servers
		// Format: deploy-123.my.obiente.cloud (deployments)
		// Format: gameserver-123.my.obiente.cloud (game servers)
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
// - Minecraft Java: _minecraft._tcp.gameserver-123.my.obiente.cloud
// - Minecraft Bedrock: _minecraft._udp.gameserver-123.my.obiente.cloud
// - Rust: _rust._udp.gameserver-123.my.obiente.cloud
func (s *DNSServer) handleSRVQuery(msg *dns.Msg, domain string, q dns.Question) bool {
	// Parse SRV query format: _service._protocol.gameserver-123.my.obiente.cloud.
	// Normalize domain - remove trailing dot if present
	domainNormalized := strings.TrimSuffix(domain, ".")
	parts := strings.Split(domainNormalized, ".")
	// Need at least: _service._protocol.gameserver-123.my.obiente.cloud = 5 parts
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

	// Check for delegated DNS records FIRST - if a self-hosted instance is pushing records,
	// those should take precedence over local database entries
	// Use domainNormalized (already normalized above) for database lookup
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
					if aErr == nil && aRecord != nil {
						var recordIPs []string
						if err := json.Unmarshal([]byte(aRecord.Records), &recordIPs); err == nil && len(recordIPs) > 0 {
							nodeIP := recordIPs[0]
							// Use delegated SRV record
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
							ip := net.ParseIP(nodeIP)
							if ip != nil {
								a := &dns.A{
									Hdr: dns.RR_Header{
										Name:   targetHostname,
										Rrtype: dns.TypeA,
										Class:  dns.ClassINET,
										Ttl:    uint32(delegatedRecord.TTL),
									},
									A: ip,
								}
								msg.Answer = append(msg.Answer, a)
							}
							log.Printf("[DNS] Resolved SRV for game server %s via delegated record", gameServerID)
							return true
						} else {
							log.Printf("[DNS] Failed to parse A record for SRV target: %v", err)
						}
					} else {
						log.Printf("[DNS] Failed to resolve A record for SRV target: %v", aErr)
					}
				} else {
					log.Printf("[DNS] Failed to parse port from SRV record")
				}
			} else {
				log.Printf("[DNS] Invalid SRV record format")
			}
		} else {
			log.Printf("[DNS] Failed to parse delegated SRV record: %v", err)
		}
	}

	// Get game server location (IP and port) from local database
	nodeIP, port, err := database.GetGameServerLocation(gameServerID)
	if err != nil {
		log.Printf("[DNS] Failed to resolve game server %s locally: %v", gameServerID, err)
		return false
	}

	// For SRV records, use the A record hostname as target
	// Format: gameserver-123.my.obiente.cloud
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
	if ip := net.ParseIP(nodeIP); ip != nil {
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
	}

	return true
}

// handleAQuery handles A record queries for deployments and game servers
// Format: deploy-123.my.obiente.cloud -> deploy-123 (deployments)
// Format: gameserver-123.my.obiente.cloud -> gameserver-123 (game servers)
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

	// Check if this is a game server (gameserver-123 or gs-123)
	var gameServerID string
	if strings.HasPrefix(resourceID, "gameserver-") {
		// Extract actual game server ID (gameserver-gs-123 -> gs-123)
		gameServerID = strings.TrimPrefix(resourceID, "gameserver-")
		// If it still has gs- prefix, use it as-is, otherwise prepend gs-
		if !strings.HasPrefix(gameServerID, "gs-") {
			gameServerID = "gs-" + gameServerID
		}
		return s.handleGameServerAQuery(msg, domain, q, gameServerID)
	} else if strings.HasPrefix(resourceID, "gs-") {
		// Direct gs-{id} format
		gameServerID = resourceID
		return s.handleGameServerAQuery(msg, domain, q, gameServerID)
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
	// Check for delegated DNS records FIRST - if a self-hosted instance is pushing records,
	// those should take precedence over local database entries
	// Normalize domain by removing trailing dot for database lookup
	domainNormalized := strings.TrimSuffix(domain, ".")
	delegatedRecord, delegationErr := database.GetDelegatedDNSRecord(domainNormalized, "A")
	if delegationErr == nil && delegatedRecord != nil {
		// Parse JSON records
		var recordIPs []string
		if err := json.Unmarshal([]byte(delegatedRecord.Records), &recordIPs); err == nil && len(recordIPs) > 0 {
			nodeIP := recordIPs[0]
			ip := net.ParseIP(nodeIP)
			if ip == nil {
				log.Printf("[DNS] Invalid IP address from delegated record for game server %s: %s", gameServerID, nodeIP)
			} else {
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
				log.Printf("[DNS] Resolved game server %s via delegated record: %s", gameServerID, nodeIP)
				return true
			}
		} else {
			log.Printf("[DNS] Failed to parse delegated DNS record for %s: %v", gameServerID, err)
		}
	}

	// Get game server IP from local database
	nodeIP, err := database.GetGameServerIP(gameServerID)
	if err != nil {
		log.Printf("[DNS] Failed to resolve game server %s locally: %v", gameServerID, err)
		return false
	}

	// Parse IP address
	ip := net.ParseIP(nodeIP)
	if ip == nil {
		log.Printf("[DNS] Invalid IP address for game server %s: %s", gameServerID, nodeIP)
		return false
	}

	// Return A record
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

	return true
}

func main() {
	// Create DNS server
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
