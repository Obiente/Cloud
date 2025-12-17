package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
	"vps-gateway/internal/metrics"
	"vps-gateway/internal/security"
	"vps-gateway/internal/sshproxy"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GatewayService implements the VPSGatewayService
type GatewayService struct {
	vpsgatewayv1connect.UnimplementedVPSGatewayServiceHandler
	dhcpManager     *dhcp.Manager
	sshProxy        *sshproxy.Proxy
	securityMgr     *security.Manager
	startTime       time.Time
	
	// Track connected VPS service instances
	connectedStreams   map[string]*connect.BidiStream[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage]
	streamsMu          sync.RWMutex
	pendingRequests    map[string]chan *vpsgatewayv1.GatewayResponse
	pendingRequestsMu  sync.RWMutex
	requestCounter     uint64
}

// NewGatewayService creates a new gateway service
func NewGatewayService(dhcpManager *dhcp.Manager, sshProxy *sshproxy.Proxy) (*GatewayService, error) {
	// Initialize security manager
	securityMgr, err := security.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize security manager: %w", err)
	}

	return &GatewayService{
		dhcpManager:      dhcpManager,
		sshProxy:         sshProxy,
		securityMgr:      securityMgr,
		startTime:        time.Now(),
		connectedStreams: make(map[string]*connect.BidiStream[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage]),
		pendingRequests:  make(map[string]chan *vpsgatewayv1.GatewayResponse),
	}, nil
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
		false, // allowPublicIP: false for regular DHCP allocations
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

// AllocatePublicIP allocates a public IP address for a VPS with security measures
func (s *GatewayService) AllocatePublicIP(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.AllocatePublicIPRequest],
) (*connect.Response[vpsgatewayv1.AllocatePublicIPResponse], error) {
	publicIP := req.Msg.GetPublicIp()
	macAddress := req.Msg.GetMacAddress()
	vpsID := req.Msg.GetVpsId()
	orgID := req.Msg.GetOrganizationId()

	if publicIP == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("public_ip is required"))
	}
	if macAddress == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("mac_address is required for public IP security"))
	}

	// Allocate the public IP (outside DHCP pool)
	_, err := s.dhcpManager.AllocateIP(
		ctx,
		vpsID,
		orgID,
		macAddress,
		publicIP,
		true, // allowPublicIP: true for public IPs
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to allocate public IP: %w", err))
	}

	// Apply security measures (firewall rules, ARP entries)
	if err := s.securityMgr.SecurePublicIP(ctx, publicIP, macAddress, vpsID); err != nil {
		// If security setup fails, release the allocation
		s.dhcpManager.ReleaseIP(ctx, vpsID, publicIP)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to secure public IP: %w", err))
	}

	// Calculate gateway and netmask if not provided
	gateway := req.Msg.GetGateway()
	netmask := req.Msg.GetNetmask()
	
	if gateway == "" {
		// Auto-calculate gateway (typically .1 in the subnet)
		ip := net.ParseIP(publicIP)
		if ip != nil && ip.To4() != nil {
			ip4 := ip.To4()
			ip4[3] = 1 // Set last octet to 1
			gateway = ip4.String()
		}
	}
	
	if netmask == "" {
		netmask = "24" // Default /24
	}

	resp := &vpsgatewayv1.AllocatePublicIPResponse{
		IpAddress: publicIP,
		Gateway:   gateway,
		Netmask:   netmask,
		Success:   true,
		Message:   fmt.Sprintf("Public IP %s allocated and secured for VPS %s", publicIP, vpsID),
	}

	logger.Info("Allocated and secured public IP %s for VPS %s (org: %s)", publicIP, vpsID, orgID)
	return connect.NewResponse(resp), nil
}

