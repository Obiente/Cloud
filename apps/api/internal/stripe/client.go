package stripe

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/stripe/stripe-go/v83"
	portalsession "github.com/stripe/stripe-go/v83/billingportal/session"
	"github.com/stripe/stripe-go/v83/checkout/session"
	"github.com/stripe/stripe-go/v83/customer"
	"github.com/stripe/stripe-go/v83/invoice"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/paymentmethod"
	"github.com/stripe/stripe-go/v83/price"
	"github.com/stripe/stripe-go/v83/product"
	"github.com/stripe/stripe-go/v83/setupintent"
	"github.com/stripe/stripe-go/v83/subscription"
)

// Client wraps Stripe API client
type Client struct {
	stripeKey string
}

// NewClient creates a new Stripe client
func NewClient() (*Client, error) {
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeKey == "" {
		return nil, errors.New("STRIPE_SECRET_KEY environment variable is required")
	}
	stripe.Key = stripeKey
	// API version is determined by the SDK version (stripe-go/v83 uses 2025-10-29.clover)
	// Webhook endpoints must be configured with the same API version in Stripe Dashboard
	return &Client{stripeKey: stripeKey}, nil
}

// CreateCheckoutSession creates a Stripe Checkout Session for purchasing credits
func (c *Client) CreateCheckoutSession(ctx context.Context, params *CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	successURL := params.SuccessURL
	if successURL == "" {
		successURL = os.Getenv("CONSOLE_URL") + "/organizations?tab=billing&payment=success"
	}
	cancelURL := params.CancelURL
	if cancelURL == "" {
		cancelURL = os.Getenv("CONSOLE_URL") + "/organizations?tab=billing&payment=canceled"
	}

	// Get or create Stripe customer
	var customerID string
	if params.CustomerID != "" {
		customerID = params.CustomerID
	} else {
		custID, err := c.getOrCreateCustomer(ctx, params.OrganizationID, params.CustomerEmail)
		if err != nil {
			return nil, fmt.Errorf("get or create customer: %w", err)
		}
		customerID = custID
	}

	// Create checkout session
	sessionParams := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String("Obiente Cloud Credits"),
						Description: stripe.String(fmt.Sprintf("Add %s credits to your account", formatAmount(params.AmountCents))),
					},
					UnitAmount: stripe.Int64(params.AmountCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Metadata: map[string]string{
			"organization_id": params.OrganizationID,
			"amount_cents":    strconv.FormatInt(params.AmountCents, 10),
		},
	}

	sess, err := session.New(sessionParams)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return sess, nil
}

// CreatePortalSession creates a Stripe Customer Portal session
func (c *Client) CreatePortalSession(ctx context.Context, customerID, returnURL string) (*stripe.BillingPortalSession, error) {
	if returnURL == "" {
		returnURL = os.Getenv("CONSOLE_URL") + "/organizations?tab=billing"
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	sess, err := portalsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("create portal session: %w", err)
	}

	return sess, nil
}

// CreateSetupIntent creates a Setup Intent for collecting payment methods without a payment
func (c *Client) CreateSetupIntent(ctx context.Context, customerID string) (*stripe.SetupIntent, error) {
	params := &stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		Usage: stripe.String(string(stripe.SetupIntentUsageOffSession)),
	}

	si, err := setupintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("create setup intent: %w", err)
	}

	return si, nil
}

// ListPaymentMethods lists all payment methods for a customer
func (c *Client) ListPaymentMethods(ctx context.Context, customerID string) ([]*stripe.PaymentMethod, error) {
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerID),
		Type:     stripe.String("card"),
	}

	iter := paymentmethod.List(params)
	var methods []*stripe.PaymentMethod
	for iter.Next() {
		methods = append(methods, iter.PaymentMethod())
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("list payment methods: %w", err)
	}

	return methods, nil
}

// AttachPaymentMethod attaches a payment method to a customer
func (c *Client) AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) (*stripe.PaymentMethod, error) {
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}

	pm, err := paymentmethod.Attach(paymentMethodID, params)
	if err != nil {
		return nil, fmt.Errorf("attach payment method: %w", err)
	}

	return pm, nil
}

// DetachPaymentMethod detaches a payment method from a customer
func (c *Client) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	_, err := paymentmethod.Detach(paymentMethodID, nil)
	if err != nil {
		return fmt.Errorf("detach payment method: %w", err)
	}
	return nil
}

// SetDefaultPaymentMethod sets the default payment method for a customer
func (c *Client) SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	params := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethodID),
		},
	}

	_, err := customer.Update(customerID, params)
	if err != nil {
		return fmt.Errorf("set default payment method: %w", err)
	}

	return nil
}

