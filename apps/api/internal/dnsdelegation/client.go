package dnsdelegation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// DelegationConfig represents configuration for DNS delegation
type DelegationConfig struct {
	APIURLs []string // List of API URLs to query (e.g., ["https://dev-api.example.com", "https://selfhosted.example.com"])
	APIKey  string   // API key for authentication
	Timeout time.Duration
}

// DNSQueryRequest represents a DNS query request
type DNSQueryRequest struct {
	Domain     string `json:"domain"`
	RecordType string `json:"record_type"` // "A" or "SRV"
}

// DNSQueryResponse represents a DNS query response
type DNSQueryResponse struct {
	Domain     string   `json:"domain"`
	RecordType string   `json:"record_type"`
	Records    []string `json:"records,omitempty"`
	Error      string   `json:"error,omitempty"`
	TTL        int64    `json:"ttl"`
}

// QueryRemoteAPI queries a remote API for DNS resolution
func QueryRemoteAPI(config DelegationConfig, domain, recordType string) (*DNSQueryResponse, error) {
	if len(config.APIURLs) == 0 {
		return nil, fmt.Errorf("no API URLs configured")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	reqBody := DNSQueryRequest{
		Domain:     domain,
		RecordType: recordType,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Try each API URL until one succeeds
	var lastErr error
	for _, apiURL := range config.APIURLs {
		apiURL = strings.TrimSuffix(apiURL, "/")
		url := fmt.Sprintf("%s/dns/query", apiURL)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		if config.APIKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[DNS Delegation] Failed to query %s: %v", url, err)
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("[DNS Delegation] API %s returned status %d: %s", url, resp.StatusCode, string(body))
			lastErr = fmt.Errorf("API returned status %d", resp.StatusCode)
			continue
		}

		var dnsResp DNSQueryResponse
		if err := json.NewDecoder(resp.Body).Decode(&dnsResp); err != nil {
			log.Printf("[DNS Delegation] Failed to decode response from %s: %v", url, err)
			lastErr = err
			continue
		}

		if dnsResp.Error != "" {
			// API returned an error (e.g., deployment not found)
			return nil, fmt.Errorf("API error: %s", dnsResp.Error)
		}

		if len(dnsResp.Records) == 0 {
			// No records found
			return nil, fmt.Errorf("no records found")
		}

		return &dnsResp, nil
	}

	return nil, fmt.Errorf("all API queries failed: %w", lastErr)
}

// ParseDelegationConfig parses DNS delegation configuration from environment variables
func ParseDelegationConfig() DelegationConfig {
	config := DelegationConfig{
		Timeout: 5 * time.Second,
	}

	// Parse DNS_DELEGATION_API_URLS (comma-separated list)
	apiURLsEnv := strings.TrimSpace(os.Getenv("DNS_DELEGATION_API_URLS"))
	if apiURLsEnv != "" {
		urls := strings.Split(apiURLsEnv, ",")
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url != "" {
				config.APIURLs = append(config.APIURLs, url)
			}
		}
	}

	// Parse DNS_DELEGATION_API_KEY
	config.APIKey = strings.TrimSpace(os.Getenv("DNS_DELEGATION_API_KEY"))

	// Parse DNS_DELEGATION_TIMEOUT (in seconds)
	timeoutStr := strings.TrimSpace(os.Getenv("DNS_DELEGATION_TIMEOUT"))
	if timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.Timeout = timeout
		}
	}

	return config
}

