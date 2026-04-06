package vps

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"connectrpc.com/connect"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	defaultJournalLines = 200
	maxJournalLines     = 1000
)

var validSystemdUnitPattern = regexp.MustCompile(`^[A-Za-z0-9@_.:-]+$`)

type journalEntryJSON struct {
	Message           string `json:"MESSAGE"`
	Priority          string `json:"PRIORITY"`
	RealtimeTimestamp string `json:"__REALTIME_TIMESTAMP"`
	SystemdUnit       string `json:"_SYSTEMD_UNIT"`
	SyslogIdentifier  string `json:"SYSLOG_IDENTIFIER"`
}

type systemdServiceJSON struct {
	Unit        string `json:"unit"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
}

func (s *Service) GetVPSJournalLogs(ctx context.Context, req *connect.Request[vpsv1.GetVPSJournalLogsRequest]) (*connect.Response[vpsv1.GetVPSJournalLogsResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSRead); err != nil {
		return nil, err
	}

	unit, err := normalizeSystemdUnit(req.Msg.GetUnit())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	lines := int(req.Msg.GetLines())
	if lines <= 0 {
		lines = defaultJournalLines
	}
	if lines > maxJournalLines {
		lines = maxJournalLines
	}

	command := fmt.Sprintf("LANG=C SYSTEMD_COLORS=0 journalctl --no-pager --output=json -n %d", lines)
	if unit != "" {
		command += " -u " + shellQuote(unit)
	}

	output, err := s.runVPSGuestCommand(ctx, vpsID, command)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to fetch journal logs: %w", err))
	}

	logs, err := parseJournalctlOutput(output)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to parse journal logs: %w", err))
	}

	return connect.NewResponse(&vpsv1.GetVPSJournalLogsResponse{
		Logs: logs,
	}), nil
}

func (s *Service) ListVPSServices(ctx context.Context, req *connect.Request[vpsv1.ListVPSServicesRequest]) (*connect.Response[vpsv1.ListVPSServicesResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSRead); err != nil {
		return nil, err
	}

	command := "LANG=C SYSTEMD_COLORS=0 systemctl list-units --type=service --plain --full --no-pager --no-legend --output=json"
	if req.Msg.GetIncludeInactive() {
		command += " --all"
	}

	output, err := s.runVPSGuestCommand(ctx, vpsID, command)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to list services: %w", err))
	}

	services, err := parseSystemctlServicesOutput(output)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to parse services list: %w", err))
	}

	sort.SliceStable(services, func(i, j int) bool {
		left := services[i]
		right := services[j]
		if left.ActiveState != right.ActiveState {
			return serviceSortWeight(left.ActiveState, left.SubState) < serviceSortWeight(right.ActiveState, right.SubState)
		}
		if left.SubState != right.SubState {
			return left.SubState < right.SubState
		}
		return left.Name < right.Name
	})

	return connect.NewResponse(&vpsv1.ListVPSServicesResponse{
		Services:  services,
		FetchedAt: timestamppb.Now(),
	}), nil
}

func (s *Service) runVPSGuestCommand(ctx context.Context, vpsID, command string) ([]byte, error) {
	if s.vpsManager == nil {
		return nil, errors.New("VPS manager is unavailable")
	}

	// Attempt 1: QEMU guest agent exec via Proxmox API (fastest, no SSH/gateway
	// required — same infrastructure as the web terminal). This is the primary
	// path for Obiente-managed VMs where cloud-init installs qemu-guest-agent.
	logger.Debug("[VPS GuestCmd] Trying hypervisor guest agent for VPS %s", vpsID)
	output, agentErr := s.runVPSGuestCommandViaAgent(ctx, vpsID, command)
	if agentErr == nil {
		return output, nil
	}
	logger.Debug("[VPS GuestCmd] Guest agent failed for VPS %s, will try SSH: %v", vpsID, agentErr)

	// Attempt 2: SSH via gateway (fallback — useful when the guest agent is not
	// installed/running, e.g. non-managed VMs). Cap the entire attempt in a
	// bounded context so a hung gateway doesn't consume the caller's deadline.
	if s.sshPool == nil {
		return nil, agentErr
	}
	sshCtx, sshCancel := context.WithTimeout(ctx, 12*time.Second)
	conn, sshConnErr := s.getVPSGuestSSHConnection(sshCtx, vpsID)
	sshCancel()
	if sshConnErr != nil {
		return nil, fmt.Errorf("guest agent failed (%v); SSH also failed: %w", agentErr, sshConnErr)
	}

	session, err := conn.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("guest agent failed (%v); SSH session error: %w", agentErr, err)
	}
	defer session.Close()

	type result struct {
		output []byte
		err    error
	}
	done := make(chan result, 1)
	go func() {
		out, runErr := session.CombinedOutput(command)
		done <- result{output: out, err: runErr}
	}()

	select {
	case <-ctx.Done():
		_ = session.Close()
		return nil, ctx.Err()
	case res := <-done:
		if res.err != nil {
			errOutput := strings.TrimSpace(string(res.output))
			if errOutput != "" {
				return nil, fmt.Errorf("%w: %s", res.err, errOutput)
			}
			return nil, res.err
		}
		return res.output, nil
	}
}

// runVPSGuestCommandViaAgent executes a shell command on the VM using the Proxmox
// QEMU guest agent (/agent/exec). This path does not require the vps-gateway service.
func (s *Service) runVPSGuestCommandViaAgent(ctx context.Context, vpsID, command string) ([]byte, error) {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, fmt.Errorf("VPS not found: %w", err)
	}
	if vps.NodeID == nil || *vps.NodeID == "" {
		return nil, errors.New("VPS has no assigned node — cannot use guest agent")
	}
	if vps.InstanceID == nil || *vps.InstanceID == "" {
		return nil, errors.New("VPS has no Proxmox instance ID — cannot use guest agent")
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, fmt.Errorf("invalid VPS instance ID %q", *vps.InstanceID)
	}

	proxmoxClient, err := s.vpsManager.GetProxmoxClientForNode(*vps.NodeID)
	if err != nil {
		return nil, fmt.Errorf("no Proxmox client for node %s: %w", *vps.NodeID, err)
	}

	output, exitCode, err := proxmoxClient.RunGuestShellCommand(ctx, *vps.NodeID, vmIDInt, command)
	if err != nil {
		return nil, fmt.Errorf("guest agent exec failed: %w", err)
	}
	if exitCode != 0 {
		return output, fmt.Errorf("command exited with code %d: %s", exitCode, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func (s *Service) getVPSGuestSSHConnection(ctx context.Context, vpsID string) (*PooledSSHConnection, error) {
	terminalKey, err := database.GetVPSTerminalKey(vpsID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("web terminal SSH key is not configured for this VPS")
		}
		return nil, fmt.Errorf("failed to load terminal key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey([]byte(terminalKey.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse terminal SSH key: %w", err)
	}

	ipv4, ipv6, err := s.vpsManager.GetVPSIPAddresses(ctx, vpsID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve VPS IP: %w", err)
	}

	vpsIP := choosePreferredVPSIP(ipv4, ipv6)
	if vpsIP == "" {
		return nil, errors.New("VPS has no reachable guest IP address yet")
	}

	conn, err := s.sshPool.GetOrCreateConnection(ctx, vpsID, vpsIP, terminalKey.ID, signer)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH connection: %w", err)
	}
	return conn, nil
}

func choosePreferredVPSIP(ipv4, ipv6 []string) string {
	for _, candidate := range ipv4 {
		ip := net.ParseIP(candidate)
		if ip == nil || ip.IsLoopback() {
			continue
		}
		if ip.To4() != nil {
			return candidate
		}
	}

	for _, candidate := range append([]string{}, ipv4...) {
		ip := net.ParseIP(candidate)
		if ip == nil || ip.IsLoopback() {
			continue
		}
		return candidate
	}

	for _, candidate := range ipv6 {
		ip := net.ParseIP(candidate)
		if ip == nil || ip.IsLoopback() {
			continue
		}
		return candidate
	}

	return ""
}

func normalizeSystemdUnit(unit string) (string, error) {
	trimmed := strings.TrimSpace(unit)
	if trimmed == "" {
		return "", nil
	}
	if !validSystemdUnitPattern.MatchString(trimmed) {
		return "", fmt.Errorf("invalid systemd unit %q", unit)
	}
	if !strings.Contains(trimmed, ".") {
		trimmed += ".service"
	}
	return trimmed, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func parseJournalctlOutput(output []byte) ([]*vpsv1.VPSLogLine, error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	logs := make([]*vpsv1.VPSLogLine, 0)
	lineNumber := int32(0)

	for scanner.Scan() {
		rawLine := strings.TrimSpace(scanner.Text())
		if rawLine == "" {
			continue
		}

		var entry journalEntryJSON
		if err := json.Unmarshal([]byte(rawLine), &entry); err != nil {
			return nil, fmt.Errorf("decode journal entry: %w", err)
		}

		message := strings.TrimSpace(entry.Message)
		if message == "" {
			continue
		}

		lineNumber++
		logs = append(logs, &vpsv1.VPSLogLine{
			Line:       message,
			Stderr:     journalPriorityIsError(entry.Priority),
			LineNumber: lineNumber,
			Timestamp:  timestamppb.New(parseJournalTimestamp(entry.RealtimeTimestamp)),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan journal output: %w", err)
	}

	return logs, nil
}

func parseSystemctlServicesOutput(output []byte) ([]*vpsv1.VPSSystemService, error) {
	var decoded []systemdServiceJSON
	if err := json.Unmarshal(output, &decoded); err != nil {
		return nil, fmt.Errorf("decode systemctl services: %w", err)
	}

	services := make([]*vpsv1.VPSSystemService, 0, len(decoded))
	for _, service := range decoded {
		if strings.TrimSpace(service.Unit) == "" {
			continue
		}
		services = append(services, &vpsv1.VPSSystemService{
			Name:        service.Unit,
			LoadState:   service.Load,
			ActiveState: service.Active,
			SubState:    service.Sub,
			Description: service.Description,
		})
	}
	return services, nil
}

func parseJournalTimestamp(raw string) time.Time {
	if raw == "" {
		return time.Now().UTC()
	}

	micros, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Now().UTC()
	}

	return time.Unix(0, micros*int64(time.Microsecond)).UTC()
}

func journalPriorityIsError(priority string) bool {
	value, err := strconv.Atoi(strings.TrimSpace(priority))
	if err != nil {
		return false
	}
	return value <= 3
}

func serviceSortWeight(activeState, subState string) int {
	switch activeState {
	case "active":
		return 0
	case "activating", "reloading":
		return 1
	case "failed":
		return 2
	case "inactive":
		if subState == "dead" {
			return 4
		}
		return 3
	default:
		return 5
	}
}
