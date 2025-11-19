package orchestrator

import (
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"net/url"
"strings"
)

// Console operations


type TermProxyInfo struct {
	WebSocketURL string
	Ticket       string
	User         string
}


func (pc *ProxmoxClient) GetVMConsole(ctx context.Context, nodeName string, vmID int, limit int32) ([]string, error) {
	// Proxmox doesn't have a direct console log API, but we can get VNC console info
	// For actual console output, we'd need to use VNC or serial console
	// This is a placeholder that returns VNC connection info
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "POST", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get console: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get console: %s (status: %d)", string(body), resp.StatusCode)
	}

	var vncResp struct {
		Data struct {
			Ticket string `json:"ticket"`
			Port   int    `json:"port"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vncResp); err != nil {
		return nil, fmt.Errorf("failed to decode console response: %w", err)
	}

	// Return VNC connection info as console output
	lines := []string{
		fmt.Sprintf("VNC Console for VM %d:", vmID),
		fmt.Sprintf("Port: %d", vncResp.Data.Port),
		fmt.Sprintf("Ticket: %s", vncResp.Data.Ticket),
		"Use a VNC client to connect to the console.",
	}

	return lines, nil
}


func (pc *ProxmoxClient) GetVNCWebSocketURL(ctx context.Context, nodeName string, vmID int) (string, string, error) {
	// First, get VNC proxy ticket
	// Proxmox API expects form-encoded data for POST requests, even if empty
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy", nodeName, vmID)
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		return "", "", fmt.Errorf("failed to get VNC proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to get VNC proxy: %s (status: %d)", string(body), resp.StatusCode)
	}

	var vncResp struct {
		Data struct {
			Ticket string      `json:"ticket"`
			Port   interface{} `json:"port"` // Can be string or int
			UPID   string      `json:"upid"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vncResp); err != nil {
		return "", "", fmt.Errorf("failed to decode VNC proxy response: %w", err)
	}

	// Convert port to int (handle both string and int from API)
	var port int
	switch v := vncResp.Data.Port.(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case string:
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			return "", "", fmt.Errorf("failed to parse port as integer: %v", vncResp.Data.Port)
		}
	default:
		return "", "", fmt.Errorf("unexpected port type: %T", vncResp.Data.Port)
	}

	// Construct WebSocket URL
	// Proxmox VNC WebSocket endpoint: /api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket
	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	wsURL := fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/vncwebsocket?port=%d&vncticket=%s", apiURL, nodeName, vmID, port, url.QueryEscape(vncResp.Data.Ticket))

	// Convert https to wss, http to ws
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)

	return wsURL, vncResp.Data.Ticket, nil
}


func (pc *ProxmoxClient) GetTermProxyWebSocketURL(ctx context.Context, nodeName string, vmID int) (string, error) {
	info, err := pc.GetTermProxyInfo(ctx, nodeName, vmID)
	if err != nil {
		return "", err
	}
	return info.WebSocketURL, nil
}


