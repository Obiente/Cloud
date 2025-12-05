package vps

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	orchestrator "github.com/obiente/cloud/apps/vps-service/orchestrator"

	"github.com/obiente/cloud/apps/shared/pkg/database"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

// ListFirewallRules lists all firewall rules for a VPS
func (s *Service) ListFirewallRules(ctx context.Context, req *connect.Request[vpsv1.ListFirewallRulesRequest]) (*connect.Response[vpsv1.ListFirewallRulesResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.view"); err != nil {
		return nil, err
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID"))
	}

	vmID, err := strconv.Atoi(*vps.InstanceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid VM ID: %w", err))
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use"))
	}

	// Get Proxmox client for the node where VPS is running
	vpsManager, err := orchestrator.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}
	defer vpsManager.Close()

	proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err))
	}

	// List firewall rules
	rules, err := proxmoxClient.ListFirewallRules(ctx, nodeName, vmID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list firewall rules: %w", err))
	}

	// Convert to proto
	protoRules := make([]*vpsv1.FirewallRule, len(rules))
	for i, rule := range rules {
		protoRules[i] = firewallRuleToProto(rule, i)
	}

	return connect.NewResponse(&vpsv1.ListFirewallRulesResponse{
		Rules: protoRules,
	}), nil
}

// GetFirewallRule gets a specific firewall rule
func (s *Service) GetFirewallRule(ctx context.Context, req *connect.Request[vpsv1.GetFirewallRuleRequest]) (*connect.Response[vpsv1.GetFirewallRuleResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.view"); err != nil {
		return nil, err
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID"))
	}

	vmID, err := strconv.Atoi(*vps.InstanceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid VM ID: %w", err))
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use"))
	}

	// Get Proxmox client for the node where VPS is running
	vpsManager, err := orchestrator.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}
	defer vpsManager.Close()

	proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err))
	}

	// Get firewall rule
	rule, err := proxmoxClient.GetFirewallRule(ctx, nodeName, vmID, int(req.Msg.GetRulePos()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get firewall rule: %w", err))
	}

	return connect.NewResponse(&vpsv1.GetFirewallRuleResponse{
		Rule: firewallRuleToProto(rule, int(req.Msg.GetRulePos())),
	}), nil
}

// CreateFirewallRule creates a new firewall rule
func (s *Service) CreateFirewallRule(ctx context.Context, req *connect.Request[vpsv1.CreateFirewallRuleRequest]) (*connect.Response[vpsv1.CreateFirewallRuleResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
		return nil, err
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID"))
	}

	vmID, err := strconv.Atoi(*vps.InstanceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid VM ID: %w", err))
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use"))
	}

	// Get Proxmox client for the node where VPS is running
	vpsManager, err := orchestrator.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}
	defer vpsManager.Close()

	proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err))
	}

	// Convert proto rule to form data
	ruleData := firewallRuleToFormData(req.Msg.GetRule())
	var pos *int
	if req.Msg.Pos != nil {
		p := int(*req.Msg.Pos)
		pos = &p
	}

	// Create firewall rule
	if err := proxmoxClient.CreateFirewallRule(ctx, nodeName, vmID, ruleData, pos); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create firewall rule: %w", err))
	}

	// Get the created rule (Proxmox returns the position)
	// We'll need to list rules to find the new one
	rules, err := proxmoxClient.ListFirewallRules(ctx, nodeName, vmID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list firewall rules after creation: %w", err))
	}

	// Find the rule we just created (it should be at the specified position or at the end)
	var createdRule map[string]interface{}
	if pos != nil && *pos < len(rules) {
		createdRule = rules[*pos]
	} else if len(rules) > 0 {
		createdRule = rules[len(rules)-1]
	} else {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find created firewall rule"))
	}

	rulePos := len(rules) - 1
	if pos != nil {
		rulePos = *pos
	}

	return connect.NewResponse(&vpsv1.CreateFirewallRuleResponse{
		Rule: firewallRuleToProto(createdRule, rulePos),
	}), nil
}

// UpdateFirewallRule updates an existing firewall rule
func (s *Service) UpdateFirewallRule(ctx context.Context, req *connect.Request[vpsv1.UpdateFirewallRuleRequest]) (*connect.Response[vpsv1.UpdateFirewallRuleResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
		return nil, err
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID"))
	}

	vmID, err := strconv.Atoi(*vps.InstanceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid VM ID: %w", err))
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use"))
	}

	// Get Proxmox client for the node where VPS is running
	vpsManager, err := orchestrator.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}
	defer vpsManager.Close()

	proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err))
	}

	// Convert proto rule to form data
	ruleData := firewallRuleToFormData(req.Msg.GetRule())

	// Update firewall rule
	if err := proxmoxClient.UpdateFirewallRule(ctx, nodeName, vmID, int(req.Msg.GetRulePos()), ruleData); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update firewall rule: %w", err))
	}

	// Get the updated rule
	rule, err := proxmoxClient.GetFirewallRule(ctx, nodeName, vmID, int(req.Msg.GetRulePos()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get updated firewall rule: %w", err))
	}

	return connect.NewResponse(&vpsv1.UpdateFirewallRuleResponse{
		Rule: firewallRuleToProto(rule, int(req.Msg.GetRulePos())),
	}), nil
}

