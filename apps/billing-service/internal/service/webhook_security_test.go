package billing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/webhook"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func billingWebhookTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("OBIENTE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requires an isolated PostgreSQL database")
	}
	base, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("billing_webhook_%d", time.Now().UnixNano())
	if err := base.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	prior := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = prior
		raw, _ := db.DB()
		raw.Close()
		base.Exec("DROP SCHEMA " + schema + " CASCADE")
		raw, _ = base.DB()
		raw.Close()
	})
	if err := db.AutoMigrate(&database.Organization{}, &database.CreditTransaction{}, &database.StripeWebhookEvent{}, &database.BillingAccount{}, &database.OrgQuota{}, &database.OrganizationPlan{}, &database.MonthlyBill{}, &database.MonthlyCreditGrant{}, &database.DNSDelegationAPIKey{}, &database.DelegatedDNSRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.OrganizationPlan{ID: "starter", Name: "Starter"}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func sendStripeEvent(t *testing.T, eventID, eventType string, object map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	secret := "synthetic-webhook-secret"
	t.Setenv("STRIPE_WEBHOOK_SECRET", secret)
	t.Setenv("BILLING_ENABLED", "true")
	payload, err := json.Marshal(map[string]any{"id": eventID, "object": "event", "api_version": stripe.APIVersion, "type": eventType, "data": map[string]any{"object": object}})
	if err != nil {
		t.Fatal(err)
	}
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{Payload: payload, Secret: secret})
	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signed.Header)
	response := httptest.NewRecorder()
	HandleStripeWebhook(response, req)
	return response
}
func checkoutObject(sessionID, status, mode string) map[string]any {
	return map[string]any{"id": sessionID, "object": "checkout.session", "mode": mode, "payment_status": status, "amount_total": 1000, "customer": "cus_synthetic", "metadata": map[string]string{"organization_id": "org-synthetic"}}
}
func seedPaymentOrganization(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Create(&database.Organization{ID: "org-synthetic", Slug: "synthetic", Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
}
func paymentBalance(t *testing.T, db *gorm.DB) (int64, int64) {
	t.Helper()
	var org database.Organization
	var count int64
	if err := db.First(&org, "id = ?", "org-synthetic").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&database.CreditTransaction{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return org.Credits, count
}
func TestCheckoutOnlyCreditsPaidOneOffPayments(t *testing.T) {
	for _, tc := range []struct{ status, mode string }{{"unpaid", "payment"}, {"no_payment_required", "payment"}, {"", "payment"}, {"paid", "subscription"}, {"paid", "setup"}} {
		t.Run(tc.status+"_"+tc.mode, func(t *testing.T) {
			db := billingWebhookTestDB(t)
			seedPaymentOrganization(t, db)
			response := sendStripeEvent(t, "evt_not_paid", "checkout.session.completed", checkoutObject("cs_synthetic", tc.status, tc.mode))
			if response.Code != 200 {
				t.Fatalf("status=%d", response.Code)
			}
			if balance, count := paymentBalance(t, db); balance != 0 || count != 0 {
				t.Fatalf("balance=%d ledger=%d", balance, count)
			}
		})
	}
}
func TestCheckoutDelayedSuccessCreditsExactlyOnce(t *testing.T) {
	db := billingWebhookTestDB(t)
	seedPaymentOrganization(t, db)
	for _, tc := range []struct{ id, kind, status string }{{"evt_pending", "checkout.session.completed", "unpaid"}, {"evt_paid", "checkout.session.async_payment_succeeded", "paid"}, {"evt_paid", "checkout.session.async_payment_succeeded", "paid"}, {"evt_duplicate", "checkout.session.completed", "paid"}} {
		if response := sendStripeEvent(t, tc.id, tc.kind, checkoutObject("cs_synthetic", tc.status, "payment")); response.Code != 200 {
			t.Fatalf("event=%s status=%d", tc.id, response.Code)
		}
	}
	if balance, count := paymentBalance(t, db); balance != 1000 || count != 1 {
		t.Fatalf("balance=%d ledger=%d", balance, count)
	}
}
func TestSubscriptionDoesNotLogDNSKey(t *testing.T) {
	db := billingWebhookTestDB(t)
	seedPaymentOrganization(t, db)
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })
	object := map[string]any{"id": "sub_synthetic", "object": "subscription", "status": "active", "customer": map[string]any{"id": "cus_synthetic", "metadata": map[string]string{"organization_id": "org-synthetic"}}, "items": map[string]any{"data": []any{map[string]any{"price": map[string]any{"unit_amount": 200, "recurring": map[string]any{"interval": "month"}}}}}}
	for _, kind := range []string{"customer.subscription.created", "customer.subscription.updated"} {
		if response := sendStripeEvent(t, "evt_"+kind, kind, object); response.Code != 200 {
			t.Fatalf("status=%d", response.Code)
		}
		if err := database.RevokeDNSDelegationAPIKeysForSubscription("sub_synthetic"); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Contains(logs.String(), "API Key:") {
		t.Fatal("raw API key was logged")
	}
	var count int64
	if err := db.Model(&database.DNSDelegationAPIKey{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("keys=%d", count)
	}
}