// ReleasePublicIP releases a public IP address and removes security measures
func (s *GatewayService) ReleasePublicIP(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.ReleasePublicIPRequest],
) (*connect.Response[vpsgatewayv1.ReleasePublicIPResponse], error) {
	publicIP := req.Msg.GetPublicIp()
	vpsID := req.Msg.GetVpsId()
	macAddress := req.Msg.GetMacAddress()

	if publicIP == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("public_ip is required"))
	}

	// Remove security measures first
	if err := s.securityMgr.RemovePublicIPSecurity(ctx, publicIP, macAddress); err != nil {
		logger.Warn("Failed to remove security for public IP %s: %v", publicIP, err)
		// Continue - try to release anyway
	}

	// Release the IP allocation
	err := s.dhcpManager.ReleaseIP(ctx, vpsID, publicIP)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to release public IP: %w", err))
	}

	resp := &vpsgatewayv1.ReleasePublicIPResponse{
		Success: true,
		Message: fmt.Sprintf("Public IP %s released and security removed", publicIP),
	}

	logger.Info("Released public IP %s for VPS %s", publicIP, vpsID)
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
		isPublic := !s.dhcpManager.IsIPInPool(alloc.IPAddress)
		protoAllocations[i] = &vpsgatewayv1.IPAllocation{
			VpsId:          alloc.VPSID,
			OrganizationId: alloc.OrganizationID,
			IpAddress:      alloc.IPAddress.String(),
			MacAddress:     alloc.MACAddress,
			AllocatedAt:    timestamppb.New(alloc.AllocatedAt),
			LeaseExpires:   timestamppb.New(alloc.LeaseExpires),
			IsPublic:       isPublic,
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
	// Map to track active data forwarding goroutines
	dataForwardGoroutines := make(map[string]context.CancelFunc)
	var mu sync.Mutex
	
	var connectionID string
	startTime := time.Now()

	logger.Info("[GatewayService] ProxySSH stream opened")

	// Create a context that will be cancelled when the handler exits
	handlerCtx, cancelHandler := context.WithCancel(ctx)
	defer cancelHandler()

	defer func() {
		// Cancel all data forwarding goroutines
		mu.Lock()
		for _, cancel := range dataForwardGoroutines {
			cancel()
		}
		// Clean up all pipes
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

			// Create context for this connection's data forwarding
			forwardCtx, cancelForward := context.WithCancel(handlerCtx)
			
			// Start data forwarding goroutine BEFORE starting the proxy connection
			// This ensures we're ready to receive data immediately when the VPS sends it
			// Handle data forwarding from target to client (read from clientPipe, send to stream)
			dataForwardReady := make(chan struct{})
			go func(connID string, pipe net.Conn, fwdCtx context.Context) {
				defer cancelForward() // Clean up when goroutine exits
				close(dataForwardReady) // Signal that we're ready to read
				logger.Debug("Data forwarding goroutine started for connection %s", connID)
				buf := make([]byte, 4096)
				
				// Start a goroutine to close the pipe when context is cancelled
				go func() {
					<-fwdCtx.Done()
					pipe.Close()
				}()
				
				for {
					n, err := pipe.Read(buf)
					if err != nil {
						// Check if error is due to context cancellation
						select {
						case <-fwdCtx.Done():
							logger.Debug("Data forwarding goroutine cancelled for connection %s", connID)
							return
						default:
						}
						if err != io.EOF {
							logger.Debug("Error reading from client pipe for connection %s: %v", connID, err)
						}
						return
					}
					if n > 0 {
						logger.Debug("Forwarding %d bytes from VPS to client for connection %s", n, connID)
						metrics.RecordSSHProxyBytes(connID, connID, "in", int64(n))
						
						// Check context before sending
						select {
						case <-fwdCtx.Done():
							logger.Debug("Context cancelled, stopping data forwarding for connection %s", connID)
							return
						default:
						}
						
						if sendErr := stream.Send(&vpsgatewayv1.ProxySSHResponse{
							ConnectionId: connID,
							Type:         "data",
							Data:         buf[:n],
						}); sendErr != nil {
							// Stream might be closed - this is expected when handler exits
							logger.Debug("Failed to send data to stream for connection %s (stream may be closed): %v", connID, sendErr)
							return
						}
					}
				}
			}(connectionID, clientPipe, forwardCtx)
			
			// Store cancel function for cleanup
			mu.Lock()
			dataForwardGoroutines[connectionID] = cancelForward
			mu.Unlock()
			
			// Wait for data forwarding goroutine to be ready before starting proxy
			<-dataForwardReady

			// Start proxying in goroutine
			go func() {
				err := s.sshProxy.ProxyConnection(handlerCtx, connectionID, target, port, serverPipe)
				if err != nil {
					logger.Error("SSH proxy error for connection %s: %v", connectionID, err)
					// Try to send error, but don't fail if stream is closed
					select {
					case <-handlerCtx.Done():
						// Handler is closing, don't send
					default:
						stream.Send(&vpsgatewayv1.ProxySSHResponse{
							ConnectionId: connectionID,
							Type:         "error",
							Error:        err.Error(),
						})
					}
				} else {
					// Try to send closed, but don't fail if stream is closed
					select {
					case <-handlerCtx.Done():
						// Handler is closing, don't send
					default:
						stream.Send(&vpsgatewayv1.ProxySSHResponse{
							ConnectionId: connectionID,
							Type:         "closed",
						})
					}
				}
				
				// Clean up pipe when connection closes
				mu.Lock()
				if pipe, exists := clientPipes[connectionID]; exists {
					pipe.Close()
					delete(clientPipes, connectionID)
				}
				if cancel, exists := dataForwardGoroutines[connectionID]; exists {
					cancel()
					delete(dataForwardGoroutines, connectionID)
				}
				mu.Unlock()
			}()

			// Send connected response AFTER starting the data forwarding goroutine
			// This ensures we're ready to receive data immediately
			if err := stream.Send(&vpsgatewayv1.ProxySSHResponse{
				ConnectionId: connectionID,
				Type:         "connected",
			}); err != nil {
				return err
			}

			// Record metrics
			metrics.RecordSSHProxyConnection(connectionID, connectionID)
			metrics.SetSSHProxyConnectionsActive(1)

		case "data":
			// Forward data from client to target (write to clientPipe)
			mu.Lock()
			clientPipe, exists := clientPipes[connectionID]
			mu.Unlock()
			
			if !exists {
				logger.Warn("Received data for unknown connection %s (connection may have been closed)", connectionID)
				// Send error response
				stream.Send(&vpsgatewayv1.ProxySSHResponse{
					ConnectionId: connectionID,
					Type:         "error",
					Error:        "connection not found",
				})
				continue
			}
			
			if len(req.Data) > 0 {
				logger.Debug("Forwarding %d bytes from client to VPS for connection %s", len(req.Data), connectionID)
				metrics.RecordSSHProxyBytes(connectionID, connectionID, "out", int64(len(req.Data)))
				// Set write deadline to prevent indefinite blocking
				if err := clientPipe.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
					logger.Debug("Failed to set write deadline for connection %s: %v", connectionID, err)
				}
				if _, err := clientPipe.Write(req.Data); err != nil {
					logger.Error("Failed to write data to client pipe for connection %s: %v", connectionID, err)
					// Remove pipe from map on error and send error response
					mu.Lock()
					delete(clientPipes, connectionID)
					mu.Unlock()
					clientPipe.Close()
					stream.Send(&vpsgatewayv1.ProxySSHResponse{
						ConnectionId: connectionID,
						Type:         "error",
						Error:        fmt.Sprintf("failed to write data: %v", err),
					})
				} else {
					// Clear write deadline on success
					clientPipe.SetWriteDeadline(time.Time{})
				}
			}

		case "close":
			// Close connection and clean up pipe
			mu.Lock()
			if pipe, exists := clientPipes[connectionID]; exists {
				pipe.Close()
				delete(clientPipes, connectionID)
			}
			if cancel, exists := dataForwardGoroutines[connectionID]; exists {
				cancel()
				delete(dataForwardGoroutines, connectionID)
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
		defer func() {
			// Clean up on disconnect
			if apiInstanceID != "" {
				s.streamsMu.Lock()
				delete(s.connectedStreams, apiInstanceID)
				s.streamsMu.Unlock()
				logger.Info("[GatewayService] Cleaned up stream for disconnected VPS instance %s", apiInstanceID)
			}
		}()
		
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

				// Store stream for this VPS service instance
				s.streamsMu.Lock()
				s.connectedStreams[apiInstanceID] = stream
				s.streamsMu.Unlock()

				// Send registration confirmation
				if err := stream.Send(&vpsgatewayv1.GatewayMessage{
					Type: "registered",
				}); err != nil {
					logger.Error("[GatewayService] Failed to send registration confirmation: %v", err)
					return
				}

				// Trigger DHCP hosts file sync now that a VPS service is connected
				// This resolves any existing DHCP leases to VPS IDs
				go func() {
					logger.Info("[GatewayService] Triggering DHCP sync after API registration")
					if err := s.dhcpManager.SyncHostsFromLeases(); err != nil {
						logger.Warn("[GatewayService] DHCP sync after registration failed: %v", err)
					} else {
						logger.Info("[GatewayService] DHCP sync completed successfully")
					}
				}()

			case "response":
				// Handle response to our request
				if msg.Response != nil {
					s.pendingRequestsMu.RLock()
					respChan, ok := s.pendingRequests[msg.Response.RequestId]
					s.pendingRequestsMu.RUnlock()

					if ok {
						// Non-blocking send
						select {
						case respChan <- msg.Response:
						default:
							logger.Warn("[GatewayService] Response channel full for request %s", msg.Response.RequestId)
						}
					} else {
						logger.Debug("[GatewayService] Received response for unknown request ID: %s", msg.Response.RequestId)
					}
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

			case "sync_allocations":
				// VPS service is requesting allocation sync from database
				if msg.SyncAllocations != nil {
					logger.Info("[GatewayService] Received sync_allocations request from API instance %s with %d allocations", apiInstanceID, len(msg.SyncAllocations.Allocations))
					
					// Process the sync (delegate to existing SyncAllocations logic)
					req := connect.NewRequest(msg.SyncAllocations)
					resp, err := s.SyncAllocations(ctx, req)
					
					var syncResp *vpsgatewayv1.SyncAllocationsResponse
					if err != nil {
						logger.Error("[GatewayService] Sync allocations failed: %v", err)
						syncResp = &vpsgatewayv1.SyncAllocationsResponse{
							Success: false,
							Message: fmt.Sprintf("Sync failed: %v", err),
						}
					} else {
						syncResp = resp.Msg
						logger.Info("[GatewayService] Sync allocations completed: added=%d removed=%d", syncResp.Added, syncResp.Removed)
					}
					
					// Send response back via stream (optional - for confirmation)
					if err := stream.Send(&vpsgatewayv1.GatewayMessage{
						Type: "sync_result",
						Response: &vpsgatewayv1.GatewayResponse{
							RequestId: "sync_allocations",
							Success:   syncResp.Success,
							Error:     syncResp.Message,
						},
					}); err != nil {
						logger.Error("[GatewayService] Failed to send sync result: %v", err)
					}
				}

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

// GetLeases retrieves all active DHCP leases from dnsmasq
func (s *GatewayService) GetLeases(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.GetLeasesRequest],
) (*connect.Response[vpsgatewayv1.GetLeasesResponse], error) {
	leases, err := s.dhcpManager.GetActiveLeases()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read leases: %w", err))
	}

	protoLeases := make([]*vpsgatewayv1.LeaseRecord, len(leases))
	for i, lease := range leases {
		protoLeases[i] = &vpsgatewayv1.LeaseRecord{
			MacAddress: lease.MAC,
			IpAddress:  lease.IP.String(),
			Hostname:   lease.Hostname,
			ExpiresAt:  timestamppb.New(lease.ExpiresAt),
		}
	}

	resp := &vpsgatewayv1.GetLeasesResponse{
		Leases: protoLeases,
	}

	logger.Debug("GetLeases: returned %d active leases", len(leases))
	return connect.NewResponse(resp), nil
}

// GetOrgLeases retrieves active DHCP leases for a specific organization
// Filters by organization ID and optionally by VPS ID for frontend display
func (s *GatewayService) GetOrgLeases(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.GetOrgLeasesRequest],
) (*connect.Response[vpsgatewayv1.GetOrgLeasesResponse], error) {
	if req.Msg.OrganizationId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Get allocations for this organization
	allocations, err := s.dhcpManager.ListIPs(ctx, req.Msg.OrganizationId, req.Msg.VpsId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list IPs: %w", err))
	}

	// Build result with lease info
	protoLeases := make([]*vpsgatewayv1.OrgLeaseRecord, len(allocations))
	for i, alloc := range allocations {
		// Determine if this is a public IP (outside DHCP pool)
		isPublic := !s.dhcpManager.IsIPInPool(alloc.IPAddress)

		protoLeases[i] = &vpsgatewayv1.OrgLeaseRecord{
			VpsId:          alloc.VPSID,
			OrganizationId: alloc.OrganizationID,
			MacAddress:     alloc.MACAddress,
			IpAddress:      alloc.IPAddress.String(),
			ExpiresAt:      timestamppb.New(alloc.LeaseExpires),
			IsPublic:       isPublic,
		}
	}

	resp := &vpsgatewayv1.GetOrgLeasesResponse{
		Leases: protoLeases,
	}

	logger.Debug("GetOrgLeases: returned %d leases for org %s", len(allocations), req.Msg.OrganizationId)
	return connect.NewResponse(resp), nil
}

