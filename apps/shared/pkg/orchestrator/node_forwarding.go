package orchestrator

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/utils"

	"nhooyr.io/websocket"
)

// NodeForwarder handles forwarding requests to other nodes in the cluster
type NodeForwarder struct {
	httpClient *http.Client
	apiBaseURL string
}

// NewNodeForwarder creates a new node forwarder
func NewNodeForwarder() *NodeForwarder {
	// Get API base URL from environment or construct from hostname
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		// Check if domain-based routing is enabled
		useDomainRouting := os.Getenv("USE_DOMAIN_ROUTING")
		domain := os.Getenv("DOMAIN")
		useTraefikRouting := os.Getenv("USE_TRAEFIK_ROUTING")
		
		// If domain routing is enabled and domain is set, use domain-based URL
		if (useDomainRouting == "true" || useDomainRouting == "1") && domain != "" && domain != "localhost" {
			scheme := "http"
			if useTraefikRouting == "true" || useTraefikRouting == "1" {
				scheme = "https"
			}
			apiBaseURL = fmt.Sprintf("%s://api.%s", scheme, domain)
		} else {
			// Fallback to service name or hostname-based URL
			port := os.Getenv("PORT")
			if port == "" {
				port = "3001"
			}
			hostname := os.Getenv("HOSTNAME")
			if hostname != "" {
				// In Swarm mode, use tasks.api service name for DNS resolution
				// Otherwise, use hostname
				if utils.IsSwarmModeEnabled() {
					apiBaseURL = fmt.Sprintf("http://tasks.api-gateway:%s", port)
				} else {
					apiBaseURL = fmt.Sprintf("http://%s:%s", hostname, port)
				}
			} else {
				// Fallback to localhost or service name
				if utils.IsSwarmModeEnabled() {
					apiBaseURL = fmt.Sprintf("http://api-gateway:%s", port)
				} else {
					apiBaseURL = fmt.Sprintf("http://localhost:%s", port)
				}
			}
		}
	}

	// Configure HTTP client with TLS support for HTTPS (domain-based routing)
	skipTLSVerify := os.Getenv("SKIP_TLS_VERIFY")
	shouldSkipVerify := skipTLSVerify == "true" || skipTLSVerify == "1"
	
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: shouldSkipVerify, // Skip TLS verification for internal Traefik certs
		},
	}
	
	return &NodeForwarder{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:  30 * time.Second,
		},
		apiBaseURL: strings.TrimSuffix(apiBaseURL, "/"),
	}
}

// GetNodeAPIURL gets the API URL for a specific node
func (nf *NodeForwarder) GetNodeAPIURL(nodeID string) (string, error) {
	// Try to get node from database
	var node database.NodeMetadata
	if err := database.DB.First(&node, "id = ?", nodeID).Error; err != nil {
		return "", fmt.Errorf("failed to find node %s: %w", nodeID, err)
	}

	// Check if node has API URL in labels (JSONB field) - highest priority
	if node.Labels != "" {
		var labels map[string]interface{}
		if err := json.Unmarshal([]byte(node.Labels), &labels); err == nil {
			// Check for explicit API URL first
			if apiURL, ok := labels["api_url"].(string); ok && apiURL != "" {
				return strings.TrimSuffix(apiURL, "/"), nil
			}
			if apiURL, ok := labels["obiente.api_url"].(string); ok && apiURL != "" {
				return strings.TrimSuffix(apiURL, "/"), nil
			}
		}
	}

	// Check if domain-based routing is enabled
	useDomainRouting := os.Getenv("USE_DOMAIN_ROUTING")
	domain := os.Getenv("DOMAIN")
	useTraefikRouting := os.Getenv("USE_TRAEFIK_ROUTING")

	// If domain routing is enabled and domain is set, try to use domain-based URL
	if (useDomainRouting == "true" || useDomainRouting == "1") && domain != "" && domain != "localhost" {
		// Determine scheme
		scheme := "http"
		if useTraefikRouting == "true" || useTraefikRouting == "1" {
			scheme = "https"
		}
		
		// Check for node-specific subdomain in labels
		var nodeSubdomain string
		if node.Labels != "" {
			var labels map[string]interface{}
			if err := json.Unmarshal([]byte(node.Labels), &labels); err == nil {
				// Check for explicit subdomain configuration
				if subdomain, ok := labels["api_subdomain"].(string); ok && subdomain != "" {
					nodeSubdomain = subdomain
				} else if subdomain, ok := labels["obiente.api_subdomain"].(string); ok && subdomain != "" {
					nodeSubdomain = subdomain
				} else if subdomain, ok := labels["subdomain"].(string); ok && subdomain != "" {
					// Use subdomain with "api" prefix (e.g., "node1" -> "node1-api")
					nodeSubdomain = subdomain + "-api"
				} else if subdomain, ok := labels["obiente.subdomain"].(string); ok && subdomain != "" {
					nodeSubdomain = subdomain + "-api"
				}
			}
		}
		
		// If no explicit subdomain, try to generate from hostname
		if nodeSubdomain == "" && node.Hostname != "" {
			// Check if hostname is already a full domain (contains dots and domain)
			if strings.Contains(node.Hostname, ".") {
				// Check if hostname already contains the domain
				if strings.HasSuffix(node.Hostname, domain) {
					// Hostname is already a subdomain of our domain (e.g., "node1.obiente.cloud")
					return fmt.Sprintf("%s://%s", scheme, node.Hostname), nil
				}
				// Hostname is a full domain but different domain - use as-is
				return fmt.Sprintf("%s://%s", scheme, node.Hostname), nil
			}
			
			// Generate subdomain from hostname
			// Sanitize hostname to be DNS-safe (lowercase, replace invalid chars with hyphens)
			sanitizedHostname := strings.ToLower(node.Hostname)
			sanitizedHostname = strings.ReplaceAll(sanitizedHostname, "_", "-")
			sanitizedHostname = strings.ReplaceAll(sanitizedHostname, " ", "-")
			
			// Use hostname as subdomain (e.g., "node1" -> "node1.obiente.cloud")
			// For API, we can use either "node1-api.obiente.cloud" or "api-node1.obiente.cloud"
			// Default to "node1-api" pattern for clarity
			nodeSubdomain = sanitizedHostname + "-api"
		}
		
		// If we have a node subdomain, use it
		if nodeSubdomain != "" {
			return fmt.Sprintf("%s://%s.%s", scheme, nodeSubdomain, domain), nil
		}
		
		// Fallback: use main API domain if no node-specific subdomain available
		// This assumes all nodes share the same API gateway domain
		return fmt.Sprintf("%s://api.%s", scheme, domain), nil
	}

	// Fallback to IP-based or hostname-based routing
	// Try to construct from node hostname/IP
	if node.IP != "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3001"
		}
		return fmt.Sprintf("http://%s:%s", node.IP, port), nil
	}

	// Use hostname if available
	if node.Hostname != "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3001"
		}
		// In Swarm mode, try to use tasks.api service DNS
		// Otherwise, use hostname directly
		if utils.IsSwarmModeEnabled() {
			// For Swarm, we need to find the specific node's API
			// Try using hostname first, fall back to tasks.api
			return fmt.Sprintf("http://%s:%s", node.Hostname, port), nil
		}
		return fmt.Sprintf("http://%s:%s", node.Hostname, port), nil
	}

	// Fallback: use API base URL pattern with node ID
	// This assumes nodes are accessible via a consistent pattern
	return "", fmt.Errorf("cannot determine API URL for node %s (no IP, hostname, or API URL in labels)", nodeID)
}

