package stripe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/stripe/stripe-go/v83"
	portalsession "github.com/stripe/stripe-go/v83/billingportal/session"
	"github.com/stripe/stripe-go/v83/checkout/session"
	"github.com/stripe/stripe-go/v83/customer"
	"github.com/stripe/stripe-go/v83/invoice"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/paymentmethod"
	"github.com/stripe/stripe-go/v83/setupintent"
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
