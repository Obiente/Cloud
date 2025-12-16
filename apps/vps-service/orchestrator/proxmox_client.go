package orchestrator

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// Core Proxmox client, authentication, and API request methods

type ProxmoxClient struct {
	config     *ProxmoxConfig
	httpClient *http.Client
	ticket     *ProxmoxTicket
	useToken   bool // If true, use API token authentication (no ticket needed)
}

type ProxmoxTicket struct {
	Ticket string
	CSRF   string
	Expiry time.Time
}

func NewProxmoxClient(config *ProxmoxConfig) (*ProxmoxClient, error) {
	// Validate that either password or token is provided
	if config.Password == "" && (config.TokenID == "" || config.Secret == "") {
		return nil, fmt.Errorf("either password or token (token_id + secret) must be provided")
	}

	// Create HTTP client with insecure TLS (Proxmox often uses self-signed certs)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // TODO: Make this configurable
		},
	}

	useToken := config.TokenID != "" && config.Secret != ""

	client := &ProxmoxClient{
		config: config,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
		useToken: useToken,
	}

	// Authenticate (only needed for password-based auth; tokens are used directly in requests)
	if !useToken {
		if err := client.authenticate(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to authenticate with Proxmox: %w", err)
		}
	} else {
		logger.Info("[ProxmoxClient] Using API token authentication (no ticket needed)")
	}

	return client, nil
}

func (pc *ProxmoxClient) GetAuthCookie() string {
	if pc.ticket != nil {
		return pc.ticket.Ticket
	}
	// For API tokens, WebSocket connections may require a ticket cookie
	// Try to get a ticket using the API token
	if pc.useToken {
		// API tokens can be used to get a ticket for WebSocket connections
		// This is a workaround for WebSocket which requires PVEAuthCookie
		return ""
	}
	return ""
}

func (pc *ProxmoxClient) GetOrCreateTicketForWebSocket(ctx context.Context) (string, error) {
	// If we already have a ticket, return it
	if pc.ticket != nil && time.Now().Before(pc.ticket.Expiry.Add(-5*time.Minute)) {
		return pc.ticket.Ticket, nil
	}

	// For API tokens, we cannot get a ticket via /access/ticket endpoint
	// The endpoint requires username/password in POST body, not API token in header
	// According to Proxmox docs, API tokens don't use tickets for regular API calls
	// However, WebSocket may require PVEAuthCookie - this is a limitation
	// We'll return empty and try with just Authorization header + vncticket
	if pc.useToken {
		// API tokens cannot obtain tickets - return empty
		// WebSocket connection will use Authorization header instead
		return "", nil
	}

	// For password-based auth, ensure we're authenticated
	if err := pc.ensureAuthenticated(ctx); err != nil {
		return "", err
	}

	if pc.ticket == nil {
		return "", fmt.Errorf("no ticket available")
	}

	return pc.ticket.Ticket, nil
}

func (pc *ProxmoxClient) GetHTTPClient() *http.Client {
	return pc.httpClient
}

func (pc *ProxmoxClient) GetAuthHeader() string {
	if !pc.useToken {
		return ""
	}
	return fmt.Sprintf("PVEAPIToken=%s!%s=%s", pc.config.Username, pc.config.TokenID, pc.config.Secret)
}

