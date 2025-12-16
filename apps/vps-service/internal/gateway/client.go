package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RequestHandler handles incoming requests from the gateway
type RequestHandler interface {
	// HandleRequest processes a request and returns response payload and error
	HandleRequest(ctx context.Context, method string, payload []byte) ([]byte, error)
}

// GatewayClient maintains persistent connections to multiple VPS gateways
type GatewayClient struct {
	apiSecret  string
	instanceID string
	version    string
	handlers   map[string]RequestHandler // method -> handler
	handlersMu sync.RWMutex
}

// parseNodeGatewayMapping parses the VPS_NODE_GATEWAY_ENDPOINTS env var
// Format: "node1:http://url1,node2:http://url2"
// Returns map of node name -> gateway URL
func parseNodeGatewayMapping() (map[string]string, error) {
	envVal := os.Getenv("VPS_NODE_GATEWAY_ENDPOINTS")
	if envVal == "" {
		return nil, fmt.Errorf("VPS_NODE_GATEWAY_ENDPOINTS environment variable is required")
	}

	result := make(map[string]string)
	pairs := strings.Split(envVal, ",")

	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format in VPS_NODE_GATEWAY_ENDPOINTS: %s (expected node:url)", pair)
		}

		nodeName := strings.TrimSpace(parts[0])
		gatewayURL := strings.TrimSpace(parts[1])

		if nodeName == "" {
			return nil, fmt.Errorf("empty node name in VPS_NODE_GATEWAY_ENDPOINTS")
		}

		// Validate URL format
		if !strings.HasPrefix(gatewayURL, "http://") && !strings.HasPrefix(gatewayURL, "https://") {
			return nil, fmt.Errorf("invalid gateway URL for node %s: %s (must start with http:// or https://)", nodeName, gatewayURL)
		}

		if _, err := url.Parse(gatewayURL); err != nil {
			return nil, fmt.Errorf("invalid gateway URL for node %s: %w", nodeName, err)
		}

		result[nodeName] = gatewayURL
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid gateway endpoints found in VPS_NODE_GATEWAY_ENDPOINTS")
	}

	return result, nil
}

// NewGatewayClient creates a new gateway client that connects to all configured gateways
func NewGatewayClient() (*GatewayClient, error) {
	apiSecret := os.Getenv("VPS_GATEWAY_API_SECRET")
	if apiSecret == "" {
		return nil, fmt.Errorf("VPS_GATEWAY_API_SECRET environment variable is required")
	}

	// Get instance ID (use hostname or generate)
	instanceID, _ := os.Hostname()
	if instanceID == "" {
		instanceID = fmt.Sprintf("vps-api-%d", time.Now().Unix())
	}

	return &GatewayClient{
		apiSecret:  apiSecret,
		instanceID: instanceID,
		version:    "1.0.0", // TODO: Get from build info
		handlers:   make(map[string]RequestHandler),
	}, nil
}

// RegisterHandler registers a handler for a specific request method
// This allows easy extension for new request types
func (c *GatewayClient) RegisterHandler(method string, handler RequestHandler) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	c.handlers[method] = handler
	logger.Info("[GatewayClient] Registered handler for method: %s", method)
}

// Start connects to all configured gateways and maintains connections
func (c *GatewayClient) Start(ctx context.Context) error {
	// Parse gateway endpoints
	gateways, err := parseNodeGatewayMapping()
	if err != nil {
		return fmt.Errorf("failed to parse gateway endpoints: %w", err)
	}

	logger.Info("[GatewayClient] Starting connections to %d gateways", len(gateways))

	// Start connection to each gateway
	var wg sync.WaitGroup
	for nodeName, gatewayURL := range gateways {
		wg.Add(1)
		go func(node, url string) {
			defer wg.Done()
			c.maintainConnection(ctx, node, url)
		}(nodeName, gatewayURL)
	}

	// Wait for all connections to finish (which should only happen on context cancellation)
	wg.Wait()
	return ctx.Err()
}

