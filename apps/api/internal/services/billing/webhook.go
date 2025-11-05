package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"api/internal/database"

	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/webhook"
	"gorm.io/gorm"
)

// HandleStripeWebhook processes Stripe webhook events
func HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
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

func addCreditsFromPayment(orgID string, amountCents int64, sessionID, customerID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Get organization
		var org database.Organization
		if err := tx.First(&org, "id = ?", orgID).Error; err != nil {
			return fmt.Errorf("organization not found: %w", err)
		}

		// Update credits
		oldBalance := org.Credits
		org.Credits += amountCents
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

		log.Printf("[Stripe Webhook] Added %d cents to organization %s (balance: %d -> %d)", amountCents, orgID, oldBalance, org.Credits)
		return nil
	})
}