// SyncAllocations syncs allocations from database as source of truth
// Releases IPs not in the list and ensures desired IPs are allocated
func (s *GatewayService) SyncAllocations(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.SyncAllocationsRequest],
) (*connect.Response[vpsgatewayv1.SyncAllocationsResponse], error) {
	// Build desired allocations map
	desiredMap := make(map[string]*vpsgatewayv1.DesiredAllocation)
	for _, alloc := range req.Msg.Allocations {
		desiredMap[alloc.VpsId] = alloc
	}

	// Get current allocations
	existing, err := s.dhcpManager.ListIPs(ctx, "", "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list current IPs: %w", err))
	}

	var removed, added int

	// Release IPs not in desired set
	for _, alloc := range existing {
		if _, ok := desiredMap[alloc.VPSID]; !ok {
			if err := s.dhcpManager.ReleaseIP(ctx, alloc.VPSID, alloc.IPAddress.String()); err != nil {
				logger.Warn("SyncAllocations: failed to release %s: %v", alloc.VPSID, err)
			} else {
				removed++
			}
		}
	}

	// Ensure desired allocations exist
	for vpsID, desired := range desiredMap {
		_, err := s.dhcpManager.AllocateIP(
			ctx,
			vpsID,
			desired.OrganizationId,
			desired.MacAddress,
			desired.IpAddress,
			desired.IsPublic,
		)
		if err != nil {
			logger.Warn("SyncAllocations: failed to allocate %s -> %s: %v", vpsID, desired.IpAddress, err)
		} else {
			added++
		}
	}

	resp := &vpsgatewayv1.SyncAllocationsResponse{
		Success: true,
		Added:   int32(added),
		Removed: int32(removed),
		Message: fmt.Sprintf("Synced allocations: added %d, removed %d", added, removed),
	}

	logger.Info("SyncAllocations: completed with added=%d removed=%d", added, removed)
	return connect.NewResponse(resp), nil
}

