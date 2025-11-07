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

	// Mark event as being processed (with a small delay to handle race conditions)
	// Use a transaction to ensure atomicity
	var processedEvent database.StripeWebhookEvent
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Double-check in transaction
		if err := tx.Where("id = ?", event.ID).First(&existingEvent).Error; err == nil {
			return fmt.Errorf("event already processed")
		}
		
		// Create record
		processedEvent = database.StripeWebhookEvent{
			ID:         event.ID,
			EventType:  string(event.Type),
			ProcessedAt: time.Now(),
			CreatedAt:  time.Now(),
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

	default:
		log.Printf("[Stripe Webhook] Unhandled event type: %s", event.Type)
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
			return nil
		}

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
