package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"

	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/webhook"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// HandleStripeWebhook processes Stripe webhook events
//
// Stripe automatically retries failed webhooks:
// - Live mode: up to 3 days with exponential backoff
// - Test mode: 3 retries over a few hours
//
// To prevent duplicate processing, we implement idempotency by:
// 1. Tracking processed event IDs in the database
// 2. Checking if an event was already processed before handling
// 3. Returning 200 OK for already-processed events (to stop retries)
// 4. Making handlers idempotent (safe to run multiple times)
func HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Check if billing is enabled
	billingEnabled := os.Getenv("BILLING_ENABLED") != "false" && os.Getenv("BILLING_ENABLED") != "0"
	if !billingEnabled {
		log.Printf("[Stripe Webhook] Billing is disabled, ignoring webhook")
		w.WriteHeader(http.StatusOK)
		return
	}

	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Stripe Webhook] Error reading request body: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Get webhook secret from environment
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Printf("[Stripe Webhook] STRIPE_WEBHOOK_SECRET not configured")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Verify webhook signature
	// Webhook endpoints must be configured with API version 2025-10-29.clover
	// in the Stripe Dashboard to match the SDK version.
	// See: https://stripe.com/docs/webhooks/best-practices#api-versioning
	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), webhookSecret)
	if err != nil {
		log.Printf("[Stripe Webhook] Signature verification failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := processStripeWebhook(ctx, &event); err != nil {
		log.Printf("[Stripe Webhook] Failed event %s (%s): %v", event.ID, event.Type, err)
		status := http.StatusInternalServerError
		var processingError *stripeWebhookError
		if errors.As(err, &processingError) {
			status = processingError.status
		}
		w.WriteHeader(status)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type stripeWebhookProcessor struct{ db *gorm.DB }
type stripeWebhookError struct {
	status int
	err    error
}

func (e *stripeWebhookError) Error() string { return e.err.Error() }
func (e *stripeWebhookError) Unwrap() error { return e.err }

// Event completion and all database side effects commit together. A failed or
// interrupted attempt leaves no completion marker, so Stripe can safely retry.
func processStripeWebhook(ctx context.Context, event *stripe.Event) error {
	if event.ID == "" || event.Data == nil {
		return &stripeWebhookError{status: http.StatusBadRequest, err: fmt.Errorf("missing event identity or data")}
	}
	return database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "stripe-event:"+event.ID).Error; err != nil {
				return err
			}
		}
		var existing database.StripeWebhookEvent
		if err := tx.First(&existing, "id = ?", event.ID).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		h := &stripeWebhookProcessor{db: tx}
		if err := h.dispatch(event); err != nil {
			return err
		}
		org, customer, subscription, invoice, session := h.extractEventIDs(string(event.Type), event.Data.Raw)
		return tx.Create(&database.StripeWebhookEvent{ID: event.ID, EventType: string(event.Type), ProcessedAt: time.Now(), CreatedAt: time.Now(), OrganizationID: org, CustomerID: customer, SubscriptionID: subscription, InvoiceID: invoice, CheckoutSessionID: session}).Error
	})
}