// DeleteFirewallRule deletes a firewall rule
func (s *Service) DeleteFirewallRule(ctx context.Context, req *connect.Request[vpsv1.DeleteFirewallRuleRequest]) (*connect.Response[vpsv1.DeleteFirewallRuleResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
		return nil, err
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID"))
	}

	vmID, err := strconv.Atoi(*vps.InstanceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid VM ID: %w", err))
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use"))
	}

	// Get Proxmox client for the node where VPS is running
	vpsManager, err := orchestrator.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}
	defer vpsManager.Close()

	proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err))
	}

	// Delete firewall rule
	if err := proxmoxClient.DeleteFirewallRule(ctx, nodeName, vmID, int(req.Msg.GetRulePos())); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete firewall rule: %w", err))
	}

	return connect.NewResponse(&vpsv1.DeleteFirewallRuleResponse{
		Success: true,
	}), nil
}

// GetFirewallOptions gets firewall options for a VPS
func (s *Service) GetFirewallOptions(ctx context.Context, req *connect.Request[vpsv1.GetFirewallOptionsRequest]) (*connect.Response[vpsv1.GetFirewallOptionsResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.view"); err != nil {
		return nil, err
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID"))
	}

	vmID, err := strconv.Atoi(*vps.InstanceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid VM ID: %w", err))
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use"))
	}

	// Get Proxmox client for the node where VPS is running
	vpsManager, err := orchestrator.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}
	defer vpsManager.Close()

	proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err))
	}

	// Get firewall options
	options, err := proxmoxClient.GetFirewallOptions(ctx, nodeName, vmID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get firewall options: %w", err))
	}

	return connect.NewResponse(&vpsv1.GetFirewallOptionsResponse{
		Options: firewallOptionsToProto(options),
	}), nil
}

// UpdateFirewallOptions updates firewall options for a VPS
func (s *Service) UpdateFirewallOptions(ctx context.Context, req *connect.Request[vpsv1.UpdateFirewallOptionsRequest]) (*connect.Response[vpsv1.UpdateFirewallOptionsResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
		return nil, err
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID"))
	}

	vmID, err := strconv.Atoi(*vps.InstanceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid VM ID: %w", err))
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use"))
	}

	// Get Proxmox client for the node where VPS is running
	vpsManager, err := orchestrator.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}
	defer vpsManager.Close()

	proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err))
	}

	// Convert proto options to form data
	optionsData := firewallOptionsToFormData(req.Msg.GetOptions())

	// Update firewall options
	if err := proxmoxClient.UpdateFirewallOptions(ctx, nodeName, vmID, optionsData); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update firewall options: %w", err))
	}

	// Get the updated options
	options, err := proxmoxClient.GetFirewallOptions(ctx, nodeName, vmID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get updated firewall options: %w", err))
	}

	return connect.NewResponse(&vpsv1.UpdateFirewallOptionsResponse{
		Options: firewallOptionsToProto(options),
	}), nil
}

// Helper functions

