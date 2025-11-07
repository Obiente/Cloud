package dnsdelegation

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"api/internal/database"
	"api/internal/metrics"
)

// HandlePushDNSRecord handles DNS record push requests from remote APIs
// This endpoint accepts DNS records pushed by dev/self-hosted APIs
func HandlePushDNSRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check API key authentication
	apiKey := r.Header.Get("Authorization")
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	// Validate API key and get key info
	apiKeyInfo, err := database.GetDNSDelegationAPIKeyByHash(apiKey)
	if err != nil {
		log.Printf("[DNS Delegation] API key validation error: %v", err)
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		Domain     string   `json:"domain"`
		RecordType string   `json:"record_type"` // "A" or "SRV"
		Records    []string `json:"records"`     // Array of record values
		TTL        int64    `json:"ttl"`         // TTL in seconds (default: 300)
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	domain := strings.TrimSpace(req.Domain)
	recordType := strings.ToUpper(strings.TrimSpace(req.RecordType))
	if recordType == "" {
		recordType = "A"
	}

	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	if len(req.Records) == 0 {
		http.Error(w, "At least one record is required", http.StatusBadRequest)
		return
	}

	// Validate domain format
	if !strings.HasSuffix(strings.ToLower(domain), ".my.obiente.cloud") {
		http.Error(w, "Domain must be a *.my.obiente.cloud domain", http.StatusBadRequest)
		return
	}

	// Support A and SRV record types
	if recordType != "A" && recordType != "SRV" {
		http.Error(w, "Only A and SRV record types are supported", http.StatusBadRequest)
		return
	}

	// Set default TTL if not provided
	ttl := req.TTL
	if ttl == 0 {
		ttl = 300 // Default: 5 minutes
	}

	// Get source API URL from request (for tracking and chain prevention)
	sourceAPI := r.Header.Get("X-Source-API")
	if sourceAPI == "" {
		sourceAPI = r.RemoteAddr // Fallback to client IP
	}

	// Prevent delegation chains: if the source API is itself using delegation, reject
	// Check if any API key has this source API as its SourceAPI (meaning it's a server using delegation)
	var existingKey database.DNSDelegationAPIKey
	result := database.DB.Where("source_api = ? AND is_active = ? AND revoked_at IS NULL", sourceAPI, true).First(&existingKey)
	if result.Error == nil {
		// This source API is itself using delegation - prevent chain
		log.Printf("[DNS Delegation] Rejected delegation chain: source API %s is itself using delegation", sourceAPI)
		metrics.RecordDNSDelegationPushError(apiKeyInfo.OrganizationID, apiKeyInfo.ID, "delegation_chain_prevented")
		http.Error(w, "Delegation chains are not allowed. Servers using DNS delegation cannot accept delegation requests from other servers.", http.StatusForbidden)
		return
	}

	// Convert records to JSON
	recordsJSON, err := json.Marshal(req.Records)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal records: %v", err), http.StatusInternalServerError)
		return
	}

	// Upsert the delegated DNS record with API key tracking
	if err := database.UpsertDelegatedDNSRecordWithAPIKey(domain, recordType, string(recordsJSON), sourceAPI, apiKeyInfo.ID, apiKeyInfo.OrganizationID, ttl); err != nil {
		log.Printf("[DNS Delegation] Failed to upsert DNS record: %v", err)
		metrics.RecordDNSDelegationPushError(apiKeyInfo.OrganizationID, apiKeyInfo.ID, "upsert_failed")
		http.Error(w, fmt.Sprintf("Failed to store DNS record: %v", err), http.StatusInternalServerError)
		return
	}

	// Record metrics
	metrics.RecordDNSDelegationPush(apiKeyInfo.OrganizationID, apiKeyInfo.ID, recordType)

	log.Printf("[DNS Delegation] Pushed DNS record: %s %s -> %v (TTL: %d)", domain, recordType, req.Records, ttl)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"domain":  domain,
		"message": "DNS record pushed successfully",
	})
}

// HandlePushDNSRecords handles batch DNS record push requests
func HandlePushDNSRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check API key authentication
	apiKey := r.Header.Get("Authorization")
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	// Validate API key and get key info
	apiKeyInfo, err := database.GetDNSDelegationAPIKeyByHash(apiKey)
	if err != nil {
		log.Printf("[DNS Delegation] API key validation error: %v", err)
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		Records []struct {
			Domain     string   `json:"domain"`
			RecordType string   `json:"record_type"`
			Records    []string `json:"records"`
			TTL        int64    `json:"ttl"`
		} `json:"records"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Records) == 0 {
		http.Error(w, "At least one record is required", http.StatusBadRequest)
		return
	}

	// Get source API URL from request (for tracking and chain prevention)
	sourceAPI := r.Header.Get("X-Source-API")
	if sourceAPI == "" {
		sourceAPI = r.RemoteAddr
	}

	// Prevent delegation chains: if the source API is itself using delegation, reject
	var existingKey database.DNSDelegationAPIKey
	result := database.DB.Where("source_api = ? AND is_active = ? AND revoked_at IS NULL", sourceAPI, true).First(&existingKey)
	if result.Error == nil {
		// This source API is itself using delegation - prevent chain
		log.Printf("[DNS Delegation] Rejected delegation chain: source API %s is itself using delegation", sourceAPI)
		metrics.RecordDNSDelegationPushError(apiKeyInfo.OrganizationID, apiKeyInfo.ID, "delegation_chain_prevented")
		http.Error(w, "Delegation chains are not allowed. Servers using DNS delegation cannot accept delegation requests from other servers.", http.StatusForbidden)
		return
	}

	successCount := 0
	errors := make([]string, 0)

	for _, recordReq := range req.Records {
		domain := strings.TrimSpace(recordReq.Domain)
		recordType := strings.ToUpper(strings.TrimSpace(recordReq.RecordType))
		if recordType == "" {
			recordType = "A"
		}

		if domain == "" || len(recordReq.Records) == 0 {
			errors = append(errors, fmt.Sprintf("Invalid record: domain=%s, records=%d", domain, len(recordReq.Records)))
			continue
		}

		if !strings.HasSuffix(strings.ToLower(domain), ".my.obiente.cloud") {
			errors = append(errors, fmt.Sprintf("Invalid domain format: %s", domain))
			continue
		}

		if recordType != "A" && recordType != "SRV" {
			errors = append(errors, fmt.Sprintf("Unsupported record type: %s", recordType))
			continue
		}

		ttl := recordReq.TTL
		if ttl == 0 {
			ttl = 300
		}

		recordsJSON, err := json.Marshal(recordReq.Records)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to marshal records for %s: %v", domain, err))
			continue
		}

		if err := database.UpsertDelegatedDNSRecordWithAPIKey(domain, recordType, string(recordsJSON), sourceAPI, apiKeyInfo.ID, apiKeyInfo.OrganizationID, ttl); err != nil {
			metrics.RecordDNSDelegationPushError(apiKeyInfo.OrganizationID, apiKeyInfo.ID, "upsert_failed")
			errors = append(errors, fmt.Sprintf("Failed to store %s: %v", domain, err))
			continue
		}

		// Record metrics
		metrics.RecordDNSDelegationPush(apiKeyInfo.OrganizationID, apiKeyInfo.ID, recordType)
		successCount++
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      len(errors) == 0,
		"success_count": successCount,
		"error_count":   len(errors),
		"errors":        errors,
	})
}