func (h *stripeWebhookProcessor) dispatch(event *stripe.Event) error {
	// Handle the event
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("[Stripe Webhook] Error parsing checkout.session.completed: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		if err := h.handleCheckoutSessionCompleted(&session, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling checkout.session.completed: %v", err)
			return &stripeWebhookError{status: http.StatusInternalServerError, err: err}
		}

	case "customer.subscription.created":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.created: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		if err := h.handleSubscriptionCreated(&subscription, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling customer.subscription.created: %v", err)
			return &stripeWebhookError{status: http.StatusInternalServerError, err: err}
		}

	case "customer.subscription.updated":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.updated: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		if err := h.handleSubscriptionUpdated(&subscription, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling customer.subscription.updated: %v", err)
			return &stripeWebhookError{status: http.StatusInternalServerError, err: err}
		}

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.deleted: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		if err := h.handleSubscriptionDeleted(&subscription); err != nil {
			log.Printf("[Stripe Webhook] Error handling customer.subscription.deleted: %v", err)
			return &stripeWebhookError{status: http.StatusInternalServerError, err: err}
		}

	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			log.Printf("[Stripe Webhook] Error parsing payment_intent.succeeded: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handlePaymentIntentSucceeded(&paymentIntent)

	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			log.Printf("[Stripe Webhook] Error parsing payment_intent.payment_failed: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handlePaymentIntentFailed(&paymentIntent)

	case "invoice.paid":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.paid: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		if err := h.handleInvoicePaid(&invoice, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling invoice.paid: %v", err)
			return &stripeWebhookError{status: http.StatusInternalServerError, err: err}
		}

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.payment_failed: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		if err := h.handleInvoicePaymentFailed(&invoice, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling invoice.payment_failed: %v", err)
			return &stripeWebhookError{status: http.StatusInternalServerError, err: err}
		}

	case "invoice.payment_action_required":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.payment_action_required: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoicePaymentActionRequired(&invoice, event.Data.Raw)

	case "invoice.finalization_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.finalization_failed: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceFinalizationFailed(&invoice, event.Data.Raw)

	case "checkout.session.expired":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("[Stripe Webhook] Error parsing checkout.session.expired: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleCheckoutSessionExpired(&session, event.Data.Raw)

	case "checkout.session.async_payment_succeeded":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("[Stripe Webhook] Error parsing checkout.session.async_payment_succeeded: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		if err := h.handleCheckoutSessionAsyncPaymentSucceeded(&session, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling checkout.session.async_payment_succeeded: %v", err)
			return &stripeWebhookError{status: http.StatusInternalServerError, err: err}
		}

	case "checkout.session.async_payment_failed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("[Stripe Webhook] Error parsing checkout.session.async_payment_failed: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleCheckoutSessionAsyncPaymentFailed(&session, event.Data.Raw)

	case "customer.subscription.trial_will_end":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.trial_will_end: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionTrialWillEnd(&subscription, event.Data.Raw)

	case "customer.subscription.paused":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.paused: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionPaused(&subscription, event.Data.Raw)

	case "customer.subscription.resumed":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.resumed: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		if err := h.handleSubscriptionResumed(&subscription, event.Data.Raw); err != nil {
			return err
		}

	case "customer.subscription.pending_update_applied":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.pending_update_applied: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionPendingUpdateApplied(&subscription, event.Data.Raw)

	case "customer.subscription.pending_update_expired":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.pending_update_expired: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionPendingUpdateExpired(&subscription, event.Data.Raw)

	case "invoice.created":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.created: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceCreated(&invoice, event.Data.Raw)

	case "invoice.deleted":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.deleted: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceDeleted(&invoice, event.Data.Raw)

	case "invoice.finalized":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.finalized: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceFinalized(&invoice, event.Data.Raw)

	case "invoice.marked_uncollectible":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.marked_uncollectible: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		if err := h.handleInvoiceMarkedUncollectible(&invoice, event.Data.Raw); err != nil {
			return err
		}

	case "invoice.overdue":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.overdue: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceOverdue(&invoice, event.Data.Raw)

	case "invoice.overpaid":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.overpaid: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceOverpaid(&invoice, event.Data.Raw)

	case "invoice.payment_succeeded":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.payment_succeeded: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoicePaymentSucceeded(&invoice, event.Data.Raw)

	case "invoice.sent":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.sent: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceSent(&invoice, event.Data.Raw)

	case "invoice.upcoming":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.upcoming: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceUpcoming(&invoice, event.Data.Raw)

	case "invoice.updated":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.updated: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceUpdated(&invoice, event.Data.Raw)

	case "invoice.voided":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.voided: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleInvoiceVoided(&invoice, event.Data.Raw)

	case "subscription_schedule.aborted":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.aborted: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionScheduleAborted(&schedule, event.Data.Raw)

	case "subscription_schedule.canceled":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.canceled: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionScheduleCanceled(&schedule, event.Data.Raw)

	case "subscription_schedule.completed":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.completed: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionScheduleCompleted(&schedule, event.Data.Raw)

	case "subscription_schedule.created":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.created: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionScheduleCreated(&schedule, event.Data.Raw)

	case "subscription_schedule.expiring":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.expiring: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionScheduleExpiring(&schedule, event.Data.Raw)

	case "subscription_schedule.released":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.released: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionScheduleReleased(&schedule, event.Data.Raw)

	case "subscription_schedule.updated":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.updated: %v", err)
			return &stripeWebhookError{status: http.StatusBadRequest, err: err}
		}
		h.handleSubscriptionScheduleUpdated(&schedule, event.Data.Raw)

	default:
		log.Printf("[Stripe Webhook] Unhandled event type: %s (ID: %s)", event.Type, event.ID)
	}

	return nil
}
func (h *stripeWebhookProcessor) handleCheckoutSessionCompleted(session *stripe.CheckoutSession, rawData []byte) error {
	// Extract organization ID from metadata
	orgID, ok := session.Metadata["organization_id"]
	if !ok {
		return fmt.Errorf("missing organization_id in metadata")
	}

	// Subscription payments and delayed checkouts are fulfilled by their own
	// lifecycle events. Credits require a successfully paid one-off checkout.
	if session.Mode != stripe.CheckoutSessionModePayment || session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		return nil
	}

	// Get payment intent amount from session (more reliable than metadata)
	amountCents := session.AmountTotal
	if amountCents <= 0 {
		return fmt.Errorf("invalid amount_total: %d", amountCents)
	}

	// Extract customer ID - Customer field can be nil, a string ID, or a Customer object
	var customerID string
	if session.Customer != nil {
		// If Customer is populated, it's a Customer object, get the ID
		customerID = session.Customer.ID
	} else {
		// Try to get from raw JSON if Customer wasn't expanded
		// This handles cases where Stripe sends customer as a string ID in webhooks
		var rawSession map[string]interface{}
		if err := json.Unmarshal(rawData, &rawSession); err == nil {
			if cust, ok := rawSession["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if customerID == "" {
		return fmt.Errorf("missing customer in checkout session")
	}

	// Add credits to organization
	if err := h.addCreditsFromPayment(orgID, amountCents, session.ID, customerID); err != nil {
		return fmt.Errorf("add credits from payment: %w", err)
	}

	log.Printf("[Stripe Webhook] Successfully added %d cents to organization %s from checkout session %s", amountCents, orgID, session.ID)
	return nil
}

// handleSubscriptionCreated handles when a DNS delegation subscription is created
func (h *stripeWebhookProcessor) handleSubscriptionCreated(subscription *stripe.Subscription, rawData []byte) error {
	// Get organization from customer
	var orgID string
	var customerID string

	// Customer might be expanded (Customer object) or not expanded (string ID in JSON)
	if subscription.Customer != nil {
		customerID = subscription.Customer.ID
		if orgIDVal, ok := subscription.Customer.Metadata["organization_id"]; ok {
			orgID = orgIDVal
		}
	}

	// If Customer wasn't expanded, try to get customer ID from raw JSON
	if customerID == "" {
		var rawSub map[string]interface{}
		if err := json.Unmarshal(rawData, &rawSub); err == nil {
			if cust, ok := rawSub["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if orgID == "" && customerID != "" {
		// Try to find organization by Stripe customer ID
		var billingAccount database.BillingAccount
		if err := h.db.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
			orgID = billingAccount.OrganizationID
		}
	}

	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for subscription %s", subscription.ID)
		return nil // Don't fail, just log
	}

	// Check if this is a DNS delegation subscription (price amount is $2/month = 200 cents)
	isDNSDelegation := false
	if subscription.Items != nil && len(subscription.Items.Data) > 0 {
		price := subscription.Items.Data[0].Price
		if price != nil && price.UnitAmount == 200 && price.Recurring != nil && price.Recurring.Interval == "month" {
			isDNSDelegation = true
		}
	}

	if !isDNSDelegation {
		log.Printf("[Stripe Webhook] Subscription %s is not a DNS delegation subscription, skipping", subscription.ID)
		return nil
	}

	// Create API key for organization if subscription is active
	if subscription.Status == "active" || subscription.Status == "trialing" {
		// Serialize subscription side effects with other organization payments.
		var org database.Organization
		if err := h.db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&org, "id = ?", orgID).Error; err != nil {
			return err
		}
		// Check if organization already has an active API key
		existingKey, err := database.GetActiveDNSDelegationAPIKeyForOrganizationTx(h.db, orgID)
		if err == nil && existingKey != nil {
			log.Printf("[Stripe Webhook] Organization %s already has an active API key, skipping creation", orgID)
		} else {
			subscriptionID := subscription.ID
			description := fmt.Sprintf("DNS Delegation API Key (Subscription: %s)", subscriptionID)
			sourceAPI := "" // Will be set by user when they configure their API

			_, err := database.CreateDNSDelegationAPIKeyTx(h.db, description, sourceAPI, orgID, &subscriptionID)
			if err != nil {
				return fmt.Errorf("create DNS delegation API key: %w", err)
			}

			log.Printf("[Stripe Webhook] Created DNS delegation API key for organization %s (subscription: %s)", orgID, subscriptionID)
		}

		// Grant monthly free credits from plan when subscription becomes active
		if err := h.grantMonthlyFreeCreditsFromPlan(orgID, subscription.ID); err != nil {
			log.Printf("[Stripe Webhook] Failed to grant free credits on subscription creation: %v", err)
			return err
		}
	}

	return nil
}