func firewallRuleToProto(rule map[string]interface{}, pos int) *vpsv1.FirewallRule {
	protoRule := &vpsv1.FirewallRule{
		Pos: int32(pos),
	}

	// Enable
	if enable, ok := rule["enable"].(float64); ok {
		protoRule.Enable = enable != 0
	} else if enable, ok := rule["enable"].(bool); ok {
		protoRule.Enable = enable
	}

	// Action
	if action, ok := rule["action"].(string); ok {
		switch action {
		case "ACCEPT":
			protoRule.Action = vpsv1.FirewallAction_ACCEPT
		case "REJECT":
			protoRule.Action = vpsv1.FirewallAction_REJECT
		case "DROP":
			protoRule.Action = vpsv1.FirewallAction_DROP
		}
	}

	// Type (direction)
	if ruleType, ok := rule["type"].(string); ok {
		switch ruleType {
		case "in":
			protoRule.Type = vpsv1.FirewallDirection_IN
		case "out":
			protoRule.Type = vpsv1.FirewallDirection_OUT
		}
	}

	// Comment
	if comment, ok := rule["comment"].(string); ok {
		protoRule.Comment = &comment
	}

	// Source
	if source, ok := rule["source"].(string); ok && source != "" {
		protoRule.Source = &source
	}

	// Dest
	if dest, ok := rule["dest"].(string); ok && dest != "" {
		protoRule.Dest = &dest
	}

	// Interface
	if iface, ok := rule["iface"].(string); ok && iface != "" {
		protoRule.Iface = &iface
	}

	// MAC source
	if macSource, ok := rule["mac-source"].(string); ok && macSource != "" {
		protoRule.MacSource = &macSource
	}

	// Protocol
	if protocol, ok := rule["protocol"].(string); ok {
		switch protocol {
		case "tcp":
			protoRule.Protocol = &[]vpsv1.FirewallProtocol{vpsv1.FirewallProtocol_TCP}[0]
		case "udp":
			protoRule.Protocol = &[]vpsv1.FirewallProtocol{vpsv1.FirewallProtocol_UDP}[0]
		case "icmp":
			protoRule.Protocol = &[]vpsv1.FirewallProtocol{vpsv1.FirewallProtocol_ICMP}[0]
		case "icmpv6":
			protoRule.Protocol = &[]vpsv1.FirewallProtocol{vpsv1.FirewallProtocol_ICMPV6}[0]
		case "all":
			protoRule.Protocol = &[]vpsv1.FirewallProtocol{vpsv1.FirewallProtocol_ALL}[0]
		}
	}

	// Destination port
	if dport, ok := rule["dport"].(string); ok && dport != "" {
		protoRule.Dport = &dport
	}

	// Source port
	if sport, ok := rule["sport"].(string); ok && sport != "" {
		protoRule.Sport = &sport
	}

	// ICMP type
	if icmpType, ok := rule["icmp-type"].(float64); ok {
		icmpTypeInt := int32(icmpType)
		protoRule.IcmpType = &icmpTypeInt
	}

	// Log
	if log, ok := rule["log"].(string); ok && log != "" {
		logBool := log != "0" && log != "nolog"
		protoRule.Log = &logBool
	} else if log, ok := rule["log"].(bool); ok {
		protoRule.Log = &log
	}

	return protoRule
}

func firewallRuleToFormData(rule *vpsv1.FirewallRule) url.Values {
	data := url.Values{}

	if rule.Enable {
		data.Set("enable", "1")
	} else {
		data.Set("enable", "0")
	}

	// Action
	switch rule.Action {
	case vpsv1.FirewallAction_ACCEPT:
		data.Set("action", "ACCEPT")
	case vpsv1.FirewallAction_REJECT:
		data.Set("action", "REJECT")
	case vpsv1.FirewallAction_DROP:
		data.Set("action", "DROP")
	}

	// Type (direction)
	switch rule.Type {
	case vpsv1.FirewallDirection_IN:
		data.Set("type", "in")
	case vpsv1.FirewallDirection_OUT:
		data.Set("type", "out")
	}

	// Comment
	if rule.Comment != nil {
		data.Set("comment", *rule.Comment)
	}

	// Source
	if rule.Source != nil {
		data.Set("source", *rule.Source)
	}

	// Dest
	if rule.Dest != nil {
		data.Set("dest", *rule.Dest)
	}

	// Interface
	if rule.Iface != nil {
		data.Set("iface", *rule.Iface)
	}

	// MAC source
	if rule.MacSource != nil {
		data.Set("mac-source", *rule.MacSource)
	}

	// Protocol
	if rule.Protocol != nil {
		switch *rule.Protocol {
		case vpsv1.FirewallProtocol_TCP:
			data.Set("protocol", "tcp")
		case vpsv1.FirewallProtocol_UDP:
			data.Set("protocol", "udp")
		case vpsv1.FirewallProtocol_ICMP:
			data.Set("protocol", "icmp")
		case vpsv1.FirewallProtocol_ICMPV6:
			data.Set("protocol", "icmpv6")
		case vpsv1.FirewallProtocol_ALL:
			data.Set("protocol", "all")
		}
	}

	// Destination port
	if rule.Dport != nil {
		data.Set("dport", *rule.Dport)
	}

	// Source port
	if rule.Sport != nil {
		data.Set("sport", *rule.Sport)
	}

	// ICMP type
	if rule.IcmpType != nil {
		data.Set("icmp-type", strconv.Itoa(int(*rule.IcmpType)))
	}

	// Log
	if rule.Log != nil && *rule.Log {
		data.Set("log", "1")
	}

	return data
}