// GetPaymentIntent retrieves a payment intent by ID
func (c *Client) GetPaymentIntent(ctx context.Context, paymentIntentID string) (*stripe.PaymentIntent, error) {
	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		return nil, fmt.Errorf("get payment intent: %w", err)
	}
	return pi, nil
}

// ListInvoices lists all invoices for a customer
func (c *Client) ListInvoices(ctx context.Context, customerID string, limit int) ([]*stripe.Invoice, bool, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	params := &stripe.InvoiceListParams{
		Customer: stripe.String(customerID),
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(int64(limit)),
		},
	}

	iter := invoice.List(params)
	var invoices []*stripe.Invoice
	for iter.Next() {
		invoices = append(invoices, iter.Invoice())
	}

	hasMore := iter.Meta().HasMore
	if err := iter.Err(); err != nil {
		return nil, false, fmt.Errorf("list invoices: %w", err)
	}

	return invoices, hasMore, nil
}

// GetCustomer retrieves a customer by ID
func (c *Client) GetCustomer(ctx context.Context, customerID string) (*stripe.Customer, error) {
	cust, err := customer.Get(customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	return cust, nil
}

// CreateCustomer creates a new Stripe customer
func (c *Client) CreateCustomer(ctx context.Context, email, organizationID string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Metadata: map[string]string{
			"organization_id": organizationID,
		},
	}

	cust, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return cust, nil
}

// UpdateCustomer updates an existing Stripe customer
func (c *Client) UpdateCustomer(ctx context.Context, customerID string, params *stripe.CustomerParams) (*stripe.Customer, error) {
	cust, err := customer.Update(customerID, params)
	if err != nil {
		return nil, fmt.Errorf("update customer: %w", err)
	}
	return cust, nil
}

// getOrCreateCustomer gets an existing customer or creates a new one
func (c *Client) getOrCreateCustomer(ctx context.Context, organizationID, email string) (string, error) {
	// First, try to find existing customer by metadata
	// Note: Stripe doesn't have a direct search by metadata, so we'll need to store
	// the customer ID in our database. For now, we'll create a new customer.
	// The billing service should handle linking customer IDs to organizations.

	if email == "" {
		return "", errors.New("email is required to create customer")
	}

	cust, err := c.CreateCustomer(ctx, email, organizationID)
	if err != nil {
		return "", err
	}

	return cust.ID, nil
}

// CheckoutSessionParams contains parameters for creating a checkout session
type CheckoutSessionParams struct {
	OrganizationID string
	CustomerEmail  string
	CustomerID     string // Optional: existing Stripe customer ID
	AmountCents    int64
	SuccessURL     string
	CancelURL      string
}

// formatAmount formats cents as a dollar amount string
func formatAmount(cents int64) string {
	dollars := cents / 100
	remainingCents := cents % 100
	return fmt.Sprintf("$%d.%02d", dollars, remainingCents)
}

// CreateSubscriptionCheckoutSession creates a Stripe Checkout Session for a subscription
func (c *Client) CreateSubscriptionCheckoutSession(ctx context.Context, params *SubscriptionCheckoutSessionParams) (*stripe.CheckoutSession, error) {
	successURL := params.SuccessURL
	if successURL == "" {
		successURL = os.Getenv("CONSOLE_URL") + "/organizations?tab=billing&payment=success"
	}
	cancelURL := params.CancelURL
	if cancelURL == "" {
		cancelURL = os.Getenv("CONSOLE_URL") + "/organizations?tab=billing&payment=canceled"
	}

	// Get or create Stripe customer
	var customerID string
	if params.CustomerID != "" {
		customerID = params.CustomerID
		// Ensure customer has organization_id in metadata
		custParams := &stripe.CustomerParams{
			Metadata: map[string]string{
				"organization_id": params.OrganizationID,
			},
		}
		if _, err := customer.Update(customerID, custParams); err != nil {
			log.Printf("[Stripe] Warning: Failed to update customer metadata: %v", err)
		}
	} else {
		custID, err := c.getOrCreateCustomer(ctx, params.OrganizationID, params.CustomerEmail)
		if err != nil {
			return nil, fmt.Errorf("get or create customer: %w", err)
		}
		customerID = custID
	}

	// Get or create DNS Delegation product and price
	priceID, err := c.getOrCreateDNSDelegationPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get or create DNS delegation price: %w", err)
	}

	// Create checkout session for subscription
	sessionParams := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Metadata: map[string]string{
			"organization_id": params.OrganizationID,
			"product_type":    "dns_delegation",
		},
	}

	sess, err := session.New(sessionParams)
	if err != nil {
		return nil, fmt.Errorf("create subscription checkout session: %w", err)
	}

	return sess, nil
}