// handleSubscriptionUpdated handles when a DNS delegation subscription is updated
func (h *stripeWebhookProcessor) handleSubscriptionUpdated(subscription *stripe.Subscription, rawData []byte) error {
	subscriptionID := subscription.ID

	// Revoke API keys if subscription is cancelled or past due
	if subscription.Status == "canceled" || subscription.Status == "unpaid" || subscription.Status == "past_due" {
		if err := database.RevokeDNSDelegationAPIKeysForSubscriptionTx(h.db, subscriptionID); err != nil {
			return fmt.Errorf("revoke DNS delegation API keys for subscription: %w", err)
		}
		log.Printf("[Stripe Webhook] Revoked DNS delegation API keys for subscription %s (status: %s)", subscriptionID, subscription.Status)
		return nil
	}

	// Reactivate API keys if subscription becomes active again
	if subscription.Status == "active" || subscription.Status == "trialing" {
		// Get organization from customer
		var orgID string
		var customerID string

		if subscription.Customer != nil {
			customerID = subscription.Customer.ID
			if orgIDVal, ok := subscription.Customer.Metadata["organization_id"]; ok {
				orgID = orgIDVal
			}
		}

		// If Customer wasn't expanded, try to get customer ID from raw JSON
		if customerID == "" {
			var rawSub map[string]interface{}
			if err := json.Unmarshal(rawData, &rawSub); err == nil {
				if cust, ok := rawSub["customer"].(string); ok {
					customerID = cust
				}
			}
		}

		if orgID == "" && customerID != "" {
			var billingAccount database.BillingAccount
			if err := h.db.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
				orgID = billingAccount.OrganizationID
			}
		}

		if orgID != "" {
			// Serialize subscription side effects with other organization payments.
			var org database.Organization
			if err := h.db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&org, "id = ?", orgID).Error; err != nil {
				return err
			}
			// Check if organization already has an active API key
			existingKey, err := database.GetActiveDNSDelegationAPIKeyForOrganizationTx(h.db, orgID)
			if err != nil || existingKey == nil {
				// Create new API key if none exists
				description := fmt.Sprintf("DNS Delegation API Key (Subscription: %s)", subscriptionID)
				_, err := database.CreateDNSDelegationAPIKeyTx(h.db, description, "", orgID, &subscriptionID)
				if err != nil {
					log.Printf("[Stripe Webhook] Failed to create API key for reactivated subscription: %v", err)
					return err
				} else {
					log.Printf("[Stripe Webhook] Created DNS delegation API key for reactivated subscription %s (org: %s)", subscriptionID, orgID)
				}
			}

			// Grant monthly free credits from plan when subscription becomes active again
			if err := h.grantMonthlyFreeCreditsFromPlan(orgID, subscriptionID); err != nil {
				log.Printf("[Stripe Webhook] Failed to grant free credits on subscription update: %v", err)
				return err
			}
		}
	}

	return nil
}

// handleSubscriptionDeleted handles when a DNS delegation subscription is deleted
func (h *stripeWebhookProcessor) handleSubscriptionDeleted(subscription *stripe.Subscription) error {
	subscriptionID := subscription.ID

	// Revoke all API keys for this subscription
	if err := database.RevokeDNSDelegationAPIKeysForSubscriptionTx(h.db, subscriptionID); err != nil {
		return fmt.Errorf("revoke DNS delegation API keys for subscription: %w", err)
	}

	log.Printf("[Stripe Webhook] Revoked DNS delegation API keys for deleted subscription %s", subscriptionID)
	return nil
}

