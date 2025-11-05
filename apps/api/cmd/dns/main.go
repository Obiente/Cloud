package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"api/internal/database"

	"github.com/miekg/dns"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	cacheTTL = 60 * time.Second // Cache DNS responses for 60 seconds
)

type cacheEntry struct {
	ips       []string
	expiresAt time.Time
}

type DNSServer struct {
	db           *gorm.DB
	traefikIPMap map[string][]string
	cache        map[string]cacheEntry
	cacheMutex   sync.RWMutex
}

func NewDNSServer() (*DNSServer, error) {
	s := &DNSServer{
		cache: make(map[string]cacheEntry),
	}

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

	return s, nil
}

func (s *DNSServer) getCached(deploymentID string) ([]string, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	entry, ok := s.cache[deploymentID]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.ips, true
}

func (s *DNSServer) setCache(deploymentID string, ips []string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache[deploymentID] = cacheEntry{
		ips:       ips,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	for _, q := range r.Question {
		domain := strings.ToLower(q.Name)
		if !strings.HasSuffix(domain, ".my.obiente.cloud.") {
			// Not our domain, return NXDOMAIN
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
			if s.handleAQuery(msg, domain, q) {
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
	parts := strings.Split(domain, ".")
	if len(parts) < 4 {
		return false
	}

	service := parts[0]  // _minecraft, _rust, etc.
	protocol := parts[1] // _tcp, _udp
	gameServerID := parts[2]

	// Extract game server ID (gameserver-123)
	if !strings.HasPrefix(gameServerID, "gameserver-") {
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

	// Get game server location (IP and port)
	nodeIP, port, err := database.GetGameServerLocation(gameServerID)
	if err != nil {
		log.Printf("[DNS] Failed to resolve game server %s locally: %v", gameServerID, err)
		
		// Try delegated DNS records if local lookup failed
		delegatedRecord, delegationErr := database.GetDelegatedDNSRecord(domain, "SRV")
		if delegationErr == nil && delegatedRecord != nil {
			// Parse JSON records (SRV format: "priority weight port target")
			var srvRecords []string
			if err := json.Unmarshal([]byte(delegatedRecord.Records), &srvRecords); err == nil && len(srvRecords) > 0 {
				// Parse SRV record: "priority weight port target"
				parts := strings.Fields(srvRecords[0])
				if len(parts) >= 4 {
					var portInt int32
					if _, err := fmt.Sscanf(parts[2], "%d", &portInt); err == nil {
						port = portInt
						// Get A record for the target hostname
						targetDomain := parts[3]
						aRecord, aErr := database.GetDelegatedDNSRecord(targetDomain, "A")
						if aErr == nil && aRecord != nil {
							var recordIPs []string
							if err := json.Unmarshal([]byte(aRecord.Records), &recordIPs); err == nil && len(recordIPs) > 0 {
								nodeIP = recordIPs[0]
								log.Printf("[DNS] Successfully resolved SRV for game server %s via delegated record", gameServerID)
							} else {
								log.Printf("[DNS] Failed to parse A record for SRV target: %v", err)
								return false
							}
						} else {
							log.Printf("[DNS] Failed to resolve A record for SRV target: %v", aErr)
							return false
						}
					} else {
						log.Printf("[DNS] Failed to parse port from SRV record")
						return false
					}
				} else {
					log.Printf("[DNS] Invalid SRV record format")
					return false
				}
			} else {
				log.Printf("[DNS] Failed to parse delegated SRV record: %v", err)
				return false
			}
		} else {
			return false
		}
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
func (s *DNSServer) handleAQuery(msg *dns.Msg, domain string, q dns.Question) bool {
	parts := strings.Split(domain, ".")
	if len(parts) < 3 {
		return false
	}

	resourceID := parts[0]

	// Check if this is a game server (gameserver-123)
	if strings.HasPrefix(resourceID, "gameserver-") {
		return s.handleGameServerAQuery(msg, domain, q, resourceID)
	}

	// Otherwise, treat as deployment (deploy-123)
	return s.handleDeploymentAQuery(msg, domain, q, resourceID)
}

// handleDeploymentAQuery handles A record queries for deployments
func (s *DNSServer) handleDeploymentAQuery(msg *dns.Msg, domain string, q dns.Question, deploymentID string) bool {
	// Check cache first
	if ips, ok := s.getCached(deploymentID); ok {
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
		
		// Try delegated DNS records if local lookup failed
		delegatedRecord, delegationErr := database.GetDelegatedDNSRecord(domain, "A")
		if delegationErr == nil && delegatedRecord != nil {
			// Parse JSON records
			var recordIPs []string
			if err := json.Unmarshal([]byte(delegatedRecord.Records), &recordIPs); err == nil && len(recordIPs) > 0 {
				ips = recordIPs
				log.Printf("[DNS] Successfully resolved deployment %s via delegated record: %v", deploymentID, ips)
			} else {
				log.Printf("[DNS] Failed to parse delegated DNS record for %s: %v", deploymentID, err)
				return false
			}
		} else {
			return false
		}
	}

	// Cache the result
	s.setCache(deploymentID, ips)

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
	// Get game server IP
	nodeIP, err := database.GetGameServerIP(gameServerID)
	if err != nil {
		log.Printf("[DNS] Failed to resolve game server %s locally: %v", gameServerID, err)
		
		// Try delegated DNS records if local lookup failed
		delegatedRecord, delegationErr := database.GetDelegatedDNSRecord(domain, "A")
		if delegationErr == nil && delegatedRecord != nil {
			// Parse JSON records
			var recordIPs []string
			if err := json.Unmarshal([]byte(delegatedRecord.Records), &recordIPs); err == nil && len(recordIPs) > 0 {
				nodeIP = recordIPs[0]
				log.Printf("[DNS] Successfully resolved game server %s via delegated record: %s", gameServerID, nodeIP)
			} else {
				log.Printf("[DNS] Failed to parse delegated DNS record for %s: %v", gameServerID, err)
				return false
			}
		} else {
			return false
		}
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
	go func() {
		udpServer := &dns.Server{
			Addr:    ":" + dnsPort,
			Net:     "udp",
			Handler: dns.DefaultServeMux,
		}
		log.Printf("[DNS] Starting DNS server on UDP port %s", dnsPort)
		if err := udpServer.ListenAndServe(); err != nil {
			log.Fatalf("[DNS] Failed to start UDP DNS server: %v", err)
		}
	}()

	// Start TCP server
	tcpServer := &dns.Server{
		Addr:    ":" + dnsPort,
		Net:     "tcp",
		Handler: dns.DefaultServeMux,
	}
	log.Printf("[DNS] Starting DNS server on TCP port %s", dnsPort)
	if err := tcpServer.ListenAndServe(); err != nil {
		log.Fatalf("[DNS] Failed to start TCP DNS server: %v", err)
	}
}
