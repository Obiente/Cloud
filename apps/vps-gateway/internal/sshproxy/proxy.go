package sshproxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
)

// queryDNSDirect performs a direct DNS query to 127.0.0.1:53
// This bypasses Go's resolver entirely to ensure we always use the gateway's dnsmasq
func queryDNSDirect(ctx context.Context, hostname string) ([]net.IP, error) {
	// Create a simple DNS query packet
	// DNS header: ID (2 bytes) + Flags (2 bytes) + Questions (2 bytes) + Answers (2 bytes) + Authority (2 bytes) + Additional (2 bytes)
	// Query: QNAME (variable) + QTYPE (2 bytes) + QCLASS (2 bytes)

	// Generate a random query ID
	queryID := uint16(time.Now().UnixNano() & 0xFFFF)

	// Build DNS query packet
	packet := make([]byte, 512)

	// Header
	binary.BigEndian.PutUint16(packet[0:2], queryID) // ID
	binary.BigEndian.PutUint16(packet[2:4], 0x0100)  // Flags: standard query, recursion desired
	binary.BigEndian.PutUint16(packet[4:6], 1)       // Questions: 1
	binary.BigEndian.PutUint16(packet[6:8], 0)       // Answers: 0
	binary.BigEndian.PutUint16(packet[8:10], 0)      // Authority: 0
	binary.BigEndian.PutUint16(packet[10:12], 0)     // Additional: 0

	// Question section
	offset := 12
	// Encode hostname (e.g., "vps-123.vps.local" -> [4]vps[3]123[7]vps[5]local[0])
	parts := []string{}
	current := ""
	for _, c := range hostname {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	for _, part := range parts {
		if len(part) > 63 {
			return nil, fmt.Errorf("hostname part too long: %s", part)
		}
		packet[offset] = byte(len(part))
		offset++
		copy(packet[offset:offset+len(part)], part)
		offset += len(part)
	}
	packet[offset] = 0 // End of QNAME
	offset++

	// QTYPE: A (1) or AAAA (28)
	binary.BigEndian.PutUint16(packet[offset:offset+2], 1) // A record
	offset += 2
	// QCLASS: IN (1)
	binary.BigEndian.PutUint16(packet[offset:offset+2], 1) // IN
	offset += 2

	packet = packet[:offset]

	// Dial UDP to 127.0.0.1:53
	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}
	conn, err := dialer.DialContext(ctx, "udp", "127.0.0.1:53")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to 127.0.0.1:53: %w", err)
	}
	defer conn.Close()

	// Set deadline
	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	} else {
		conn.SetDeadline(time.Now().Add(5 * time.Second))
	}

	// Send query
	_, err = conn.Write(packet)
	if err != nil {
		return nil, fmt.Errorf("failed to send DNS query: %w", err)
	}

	// Read response
	response := make([]byte, 512)
	n, err := conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("failed to read DNS response: %w", err)
	}
	response = response[:n]

	// Parse response
	if len(response) < 12 {
		return nil, fmt.Errorf("DNS response too short")
	}

	// Check query ID matches
	respID := binary.BigEndian.Uint16(response[0:2])
	if respID != queryID {
		return nil, fmt.Errorf("DNS response ID mismatch: expected %d, got %d", queryID, respID)
	}

	// Check flags (response code is in bits 0-3 of byte 3)
	flags := binary.BigEndian.Uint16(response[2:4])
	rcode := flags & 0x0F
	if rcode != 0 {
		return nil, fmt.Errorf("DNS query failed with RCODE %d", rcode)
	}

	// Get number of answers
	ancount := binary.BigEndian.Uint16(response[6:8])
	if ancount == 0 {
		return nil, fmt.Errorf("no DNS answers for %s", hostname)
	}

	// Skip question section to get to answers
	// Question section starts at offset 12
	offset = 12
	// Skip QNAME
	for offset < len(response) && response[offset] != 0 {
		labelLen := int(response[offset])
		if labelLen > 63 {
			return nil, fmt.Errorf("invalid DNS label length")
		}
		offset += labelLen + 1
	}
	if offset >= len(response) {
		return nil, fmt.Errorf("DNS response truncated in question section")
	}
	offset++    // Skip null terminator
	offset += 4 // Skip QTYPE and QCLASS

	// Parse answers
	var ips []net.IP
	for i := uint16(0); i < ancount && offset < len(response); i++ {
		// Check for compression pointer
		if offset+1 >= len(response) {
			break
		}
		if (response[offset] & 0xC0) == 0xC0 {
			// Compression pointer - skip name
			offset += 2
		} else {
			// Skip name
			for offset < len(response) && response[offset] != 0 {
				labelLen := int(response[offset] & 0x3F)
				offset += labelLen + 1
			}
			if offset >= len(response) {
				break
			}
			offset++
		}

		if offset+10 > len(response) {
			break
		}

		// Read TYPE, CLASS, TTL, RDLENGTH
		rrType := binary.BigEndian.Uint16(response[offset : offset+2])
		offset += 2
		offset += 2 // Skip CLASS
		offset += 4 // Skip TTL
		rdlength := binary.BigEndian.Uint16(response[offset : offset+2])
		offset += 2

		// Read RDATA
		if offset+int(rdlength) > len(response) {
			break
		}

		if rrType == 1 && rdlength == 4 {
			// A record (IPv4)
			ip := net.IP(response[offset : offset+4])
			ips = append(ips, ip)
		} else if rrType == 28 && rdlength == 16 {
			// AAAA record (IPv6)
			ip := net.IP(response[offset : offset+16])
			ips = append(ips, ip)
		}

		offset += int(rdlength)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP addresses found in DNS response for %s", hostname)
	}

	return ips, nil
}