func (h *stripeWebhookProcessor) handlePaymentIntentSucceeded(paymentIntent *stripe.PaymentIntent) {
	// Payment intents are handled via checkout.session.completed
	// This handler is here for completeness but typically won't add credits
	// as checkout.session.completed already handles it
	log.Printf("[Stripe Webhook] payment_intent.succeeded: %s", paymentIntent.ID)
}

func (h *stripeWebhookProcessor) handlePaymentIntentFailed(paymentIntent *stripe.PaymentIntent) {
	log.Printf("[Stripe Webhook] payment_intent.payment_failed: %s", paymentIntent.ID)
	// Could send notification to user, update billing account status, etc.
}

// handleInvoicePaid handles when an invoice is paid
func (h *stripeWebhookProcessor) handleInvoicePaid(invoice *stripe.Invoice, rawData []byte) error {
	// Get customer ID
	var customerID string
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	} else {
		// Try to get from raw JSON if Customer wasn't expanded
		var rawInvoice map[string]interface{}
		if err := json.Unmarshal(rawData, &rawInvoice); err == nil {
			if cust, ok := rawInvoice["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if customerID == "" {
		return fmt.Errorf("missing customer in invoice")
	}

	// Find organization by Stripe customer ID
	var billingAccount database.BillingAccount
	if err := h.db.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[Stripe Webhook] No billing account found for customer %s (invoice %s)", customerID, invoice.ID)
			return nil // Don't fail, just log - invoice might be for a different system
		}
		return fmt.Errorf("find billing account: %w", err)
	}

	// Only create transaction if this invoice actually resulted in credits being added
	// Check if invoice amount is positive and invoice is paid
	if invoice.AmountPaid <= 0 {
		log.Printf("[Stripe Webhook] Invoice %s has no amount paid, skipping transaction", invoice.ID)
		return nil
	}

	var org database.Organization
	if err := h.db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&org, "id = ?", billingAccount.OrganizationID).Error; err != nil {
		return err
	}
	reference := "invoice:" + invoice.ID
	if invoice.ID == "" {
		return fmt.Errorf("missing invoice id")
	}
	note := "Payment via Stripe Invoice " + invoice.ID
	if duplicate, err := recordedStripePayment(h.db, billingAccount.OrganizationID, reference, note); err != nil || duplicate {
		return err
	}
	transaction := &database.CreditTransaction{ID: generateID("ct"), OrganizationID: billingAccount.OrganizationID, ExternalReference: &reference, AmountCents: invoice.AmountPaid, Type: "payment", Source: "stripe", Note: &note, CreatedAt: time.Now()}

	// Only add credits if this is a credit purchase invoice (not a subscription invoice)
	// Check invoice metadata or description to determine if it's a credit purchase
	isCreditPurchase := false
	if invoice.Metadata != nil {
		if orgID, ok := invoice.Metadata["organization_id"]; ok && orgID == billingAccount.OrganizationID {
			isCreditPurchase = true
		}
	}
	// Also check if invoice description mentions credits
	if !isCreditPurchase && invoice.Description != "" {
		if strings.Contains(strings.ToLower(invoice.Description), "credit") {
			isCreditPurchase = true
		}
	}

	if isCreditPurchase {
		// Update credits and total paid, then check for plan upgrade
		oldBalance := org.Credits
		org.Credits += invoice.AmountPaid

		// Update total paid for safety check/auto-upgrade
		org.TotalPaidCents += invoice.AmountPaid

		if err := h.db.Model(&org).Select("credits", "total_paid_cents").Updates(&org).Error; err != nil {
			return fmt.Errorf("update credits: %w", err)
		}
		transaction.BalanceAfter = org.Credits
		log.Printf("[Stripe Webhook] Added %d cents to organization %s from invoice %s (balance: %d -> %d, total paid: %d)",
			invoice.AmountPaid, billingAccount.OrganizationID, invoice.ID, oldBalance, org.Credits, org.TotalPaidCents)

		// Check and upgrade plan if eligible
		if err := checkAndUpgradePlan(billingAccount.OrganizationID, h.db); err != nil {
			log.Printf("[Stripe Webhook] Warning: failed to check/upgrade plan: %v", err)
			return err
		}
	} else {
		// For non-credit invoices (like subscriptions), update total paid and check for upgrade
		// This handles the case where users pay for usage directly (safety check)
		org.TotalPaidCents += invoice.AmountPaid
		if err := h.db.Model(&org).Select("credits", "total_paid_cents").Updates(&org).Error; err != nil {
			return fmt.Errorf("update total paid: %w", err)
		}
		transaction.BalanceAfter = org.Credits
		log.Printf("[Stripe Webhook] Recorded payment transaction for invoice %s (subscription/service payment, total paid: %d)",
			invoice.ID, org.TotalPaidCents)

		// Check and upgrade plan if eligible
		if err := checkAndUpgradePlan(billingAccount.OrganizationID, h.db); err != nil {
			log.Printf("[Stripe Webhook] Warning: failed to check/upgrade plan: %v", err)
			return err
		}
	}

	// Create transaction record
	if err := h.db.Create(transaction).Error; err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	return nil
}

// handleInvoicePaymentFailed handles when an invoice payment fails
func (h *stripeWebhookProcessor) handleInvoicePaymentFailed(invoice *stripe.Invoice, rawData []byte) error {
	// Get customer ID
	var customerID string
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	} else {
		var rawInvoice map[string]interface{}
		if err := json.Unmarshal(rawData, &rawInvoice); err == nil {
			if cust, ok := rawInvoice["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if customerID == "" {
		return fmt.Errorf("missing customer in invoice")
	}

	// Find organization by Stripe customer ID
	var billingAccount database.BillingAccount
	if err := h.db.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[Stripe Webhook] No billing account found for customer %s (invoice %s)", customerID, invoice.ID)
			return nil
		}
		return fmt.Errorf("find billing account: %w", err)
	}

	// Update billing account status if needed
	billingAccount.Status = "PAYMENT_FAILED"
	billingAccount.UpdatedAt = time.Now()
	if err := h.db.Save(&billingAccount).Error; err != nil {
		log.Printf("[Stripe Webhook] Failed to update billing account status: %v", err)
		return err
	}

	log.Printf("[Stripe Webhook] Invoice payment failed for organization %s (invoice %s, amount: %d cents)",
		billingAccount.OrganizationID, invoice.ID, invoice.AmountDue)

	// TODO: Could send notification email to user about failed payment
	return nil
}

