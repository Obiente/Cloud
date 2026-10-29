package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestDNSPusherBatchesWithinCountAndByteLimits(t *testing.T) {
	for _, tc := range []struct {
		name          string
		count, values int
	}{{"record count", 2001, 1}, {"encoded bytes", 900, 100}} {
		t.Run(tc.name, func(t *testing.T) {
			var records []map[string]interface{}
			for i := range tc.count {
				values := make([]string, tc.values)
				for j := range values {
					values[j] = "192.0.2.123"
				}
				records = append(records, map[string]interface{}{"domain": fmt.Sprintf("deploy-synthetic-%d.my.obiente.cloud", i), "record_type": "A", "records": values, "ttl": int64(300)})
			}
			seen := map[string]bool{}
			var mu sync.Mutex
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests++
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				var request struct {
					Records []struct {
						Domain string `json:"domain"`
					} `json:"records"`
				}
				if len(body) > maxDNSPushBodyBytes || json.Unmarshal(body, &request) != nil || len(request.Records) > maxDNSPushRecords || len(request.Records) == 0 {
					t.Errorf("rejected batch: bytes=%d records=%d", len(body), len(request.Records))
					http.Error(w, "too large", 400)
					return
				}
				if r.Header.Get("Authorization") != "Bearer synthetic-key" || r.Header.Get("X-Source-API") != "https://synthetic.example.test" {
					t.Error("missing push credentials/source")
				}
				for _, record := range request.Records {
					if seen[record.Domain] {
						t.Errorf("duplicate %s", record.Domain)
					}
					seen[record.Domain] = true
				}
				// Do not write local mirrors: this test verifies the HTTP transport.
				fmt.Fprintf(w, `{"success_count":0,"total":%d,"errors":[]}`, len(request.Records))
			}))
			defer server.Close()
			pushDNSRecordBatches(context.Background(), server.Client(), server.URL, "synthetic-key", "https://synthetic.example.test", 300, records)
			mu.Lock()
			defer mu.Unlock()
			if requests < 2 || len(seen) != tc.count {
				t.Fatalf("requests=%d records=%d want=%d", requests, len(seen), tc.count)
			}
		})
	}
}

func TestDNSPushBatchRejectsSingleOversizedRecord(t *testing.T) {
	if _, err := splitDNSPushBatches([]map[string]interface{}{{"domain": strings.Repeat("x", maxDNSPushBodyBytes)}}); err == nil {
		t.Fatal("oversized record accepted")
	}
}
