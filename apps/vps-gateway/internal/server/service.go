package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
	"vps-gateway/internal/metrics"
	"vps-gateway/internal/sshproxy"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GatewayService implements the VPSGatewayService
type GatewayService struct {
	vpsgatewayv1connect.UnimplementedVPSGatewayServiceHandler
	dhcpManager *dhcp.Manager
	sshProxy    *sshproxy.Proxy
	startTime   time.Time
}

// NewGatewayService creates a new gateway service
func NewGatewayService(dhcpManager *dhcp.Manager, sshProxy *sshproxy.Proxy) *GatewayService {
	return &GatewayService{
		dhcpManager: dhcpManager,
		sshProxy:    sshProxy,
		startTime:   time.Now(),
	}
}

// AllocateIP allocates a DHCP IP address for a VPS
func (s *GatewayService) AllocateIP(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.AllocateIPRequest],
) (*connect.Response[vpsgatewayv1.AllocateIPResponse], error) {
	alloc, err := s.dhcpManager.AllocateIP(
		ctx,
		req.Msg.VpsId,
		req.Msg.OrganizationId,
		req.Msg.MacAddress,
		req.Msg.PreferredIp,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to allocate IP: %w", err))
	}

	// Record metrics
	metrics.RecordDHCPAllocation(req.Msg.OrganizationId)

	// Update active allocations count for this org
	orgAllocs, _ := s.dhcpManager.ListIPs(ctx, req.Msg.OrganizationId, "")
	metrics.SetDHCPAllocationsActive(req.Msg.OrganizationId, float64(len(orgAllocs)))

	// Update pool metrics
	totalIPs, allocatedIPs, _ := s.dhcpManager.GetStats()
	metrics.SetDHCPPoolSize(float64(totalIPs))
	metrics.SetDHCPPoolAvailable(float64(totalIPs - allocatedIPs))

	_, _, subnetMask, gateway, dnsServers := s.dhcpManager.GetConfig()

	resp := &vpsgatewayv1.AllocateIPResponse{
		IpAddress:    alloc.IPAddress.String(),
		SubnetMask:   subnetMask,
		Gateway:      gateway,
		DnsServers:   dnsServers,
		LeaseExpires: timestamppb.New(alloc.LeaseExpires),
	}

	return connect.NewResponse(resp), nil
}

// ReleaseIP releases a DHCP IP address for a VPS
func (s *GatewayService) ReleaseIP(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.ReleaseIPRequest],
) (*connect.Response[vpsgatewayv1.ReleaseIPResponse], error) {
	err := s.dhcpManager.ReleaseIP(ctx, req.Msg.VpsId, req.Msg.IpAddress)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to release IP: %w", err))
	}

	// Get org ID from allocation before deleting
	var orgID string
	allocations, _ := s.dhcpManager.ListIPs(ctx, "", req.Msg.VpsId)
	if len(allocations) > 0 {
		orgID = allocations[0].OrganizationID
	}

	// Record metrics
	if orgID != "" {
		metrics.RecordDHCPRelease(orgID)
		// Update active allocations count for this org
		orgAllocs, _ := s.dhcpManager.ListIPs(ctx, orgID, "")
		metrics.SetDHCPAllocationsActive(orgID, float64(len(orgAllocs)))
	}

	// Update pool metrics
	totalIPs, allocatedIPs, _ := s.dhcpManager.GetStats()
	metrics.SetDHCPPoolSize(float64(totalIPs))
	metrics.SetDHCPPoolAvailable(float64(totalIPs - allocatedIPs))

	resp := &vpsgatewayv1.ReleaseIPResponse{
		Success: true,
		Message: fmt.Sprintf("Released IP for VPS %s", req.Msg.VpsId),
	}

	return connect.NewResponse(resp), nil
}

// ListIPs lists all allocated IP addresses
func (s *GatewayService) ListIPs(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.ListIPsRequest],
) (*connect.Response[vpsgatewayv1.ListIPsResponse], error) {
	allocations, err := s.dhcpManager.ListIPs(ctx, req.Msg.OrganizationId, req.Msg.VpsId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list IPs: %w", err))
	}

	protoAllocations := make([]*vpsgatewayv1.IPAllocation, len(allocations))
	for i, alloc := range allocations {
		protoAllocations[i] = &vpsgatewayv1.IPAllocation{
			VpsId:          alloc.VPSID,
			OrganizationId: alloc.OrganizationID,
			IpAddress:      alloc.IPAddress.String(),
			MacAddress:     alloc.MACAddress,
			AllocatedAt:    timestamppb.New(alloc.AllocatedAt),
			LeaseExpires:   timestamppb.New(alloc.LeaseExpires),
		}
	}

	resp := &vpsgatewayv1.ListIPsResponse{
		Allocations: protoAllocations,
	}

	return connect.NewResponse(resp), nil
}