// handleInvoicePaymentActionRequired handles when payment requires user action (e.g., 3D Secure)
func (h *stripeWebhookProcessor) handleInvoicePaymentActionRequired(invoice *stripe.Invoice, rawData []byte) {
	// Get customer ID
	var customerID string
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	} else {
		var rawInvoice map[string]interface{}
		if err := json.Unmarshal(rawData, &rawInvoice); err == nil {
			if cust, ok := rawInvoice["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if customerID == "" {
		log.Printf("[Stripe Webhook] Missing customer in invoice.payment_action_required (invoice %s)", invoice.ID)
		return
	}

	log.Printf("[Stripe Webhook] Payment action required for customer %s (invoice %s)", customerID, invoice.ID)
	// TODO: Could send notification email to user with payment link
}

// handleInvoiceFinalizationFailed handles when invoice finalization fails
func (h *stripeWebhookProcessor) handleInvoiceFinalizationFailed(invoice *stripe.Invoice, rawData []byte) {
	// Get customer ID
	var customerID string
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	} else {
		var rawInvoice map[string]interface{}
		if err := json.Unmarshal(rawData, &rawInvoice); err == nil {
			if cust, ok := rawInvoice["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if customerID == "" {
		log.Printf("[Stripe Webhook] Missing customer in invoice.finalization_failed (invoice %s)", invoice.ID)
		return
	}

	log.Printf("[Stripe Webhook] Invoice finalization failed for customer %s (invoice %s)", customerID, invoice.ID)
	// TODO: Could send notification email to admin about finalization failure
}

// handleCheckoutSessionExpired handles when a checkout session expires
func (h *stripeWebhookProcessor) handleCheckoutSessionExpired(session *stripe.CheckoutSession, rawData []byte) {
	// Extract organization ID from metadata if available
	orgID, ok := session.Metadata["organization_id"]
	if !ok {
		log.Printf("[Stripe Webhook] Checkout session expired (session %s) - no organization_id in metadata", session.ID)
		return
	}

	log.Printf("[Stripe Webhook] Checkout session expired for organization %s (session %s)", orgID, session.ID)
	// TODO: Could send notification email to user that their checkout session expired
}

// handleCheckoutSessionAsyncPaymentSucceeded handles when an async payment (e.g., bank transfer) succeeds
func (h *stripeWebhookProcessor) handleCheckoutSessionAsyncPaymentSucceeded(session *stripe.CheckoutSession, rawData []byte) error {
	return h.handleCheckoutSessionCompleted(session, rawData)
}

// handleCheckoutSessionAsyncPaymentFailed handles when an async payment fails
func (h *stripeWebhookProcessor) handleCheckoutSessionAsyncPaymentFailed(session *stripe.CheckoutSession, rawData []byte) {
	// Extract organization ID from metadata if available
	orgID, ok := session.Metadata["organization_id"]
	if !ok {
		log.Printf("[Stripe Webhook] Async payment failed (session %s) - no organization_id in metadata", session.ID)
		return
	}

	log.Printf("[Stripe Webhook] Async payment failed for organization %s (session %s)", orgID, session.ID)
	// TODO: Could send notification email to user about failed async payment
}

// handleSubscriptionTrialWillEnd handles when a subscription trial is about to end
func (h *stripeWebhookProcessor) handleSubscriptionTrialWillEnd(subscription *stripe.Subscription, rawData []byte) {
	// Get organization from customer
	var orgID string
	var customerID string

	if subscription.Customer != nil {
		customerID = subscription.Customer.ID
		if orgIDVal, ok := subscription.Customer.Metadata["organization_id"]; ok {
			orgID = orgIDVal
		}
	}

	if customerID == "" {
		var rawSub map[string]interface{}
		if err := json.Unmarshal(rawData, &rawSub); err == nil {
			if cust, ok := rawSub["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if orgID == "" && customerID != "" {
		var billingAccount database.BillingAccount
		if err := h.db.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
			orgID = billingAccount.OrganizationID
		}
	}

	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for subscription trial_will_end %s", subscription.ID)
		return
	}

	// Get trial end date
	trialEnd := time.Unix(subscription.TrialEnd, 0)
	daysUntilEnd := int(time.Until(trialEnd).Hours() / 24)

	log.Printf("[Stripe Webhook] Subscription trial will end for organization %s (subscription %s, ends in %d days)",
		orgID, subscription.ID, daysUntilEnd)
	// TODO: Could send notification email to user about trial ending
}

// grantMonthlyFreeCreditsFromPlan grants monthly free credits to an organization based on their plan
// This is called when a subscription becomes active to grant the plan's monthly free credits
func (h *stripeWebhookProcessor) grantMonthlyFreeCreditsFromPlan(orgID string, subscriptionID string) error {
	// Get organization's quota to find their plan
	var quota database.OrgQuota
	if err := h.db.Where("organization_id = ?", orgID).First(&quota).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[Stripe Webhook] No quota found for organization %s, skipping free credits grant", orgID)
			return nil
		}
		return fmt.Errorf("get quota: %w", err)
	}

	if quota.PlanID == "" {
		log.Printf("[Stripe Webhook] Organization %s has no plan assigned, skipping free credits grant", orgID)
		return nil
	}

	// Get the plan
	var plan database.OrganizationPlan
	if err := h.db.First(&plan, "id = ?", quota.PlanID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[Stripe Webhook] Plan %s not found, skipping free credits grant", quota.PlanID)
			return nil
		}
		return fmt.Errorf("get plan: %w", err)
	}

	// Skip if plan has no monthly free credits
	if plan.MonthlyFreeCreditsCents <= 0 {
		return nil
	}

	// Grant credits in a transaction
	return h.db.Transaction(func(tx *gorm.DB) error {
		// Get organization
		var org database.Organization
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&org, "id = ?", orgID).Error; err != nil {
			return fmt.Errorf("organization not found: %w", err)
		}

		// Check if we've already granted credits this month
		now := time.Now()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

		var existingGrant database.MonthlyCreditGrant
		if err := tx.Where("organization_id = ? AND plan_id = ? AND grant_month = ?",
			orgID, plan.ID, monthStart).First(&existingGrant).Error; err == nil {
			log.Printf("[Stripe Webhook] Credits already granted to org %s for %s, skipping", orgID, monthStart.Format("2006-01"))
			return nil // Already granted this month
		}

		// Grant credits
		oldBalance := org.Credits
		org.Credits += plan.MonthlyFreeCreditsCents
		if err := tx.Model(&org).Select("credits", "total_paid_cents").Updates(&org).Error; err != nil {
			return fmt.Errorf("update credits: %w", err)
		}

		// Record credit transaction
		monthStr := monthStart.Format("2006-01")
		note := fmt.Sprintf("Monthly free credits for %s (plan: %s, subscription: %s)", monthStr, plan.Name, subscriptionID)
		transactionID := generateID("ct")
		transaction := &database.CreditTransaction{
			ID:             transactionID,
			OrganizationID: orgID,
			AmountCents:    plan.MonthlyFreeCreditsCents,
			BalanceAfter:   org.Credits,
			Type:           "admin_add",
			Source:         "stripe",
			Note:           &note,
			CreatedAt:      time.Now(),
		}
		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		// Record grant in tracking table
		grant := &database.MonthlyCreditGrant{
			OrganizationID: orgID,
			PlanID:         plan.ID,
			GrantMonth:     monthStart,
			AmountCents:    plan.MonthlyFreeCreditsCents,
			GrantedAt:      time.Now(),
			CreatedAt:      time.Now(),
		}
		if err := tx.Create(grant).Error; err != nil {
			return fmt.Errorf("create grant record: %w", err)
		}

		log.Printf("[Stripe Webhook] Granted %d cents monthly free credits to org %s (plan: %s, subscription: %s, balance: %d -> %d)",
			plan.MonthlyFreeCreditsCents, orgID, plan.Name, subscriptionID, oldBalance, org.Credits)

		return nil
	})
}

