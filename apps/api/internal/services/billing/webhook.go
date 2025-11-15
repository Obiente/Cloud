package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"api/internal/database"
	"api/internal/services/organizations"

	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/webhook"
	"gorm.io/gorm"
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

	// Check if event has already been processed (idempotency)
	var existingEvent database.StripeWebhookEvent
	if err := database.DB.Where("id = ?", event.ID).First(&existingEvent).Error; err == nil {
		log.Printf("[Stripe Webhook] Event %s (type: %s) already processed at %s, skipping", 
			event.ID, event.Type, existingEvent.ProcessedAt.Format(time.RFC3339))
		w.WriteHeader(http.StatusOK) // Return 200 to stop Stripe from retrying
		return
	}

	// Extract organization/customer info from event data for tracking
	orgID, customerID, subscriptionID, invoiceID, checkoutSessionID := extractEventIDs(string(event.Type), event.Data.Raw)
	
	// Mark event as being processed (with a small delay to handle race conditions)
	// Use a transaction to ensure atomicity
	var processedEvent database.StripeWebhookEvent
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Double-check in transaction
		if err := tx.Where("id = ?", event.ID).First(&existingEvent).Error; err == nil {
			return fmt.Errorf("event already processed")
		}
		
		// Create record with extracted IDs
		processedEvent = database.StripeWebhookEvent{
			ID:               event.ID,
			EventType:        string(event.Type),
			ProcessedAt:      time.Now(),
			CreatedAt:        time.Now(),
			OrganizationID:   orgID,
			CustomerID:       customerID,
			SubscriptionID:   subscriptionID,
			InvoiceID:        invoiceID,
			CheckoutSessionID: checkoutSessionID,
		}
		return tx.Create(&processedEvent).Error
	})
	
	if err != nil {
		// Event was already processed (race condition)
		if err.Error() == "event already processed" {
			log.Printf("[Stripe Webhook] Event %s already processed (race condition), skipping", event.ID)
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Printf("[Stripe Webhook] Error recording event %s: %v", event.ID, err)
		// Continue processing - we'll try to be idempotent in handlers
	}

	// Handle the event
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("[Stripe Webhook] Error parsing checkout.session.completed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := handleCheckoutSessionCompleted(&session, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling checkout.session.completed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case "customer.subscription.created":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.created: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := handleSubscriptionCreated(&subscription, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling customer.subscription.created: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case "customer.subscription.updated":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.updated: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := handleSubscriptionUpdated(&subscription, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling customer.subscription.updated: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.deleted: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := handleSubscriptionDeleted(&subscription); err != nil {
			log.Printf("[Stripe Webhook] Error handling customer.subscription.deleted: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			log.Printf("[Stripe Webhook] Error parsing payment_intent.succeeded: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handlePaymentIntentSucceeded(&paymentIntent)

	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			log.Printf("[Stripe Webhook] Error parsing payment_intent.payment_failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handlePaymentIntentFailed(&paymentIntent)

	case "invoice.paid":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.paid: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := handleInvoicePaid(&invoice, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling invoice.paid: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.payment_failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := handleInvoicePaymentFailed(&invoice, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling invoice.payment_failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case "invoice.payment_action_required":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.payment_action_required: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoicePaymentActionRequired(&invoice, event.Data.Raw)

	case "invoice.finalization_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.finalization_failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceFinalizationFailed(&invoice, event.Data.Raw)

	case "checkout.session.expired":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("[Stripe Webhook] Error parsing checkout.session.expired: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleCheckoutSessionExpired(&session, event.Data.Raw)

	case "checkout.session.async_payment_succeeded":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("[Stripe Webhook] Error parsing checkout.session.async_payment_succeeded: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := handleCheckoutSessionAsyncPaymentSucceeded(&session, event.Data.Raw); err != nil {
			log.Printf("[Stripe Webhook] Error handling checkout.session.async_payment_succeeded: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case "checkout.session.async_payment_failed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("[Stripe Webhook] Error parsing checkout.session.async_payment_failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleCheckoutSessionAsyncPaymentFailed(&session, event.Data.Raw)

	case "customer.subscription.trial_will_end":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.trial_will_end: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionTrialWillEnd(&subscription, event.Data.Raw)

	case "customer.subscription.paused":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.paused: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionPaused(&subscription, event.Data.Raw)

	case "customer.subscription.resumed":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.resumed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionResumed(&subscription, event.Data.Raw)

	case "customer.subscription.pending_update_applied":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.pending_update_applied: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionPendingUpdateApplied(&subscription, event.Data.Raw)

	case "customer.subscription.pending_update_expired":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			log.Printf("[Stripe Webhook] Error parsing customer.subscription.pending_update_expired: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionPendingUpdateExpired(&subscription, event.Data.Raw)

	case "invoice.created":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.created: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceCreated(&invoice, event.Data.Raw)

	case "invoice.deleted":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.deleted: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceDeleted(&invoice, event.Data.Raw)

	case "invoice.finalized":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.finalized: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceFinalized(&invoice, event.Data.Raw)

	case "invoice.marked_uncollectible":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.marked_uncollectible: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceMarkedUncollectible(&invoice, event.Data.Raw)

	case "invoice.overdue":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.overdue: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceOverdue(&invoice, event.Data.Raw)

	case "invoice.overpaid":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.overpaid: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceOverpaid(&invoice, event.Data.Raw)

	case "invoice.payment_succeeded":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.payment_succeeded: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoicePaymentSucceeded(&invoice, event.Data.Raw)

	case "invoice.sent":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.sent: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceSent(&invoice, event.Data.Raw)

	case "invoice.upcoming":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.upcoming: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceUpcoming(&invoice, event.Data.Raw)

	case "invoice.updated":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.updated: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceUpdated(&invoice, event.Data.Raw)

	case "invoice.voided":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			log.Printf("[Stripe Webhook] Error parsing invoice.voided: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleInvoiceVoided(&invoice, event.Data.Raw)

	case "subscription_schedule.aborted":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.aborted: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionScheduleAborted(&schedule, event.Data.Raw)

	case "subscription_schedule.canceled":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.canceled: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionScheduleCanceled(&schedule, event.Data.Raw)

	case "subscription_schedule.completed":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.completed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionScheduleCompleted(&schedule, event.Data.Raw)

	case "subscription_schedule.created":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.created: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionScheduleCreated(&schedule, event.Data.Raw)

	case "subscription_schedule.expiring":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.expiring: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionScheduleExpiring(&schedule, event.Data.Raw)

	case "subscription_schedule.released":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.released: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionScheduleReleased(&schedule, event.Data.Raw)

	case "subscription_schedule.updated":
		var schedule stripe.SubscriptionSchedule
		if err := json.Unmarshal(event.Data.Raw, &schedule); err != nil {
			log.Printf("[Stripe Webhook] Error parsing subscription_schedule.updated: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handleSubscriptionScheduleUpdated(&schedule, event.Data.Raw)

	default:
		log.Printf("[Stripe Webhook] Unhandled event type: %s (ID: %s)", event.Type, event.ID)
	}

	w.WriteHeader(http.StatusOK)
}

func handleCheckoutSessionCompleted(session *stripe.CheckoutSession, rawData []byte) error {
	// Extract organization ID from metadata
	orgID, ok := session.Metadata["organization_id"]
	if !ok {
		return fmt.Errorf("missing organization_id in metadata")
	}

	// Check if this is a subscription checkout (for DNS delegation)
	if session.Mode == stripe.CheckoutSessionModeSubscription {
		productType, isDNSDelegation := session.Metadata["product_type"]
		if isDNSDelegation && productType == "dns_delegation" {
			// Subscription checkout completed - subscription will be created separately
			// We'll handle API key creation in customer.subscription.created event
			log.Printf("[Stripe Webhook] DNS delegation subscription checkout completed for organization %s", orgID)
			return nil
		}
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
	if err := addCreditsFromPayment(orgID, amountCents, session.ID, customerID); err != nil {
		return fmt.Errorf("add credits from payment: %w", err)
	}

	log.Printf("[Stripe Webhook] Successfully added %d cents to organization %s from checkout session %s", amountCents, orgID, session.ID)
	return nil
}

// handleSubscriptionCreated handles when a DNS delegation subscription is created
func handleSubscriptionCreated(subscription *stripe.Subscription, rawData []byte) error {
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
		if err := database.DB.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
			orgID = billingAccount.OrganizationID
		}
	}

	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for subscription %s", subscription.ID)
		return nil // Don't fail, just log
	}

	// Check if this is a DNS delegation subscription (price amount is $2/month = 200 cents)
	isDNSDelegation := false
	if len(subscription.Items.Data) > 0 {
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
		// Check if organization already has an active API key
		existingKey, err := database.GetActiveDNSDelegationAPIKeyForOrganization(orgID)
		if err == nil && existingKey != nil {
			log.Printf("[Stripe Webhook] Organization %s already has an active API key, skipping creation", orgID)
		} else {
		subscriptionID := subscription.ID
		description := fmt.Sprintf("DNS Delegation API Key (Subscription: %s)", subscriptionID)
		sourceAPI := "" // Will be set by user when they configure their API

		apiKey, err := database.CreateDNSDelegationAPIKey(description, sourceAPI, orgID, &subscriptionID)
		if err != nil {
			return fmt.Errorf("create DNS delegation API key: %w", err)
		}

		log.Printf("[Stripe Webhook] Created DNS delegation API key for organization %s (subscription: %s)", orgID, subscriptionID)
		log.Printf("[Stripe Webhook] API Key: %s (save this securely!)", apiKey)
		}

		// Grant monthly free credits from plan when subscription becomes active
		if err := grantMonthlyFreeCreditsFromPlan(orgID, subscription.ID); err != nil {
			log.Printf("[Stripe Webhook] Failed to grant free credits on subscription creation: %v", err)
			// Don't fail the whole handler if free credits grant fails
		}
	}

	return nil
}

// handleSubscriptionUpdated handles when a DNS delegation subscription is updated
func handleSubscriptionUpdated(subscription *stripe.Subscription, rawData []byte) error {
	subscriptionID := subscription.ID

	// Revoke API keys if subscription is cancelled or past due
	if subscription.Status == "canceled" || subscription.Status == "unpaid" || subscription.Status == "past_due" {
		if err := database.RevokeDNSDelegationAPIKeysForSubscription(subscriptionID); err != nil {
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
			if err := database.DB.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
				orgID = billingAccount.OrganizationID
			}
		}

		if orgID != "" {
			// Check if organization already has an active API key
			existingKey, err := database.GetActiveDNSDelegationAPIKeyForOrganization(orgID)
			if err != nil || existingKey == nil {
				// Create new API key if none exists
				description := fmt.Sprintf("DNS Delegation API Key (Subscription: %s)", subscriptionID)
				apiKey, err := database.CreateDNSDelegationAPIKey(description, "", orgID, &subscriptionID)
				if err != nil {
					log.Printf("[Stripe Webhook] Failed to create API key for reactivated subscription: %v", err)
				} else {
					log.Printf("[Stripe Webhook] Created DNS delegation API key for reactivated subscription %s (org: %s)", subscriptionID, orgID)
					log.Printf("[Stripe Webhook] API Key: %s", apiKey)
				}
			}

			// Grant monthly free credits from plan when subscription becomes active again
			if err := grantMonthlyFreeCreditsFromPlan(orgID, subscriptionID); err != nil {
				log.Printf("[Stripe Webhook] Failed to grant free credits on subscription update: %v", err)
				// Don't fail the whole handler if free credits grant fails
			}
		}
	}

	return nil
}

// handleSubscriptionDeleted handles when a DNS delegation subscription is deleted
func handleSubscriptionDeleted(subscription *stripe.Subscription) error {
	subscriptionID := subscription.ID

	// Revoke all API keys for this subscription
	if err := database.RevokeDNSDelegationAPIKeysForSubscription(subscriptionID); err != nil {
		return fmt.Errorf("revoke DNS delegation API keys for subscription: %w", err)
	}

	log.Printf("[Stripe Webhook] Revoked DNS delegation API keys for deleted subscription %s", subscriptionID)
	return nil
}

func handlePaymentIntentSucceeded(paymentIntent *stripe.PaymentIntent) {
	// Payment intents are handled via checkout.session.completed
	// This handler is here for completeness but typically won't add credits
	// as checkout.session.completed already handles it
	log.Printf("[Stripe Webhook] payment_intent.succeeded: %s", paymentIntent.ID)
}

func handlePaymentIntentFailed(paymentIntent *stripe.PaymentIntent) {
	log.Printf("[Stripe Webhook] payment_intent.payment_failed: %s", paymentIntent.ID)
	// Could send notification to user, update billing account status, etc.
}

// handleInvoicePaid handles when an invoice is paid
func handleInvoicePaid(invoice *stripe.Invoice, rawData []byte) error {
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
	if err := database.DB.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err != nil {
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

	// Check if transaction already exists for this invoice
	var existingTransaction database.CreditTransaction
	notePattern := fmt.Sprintf("%%Invoice %s%%", invoice.ID)
	if err := database.DB.Where("organization_id = ? AND note LIKE ?", billingAccount.OrganizationID, notePattern).First(&existingTransaction).Error; err == nil {
		log.Printf("[Stripe Webhook] Transaction already exists for invoice %s", invoice.ID)
		return nil // Transaction already exists
	}

	// Create transaction for invoice payment
	note := fmt.Sprintf("Payment via Stripe Invoice %s", invoice.ID)
	transaction := &database.CreditTransaction{
		ID:             generateID("ct"),
		OrganizationID: billingAccount.OrganizationID,
		AmountCents:    invoice.AmountPaid,
		Type:           "payment",
		Source:         "stripe",
		Note:           &note,
		CreatedAt:      time.Now(),
	}

	// Get current organization credits to calculate balance after
	var org database.Organization
	if err := database.DB.First(&org, "id = ?", billingAccount.OrganizationID).Error; err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

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
		
		if err := database.DB.Save(&org).Error; err != nil {
			return fmt.Errorf("update credits: %w", err)
		}
		transaction.BalanceAfter = org.Credits
		log.Printf("[Stripe Webhook] Added %d cents to organization %s from invoice %s (balance: %d -> %d, total paid: %d)", 
			invoice.AmountPaid, billingAccount.OrganizationID, invoice.ID, oldBalance, org.Credits, org.TotalPaidCents)
		
		// Check and upgrade plan if eligible
		if err := checkAndUpgradePlan(billingAccount.OrganizationID, database.DB); err != nil {
			log.Printf("[Stripe Webhook] Warning: failed to check/upgrade plan: %v", err)
		}
	} else {
		// For non-credit invoices (like subscriptions), update total paid and check for upgrade
		// This handles the case where users pay for usage directly (safety check)
		org.TotalPaidCents += invoice.AmountPaid
		if err := database.DB.Save(&org).Error; err != nil {
			return fmt.Errorf("update total paid: %w", err)
		}
		transaction.BalanceAfter = org.Credits
		log.Printf("[Stripe Webhook] Recorded payment transaction for invoice %s (subscription/service payment, total paid: %d)", 
			invoice.ID, org.TotalPaidCents)
		
		// Check and upgrade plan if eligible
		if err := checkAndUpgradePlan(billingAccount.OrganizationID, database.DB); err != nil {
			log.Printf("[Stripe Webhook] Warning: failed to check/upgrade plan: %v", err)
		}
	}

	// Create transaction record
	if err := database.DB.Create(transaction).Error; err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	return nil
}

// handleInvoicePaymentFailed handles when an invoice payment fails
func handleInvoicePaymentFailed(invoice *stripe.Invoice, rawData []byte) error {
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
	if err := database.DB.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[Stripe Webhook] No billing account found for customer %s (invoice %s)", customerID, invoice.ID)
			return nil
		}
		return fmt.Errorf("find billing account: %w", err)
	}

	// Update billing account status if needed
	billingAccount.Status = "PAYMENT_FAILED"
	billingAccount.UpdatedAt = time.Now()
	if err := database.DB.Save(&billingAccount).Error; err != nil {
		log.Printf("[Stripe Webhook] Failed to update billing account status: %v", err)
	}

	log.Printf("[Stripe Webhook] Invoice payment failed for organization %s (invoice %s, amount: %d cents)", 
		billingAccount.OrganizationID, invoice.ID, invoice.AmountDue)
	
	// TODO: Could send notification email to user about failed payment
	return nil
}

// handleInvoicePaymentActionRequired handles when payment requires user action (e.g., 3D Secure)
func handleInvoicePaymentActionRequired(invoice *stripe.Invoice, rawData []byte) {
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
func handleInvoiceFinalizationFailed(invoice *stripe.Invoice, rawData []byte) {
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
func handleCheckoutSessionExpired(session *stripe.CheckoutSession, rawData []byte) {
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
func handleCheckoutSessionAsyncPaymentSucceeded(session *stripe.CheckoutSession, rawData []byte) error {
	// Extract organization ID from metadata
	orgID, ok := session.Metadata["organization_id"]
	if !ok {
		return fmt.Errorf("missing organization_id in metadata")
	}

	// Get payment intent amount from session
	amountCents := session.AmountTotal
	if amountCents <= 0 {
		return fmt.Errorf("invalid amount_total: %d", amountCents)
	}

	// Extract customer ID
	var customerID string
	if session.Customer != nil {
		customerID = session.Customer.ID
	} else {
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

	// Add credits to organization (same as regular checkout.session.completed)
	if err := addCreditsFromPayment(orgID, amountCents, session.ID, customerID); err != nil {
		return fmt.Errorf("add credits from payment: %w", err)
	}

	log.Printf("[Stripe Webhook] Successfully added %d cents to organization %s from async payment checkout session %s", 
		amountCents, orgID, session.ID)
	return nil
}

// handleCheckoutSessionAsyncPaymentFailed handles when an async payment fails
func handleCheckoutSessionAsyncPaymentFailed(session *stripe.CheckoutSession, rawData []byte) {
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
func handleSubscriptionTrialWillEnd(subscription *stripe.Subscription, rawData []byte) {
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
		if err := database.DB.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
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
func grantMonthlyFreeCreditsFromPlan(orgID string, subscriptionID string) error {
	// Get organization's quota to find their plan
	var quota database.OrgQuota
	if err := database.DB.Where("organization_id = ?", orgID).First(&quota).Error; err != nil {
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
	if err := database.DB.First(&plan, "id = ?", quota.PlanID).Error; err != nil {
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
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Get organization
		var org database.Organization
		if err := tx.First(&org, "id = ?", orgID).Error; err != nil {
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
		if err := tx.Save(&org).Error; err != nil {
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
func getOrgIDFromSubscription(subscription *stripe.Subscription, rawData []byte) (string, string) {
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
		if err := database.DB.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
			orgID = billingAccount.OrganizationID
		}
	}

	return orgID, customerID
}

// handleSubscriptionPaused handles when a subscription is paused
func handleSubscriptionPaused(subscription *stripe.Subscription, rawData []byte) {
	orgID, _ := getOrgIDFromSubscription(subscription, rawData)
	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for paused subscription %s", subscription.ID)
		return
	}

	log.Printf("[Stripe Webhook] Subscription paused for organization %s (subscription %s)", orgID, subscription.ID)
	// TODO: Could pause services or send notification
}

// handleSubscriptionResumed handles when a subscription is resumed
func handleSubscriptionResumed(subscription *stripe.Subscription, rawData []byte) {
	orgID, _ := getOrgIDFromSubscription(subscription, rawData)
	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for resumed subscription %s", subscription.ID)
		return
	}

	log.Printf("[Stripe Webhook] Subscription resumed for organization %s (subscription %s)", orgID, subscription.ID)
	
	// Grant monthly free credits if subscription is active
	if subscription.Status == "active" || subscription.Status == "trialing" {
		if err := grantMonthlyFreeCreditsFromPlan(orgID, subscription.ID); err != nil {
			log.Printf("[Stripe Webhook] Failed to grant free credits on resume: %v", err)
		}
	}
}

// handleSubscriptionPendingUpdateApplied handles when a pending subscription update is applied
func handleSubscriptionPendingUpdateApplied(subscription *stripe.Subscription, rawData []byte) {
	orgID, _ := getOrgIDFromSubscription(subscription, rawData)
	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for subscription pending_update_applied %s", subscription.ID)
		return
	}

	log.Printf("[Stripe Webhook] Subscription pending update applied for organization %s (subscription %s)", orgID, subscription.ID)
}

// handleSubscriptionPendingUpdateExpired handles when a pending subscription update expires
func handleSubscriptionPendingUpdateExpired(subscription *stripe.Subscription, rawData []byte) {
	orgID, _ := getOrgIDFromSubscription(subscription, rawData)
	if orgID == "" {
		log.Printf("[Stripe Webhook] Could not find organization for subscription pending_update_expired %s", subscription.ID)
		return
	}

	log.Printf("[Stripe Webhook] Subscription pending update expired for organization %s (subscription %s)", orgID, subscription.ID)
}

// Invoice event handlers
func handleInvoiceCreated(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice created: %s (amount: %d cents)", invoice.ID, invoice.AmountDue)
	// Invoice is created, will be finalized and paid later
}

func handleInvoiceDeleted(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice deleted: %s", invoice.ID)
	// Invoice was deleted, typically means it was a draft
}

func handleInvoiceFinalized(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice finalized: %s (amount: %d cents)", invoice.ID, invoice.AmountDue)
	// Invoice is finalized and ready for payment
}

func handleInvoiceMarkedUncollectible(invoice *stripe.Invoice, rawData []byte) {
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
		if err := database.DB.Where("stripe_customer_id = ?", customerID).First(&billingAccount).Error; err == nil {
			billingAccount.Status = "UNCOLLECTIBLE"
			billingAccount.UpdatedAt = time.Now()
			database.DB.Save(&billingAccount)
			log.Printf("[Stripe Webhook] Invoice marked uncollectible for organization %s (invoice %s)", 
				billingAccount.OrganizationID, invoice.ID)
		}
	}
}

func handleInvoiceOverdue(invoice *stripe.Invoice, rawData []byte) {
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

func handleInvoiceOverpaid(invoice *stripe.Invoice, rawData []byte) {
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

func handleInvoicePaymentSucceeded(invoice *stripe.Invoice, rawData []byte) {
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

func handleInvoiceSent(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice sent: %s", invoice.ID)
	// Invoice was sent to customer
}

func handleInvoiceUpcoming(invoice *stripe.Invoice, rawData []byte) {
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

func handleInvoiceUpdated(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice updated: %s", invoice.ID)
	// Invoice was updated (amount, status, etc.)
}

func handleInvoiceVoided(invoice *stripe.Invoice, rawData []byte) {
	log.Printf("[Stripe Webhook] Invoice voided: %s", invoice.ID)
	// Invoice was voided and will not be paid
}

// Subscription schedule event handlers
func handleSubscriptionScheduleAborted(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule aborted: %s", schedule.ID)
}

func handleSubscriptionScheduleCanceled(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule canceled: %s", schedule.ID)
}

func handleSubscriptionScheduleCompleted(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule completed: %s", schedule.ID)
}

func handleSubscriptionScheduleCreated(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule created: %s", schedule.ID)
}

func handleSubscriptionScheduleExpiring(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule expiring: %s", schedule.ID)
}

func handleSubscriptionScheduleReleased(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule released: %s", schedule.ID)
}

func handleSubscriptionScheduleUpdated(schedule *stripe.SubscriptionSchedule, rawData []byte) {
	log.Printf("[Stripe Webhook] Subscription schedule updated: %s", schedule.ID)
}

// extractEventIDs extracts organization, customer, subscription, invoice, and checkout session IDs from webhook event data
func extractEventIDs(eventType string, rawData []byte) (*string, *string, *string, *string, *string) {
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
			if err := database.DB.Where("stripe_customer_id = ?", *customerID).First(&billingAccount).Error; err == nil {
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
			if err := database.DB.Where("stripe_customer_id = ?", *customerID).First(&billingAccount).Error; err == nil {
				orgID = &billingAccount.OrganizationID
			}
		}
		
	case strings.HasPrefix(eventType, "payment_intent."):
		// Try to get organization from customer if available
		if orgID == nil && customerID != nil {
			var billingAccount database.BillingAccount
			if err := database.DB.Where("stripe_customer_id = ?", *customerID).First(&billingAccount).Error; err == nil {
				orgID = &billingAccount.OrganizationID
			}
		}
	}
	
	return orgID, customerID, subscriptionID, invoiceID, checkoutSessionID
}

func addCreditsFromPayment(orgID string, amountCents int64, sessionID, customerID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Check if transaction already exists for this checkout session (idempotency)
		var existingTransaction database.CreditTransaction
		notePattern := fmt.Sprintf("%%Checkout Session %s%%", sessionID)
		if err := tx.Where("organization_id = ? AND note LIKE ?", orgID, notePattern).First(&existingTransaction).Error; err == nil {
			log.Printf("[Stripe Webhook] Transaction already exists for checkout session %s", sessionID)
			return nil // Already processed, return success
		}

		// Get organization
		var org database.Organization
		if err := tx.First(&org, "id = ?", orgID).Error; err != nil {
			return fmt.Errorf("organization not found: %w", err)
		}

		// Update credits and total paid
		org.Credits += amountCents
		org.TotalPaidCents += amountCents
		if err := tx.Save(&org).Error; err != nil {
			return fmt.Errorf("update credits: %w", err)
		}

		// Record transaction
		note := fmt.Sprintf("Payment via Stripe Checkout Session %s", sessionID)
		transaction := &database.CreditTransaction{
			ID:             generateID("ct"),
			OrganizationID: orgID,
			AmountCents:    amountCents,
			BalanceAfter:   org.Credits,
			Type:           "payment",
			Source:         "stripe",
			Note:           &note,
			CreatedAt:      time.Now(),
		}
		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		// Check and upgrade plan if eligible (after credits are added)
		if err := checkAndUpgradePlan(orgID, tx); err != nil {
			log.Printf("[Stripe Webhook] Warning: failed to check/upgrade plan: %v", err)
			// Don't fail the transaction if upgrade check fails
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
					// Don't fail the transaction for this
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
				// Don't fail the transaction for this
			}
		}

		return nil
	})
	
	// Ensure organization has a plan assigned (defaults to Starter plan)
	// This is called after the transaction commits to ensure plan is assigned
	if err := organizations.EnsurePlanAssigned(orgID); err != nil {
		log.Printf("[Stripe Webhook] Warning: failed to ensure plan assigned: %v", err)
		// Don't fail the payment processing if plan assignment fails
	}
	
	return nil
}
