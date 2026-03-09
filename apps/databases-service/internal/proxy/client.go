package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client provides methods to interact with the db-proxy registry
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new proxy client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://db-proxy:8080"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// RegisterRoute registers a database route with the proxy
func (c *Client) RegisterRoute(route *Route) error {
	data, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("failed to marshal route: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/routes/register", c.baseURL),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf("failed to register route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("proxy returned status %d", resp.StatusCode)
	}

	return nil
}

// UnregisterRoute removes a database route from the proxy
func (c *Client) UnregisterRoute(databaseID string) error {
	data, err := json.Marshal(map[string]string{"database_id": databaseID})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/routes/unregister", c.baseURL),
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to unregister route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy returned status %d", resp.StatusCode)
	}

	return nil
}

// HealthCheck checks if the proxy is healthy
func (c *Client) HealthCheck() error {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/health", c.baseURL))
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
