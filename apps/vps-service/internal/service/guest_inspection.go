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
	if s.sshPool == nil {
		return nil, errors.New("SSH connection pool is unavailable")
	}

	conn, err := s.getVPSGuestSSHConnection(ctx, vpsID)
	if err != nil {
		return nil, err
	}

	session, err := conn.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	type result struct {
		output []byte
		err    error
	}
	done := make(chan result, 1)
	go func() {
		output, runErr := session.CombinedOutput(command)
		done <- result{output: output, err: runErr}
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
