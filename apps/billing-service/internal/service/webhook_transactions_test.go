package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/stripe/stripe-go/v83"
	"gorm.io/gorm"
)

func TestWebhookRetryAfterDependencyRecovers(t *testing.T) {
	db := billingWebhookTestDB(t)
	object := checkoutObject("cs_retry", "paid", "payment")
	if response := sendStripeEvent(t, "evt_retry", "checkout.session.completed", object); response.Code != 500 {
		t.Fatalf("first status=%d", response.Code)
	}
	seedPaymentOrganization(t, db)
	if response := sendStripeEvent(t, "evt_retry", "checkout.session.completed", object); response.Code != 200 {
		t.Fatalf("retry status=%d", response.Code)
	}
	if balance, count := paymentBalance(t, db); balance != 1000 || count != 1 {
		t.Fatalf("balance=%d ledger=%d", balance, count)
	}
}

func TestWebhookFailureRollsBackPaymentAndCompletion(t *testing.T) {
	for _, table := range []string{"credit_transactions", "billing_accounts", "stripe_webhook_events"} {
		t.Run(table, func(t *testing.T) {
			db := billingWebhookTestDB(t)
			seedPaymentOrganization(t, db)
			if err := db.Callback().Create().Before("gorm:create").Register("test:write_failure", func(tx *gorm.DB) {
				if tx.Statement.Table == table {
					tx.AddError(errors.New("synthetic write failure"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			object := checkoutObject("cs_rollback", "paid", "payment")
			if response := sendStripeEvent(t, "evt_rollback", "checkout.session.completed", object); response.Code != 500 {
				t.Fatalf("failure status=%d", response.Code)
			}
			if balance, count := paymentBalance(t, db); balance != 0 || count != 0 {
				t.Fatalf("partial commit balance=%d ledger=%d", balance, count)
			}
			var events int64
			if err := db.Model(&database.StripeWebhookEvent{}).Count(&events).Error; err != nil {
				t.Fatal(err)
			}
			if events != 0 {
				t.Fatal("failed event marked complete")
			}
			db.Callback().Create().Remove("test:write_failure")
			if response := sendStripeEvent(t, "evt_rollback", "checkout.session.completed", object); response.Code != 200 {
				t.Fatalf("retry status=%d", response.Code)
			}
			if balance, count := paymentBalance(t, db); balance != 1000 || count != 1 {
				t.Fatalf("retry balance=%d ledger=%d", balance, count)
			}
		})
	}
}

func TestWebhookEventLookupFailureStopsProcessing(t *testing.T) {
	db := billingWebhookTestDB(t)
	seedPaymentOrganization(t, db)
	db.Callback().Query().Before("gorm:query").Register("test:lookup_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "stripe_webhook_events" {
			tx.AddError(errors.New("synthetic read failure"))
		}
	})
	if response := sendStripeEvent(t, "evt_lookup", "checkout.session.completed", checkoutObject("cs_lookup", "paid", "payment")); response.Code != 500 {
		t.Fatalf("status=%d", response.Code)
	}
	if balance, count := paymentBalance(t, db); balance != 0 || count != 0 {
		t.Fatal("payment applied despite unavailable event store")
	}
}

func syntheticStripeEvent(t *testing.T, id, kind string, object map[string]any) *stripe.Event {
	t.Helper()
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return &stripe.Event{ID: id, Type: stripe.EventType(kind), Data: &stripe.EventData{Raw: raw}}
}

func TestConcurrentStripePayments(t *testing.T) {
	for _, mode := range []string{"distinct", "duplicate_event", "duplicate_session", "checkout_and_invoice"} {
		t.Run(mode, func(t *testing.T) {
			db := billingWebhookTestDB(t)
			seedPaymentOrganization(t, db)
			customer := "cus_synthetic"
			if err := db.Create(&database.BillingAccount{ID: "billing", OrganizationID: "org-synthetic", StripeCustomerID: &customer}).Error; err != nil {
				t.Fatal(err)
			}
			// Hold the first balance read briefly to exercise competing writes.
			var reads atomic.Int32
			db.Callback().Query().After("gorm:query").Register("test:overlap", func(tx *gorm.DB) {
				if tx.Statement.Table == "organizations" && reads.Add(1) == 1 {
					time.Sleep(100 * time.Millisecond)
				}
			})
			first := syntheticStripeEvent(t, "evt_first", "checkout.session.completed", checkoutObject("cs_first", "paid", "payment"))
			second := syntheticStripeEvent(t, "evt_second", "checkout.session.completed", checkoutObject("cs_second", "paid", "payment"))
			wantBalance, wantLedger := int64(2000), int64(2)
			switch mode {
			case "duplicate_event":
				second = first
				wantBalance, wantLedger = 1000, 1
			case "duplicate_session":
				second = syntheticStripeEvent(t, "evt_second", "checkout.session.async_payment_succeeded", checkoutObject("cs_first", "paid", "payment"))
				wantBalance, wantLedger = 1000, 1
			case "checkout_and_invoice":
				second = syntheticStripeEvent(t, "evt_invoice", "invoice.paid", map[string]any{"id": "in_synthetic", "object": "invoice", "customer": "cus_synthetic", "amount_paid": 1000, "metadata": map[string]string{"organization_id": "org-synthetic"}})
			}
			start := make(chan struct{})
			errs := make(chan error, 2)
			var wg sync.WaitGroup
			for _, event := range []*stripe.Event{first, second} {
				wg.Add(1)
				go func(event *stripe.Event) {
					defer wg.Done()
					<-start
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					errs <- processStripeWebhook(ctx, event)
				}(event)
			}
			close(start)
			wg.Wait()
			close(errs)
			for err := range errs {
				if err != nil {
					t.Fatal(err)
				}
			}
			if balance, count := paymentBalance(t, db); balance != wantBalance || count != wantLedger {
				t.Fatalf("balance=%d ledger=%d want %d/%d", balance, count, wantBalance, wantLedger)
			}
			var org database.Organization
			if err := db.First(&org, "id = ?", "org-synthetic").Error; err != nil {
				t.Fatal(err)
			}
			if org.TotalPaidCents != wantBalance {
				t.Fatalf("total paid=%d", org.TotalPaidCents)
			}
			var ledger []database.CreditTransaction
			db.Order("balance_after").Find(&ledger)
			for i, row := range ledger {
				if row.BalanceAfter != int64((i+1)*1000) {
					t.Fatalf("non-serial ledger balance=%d", row.BalanceAfter)
				}
			}
		})
	}
}

func TestStripeLegacyPaymentDeduplicationIsExact(t *testing.T) {
	db := billingWebhookTestDB(t)
	seedPaymentOrganization(t, db)
	note := "Payment via Stripe Checkout Session cs_legacy"
	if err := db.Create(&database.CreditTransaction{ID: "legacy", OrganizationID: "org-synthetic", Source: "stripe", Type: "payment", Note: &note, AmountCents: 1000, BalanceAfter: 1000}).Error; err != nil {
		t.Fatal(err)
	}
	db.Model(&database.Organization{}).Where("id = ?", "org-synthetic").Update("credits", 1000)
	for i, session := range []string{"cs_legacy", "cs_legac", "csXlegacy"} {
		if response := sendStripeEvent(t, fmt.Sprintf("evt_legacy_%d", i), "checkout.session.completed", checkoutObject(session, "paid", "payment")); response.Code != 200 {
			t.Fatalf("status=%d", response.Code)
		}
	}
	if balance, count := paymentBalance(t, db); balance != 3000 || count != 3 {
		t.Fatalf("balance=%d ledger=%d", balance, count)
	}
}

func TestConcurrentBillPaymentChargesOnce(t *testing.T) {
	db := billingWebhookTestDB(t)
	seedPaymentOrganization(t, db)
	db.Model(&database.Organization{}).Where("id = ?", "org-synthetic").Update("credits", 2000)
	if err := db.Create(&database.MonthlyBill{ID: "bill", OrganizationID: "org-synthetic", Status: "PENDING", AmountCents: 1000, UsageBreakdown: "{}"}).Error; err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		go func() { <-start; errs <- PayBillPrematurely("bill", "org-synthetic") }()
	}
	close(start)
	success := 0
	for range 2 {
		if err := <-errs; err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("successful duplicate payments=%d", success)
	}
	if balance, count := paymentBalance(t, db); balance != 1000 || count != 1 {
		t.Fatalf("balance=%d ledger=%d", balance, count)
	}
}

func TestSubscriptionKeyCreationRollsBackWithEvent(t *testing.T) {
	db := billingWebhookTestDB(t)
	seedPaymentOrganization(t, db)
	object := map[string]any{"id": "sub_atomic", "object": "subscription", "status": "active", "customer": map[string]any{"id": "cus_synthetic", "metadata": map[string]string{"organization_id": "org-synthetic"}}, "items": map[string]any{"data": []any{map[string]any{"price": map[string]any{"unit_amount": 200, "recurring": map[string]any{"interval": "month"}}}}}}
	db.Callback().Create().Before("gorm:create").Register("test:completion_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "stripe_webhook_events" {
			tx.AddError(errors.New("synthetic completion failure"))
		}
	})
	if response := sendStripeEvent(t, "evt_subscription_atomic", "customer.subscription.created", object); response.Code != 500 {
		t.Fatalf("failure status=%d", response.Code)
	}
	var count int64
	if err := db.Model(&database.DNSDelegationAPIKey{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("DNS key escaped failed event transaction")
	}
	db.Callback().Create().Remove("test:completion_failure")
	if response := sendStripeEvent(t, "evt_subscription_atomic", "customer.subscription.created", object); response.Code != 200 {
		t.Fatalf("retry status=%d", response.Code)
	}
	if err := db.Model(&database.DNSDelegationAPIKey{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("keys=%d", count)
	}
}

func TestMonthlyGrantAndCheckoutPreserveBalance(t *testing.T) {
	db := billingWebhookTestDB(t)
	seedPaymentOrganization(t, db)
	if err := db.Model(&database.OrganizationPlan{}).Where("id = ?", "starter").Update("monthly_free_credits_cents", 100).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.OrgQuota{OrganizationID: "org-synthetic", PlanID: "starter"}).Error; err != nil {
		t.Fatal(err)
	}
	processor := &stripeWebhookProcessor{db: db}
	start := make(chan struct{})
	errs := make(chan error, 2)
	go func() {
		<-start
		errs <- processor.addCreditsFromPayment("org-synthetic", 1000, "cs_with_grant", "cus_synthetic")
	}()
	go func() { <-start; errs <- processor.grantMonthlyFreeCreditsFromPlan("org-synthetic", "sub_synthetic") }()
	close(start)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if balance, count := paymentBalance(t, db); balance != 1100 || count != 2 {
		t.Fatalf("balance=%d ledger=%d", balance, count)
	}
	if err := processor.grantMonthlyFreeCreditsFromPlan("org-synthetic", "sub_synthetic"); err != nil {
		t.Fatal(err)
	}
	if balance, count := paymentBalance(t, db); balance != 1100 || count != 2 {
		t.Fatal("monthly grant was duplicated")
	}
}