// ForwardConnectRPCRequest forwards a ConnectRPC request to another node
func (nf *NodeForwarder) ForwardConnectRPCRequest(ctx context.Context, nodeID string, method string, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	nodeURL, err := nf.GetNodeAPIURL(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node API URL: %w", err)
	}

	url := fmt.Sprintf("%s%s", nodeURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/connect+json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Forward authorization headers if present
	if auth := headers["Authorization"]; auth != "" {
		req.Header.Set("Authorization", auth)
	}

	logger.Info("[NodeForwarder] Forwarding %s request to node %s: %s", method, nodeID, url)

	resp, err := nf.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to forward request to node %s: %w", nodeID, err)
	}

	return resp, nil
}

// ForwardConnectRPCStream forwards a ConnectRPC streaming request
// This is a simplified version - full streaming support would require more complex handling
func (nf *NodeForwarder) ForwardConnectRPCStream(ctx context.Context, nodeID string, method string, path string, body io.Reader, headers map[string]string) (io.ReadCloser, error) {
	resp, err := nf.ForwardConnectRPCRequest(ctx, nodeID, method, path, body, headers)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("node %s returned status %d", nodeID, resp.StatusCode)
	}

	return resp.Body, nil
}

// CanForward checks if forwarding is possible for a node
func (nf *NodeForwarder) CanForward(nodeID string) bool {
	_, err := nf.GetNodeAPIURL(nodeID)
	return err == nil
}

// ForwardWebSocket forwards a WebSocket connection to another node
// It dials a WebSocket connection to the target node and proxies messages bidirectionally
func (nf *NodeForwarder) ForwardWebSocket(ctx context.Context, nodeID string, path string, headers http.Header) (*websocket.Conn, error) {
	nodeURL, err := nf.GetNodeAPIURL(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node API URL: %w", err)
	}

	// Construct WebSocket URL
	wsURL := nodeURL
	if strings.HasPrefix(wsURL, "http://") {
		wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	} else if strings.HasPrefix(wsURL, "https://") {
		wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	} else {
		wsURL = "ws://" + wsURL
	}

	wsURL = wsURL + path

	logger.Info("[NodeForwarder] Forwarding WebSocket connection to node %s: %s", nodeID, wsURL)

	// Parse URL
	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WebSocket URL: %w", err)
	}

	// Dial WebSocket connection
	dialOptions := &websocket.DialOptions{
		HTTPHeader: headers,
	}

	conn, _, err := websocket.Dial(ctx, u.String(), dialOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to dial WebSocket to node %s: %w", nodeID, err)
	}

	return conn, nil
}