// Helper function to get organization ID from subscription
func (h *stripeWebhookProcessor) getOrgIDFromSubscription(subscription *stripe.Subscription, rawData []byte) (string, string) {
	var orgID string
	var customerID string

	if subscription.Customer != nil {
		customerID = subscription.Customer.ID
		if orgIDVal, ok := subscription.Customer.Metadata["organization_id"]; ok {
			orgID = orgIDVal
		}
	}

	if customerID == "" {
		var rawSub map[string]interface{}
		if err := json.Unmarshal(rawData, &rawSub); err == nil {
			if cust, ok := rawSub["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if orgID == "" && customerID != "" {
		var billingAccount database.BillingAccount
		if err := h.db.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
			orgID = billingAccount.OrganizationID
		}
	}

	return orgID, customerID
}

// handleSubscriptionPaused handles when a subscription is paused
func (h *stripeWebhookProcessor) handleSubscriptionPaused(subscription *stripe.Subscription, rawData []byte) {
	orgID, _ := h.getOrgIDFromSubscription(subscription, rawData)
	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for paused subscription %s", subscription.ID)
		return
	}

	log.Printf("[Stripe Webhook] Subscription paused for organization %s (subscription %s)", orgID, subscription.ID)
	// TODO: Could pause services or send notification
}

// handleSubscriptionResumed handles when a subscription is resumed
func (h *stripeWebhookProcessor) handleSubscriptionResumed(subscription *stripe.Subscription, rawData []byte) error {
	orgID, _ := h.getOrgIDFromSubscription(subscription, rawData)
	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for resumed subscription %s", subscription.ID)
		return fmt.Errorf("organization not found for subscription")
	}

	log.Printf("[Stripe Webhook] Subscription resumed for organization %s (subscription %s)", orgID, subscription.ID)

	// Grant monthly free credits if subscription is active
	if subscription.Status == "active" || subscription.Status == "trialing" {
		if err := h.grantMonthlyFreeCreditsFromPlan(orgID, subscription.ID); err != nil {
			log.Printf("[Stripe Webhook] Failed to grant free credits on resume: %v", err)
			return err
		}
	}
	return nil
}

// handleSubscriptionPendingUpdateApplied handles when a pending subscription update is applied
func (h *stripeWebhookProcessor) handleSubscriptionPendingUpdateApplied(subscription *stripe.Subscription, rawData []byte) {
	orgID, _ := h.getOrgIDFromSubscription(subscription, rawData)
	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for subscription pending_update_applied %s", subscription.ID)
		return
	}

	log.Printf("[Stripe Webhook] Subscription pending update applied for organization %s (subscription %s)", orgID, subscription.ID)
}

// handleSubscriptionPendingUpdateExpired handles when a pending subscription update expires
func (h *stripeWebhookProcessor) handleSubscriptionPendingUpdateExpired(subscription *stripe.Subscription, rawData []byte) {
	orgID, _ := h.getOrgIDFromSubscription(subscription, rawData)
	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for subscription pending_update_expired %s", subscription.ID)
		return
	}

	log.Printf("[Stripe Webhook] Subscription pending update expired for organization %s (subscription %s)", orgID, subscription.ID)
}