// ProxySSH proxies SSH connections via bidirectional stream
func (s *GatewayService) ProxySSH(
	ctx context.Context,
	stream *connect.BidiStream[vpsgatewayv1.ProxySSHRequest, vpsgatewayv1.ProxySSHResponse],
) error {
	// Map to track client pipes by connection ID
	clientPipes := make(map[string]net.Conn)
	var mu sync.Mutex
	
	var connectionID string
	startTime := time.Now()

	logger.Info("[GatewayService] ProxySSH stream opened")

	defer func() {
		// Clean up all pipes
		mu.Lock()
		for _, pipe := range clientPipes {
			pipe.Close()
		}
		mu.Unlock()
		
		if connectionID != "" {
			duration := time.Since(startTime).Seconds()
			metrics.RecordSSHProxyConnectionDuration(connectionID, connectionID, duration)
			metrics.SetSSHProxyConnectionsActive(-1)
		}
	}()

	for {
		req, err := stream.Receive()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to receive request: %w", err))
		}

		connectionID = req.ConnectionId

		switch req.Type {
		case "connect":
			// Create new connection
			target := req.Target
			port := int(req.Port)
			if port == 0 {
				port = 22
			}

			// Create a pipe for bidirectional communication
			clientPipe, serverPipe := net.Pipe()

			// Store client pipe for this connection
			mu.Lock()
			clientPipes[connectionID] = clientPipe
			mu.Unlock()

			// Start proxying in goroutine
			go func() {
				err := s.sshProxy.ProxyConnection(ctx, connectionID, target, port, serverPipe)
				if err != nil {
					logger.Error("SSH proxy error for connection %s: %v", connectionID, err)
					stream.Send(&vpsgatewayv1.ProxySSHResponse{
						ConnectionId: connectionID,
						Type:         "error",
						Error:        err.Error(),
					})
				} else {
					stream.Send(&vpsgatewayv1.ProxySSHResponse{
						ConnectionId: connectionID,
						Type:         "closed",
					})
				}
				
				// Clean up pipe when connection closes
				mu.Lock()
				if pipe, exists := clientPipes[connectionID]; exists {
					pipe.Close()
					delete(clientPipes, connectionID)
				}
				mu.Unlock()
			}()

			// Send connected response
			if err := stream.Send(&vpsgatewayv1.ProxySSHResponse{
				ConnectionId: connectionID,
				Type:         "connected",
			}); err != nil {
				return err
			}

			// Record metrics
			metrics.RecordSSHProxyConnection(connectionID, connectionID)
			metrics.SetSSHProxyConnectionsActive(1)

			// Handle data forwarding from target to client (read from clientPipe, send to stream)
			go func(connID string, pipe net.Conn) {
				buf := make([]byte, 4096)
				for {
					n, err := pipe.Read(buf)
					if err != nil {
						if err != io.EOF {
							logger.Debug("Error reading from client pipe for connection %s: %v", connID, err)
						}
						return
					}
					if n > 0 {
						metrics.RecordSSHProxyBytes(connID, connID, "in", int64(n))
						if sendErr := stream.Send(&vpsgatewayv1.ProxySSHResponse{
							ConnectionId: connID,
							Type:         "data",
							Data:         buf[:n],
						}); sendErr != nil {
							logger.Error("Failed to send data to stream for connection %s: %v", connID, sendErr)
							return
						}
					}
				}
			}(connectionID, clientPipe)

		case "data":
			// Forward data from client to target (write to clientPipe)
			mu.Lock()
			clientPipe, exists := clientPipes[connectionID]
			mu.Unlock()
			
			if !exists {
				logger.Warn("Received data for unknown connection %s", connectionID)
				continue
			}
			
			if len(req.Data) > 0 {
				metrics.RecordSSHProxyBytes(connectionID, connectionID, "out", int64(len(req.Data)))
				if _, err := clientPipe.Write(req.Data); err != nil {
					logger.Error("Failed to write data to client pipe for connection %s: %v", connectionID, err)
					// Remove pipe from map on error
					mu.Lock()
					delete(clientPipes, connectionID)
					mu.Unlock()
					clientPipe.Close()
				}
			}

		case "close":
			// Close connection and clean up pipe
			mu.Lock()
			if pipe, exists := clientPipes[connectionID]; exists {
				pipe.Close()
				delete(clientPipes, connectionID)
			}
			mu.Unlock()
			return nil
		}
	}
}

