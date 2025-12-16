package gateway

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"strings"

	"google.golang.org/protobuf/proto"
)

// FindVPSByLeaseHandler handles FindVPSByLease requests from the gateway
type FindVPSByLeaseHandler struct{}

// NewFindVPSByLeaseHandler creates a new FindVPSByLease handler
func NewFindVPSByLeaseHandler() *FindVPSByLeaseHandler {
	return &FindVPSByLeaseHandler{}
}

// HandleRequest implements RequestHandler interface
func (h *FindVPSByLeaseHandler) HandleRequest(ctx context.Context, method string, payload []byte) ([]byte, error) {
	// Unmarshal request
	var req vpsv1.FindVPSByLeaseRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FindVPSByLease request: %w", err)
	}

	// Query database directly to find VPS by lease
	mac := strings.ToLower(strings.TrimSpace(req.GetMac()))
	ip := strings.TrimSpace(req.GetIp())

	var lease database.DHCPLease
	var found bool

	if mac != "" {
		if err := database.DB.WithContext(ctx).Where("mac_address = ?", mac).First(&lease).Error; err == nil {
			found = true
		}
	}
	if !found && ip != "" {
		if err := database.DB.WithContext(ctx).Where("ip_address = ?", ip).First(&lease).Error; err == nil {
			found = true
		}
	}

	// Create response
	resp := &vpsv1.FindVPSByLeaseResponse{}
	if found {
		resp.VpsId = lease.VPSID
		resp.OrganizationId = lease.OrganizationID
	}

	// Marshal response
	respPayload, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FindVPSByLease response: %w", err)
	}

	return respPayload, nil
}