// Invoice event handlers
func (h *stripeWebhookProcessor) handleInvoiceCreated(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice created: %s (amount: %d cents)", invoice.ID, invoice.AmountDue)
	// Invoice is created, will be finalized and paid later
}

func (h *stripeWebhookProcessor) handleInvoiceDeleted(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice deleted: %s", invoice.ID)
	// Invoice was deleted, typically means it was a draft
}

func (h *stripeWebhookProcessor) handleInvoiceFinalized(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice finalized: %s (amount: %d cents)", invoice.ID, invoice.AmountDue)
	// Invoice is finalized and ready for payment
}

func (h *stripeWebhookProcessor) handleInvoiceMarkedUncollectible(invoice *stripe.Invoice, rawData []byte) error {
	var customerID string
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	} else {
		var rawInvoice map[string]interface{}
		if err := json.Unmarshal(rawData, &rawInvoice); err == nil {
			if cust, ok := rawInvoice["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if customerID != "" {
		var billingAccount database.BillingAccount
		if err := h.db.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
			billingAccount.Status = "UNCOLLECTIBLE"
			billingAccount.UpdatedAt = time.Now()
			if err := h.db.Save(&billingAccount).Error; err != nil {
				return err
			}
			log.Printf("[Stripe Webhook] Invoice marked uncollectible for organization %s (invoice %s)",
				billingAccount.OrganizationID, invoice.ID)
		}
	}
	return nil
}