func (pc *ProxmoxClient) GetTermProxyInfo(ctx context.Context, nodeName string, vmID int) (*TermProxyInfo, error) {
	// Get terminal proxy ticket from termproxy endpoint
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/termproxy", nodeName, vmID)
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("failed to get term proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get term proxy: %s (status: %d)", string(body), resp.StatusCode)
	}

	var termResp struct {
		Data struct {
			Ticket string      `json:"ticket"`
			Port   interface{} `json:"port"` // Can be string or int
			User   string      `json:"user,omitempty"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&termResp); err != nil {
		return nil, fmt.Errorf("failed to decode term proxy response: %w", err)
	}

	// Convert port to int
	var port int
	switch v := termResp.Data.Port.(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case string:
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			return nil, fmt.Errorf("failed to parse port as integer: %v", termResp.Data.Port)
		}
	default:
		return nil, fmt.Errorf("unexpected port type: %T", termResp.Data.Port)
	}

	// Construct WebSocket URL for termproxy
	// According to Proxmox API documentation, termproxy uses vncwebsocket endpoint
	// but with the termproxy ticket (not vncticket parameter name)
	// Format: /api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket?port={port}&vncticket={ticket}
	// Note: termproxy ticket is used as vncticket parameter
	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	params := url.Values{}
	params.Set("port", fmt.Sprintf("%d", port))
	params.Set("vncticket", termResp.Data.Ticket)
	// Note: termproxy doesn't use serial=1, it's already a terminal proxy
	wsURL := fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/vncwebsocket?%s", apiURL, nodeName, vmID, params.Encode())

	// Convert https to wss, http to ws
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)

	// Get username from config if not provided in response
	user := termResp.Data.User
	if user == "" {
		// Try to get username from config
		if pc.config.Username != "" {
			user = pc.config.Username
			// For API tokens, keep the token ID in the username (format: username@realm!tokenid)
			// For password auth, use just username@realm
			if pc.config.TokenID == "" {
				// Password auth - remove token ID if present
				if idx := strings.Index(user, "!"); idx != -1 {
					user = user[:idx]
				}
			}
			// For API tokens, keep the full format including token ID
		} else {
			// Default to root@pam if no user specified
			user = "root@pam"
		}
	} else {
		// User from termproxy response - check if we need to add token ID for API tokens
		if pc.config.TokenID != "" && pc.config.Username != "" {
			// API token auth - ensure username includes token ID
			if !strings.Contains(user, "!") && strings.Contains(pc.config.Username, "!") {
				// Extract token ID from config username and add to termproxy user
				if idx := strings.Index(pc.config.Username, "!"); idx != -1 {
					tokenID := pc.config.Username[idx:]
					user = user + tokenID
				}
			}
		} else {
			// Password auth - remove token ID if present
			if idx := strings.Index(user, "!"); idx != -1 {
				user = user[:idx]
			}
		}
	}

	return &TermProxyInfo{
		WebSocketURL: wsURL,
		Ticket:       termResp.Data.Ticket,
		User:         user,
	}, nil
}


func (pc *ProxmoxClient) GetSerialConsoleWebSocketURL(ctx context.Context, nodeName string, vmID int) (string, error) {
	// Get VNC proxy with websocket=1 parameter (required for serial terminal)
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy", nodeName, vmID)
	formData := url.Values{}
	formData.Set("websocket", "1") // Required for serial terminal per API docs
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, formData)
	if err != nil {
		return "", fmt.Errorf("failed to get VNC proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get VNC proxy: %s (status: %d)", string(body), resp.StatusCode)
	}

	var vncResp struct {
		Data struct {
			Ticket string      `json:"ticket"`
			Port   interface{} `json:"port"` // Can be string or int
			UPID   string      `json:"upid"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vncResp); err != nil {
		return "", fmt.Errorf("failed to decode VNC proxy response: %w", err)
	}

	// Convert port to int
	var port int
	switch v := vncResp.Data.Port.(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case string:
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			return "", fmt.Errorf("failed to parse port as integer: %v", vncResp.Data.Port)
		}
	default:
		return "", fmt.Errorf("unexpected port type: %T", vncResp.Data.Port)
	}

	// Construct Serial Console WebSocket URL
	// Required parameters per API docs:
	// - node: string (in path)
	// - port: integer 5900-5999 (query parameter)
	// - vmid: integer 100-999999999 (in path)
	// - vncticket: string (query parameter)
	// Format: /api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket?port={port}&vncticket={ticket}
	// Note: When websocket=1 is used in vncproxy, the connection can be used for serial console
	// The RFB protocol handshake may appear initially but should be followed by serial console data
	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	params := url.Values{}
	params.Set("port", fmt.Sprintf("%d", port))
	params.Set("vncticket", vncResp.Data.Ticket)
	wsURL := fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/vncwebsocket?%s", apiURL, nodeName, vmID, params.Encode())

	// Validate required parameters are present
	if nodeName == "" {
		return "", fmt.Errorf("node parameter is required")
	}
	if port < 5900 || port > 5999 {
		return "", fmt.Errorf("port must be between 5900 and 5999, got %d", port)
	}
	if vmID < 100 || vmID > 999999999 {
		return "", fmt.Errorf("vmid must be between 100 and 999999999, got %d", vmID)
	}
	if vncResp.Data.Ticket == "" {
		return "", fmt.Errorf("vncticket is required")
	}

	// Convert https to wss, http to ws
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)

	return wsURL, nil
}

