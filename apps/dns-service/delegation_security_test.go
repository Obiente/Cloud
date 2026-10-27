package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func delegationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("OBIENTE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requires an isolated PostgreSQL database")
	}
	base, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("dns_security_%d", time.Now().UnixNano())
	if err := base.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previous
		raw, _ := db.DB()
		raw.Close()
		base.Exec("DROP SCHEMA " + schema + " CASCADE")
		raw, _ = base.DB()
		raw.Close()
	})
	if err := db.AutoMigrate(&database.Organization{}, &database.Deployment{}, &database.GameServer{}, &database.DatabaseInstance{}, &database.DNSDelegationAPIKey{}, &database.DelegatedDNSRecord{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"owner", "other"} {
		if err := db.Create(&database.Organization{ID: id, Slug: id, Status: "active"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func delegationKey(t *testing.T, org string) (string, *database.DNSDelegationAPIKey) {
	t.Helper()
	key, err := database.CreateDNSDelegationAPIKey("Test key", "", org, nil)
	if err != nil {
		t.Fatal(err)
	}
	info, err := database.GetDNSDelegationAPIKeyByHash(key)
	if err != nil {
		t.Fatal(err)
	}
	return key, info
}

func pushDelegation(t *testing.T, key string, batch bool, domain, kind string, records []string, ttl int64) *httptest.ResponseRecorder {
	t.Helper()
	var body any = map[string]any{"domain": domain, "record_type": kind, "records": records, "ttl": ttl}
	if batch {
		body = map[string]any{"records": []any{body}}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/dns/push", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+key)
	rec := httptest.NewRecorder()
	if batch {
		handlePushDNSRecords(rec, req)
	} else {
		handlePushDNSRecord(rec, req)
	}
	return rec
}

func TestDelegationTenantIsolation(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			db := delegationTestDB(t)
			owner, _ := delegationKey(t, "owner")
			other, _ := delegationKey(t, "other")
			domain := "deploy-owner.my.obiente.cloud"
			if response := pushDelegation(t, owner, false, domain, "A", []string{"192.0.2.1"}, 300); response.Code != 200 {
				t.Fatal(response.Body.String())
			}
			response := pushDelegation(t, other, batch, "DEPLOY-OWNER.MY.OBIENTE.CLOUD.", "A", []string{"192.0.2.2"}, 300)
			if batch {
				var result struct {
					Success int `json:"success_count"`
				}
				json.Unmarshal(response.Body.Bytes(), &result)
				if result.Success != 0 {
					t.Fatal("cross-tenant batch succeeded")
				}
			} else if response.Code != 403 {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var record database.DelegatedDNSRecord
			if err := db.First(&record, "domain = ?", domain).Error; err != nil {
				t.Fatal(err)
			}
			if record.OrganizationID != "owner" || record.Records != `["192.0.2.1"]` {
				t.Fatalf("record changed: %+v", record)
			}
			if response := pushDelegation(t, owner, batch, domain, "A", []string{"192.0.2.3"}, 300); response.Code != 200 {
				t.Fatal(response.Body.String())
			}
		})
	}
}

func TestDelegationOrganizationRestriction(t *testing.T) {
	for _, status := range []string{"banned", "suspended"} {
		t.Run(status, func(t *testing.T) {
			db := delegationTestDB(t)
			key, info := delegationKey(t, "owner")
			domain := "deploy-restricted.my.obiente.cloud"
			if response := pushDelegation(t, key, false, domain, "A", []string{"192.0.2.1"}, 300); response.Code != 200 {
				t.Fatal(response.Body.String())
			}
			db.Model(&database.Organization{}).Where("id = ?", "owner").Update("status", status)
			for _, batch := range []bool{false, true} {
				if response := pushDelegation(t, key, batch, domain, "A", []string{"192.0.2.2"}, 300); response.Code != 401 {
					t.Fatalf("restricted status=%d", response.Code)
				}
			}
			if _, err := database.GetDelegatedDNSRecord(domain, "A"); err == nil {
				t.Fatal("restricted DNS still resolves")
			}
			if err := database.UpsertDelegatedDNSRecordWithAPIKey(domain, "A", `["192.0.2.2"]`, "test", info.ID, info.OrganizationID, 300); err == nil {
				t.Fatal("stale authenticated key bypassed restriction")
			}
			if status == "suspended" {
				db.Model(&database.Organization{}).Where("id = ?", "owner").Update("suspension_expires", time.Now().Add(-time.Minute))
				if response := pushDelegation(t, key, false, domain, "A", []string{"192.0.2.2"}, 300); response.Code != 200 {
					t.Fatal("expired suspension did not recover")
				}
			}
		})
	}
}

func TestDelegationRevocation(t *testing.T) {
	for _, subscription := range []bool{false, true} {
		t.Run(fmt.Sprint(subscription), func(t *testing.T) {
			db := delegationTestDB(t)
			key, info := delegationKey(t, "owner")
			db.Model(info).Update("stripe_subscription_id", "sub_test")
			domain := "deploy-revoked.my.obiente.cloud"
			if response := pushDelegation(t, key, false, domain, "A", []string{"192.0.2.1"}, 300); response.Code != 200 {
				t.Fatal(response.Body.String())
			}
			if _, err := database.GetDelegatedDNSRecord(domain, "A"); err != nil {
				t.Fatal(err)
			}
			var err error
			if subscription {
				err = database.RevokeDNSDelegationAPIKeysForSubscription("sub_test")
			} else {
				err = database.RevokeDNSDelegationAPIKey(database.HashAPIKeyForDelegation(key))
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := database.GetDelegatedDNSRecord(domain, "A"); err == nil {
				t.Fatal("revoked DNS still resolves")
			}
			if err := database.UpsertDelegatedDNSRecordWithAPIKey(domain, "A", `["192.0.2.2"]`, "test", info.ID, info.OrganizationID, 300); err == nil {
				t.Fatal("revoked key accepted by write")
			}
		})
	}
}

func TestDelegationDomainEntitlementAndLimits(t *testing.T) {
	db := delegationTestDB(t)
	owner, _ := delegationKey(t, "owner")
	other, _ := delegationKey(t, "other")
	localID := "deploy-11111111-2222-3333-4444-555555555555"
	if err := db.Model(&database.Deployment{}).Create(map[string]any{"id": localID, "organization_id": "owner", "domain": database.DefaultMyObienteCloudDomain(localID), "custom_domains": `["app.example.test:verified","token.example.test:token:synthetic:verified","pending.example.test:token:verified:pending"]`}).Error; err != nil {
		t.Fatal(err)
	}
	for _, domain := range []string{localID + ".my.obiente.cloud", database.DefaultMyObienteCloudDomain(localID), "app.example.test", "pending.example.test", "_acme-challenge.my.obiente.cloud", "arbitrary.my.obiente.cloud"} {
		if response := pushDelegation(t, other, false, domain, "A", []string{"192.0.2.1"}, 300); response.Code == 200 {
			t.Fatalf("unauthorized domain accepted: %s", domain)
		}
	}
	if response := pushDelegation(t, owner, false, "app.example.test", "A", []string{"192.0.2.1"}, 300); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if response := pushDelegation(t, owner, false, "token.example.test", "A", []string{"192.0.2.1"}, 300); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	for _, ttl := range []int64{-1, 3601, 31536000, 1 << 62} {
		if response := pushDelegation(t, owner, false, "deploy-limits.my.obiente.cloud", "A", []string{"192.0.2.1"}, ttl); response.Code != 400 {
			t.Fatalf("ttl %d status=%d", ttl, response.Code)
		}
	}
	if response := pushDelegation(t, owner, false, "deploy-limits.my.obiente.cloud", "A", []string{"not-an-ip"}, 300); response.Code != 400 {
		t.Fatalf("invalid IP status=%d", response.Code)
	}
	for _, ttl := range []int64{0, 1, 3600} {
		if response := pushDelegation(t, owner, false, "deploy-limits.my.obiente.cloud", "A", []string{"192.0.2.1"}, ttl); response.Code != 200 {
			t.Fatalf("ttl %d: %s", ttl, response.Body.String())
		}
	}
}

func TestDelegationConcurrentResourceClaims(t *testing.T) {
	for _, srv := range []bool{false, true} {
		t.Run(fmt.Sprint(srv), func(t *testing.T) {
			db := delegationTestDB(t)
			_, owner := delegationKey(t, "owner")
			_, other := delegationKey(t, "other")
			base := "gs-concurrent.my.obiente.cloud"
			start := make(chan struct{})
			results := make(chan error, 2)
			var wg sync.WaitGroup
			for i, key := range []*database.DNSDelegationAPIKey{owner, other} {
				wg.Add(1)
				go func(i int, key *database.DNSDelegationAPIKey) {
					defer wg.Done()
					<-start
					domain, kind, values := base, "A", `["192.0.2.1"]`
					if srv && i == 1 {
						domain, kind, values = "_minecraft._tcp."+base, "SRV", `["0 0 25565 `+base+`"]`
					}
					results <- database.UpsertDelegatedDNSRecordWithAPIKey(domain, kind, values, "test", key.ID, key.OrganizationID, 300)
				}(i, key)
			}
			close(start)
			wg.Wait()
			close(results)
			success := 0
			for err := range results {
				if err == nil {
					success++
				}
			}
			if success != 1 {
				t.Fatalf("successful competing owners=%d", success)
			}
			var owners []string
			db.Model(&database.DelegatedDNSRecord{}).Distinct("organization_id").Pluck("organization_id", &owners)
			if len(owners) != 1 {
				t.Fatalf("owners=%v", owners)
			}
		})
	}
}

func TestDelegationServiceRecordsAndLegacyTTL(t *testing.T) {
	db := delegationTestDB(t)
	key, info := delegationKey(t, "owner")
	base := "gs-synthetic.my.obiente.cloud"
	for _, kind := range []string{"A", "SRV"} {
		domain, values := base, []string{"192.0.2.1"}
		if kind == "SRV" {
			domain, values = "_minecraft._tcp."+base, []string{"0 0 25565 " + base}
		}
		if response := pushDelegation(t, key, false, domain, kind, values, 300); response.Code != 200 {
			t.Fatal(response.Body.String())
		}
		if _, err := database.GetDelegatedDNSRecord(domain, kind); err != nil {
			t.Fatal(err)
		}
	}
	legacy := database.DelegatedDNSRecord{ID: "legacy", Domain: "deploy-legacy.my.obiente.cloud", RecordType: "A", Records: `["192.0.2.1"]`, APIKeyID: info.ID, OrganizationID: info.OrganizationID, TTL: 31536000, LastUpdated: time.Now().Add(-2 * time.Hour), ExpiresAt: time.Now().Add(365 * 24 * time.Hour)}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := database.GetDelegatedDNSRecord(legacy.Domain, "A"); err == nil {
		t.Fatal("legacy excessive TTL still resolves after maximum lease")
	}
}

func TestDelegationExpiryPreservesOwnership(t *testing.T) {
	db := delegationTestDB(t)
	owner, _ := delegationKey(t, "owner")
	other, _ := delegationKey(t, "other")
	domain := "deploy-recover.my.obiente.cloud"
	if response := pushDelegation(t, owner, false, domain, "A", []string{"192.0.2.1"}, 300); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	db.Model(&database.DelegatedDNSRecord{}).Where("domain = ?", domain).UpdateColumn("expires_at", time.Now().Add(-time.Hour))
	if err := database.CleanupExpiredDelegatedRecords(); err != nil {
		t.Fatal(err)
	}
	if _, err := database.GetDelegatedDNSRecord(domain, "A"); err == nil {
		t.Fatal("expired record still resolves")
	}
	if response := pushDelegation(t, other, false, domain, "A", []string{"192.0.2.2"}, 300); response.Code != 403 {
		t.Fatalf("expired hostname claimed by another tenant: %d", response.Code)
	}
	if response := pushDelegation(t, owner, false, domain, "A", []string{"192.0.2.3"}, 300); response.Code != 200 {
		t.Fatal("owner could not recover after expiry")
	}
}