func (h *stripeWebhookProcessor) handleInvoiceOverdue(invoice *stripe.Invoice, rawData []byte) {
	var customerID string
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	} else {
		var rawInvoice map[string]interface{}
		if err := json.Unmarshal(rawData, &rawInvoice); err == nil {
			if cust, ok := rawInvoice["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if customerID != "" {
		log.Printf("[Stripe Webhook] Invoice overdue for customer %s (invoice %s, amount: %d cents)",
			customerID, invoice.ID, invoice.AmountDue)
		// TODO: Could send notification email to user
	}
}

func (h *stripeWebhookProcessor) handleInvoiceOverpaid(invoice *stripe.Invoice, rawData []byte) {
	var customerID string
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	} else {
		var rawInvoice map[string]interface{}
		if err := json.Unmarshal(rawData, &rawInvoice); err == nil {
			if cust, ok := rawInvoice["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if customerID != "" {
		log.Printf("[Stripe Webhook] Invoice overpaid for customer %s (invoice %s, overpaid: %d cents)",
			customerID, invoice.ID, invoice.AmountRemaining)
		// TODO: Could add overpaid amount as credits or refund
	}
}

func (h *stripeWebhookProcessor) handleInvoicePaymentSucceeded(invoice *stripe.Invoice, rawData []byte) {
	// This is similar to invoice.paid but fires for all successful payments
	// We already handle invoice.paid, so this is mainly for logging
	var customerID string
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	} else {
		var rawInvoice map[string]interface{}
		if err := json.Unmarshal(rawData, &rawInvoice); err == nil {
			if cust, ok := rawInvoice["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	log.Printf("[Stripe Webhook] Invoice payment succeeded for customer %s (invoice %s, amount: %d cents)",
		customerID, invoice.ID, invoice.AmountPaid)
}

func (h *stripeWebhookProcessor) handleInvoiceSent(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice sent: %s", invoice.ID)
	// Invoice was sent to customer
}

func (h *stripeWebhookProcessor) handleInvoiceUpcoming(invoice *stripe.Invoice, rawData []byte) {
	var customerID string
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	} else {
		var rawInvoice map[string]interface{}
		if err := json.Unmarshal(rawData, &rawInvoice); err == nil {
			if cust, ok := rawInvoice["customer"].(string); ok {
				customerID = cust
			}
		}
	}

	if customerID != "" {
		log.Printf("[Stripe Webhook] Invoice upcoming for customer %s (invoice %s, amount: %d cents)",
			customerID, invoice.ID, invoice.AmountDue)
		// TODO: Could send notification email to user about upcoming invoice
	}
}

func (h *stripeWebhookProcessor) handleInvoiceUpdated(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice updated: %s", invoice.ID)
	// Invoice was updated (amount, status, etc.)
}

func (h *stripeWebhookProcessor) handleInvoiceVoided(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice voided: %s", invoice.ID)
	// Invoice was voided and will not be paid
}

// Subscription schedule event handlers
func (h *stripeWebhookProcessor) handleSubscriptionScheduleAborted(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule aborted: %s", schedule.ID)
}

func (h *stripeWebhookProcessor) handleSubscriptionScheduleCanceled(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule canceled: %s", schedule.ID)
}

func (h *stripeWebhookProcessor) handleSubscriptionScheduleCompleted(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule completed: %s", schedule.ID)
}

func (h *stripeWebhookProcessor) handleSubscriptionScheduleCreated(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule created: %s", schedule.ID)
}

func (h *stripeWebhookProcessor) handleSubscriptionScheduleExpiring(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule expiring: %s", schedule.ID)
}

func (h *stripeWebhookProcessor) handleSubscriptionScheduleReleased(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule released: %s", schedule.ID)
}

func (h *stripeWebhookProcessor) handleSubscriptionScheduleUpdated(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule updated: %s", schedule.ID)
}

// extractEventIDs extracts organization, customer, subscription, invoice, and checkout session IDs from webhook event data
func (h *stripeWebhookProcessor) extractEventIDs(eventType string, rawData []byte) (*string, *string, *string, *string, *string) {
	var orgID, customerID, subscriptionID, invoiceID, checkoutSessionID *string

	// Parse raw JSON to extract IDs
	var eventData map[string]interface{}
	if err := json.Unmarshal(rawData, &eventData); err != nil {
		return nil, nil, nil, nil, nil
	}

	// Extract customer ID (common across many event types)
	if cust, ok := eventData["customer"].(string); ok {
		customerID = &cust
	} else if custObj, ok := eventData["customer"].(map[string]interface{}); ok {
		if custID, ok := custObj["id"].(string); ok {
			customerID = &custID
			// Try to get organization_id from customer metadata
			if metadata, ok := custObj["metadata"].(map[string]interface{}); ok {
				if orgIDVal, ok := metadata["organization_id"].(string); ok {
					orgID = &orgIDVal
				}
			}
		}
	}

	// Extract IDs based on event type
	switch {
	case strings.HasPrefix(eventType, "checkout.session."):
		if sessionID, ok := eventData["id"].(string); ok {
			checkoutSessionID = &sessionID
		}
		// Try to get organization_id from metadata
		if metadata, ok := eventData["metadata"].(map[string]interface{}); ok {
			if orgIDVal, ok := metadata["organization_id"].(string); ok {
				orgID = &orgIDVal
			}
		}

	case strings.HasPrefix(eventType, "customer.subscription."):
		if subID, ok := eventData["id"].(string); ok {
			subscriptionID = &subID
		}
		// Try to get organization from customer if not already found
		if orgID == nil && customerID != nil {
			var billingAccount database.BillingAccount
			if err := h.db.Where("stripe_customer_id = ?", *customerID).First(&billingAccount).Error; err == nil {
				orgID = &billingAccount.OrganizationID
			}
		}

	case strings.HasPrefix(eventType, "invoice."):
		if invID, ok := eventData["id"].(string); ok {
			invoiceID = &invID
		}
		// Try to get organization from customer if not already found
		if orgID == nil && customerID != nil {
			var billingAccount database.BillingAccount
			if err := h.db.Where("stripe_customer_id = ?", *customerID).First(&billingAccount).Error; err == nil {
				orgID = &billingAccount.OrganizationID
			}
		}

	case strings.HasPrefix(eventType, "payment_intent."):
		// Try to get organization from customer if available
		if orgID == nil && customerID != nil {
			var billingAccount database.BillingAccount
			if err := h.db.Where("stripe_customer_id = ?", *customerID).First(&billingAccount).Error; err == nil {
				orgID = &billingAccount.OrganizationID
			}
		}
	}

	return orgID, customerID, subscriptionID, invoiceID, checkoutSessionID
}

func (h *stripeWebhookProcessor) addCreditsFromPayment(orgID string, amountCents int64, sessionID, customerID string) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		// Get organization
		var org database.Organization
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&org, "id = ?", orgID).Error; err != nil {
			return fmt.Errorf("organization not found: %w", err)
		}

		reference := "checkout:" + sessionID
		if sessionID == "" || amountCents <= 0 {
			return fmt.Errorf("invalid checkout payment")
		}
		note := "Payment via Stripe Checkout Session " + sessionID
		if duplicate, err := recordedStripePayment(tx, orgID, reference, note); err != nil || duplicate {
			return err
		}

		// Update credits and total paid
		org.Credits += amountCents
		org.TotalPaidCents += amountCents
		if err := tx.Model(&org).Select("credits", "total_paid_cents").Updates(&org).Error; err != nil {
			return fmt.Errorf("update credits: %w", err)
		}

		// Record transaction
		transaction := &database.CreditTransaction{
			ID:                generateID("ct"),
			OrganizationID:    orgID,
			ExternalReference: &reference,
			AmountCents:       amountCents,
			BalanceAfter:      org.Credits,
			Type:              "payment",
			Source:            "stripe",
			Note:              &note,
			CreatedAt:         time.Now(),
		}
		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		// Check and upgrade plan if eligible (after credits are added)
		if err := checkAndUpgradePlan(orgID, tx); err != nil {
			log.Printf("[Stripe Webhook] Warning: failed to check/upgrade plan: %v", err)
			return err
		}

		// Update billing account with customer ID if not set
		var billingAccount database.BillingAccount
		if err := tx.Where("organization_id = ?", orgID).First(&billingAccount).Error; err == nil {
			if billingAccount.StripeCustomerID == nil || *billingAccount.StripeCustomerID == "" {
				billingAccount.StripeCustomerID = &customerID
				billingAccount.Status = "ACTIVE"
				billingAccount.UpdatedAt = time.Now()
				if err := tx.Save(&billingAccount).Error; err != nil {
					log.Printf("[Stripe Webhook] Failed to update billing account: %v", err)
					return err
				}
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create billing account if it doesn't exist
			billingAccount = database.BillingAccount{
				ID:               generateID("ba"),
				OrganizationID:   orgID,
				StripeCustomerID: &customerID,
				Status:           "ACTIVE",
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			if err := tx.Create(&billingAccount).Error; err != nil {
				log.Printf("[Stripe Webhook] Failed to create billing account: %v", err)
				return err
			}
		}

		return nil
	})

}

// Exact legacy notes keep pre-migration payments idempotent. New payments use a
// unique external reference instead of SQL LIKE matching on human-readable text.
func recordedStripePayment(tx *gorm.DB, orgID, reference, legacyNote string) (bool, error) {
	var transaction database.CreditTransaction
	err := tx.Where("external_reference = ? OR (organization_id = ? AND source = ? AND type = ? AND note = ?)", reference, orgID, "stripe", "payment", legacyNote).First(&transaction).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if transaction.OrganizationID != orgID {
		return false, fmt.Errorf("payment already belongs to another organization")
	}
	return true, nil
}