// GetGatewayInfo returns gateway status and configuration
func (s *GatewayService) GetGatewayInfo(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.GetGatewayInfoRequest],
) (*connect.Response[vpsgatewayv1.GetGatewayInfoResponse], error) {
	poolStart, poolEnd, subnetMask, gateway, dnsServers := s.dhcpManager.GetConfig()
	totalIPs, allocatedIPs, dhcpStatus := s.dhcpManager.GetStats()
	_, sshProxyStatus := s.sshProxy.GetStats()

	// Update metrics
	metrics.SetGatewayUptime(time.Since(s.startTime).Seconds())
	metrics.SetDHCPServerStatus(dhcpStatus == "running")

	resp := &vpsgatewayv1.GetGatewayInfoResponse{
		Version:        "1.0.0",
		DhcpPoolStart:  poolStart,
		DhcpPoolEnd:    poolEnd,
		SubnetMask:     subnetMask,
		GatewayIp:      gateway,
		DnsServers:     dnsServers,
		TotalIps:       int32(totalIPs),
		AllocatedIps:   int32(allocatedIPs),
		DhcpStatus:     dhcpStatus,
		SshProxyStatus: sshProxyStatus,
	}

	return connect.NewResponse(resp), nil
}

// RegisterGateway handles API instance registration via bidirectional stream (forward connection pattern)
// API instances connect to the gateway and register themselves
func (s *GatewayService) RegisterGateway(
	ctx context.Context,
	stream *connect.BidiStream[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage],
) error {
	var apiInstanceID string
	startTime := time.Now()

	logger.Info("[GatewayService] New API connection attempt")

	// Handle incoming messages from API
	go func() {
		for {
			msg, err := stream.Receive()
			if err == io.EOF {
				logger.Info("[GatewayService] API instance %s disconnected", apiInstanceID)
				return
			}
			if err != nil {
				logger.Error("[GatewayService] Error receiving from API: %v", err)
				return
			}

			switch msg.Type {
			case "register":
				if msg.Registration == nil {
					logger.Error("[GatewayService] Registration message missing registration data")
					continue
				}
				reg := msg.Registration
				apiInstanceID = reg.GatewayId // Reusing GatewayId field for API instance ID

				logger.Info("[GatewayService] API instance %s registered (version: %s)", apiInstanceID, reg.Version)

				// Send registration confirmation
				if err := stream.Send(&vpsgatewayv1.GatewayMessage{
					Type: "registered",
				}); err != nil {
					logger.Error("[GatewayService] Failed to send registration confirmation: %v", err)
					return
				}

			case "metrics":
				// API can send metrics if needed
				logger.Debug("[GatewayService] Received metrics from API instance %s", apiInstanceID)

			case "request":
				// Handle requests from API (forwarded RPCs)
				if msg.Request != nil {
					logger.Debug("[GatewayService] Received request %s from API instance %s", msg.Request.Method, apiInstanceID)
					// Requests are handled directly via the RPC methods, not through the stream
				}

			case "heartbeat":
				logger.Debug("[GatewayService] Received heartbeat from API instance %s", apiInstanceID)

			default:
				logger.Warn("[GatewayService] Unknown message type: %s", msg.Type)
			}
		}
	}()

	// Send periodic heartbeats to API
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := stream.Send(&vpsgatewayv1.GatewayMessage{
					Type:      "heartbeat",
					Heartbeat: timestamppb.Now(),
				}); err != nil {
					logger.Debug("[GatewayService] Failed to send heartbeat: %v", err)
					return
				}
			}
		}
	}()

	// Send gateway info periodically
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Get gateway info
				_, _, _, _, _ = s.dhcpManager.GetConfig()
				totalIPs, allocatedIPs, dhcpStatus := s.dhcpManager.GetStats()
				_, sshProxyStatus := s.sshProxy.GetStats()

				// Log gateway status (could be used for monitoring)
				logger.Debug("[GatewayService] Gateway status: DHCP=%s, SSH=%s, IPs=%d/%d",
					dhcpStatus, sshProxyStatus, allocatedIPs, totalIPs)
			}
		}
	}()

	// Keep connection alive until context is cancelled
	<-ctx.Done()
	duration := time.Since(startTime).Seconds()
	logger.Info("[GatewayService] API instance %s connection closed (duration: %.2fs)", apiInstanceID, duration)
	return nil
}