// FindVPSByLease sends a FindVPSByLease request to ALL connected VPS service instances and returns the first valid response
// This method implements the dhcp.APIClient interface so DHCP manager can use it
func (s *GatewayService) FindVPSByLease(ctx context.Context, ip string, mac string) (*vpsv1.FindVPSByLeaseResponse, error) {
	// Marshal request payload once
	findReq := &vpsv1.FindVPSByLeaseRequest{
		Ip:  ip,
		Mac: mac,
	}
	payloadBytes, err := proto.Marshal(findReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FindVPSByLease request: %w", err)
	}

	// Send to all connected VPS service instances with unique request IDs
	s.streamsMu.RLock()
	var requestIDs []string
	var responseChans []chan *vpsgatewayv1.GatewayResponse
	
	for instanceID, stream := range s.connectedStreams {
		if stream == nil {
			continue
		}

		// Generate unique request ID for this instance
		requestID := fmt.Sprintf("gateway-findvps-%d-%s", atomic.AddUint64(&s.requestCounter, 1), instanceID)
		
		// Create response channel for this request
		respChan := make(chan *vpsgatewayv1.GatewayResponse, 1)
		responseChans = append(responseChans, respChan)
		requestIDs = append(requestIDs, requestID)
		
		s.pendingRequestsMu.Lock()
		s.pendingRequests[requestID] = respChan
		s.pendingRequestsMu.Unlock()

		// Create and send request
		gatewayReq := &vpsgatewayv1.GatewayRequest{
			RequestId: requestID,
			Method:    "FindVPSByLease",
			Payload:   payloadBytes,
		}
		
		msg := &vpsgatewayv1.GatewayMessage{
			Type:    "request",
			Request: gatewayReq,
		}
		
		if sendErr := stream.Send(msg); sendErr != nil {
			logger.Debug("[GatewayService] Failed to send FindVPSByLease request to VPS instance %s: %v", instanceID, sendErr)
			// Clean up failed request
			s.pendingRequestsMu.Lock()
			delete(s.pendingRequests, requestID)
			s.pendingRequestsMu.Unlock()
			close(respChan)
			// Remove from slices
			requestIDs = requestIDs[:len(requestIDs)-1]
			responseChans = responseChans[:len(responseChans)-1]
		}
	}
	s.streamsMu.RUnlock()

	// Clean up all pending requests when done
	defer func() {
		s.pendingRequestsMu.Lock()
		for _, reqID := range requestIDs {
			delete(s.pendingRequests, reqID)
		}
		s.pendingRequestsMu.Unlock()
		for _, ch := range responseChans {
			close(ch)
		}
	}()

	if len(responseChans) == 0 {
		logger.Debug("[GatewayService] No connected VPS service instances for FindVPSByLease")
		return &vpsv1.FindVPSByLeaseResponse{}, nil
	}

	// Wait for responses from all instances, return first valid one
	timeout := 5 * time.Second
	deadline := time.After(timeout)
	responsesReceived := 0
	
	for responsesReceived < len(responseChans) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			// Timeout - return empty response if no valid results found
			logger.Debug("[GatewayService] Timeout waiting for FindVPSByLease responses (received %d/%d)", responsesReceived, len(responseChans))
			return &vpsv1.FindVPSByLeaseResponse{}, nil
		case gatewayResp := <-mergeResponseChannels(responseChans):
			responsesReceived++
			
			if gatewayResp.Error != "" {
				logger.Debug("[GatewayService] FindVPSByLease error from VPS instance: %s", gatewayResp.Error)
				continue
			}

			// Unmarshal response payload
			var findResp vpsv1.FindVPSByLeaseResponse
			if err := proto.Unmarshal(gatewayResp.Payload, &findResp); err != nil {
				logger.Debug("[GatewayService] Failed to unmarshal FindVPSByLease response: %v", err)
				continue
			}

			// If this response has a VPS ID, return it immediately
			if findResp.GetVpsId() != "" {
				logger.Debug("[GatewayService] Found VPS %s for lease IP=%s MAC=%s", findResp.GetVpsId(), ip, mac)
				return &findResp, nil
			}
		}
	}

	// All instances responded but none had the VPS
	return &vpsv1.FindVPSByLeaseResponse{}, nil
}

// mergeResponseChannels merges multiple response channels into a single channel
func mergeResponseChannels(channels []chan *vpsgatewayv1.GatewayResponse) <-chan *vpsgatewayv1.GatewayResponse {
	merged := make(chan *vpsgatewayv1.GatewayResponse, len(channels))
	
	for _, ch := range channels {
		go func(c chan *vpsgatewayv1.GatewayResponse) {
			if resp, ok := <-c; ok {
				merged <- resp
			}
		}(ch)
	}
	
	return merged
}
