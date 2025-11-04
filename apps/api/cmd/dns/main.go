package main

import (
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
		log.Printf("[DNS] Configured DNS server IPs: %v", dnsIPs)
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
		// Only handle A record queries
		if q.Qtype != dns.TypeA {
			continue
		}

		// Check if this is a *.my.obiente.cloud domain
		domain := strings.ToLower(q.Name)
		if !strings.HasSuffix(domain, ".my.obiente.cloud.") {
			// Not our domain, return NXDOMAIN
			msg.SetRcode(r, dns.RcodeNameError)
			w.WriteMsg(msg)
			return
		}

		// Extract deployment ID from domain
		// Format: deploy-123.my.obiente.cloud -> deploy-123
		parts := strings.Split(domain, ".")
		if len(parts) < 3 {
			msg.SetRcode(r, dns.RcodeNameError)
			w.WriteMsg(msg)
			return
		}

		deploymentID := parts[0]

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
				w.WriteMsg(msg)
				return
			}
		}

		// Query database for deployment location using shared database functions
		ips, err := database.GetDeploymentTraefikIP(deploymentID, s.traefikIPMap)
		if err != nil {
			log.Printf("[DNS] Failed to resolve deployment %s: %v", deploymentID, err)
			msg.SetRcode(r, dns.RcodeNameError)
			w.WriteMsg(msg)
			return
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

		if len(msg.Answer) == 0 {
			msg.SetRcode(r, dns.RcodeNameError)
			w.WriteMsg(msg)
			return
		}
	}

	w.WriteMsg(msg)
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
