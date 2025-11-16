package orchestrator

import (
	"context"
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
		// Try to construct from hostname and port
		hostname := os.Getenv("HOSTNAME")
		port := os.Getenv("PORT")
		if port == "" {
			port = "3001"
		}
		if hostname != "" {
			// In Swarm mode, use tasks.api service name for DNS resolution
			// Otherwise, use hostname
			if os.Getenv("ENABLE_SWARM") != "false" {
				apiBaseURL = fmt.Sprintf("http://tasks.api:%s", port)
			} else {
				apiBaseURL = fmt.Sprintf("http://%s:%s", hostname, port)
			}
		} else {
			// Fallback to localhost
			apiBaseURL = fmt.Sprintf("http://localhost:%s", port)
		}
	}

	return &NodeForwarder{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
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

	// Check if node has API URL in labels (JSONB field)
	if node.Labels != "" {
		var labels map[string]interface{}
		if err := json.Unmarshal([]byte(node.Labels), &labels); err == nil {
			if apiURL, ok := labels["api_url"].(string); ok && apiURL != "" {
				return strings.TrimSuffix(apiURL, "/"), nil
			}
			if apiURL, ok := labels["obiente.api_url"].(string); ok && apiURL != "" {
				return strings.TrimSuffix(apiURL, "/"), nil
			}
		}
	}

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
		if os.Getenv("ENABLE_SWARM") != "false" {
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