// Proxy handles SSH TCP proxying
type Proxy struct {
	dhcpManager *dhcp.Manager
	connections map[string]*Connection
	mu          sync.RWMutex
}

// Connection represents an active SSH proxy connection
type Connection struct {
	ID           string
	Target       string
	Port         int
	ClientConn   net.Conn
	TargetConn   net.Conn
	CreatedAt    time.Time
	LastActivity time.Time
}

// NewProxy creates a new SSH proxy
func NewProxy(dhcpManager *dhcp.Manager) (*Proxy, error) {
	return &Proxy{
		dhcpManager: dhcpManager,
		connections: make(map[string]*Connection),
	}, nil
}

// ProxyConnection proxies an SSH connection to a target VPS
func (p *Proxy) ProxyConnection(ctx context.Context, connectionID, target string, port int, clientConn net.Conn) error {
	// Resolve target (could be IP or hostname)
	if port == 0 {
		port = 22
	}

	// Resolve hostname to IP using gateway's own dnsmasq (localhost:53)
	// This ensures VPS hostnames are resolved by the gateway's dnsmasq, not the host DNS
	// IMPORTANT: If target is already an IP, use it directly - don't resolve
	var targetIP string
	parsedIP := net.ParseIP(target)
	if parsedIP != nil {
		// Target is already an IP address - use it directly
		targetIP = target
		logger.Debug("Using provided IP address directly: %s", targetIP)
	} else {
		// Target is a hostname - resolve using gateway's dnsmasq via direct DNS query
		logger.Info("Resolving hostname %s using direct DNS query to 127.0.0.1:53", target)

		// First, verify dnsmasq is listening on 127.0.0.1:53
		testConn, err := net.DialTimeout("udp", "127.0.0.1:53", 1*time.Second)
		if err != nil {
			logger.Warn("dnsmasq not reachable on 127.0.0.1:53: %v. DNS resolution may fail. Please check dnsmasq is running and configured to listen on 127.0.0.1", err)
		} else {
			testConn.Close()
			logger.Debug("dnsmasq is reachable on 127.0.0.1:53")
		}

		// Use direct DNS query with explicit timeout
		// Create a new context with a timeout to prevent hanging
		resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		ips, err := queryDNSDirect(resolveCtx, target)
		if err != nil {
			logger.Debug("First resolution attempt failed for %s: %v", target, err)
			// Try with domain suffix if resolution fails
			domain := os.Getenv("GATEWAY_DHCP_DOMAIN")
			if domain == "" {
				domain = "vps.local"
			}
			fqdn := fmt.Sprintf("%s.%s", target, domain)
			logger.Debug("Trying FQDN: %s", fqdn)

			// Try FQDN with new context
			resolveCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
			defer cancel2()
			ips, err = queryDNSDirect(resolveCtx2, fqdn)
			if err != nil {
				logger.Error("DNS resolution failed for %s and %s: %v (check if dnsmasq is running on 127.0.0.1:53)", target, fqdn, err)
				return fmt.Errorf("failed to resolve hostname %s (tried %s and %s): %w", target, target, fqdn, err)
			}
		}
		if len(ips) == 0 {
			return fmt.Errorf("no IP addresses found for hostname %s", target)
		}
		// Use the first IP address (prefer IPv4 if available)
		var ip net.IP
		for _, candidate := range ips {
			if candidate.To4() != nil {
				ip = candidate
				break
			}
		}
		if ip == nil && len(ips) > 0 {
			ip = ips[0] // Use first IPv6 if no IPv4
		}
		if ip == nil {
			return fmt.Errorf("no valid IP address found for hostname %s", target)
		}
		logger.Info("Resolved %s to %s", target, ip.String())
		targetIP = ip.String()
	}

	// Use net.JoinHostPort to properly format IPv6 addresses
	targetAddr := net.JoinHostPort(targetIP, fmt.Sprintf("%d", port))
	logger.Info("Proxying SSH connection %s to %s (%s)", connectionID, targetAddr, target)

	// Dial target using resolved IP
	targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to target %s (%s): %w", targetAddr, target, err)
	}

	// Create connection record
	conn := &Connection{
		ID:           connectionID,
		Target:       target,
		Port:         port,
		ClientConn:   clientConn,
		TargetConn:   targetConn,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	p.mu.Lock()
	p.connections[connectionID] = conn
	p.mu.Unlock()

	// Cleanup on exit
	defer func() {
		p.mu.Lock()
		delete(p.connections, connectionID)
		p.mu.Unlock()
		targetConn.Close()
		logger.Info("Closed SSH proxy connection %s", connectionID)
	}()

	// Forward data bidirectionally
	errChan := make(chan error, 2)

	// Forward client -> target
	go func() {
		_, err := io.Copy(targetConn, clientConn)
		if err != nil {
			errChan <- fmt.Errorf("client->target copy error: %w", err)
		}
	}()

	// Forward target -> client
	go func() {
		_, err := io.Copy(clientConn, targetConn)
		if err != nil {
			errChan <- fmt.Errorf("target->client copy error: %w", err)
		}
	}()

	// Wait for connection to close or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			logger.Debug("SSH proxy connection %s error: %v", connectionID, err)
		}
		return nil
	}
}

// GetConnection returns a connection by ID
func (p *Proxy) GetConnection(connectionID string) (*Connection, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	conn, exists := p.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", connectionID)
	}

	return conn, nil
}

// ListConnections returns all active connections
func (p *Proxy) ListConnections() []*Connection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	connections := make([]*Connection, 0, len(p.connections))
	for _, conn := range p.connections {
		connections = append(connections, conn)
	}

	return connections
}

// Close closes the proxy and all connections
func (p *Proxy) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		if conn.ClientConn != nil {
			conn.ClientConn.Close()
		}
		if conn.TargetConn != nil {
			conn.TargetConn.Close()
		}
	}

	p.connections = make(map[string]*Connection)
	return nil
}

// GetStats returns proxy statistics
func (p *Proxy) GetStats() (activeConnections int, status string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	activeConnections = len(p.connections)
	status = "running"
	if activeConnections == 0 {
		status = "idle"
	}

	return activeConnections, status
}