func firewallOptionsToProto(options map[string]interface{}) *vpsv1.FirewallOptions {
	protoOptions := &vpsv1.FirewallOptions{}

	// Enable
	if enable, ok := options["enable"].(float64); ok {
		protoOptions.Enable = enable != 0
	} else if enable, ok := options["enable"].(bool); ok {
		protoOptions.Enable = enable
	}

	// Policy in
	if policyIn, ok := options["policy_in"].(string); ok {
		protoOptions.PolicyIn = &policyIn
	}

	// Policy out
	if policyOut, ok := options["policy_out"].(string); ok {
		protoOptions.PolicyOut = &policyOut
	}

	// Log level in
	if logLevelIn, ok := options["log_level_in"].(float64); ok {
		logLevelInBool := logLevelIn != 0
		protoOptions.LogLevelIn = &logLevelInBool
	} else if logLevelIn, ok := options["log_level_in"].(bool); ok {
		protoOptions.LogLevelIn = &logLevelIn
	}

	// Log level out
	if logLevelOut, ok := options["log_level_out"].(float64); ok {
		logLevelOutBool := logLevelOut != 0
		protoOptions.LogLevelOut = &logLevelOutBool
	} else if logLevelOut, ok := options["log_level_out"].(bool); ok {
		protoOptions.LogLevelOut = &logLevelOut
	}

	// NF log
	if nfLog, ok := options["nf_log"].(float64); ok {
		nfLogBool := nfLog != 0
		protoOptions.NfLog = &nfLogBool
	} else if nfLog, ok := options["nf_log"].(bool); ok {
		protoOptions.NfLog = &nfLog
	}

	// DHCP
	if dhcp, ok := options["dhcp"].(float64); ok {
		dhcpBool := dhcp != 0
		protoOptions.Dhcp = &dhcpBool
	} else if dhcp, ok := options["dhcp"].(bool); ok {
		protoOptions.Dhcp = &dhcp
	}

	// NDP
	if ndp, ok := options["ndp"].(float64); ok {
		ndpBool := ndp != 0
		protoOptions.Ndp = &ndpBool
	} else if ndp, ok := options["ndp"].(bool); ok {
		protoOptions.Ndp = &ndp
	}

	// RADV
	if radv, ok := options["radv"].(float64); ok {
		radvBool := radv != 0
		protoOptions.Radv = &radvBool
	} else if radv, ok := options["radv"].(bool); ok {
		protoOptions.Radv = &radv
	}

	// IP filter
	if ipfilter, ok := options["ipfilter"].(float64); ok {
		ipfilterBool := ipfilter != 0
		protoOptions.Ipfilter = &ipfilterBool
	} else if ipfilter, ok := options["ipfilter"].(bool); ok {
		protoOptions.Ipfilter = &ipfilter
	}

	// IP filter rules
	if ipfilterRules, ok := options["ipfilter_rules"].(float64); ok {
		ipfilterRulesBool := ipfilterRules != 0
		protoOptions.IpfilterRules = &ipfilterRulesBool
	} else if ipfilterRules, ok := options["ipfilter_rules"].(bool); ok {
		protoOptions.IpfilterRules = &ipfilterRules
	}

	return protoOptions
}

func firewallOptionsToFormData(options *vpsv1.FirewallOptions) url.Values {
	data := url.Values{}

	if options.Enable {
		data.Set("enable", "1")
	} else {
		data.Set("enable", "0")
	}

	if options.PolicyIn != nil {
		data.Set("policy_in", *options.PolicyIn)
	}

	if options.PolicyOut != nil {
		data.Set("policy_out", *options.PolicyOut)
	}

	if options.LogLevelIn != nil {
		if *options.LogLevelIn {
			data.Set("log_level_in", "1")
		} else {
			data.Set("log_level_in", "0")
		}
	}

	if options.LogLevelOut != nil {
		if *options.LogLevelOut {
			data.Set("log_level_out", "1")
		} else {
			data.Set("log_level_out", "0")
		}
	}

	if options.NfLog != nil {
		if *options.NfLog {
			data.Set("nf_log", "1")
		} else {
			data.Set("nf_log", "0")
		}
	}

	if options.Dhcp != nil {
		if *options.Dhcp {
			data.Set("dhcp", "1")
		} else {
			data.Set("dhcp", "0")
		}
	}

	if options.Ndp != nil {
		if *options.Ndp {
			data.Set("ndp", "1")
		} else {
			data.Set("ndp", "0")
		}
	}

	if options.Radv != nil {
		if *options.Radv {
			data.Set("radv", "1")
		} else {
			data.Set("radv", "0")
		}
	}

	if options.Ipfilter != nil {
		if *options.Ipfilter {
			data.Set("ipfilter", "1")
		} else {
			data.Set("ipfilter", "0")
		}
	}

	if options.IpfilterRules != nil {
		if *options.IpfilterRules {
			data.Set("ipfilter_rules", "1")
		} else {
			data.Set("ipfilter_rules", "0")
		}
	}

	return data
}