func (c *GatewayClient) maintainConnection(ctx context.Context, nodeName, gatewayURL string) {
	logger.Info("[GatewayClient] Starting connection to %s at %s", nodeName, gatewayURL)

	for {
		select {
		case <-ctx.Done():
			logger.Info("[GatewayClient] Stopping connection to %s", nodeName)
			return
		default:
			if err := c.connectAndServe(ctx, nodeName, gatewayURL); err != nil {
				logger.Error("[GatewayClient] Connection to %s error: %v, reconnecting in 5s...", nodeName, err)
				time.Sleep(5 * time.Second)
				continue
			}
		}
	}
}

func (c *GatewayClient) connectAndServe(ctx context.Context, nodeName, gatewayURL string) error {
	// Create HTTP client with h2c support and keep-alive settings
	transport := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
		// Enable HTTP/2 ping frames to keep connection alive
		ReadIdleTimeout: 45 * time.Second, // Send ping if no reads for 45s
		PingTimeout:     15 * time.Second, // Wait 15s for pong response
	}
	httpClient := &http.Client{
		Transport: transport,
	}

	// Create Connect client
	client := vpsgatewayv1connect.NewVPSGatewayServiceClient(
		httpClient,
		gatewayURL,
	)

	logger.Info("[GatewayClient] Connecting to gateway %s at %s", nodeName, gatewayURL)

	// Establish bidirectional stream
	// Note: For streaming calls, we need to set auth header using request options
	stream := client.RegisterGateway(ctx)

	// Set auth header on the stream
	stream.RequestHeader().Set("x-api-secret", c.apiSecret)

	// Send registration
	regMsg := &vpsgatewayv1.GatewayMessage{
		Type: "register",
		Registration: &vpsgatewayv1.GatewayRegistration{
			GatewayId: c.instanceID, // Reusing this field for API instance ID
			Version:   c.version,
		},
	}

	if err := stream.Send(regMsg); err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}

	logger.Info("[GatewayClient] Sent registration to gateway %s", nodeName)

	// Start heartbeat sender
	go c.sendHeartbeats(ctx, stream)

	// Handle incoming messages
	for {
		msg, err := stream.Receive()
		if err == io.EOF {
			logger.Info("[GatewayClient] Gateway closed connection")
			return nil
		}
		if err != nil {
			return fmt.Errorf("receive error: %w", err)
		}

		switch msg.Type {
		case "registered":
			logger.Info("[GatewayClient] Successfully registered with gateway %s", nodeName)

		case "request":
			if msg.Request != nil {
				go c.handleRequest(ctx, stream, msg.Request, nodeName)
			}

		case "heartbeat":
			logger.Debug("[GatewayClient] Received heartbeat from gateway %s", nodeName)

		default:
			logger.Warn("[GatewayClient] Unknown message type from %s: %s", nodeName, msg.Type)
		}
	}
}

func (c *GatewayClient) sendHeartbeats(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage]) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msg := &vpsgatewayv1.GatewayMessage{
				Type:      "heartbeat",
				Heartbeat: timestamppb.Now(),
			}
			if err := stream.Send(msg); err != nil {
				logger.Debug("[GatewayClient] Failed to send heartbeat: %v", err)
				return
			}
		}
	}
}

func (c *GatewayClient) handleRequest(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage], req *vpsgatewayv1.GatewayRequest, nodeName string) {
	var respPayload []byte
	var respError string

	// Look up handler for this method
	c.handlersMu.RLock()
	handler, ok := c.handlers[req.Method]
	c.handlersMu.RUnlock()

	if !ok {
		respError = fmt.Sprintf("no handler registered for method: %s", req.Method)
	} else {
		// Call handler
		payload, err := handler.HandleRequest(ctx, req.Method, req.Payload)
		if err != nil {
			respError = err.Error()
		} else {
			respPayload = payload
		}
	}

	// Send response
	resp := &vpsgatewayv1.GatewayResponse{
		RequestId: req.RequestId,
		Success:   respError == "",
		Error:     respError,
		Payload:   respPayload,
	}

	msg := &vpsgatewayv1.GatewayMessage{
		Type:     "response",
		Response: resp,
	}

	if err := stream.Send(msg); err != nil {
		logger.Error("[GatewayClient] Failed to send response: %v", err)
	}
}
