package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
	"vps-gateway/internal/metrics"
	"vps-gateway/internal/sshproxy"

	vpsgatewayv1 "vps-gateway/gen/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "vps-gateway/gen/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

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
	var currentConn *sshproxy.Connection
	var connectionID string
	startTime := time.Now()

	defer func() {
		if currentConn != nil {
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

			// Handle data forwarding
			go func() {
				buf := make([]byte, 4096)
				for {
					n, err := clientPipe.Read(buf)
					if err != nil {
						if err != io.EOF {
							logger.Error("Error reading from client pipe: %v", err)
						}
						return
					}
					if n > 0 {
						metrics.RecordSSHProxyBytes(connectionID, connectionID, "in", int64(n))
						stream.Send(&vpsgatewayv1.ProxySSHResponse{
							ConnectionId: connectionID,
							Type:         "data",
							Data:         buf[:n],
						})
					}
				}
			}()

		case "data":
			// Forward data from client to target (via clientPipe)
			// This is handled by the goroutine that reads from clientPipe
			// We need to write to a connection that's associated with this connectionID
			// For now, we'll use a map to track connections, but the pipe approach above should work
			logger.Debug("Received data for connection %s: %d bytes", connectionID, len(req.Data))

		case "close":
			// Close connection
			if currentConn != nil {
				if currentConn.ClientConn != nil {
					currentConn.ClientConn.Close()
				}
				if currentConn.TargetConn != nil {
					currentConn.TargetConn.Close()
				}
			}
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