func (pc *ProxmoxClient) authenticate(ctx context.Context) error {
	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	authURL := fmt.Sprintf("%s/api2/json/access/ticket", apiURL)

	// Password-based authentication only (tokens don't use tickets)
	authData := url.Values{}
	authData.Set("username", pc.config.Username)
	authData.Set("password", pc.config.Password)

	req, err := http.NewRequestWithContext(ctx, "POST", authURL, strings.NewReader(authData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var authResp struct {
		Data struct {
			Ticket string `json:"ticket"`
			CSRF   string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	pc.ticket = &ProxmoxTicket{
		Ticket: authResp.Data.Ticket,
		CSRF:   authResp.Data.CSRF,
		Expiry: time.Now().Add(2 * time.Hour), // Proxmox tickets typically last 2 hours
	}

	logger.Info("[ProxmoxClient] Successfully authenticated with Proxmox API (password-based)")
	return nil
}

func (pc *ProxmoxClient) ensureAuthenticated(ctx context.Context) error {
	if pc.useToken {
		// API tokens don't need tickets - they're used directly in requests
		return nil
	}
	if pc.ticket == nil || time.Now().After(pc.ticket.Expiry.Add(-5*time.Minute)) {
		// Ticket expired or about to expire, re-authenticate
		return pc.authenticate(ctx)
	}
	return nil
}

func (pc *ProxmoxClient) apiRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	if err := pc.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	reqURL := fmt.Sprintf("%s/api2/json%s", apiURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if pc.useToken {
		// API token authentication: Use Authorization header
		// Format: PVEAPIToken=USER@REALM!TOKENID=SECRET
		authHeader := fmt.Sprintf("PVEAPIToken=%s!%s=%s", pc.config.Username, pc.config.TokenID, pc.config.Secret)
		req.Header.Set("Authorization", authHeader)
		// API tokens don't need CSRF tokens
	} else {
		// Password-based authentication: Use ticket cookie
		req.AddCookie(&http.Cookie{
			Name:  "PVEAuthCookie",
			Value: pc.ticket.Ticket,
		})

		// Set CSRF token for write operations
		if method != "GET" {
			req.Header.Set("CSRFPreventionToken", pc.ticket.CSRF)
		}
	}

	// Only set Content-Type if there's a body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return pc.httpClient.Do(req)
}

func (pc *ProxmoxClient) APIRequestRaw(ctx context.Context, method, endpoint string, bodyJSON []byte) (*http.Response, error) {
	if err := pc.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	reqURL := fmt.Sprintf("%s/api2/json%s", apiURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication
	if pc.useToken {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s!%s=%s", pc.config.Username, pc.config.TokenID, pc.config.Secret))
	} else {
		req.AddCookie(&http.Cookie{
			Name:  "PVEAuthCookie",
			Value: pc.ticket.Ticket,
		})
		if method != "GET" {
			req.Header.Set("CSRFPreventionToken", pc.ticket.CSRF)
		}
	}

	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func (pc *ProxmoxClient) APIRequestForm(ctx context.Context, method, endpoint string, formData url.Values) (*http.Response, error) {
	return pc.apiRequestForm(ctx, method, endpoint, formData)
}

func (pc *ProxmoxClient) apiRequestForm(ctx context.Context, method, endpoint string, formData url.Values) (*http.Response, error) {
	if err := pc.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	reqURL := fmt.Sprintf("%s/api2/json%s", apiURL, endpoint)

	var body io.Reader
	if len(formData) > 0 {
		// Check if sshkeys is pre-encoded (we manually encoded it with %20)
		// If so, manually construct form data to avoid double encoding
		if sshKeysVal, ok := formData["sshkeys"]; ok && len(sshKeysVal) > 0 {
			// sshkeys is already URL-encoded with %20, manually construct form data
			var formParts []string
			for key, values := range formData {
				if key == "sshkeys" {
					// sshkeys is already encoded with %20, use as-is
					formParts = append(formParts, fmt.Sprintf("%s=%s", url.QueryEscape(key), sshKeysVal[0]))
				} else {
					// Other parameters: use normal form encoding
					for _, value := range values {
						tempForm := url.Values{}
						tempForm.Set(key, value)
						encoded := tempForm.Encode()
						formParts = append(formParts, encoded)
					}
				}
			}
			bodyStr := strings.Join(formParts, "&")
			logger.Debug("[ProxmoxClient] Form data body: %s", bodyStr)
			body = strings.NewReader(bodyStr)
		} else {
			// No sshkeys parameter, use standard form encoding
			encodedBody := formData.Encode()
			logger.Debug("[ProxmoxClient] Form data body: %s", encodedBody)
			body = strings.NewReader(encodedBody)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if pc.useToken {
		// API token authentication: Use Authorization header
		// Format: PVEAPIToken=USER@REALM!TOKENID=SECRET
		authHeader := fmt.Sprintf("PVEAPIToken=%s!%s=%s", pc.config.Username, pc.config.TokenID, pc.config.Secret)
		req.Header.Set("Authorization", authHeader)
		// API tokens don't need CSRF tokens
	} else {
		// Password-based authentication: Use ticket cookie
		req.AddCookie(&http.Cookie{
			Name:  "PVEAuthCookie",
			Value: pc.ticket.Ticket,
		})

		// Set CSRF token for write operations
		if method != "GET" {
			req.Header.Set("CSRFPreventionToken", pc.ticket.CSRF)
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return pc.httpClient.Do(req)
}