// GetSubscription retrieves a subscription by ID
func (c *Client) GetSubscription(ctx context.Context, subscriptionID string) (*stripe.Subscription, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return sub, nil
}

// ListSubscriptionsForCustomer lists all subscriptions for a Stripe customer
func (c *Client) ListSubscriptionsForCustomer(ctx context.Context, customerID string) ([]*stripe.Subscription, error) {
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String("active"), // Only get active subscriptions
	}
	
	iter := subscription.List(params)
	var subscriptions []*stripe.Subscription
	for iter.Next() {
		subscriptions = append(subscriptions, iter.Subscription())
	}
	
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	
	return subscriptions, nil
}

// FindDNSDelegationSubscription finds an active DNS delegation subscription ($2/month) for a customer
func (c *Client) FindDNSDelegationSubscription(ctx context.Context, customerID string) (*stripe.Subscription, error) {
	subscriptions, err := c.ListSubscriptionsForCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	
	// Look for DNS delegation subscription ($2/month = 200 cents)
	for _, sub := range subscriptions {
		if len(sub.Items.Data) > 0 {
			price := sub.Items.Data[0].Price
			if price != nil && price.UnitAmount == 200 && price.Recurring != nil && price.Recurring.Interval == "month" {
				// Check if subscription is active or trialing
				if sub.Status == "active" || sub.Status == "trialing" {
					return sub, nil
				}
			}
		}
	}
	
	return nil, nil // No active DNS delegation subscription found
}

// CancelSubscription cancels a subscription
func (c *Client) CancelSubscription(ctx context.Context, subscriptionID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true), // Cancel at end of billing period
	}
	sub, err := subscription.Update(subscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("cancel subscription: %w", err)
	}
	return sub, nil
}

// getOrCreateDNSDelegationPrice gets or creates the DNS Delegation subscription price ($2/month)
func (c *Client) getOrCreateDNSDelegationPrice(ctx context.Context) (string, error) {
	// Check if we have a price ID in environment variable
	priceID := os.Getenv("STRIPE_DNS_DELEGATION_PRICE_ID")
	if priceID != "" {
		return priceID, nil
	}

	// Try to find existing DNS Delegation product
	productName := "DNS Delegation"
	productParams := &stripe.ProductListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(100),
		},
	}
	products := product.List(productParams)
	
	var dnsProduct *stripe.Product
	for products.Next() {
		p := products.Product()
		if p.Name == productName && p.Active {
			dnsProduct = p
			break
		}
	}

	// Create product if it doesn't exist
	if dnsProduct == nil {
		prodParams := &stripe.ProductParams{
			Name:        stripe.String(productName),
			Description: stripe.String("DNS Delegation for Self-Hosted Obiente Cloud instances"),
			Active:      stripe.Bool(true),
		}
		prod, err := product.New(prodParams)
		if err != nil {
			return "", fmt.Errorf("create DNS delegation product: %w", err)
		}
		dnsProduct = prod
	}

	// Try to find existing $2/month price
	priceParams := &stripe.PriceListParams{
		Product: stripe.String(dnsProduct.ID),
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(100),
		},
	}
	prices := price.List(priceParams)

	var dnsPrice *stripe.Price
	for prices.Next() {
		p := prices.Price()
		if p.UnitAmount == 200 && p.Currency == "usd" && p.Recurring != nil && p.Recurring.Interval == "month" && p.Active {
			dnsPrice = p
			break
		}
	}

	// Create price if it doesn't exist
	if dnsPrice == nil {
		priceParams := &stripe.PriceParams{
			Product:    stripe.String(dnsProduct.ID),
			Currency:   stripe.String("usd"),
			UnitAmount: stripe.Int64(200), // $2.00
			Recurring: &stripe.PriceRecurringParams{
				Interval: stripe.String("month"),
			},
			Active: stripe.Bool(true),
		}
		p, err := price.New(priceParams)
		if err != nil {
			return "", fmt.Errorf("create DNS delegation price: %w", err)
		}
		dnsPrice = p
	}

	return dnsPrice.ID, nil
}

// SubscriptionCheckoutSessionParams contains parameters for creating a subscription checkout session
type SubscriptionCheckoutSessionParams struct {
	OrganizationID string
	CustomerEmail  string
	CustomerID     string // Optional: existing Stripe customer ID
	SuccessURL     string
	CancelURL      string
}
