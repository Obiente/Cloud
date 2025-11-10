package sshproxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
)

// Custom DNS resolver that uses localhost (gateway's own dnsmasq)
var localDNSResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{
			Timeout: time.Millisecond * time.Duration(10000),
		}
		// Use localhost:53 to query gateway's own dnsmasq
		return d.DialContext(ctx, network, "127.0.0.1:53")
	},
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
	var targetIP string
	if net.ParseIP(target) != nil {
		// Target is already an IP address
		targetIP = target
	} else {
		// Target is a hostname - resolve using gateway's dnsmasq
		logger.Info("Resolving hostname %s using gateway's dnsmasq (127.0.0.1:53)", target)
		
		// First, verify dnsmasq is listening on 127.0.0.1:53
		testConn, err := net.DialTimeout("udp", "127.0.0.1:53", 1*time.Second)
		if err != nil {
			logger.Warn("dnsmasq not reachable on 127.0.0.1:53: %v. DNS resolution may fail. Please check dnsmasq is running and configured to listen on 127.0.0.1", err)
		} else {
			testConn.Close()
			logger.Debug("dnsmasq is reachable on 127.0.0.1:53")
		}
		
		ips, err := localDNSResolver.LookupIPAddr(ctx, target)
		if err != nil {
			logger.Debug("First resolution attempt failed for %s: %v", target, err)
			// Try with domain suffix if resolution fails
			domain := os.Getenv("GATEWAY_DHCP_DOMAIN")
			if domain == "" {
				domain = "vps.local"
			}
			fqdn := fmt.Sprintf("%s.%s", target, domain)
			logger.Debug("Trying FQDN: %s", fqdn)
			ips, err = localDNSResolver.LookupIPAddr(ctx, fqdn)
			if err != nil {
				logger.Error("DNS resolution failed for %s and %s: %v", target, fqdn, err)
				return fmt.Errorf("failed to resolve hostname %s (tried %s and %s): %w", target, target, fqdn, err)
			}
		}
		if len(ips) == 0 {
			return fmt.Errorf("no IP addresses found for hostname %s", target)
		}
		targetIP = ips[0].IP.String()
		logger.Info("Resolved %s to %s", target, targetIP)
	}

	targetAddr := fmt.Sprintf("%s:%d", targetIP, port)
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

