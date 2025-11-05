package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	billingv1 "api/gen/proto/obiente/cloud/billing/v1"
	billingv1connect "api/gen/proto/obiente/cloud/billing/v1/billingv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/stripe"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Service struct {
	billingv1connect.UnimplementedBillingServiceHandler
	stripeClient *stripe.Client
	consoleURL   string
}

func NewService(stripeClient *stripe.Client, consoleURL string) billingv1connect.BillingServiceHandler {
	return &Service{
		stripeClient: stripeClient,
		consoleURL:   strings.TrimSuffix(strings.TrimSpace(consoleURL), "/"),
	}
}

// checkStripeConfigured returns an error if Stripe is not configured
func (s *Service) checkStripeConfigured() error {
	if s.stripeClient == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("Stripe is not configured. Please set STRIPE_SECRET_KEY environment variable"))
	}
	return nil
}

func (s *Service) CreateCheckoutSession(ctx context.Context, req *connect.Request[billingv1.CreateCheckoutSessionRequest]) (*connect.Response[billingv1.CreateCheckoutSessionResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can create checkout sessions
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	amountCents := req.Msg.GetAmountCents()
	if amountCents <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("amount_cents must be positive"))
	}

	// Get or create billing account
	billingAccount, err := s.getOrCreateBillingAccount(orgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	// Get user email for Stripe customer
	var userEmail string
	if user.Email != "" {
		userEmail = user.Email
	} else {
		// Fallback: try to get from billing account
		if billingAccount.BillingEmail != nil {
			userEmail = *billingAccount.BillingEmail
		}
	}

	if userEmail == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email is required for billing"))
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	successURL := req.Msg.GetSuccessUrl()
	cancelURL := req.Msg.GetCancelUrl()

	// Create checkout session
	sessionParams := &stripe.CheckoutSessionParams{
		OrganizationID: orgID,
		CustomerEmail:  userEmail,
		AmountCents:    amountCents,
		SuccessURL:     successURL,
		CancelURL:      cancelURL,
	}

	// If billing account already has a Stripe customer ID, use it
	if billingAccount.StripeCustomerID != nil && *billingAccount.StripeCustomerID != "" {
		sessionParams.CustomerID = *billingAccount.StripeCustomerID
	}

	checkoutSession, err := s.stripeClient.CreateCheckoutSession(ctx, sessionParams)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create checkout session: %w", err))
	}

	// Update billing account with Stripe customer ID if not set
	if billingAccount.StripeCustomerID == nil || *billingAccount.StripeCustomerID == "" {
		if checkoutSession.Customer != nil {
			customerID := checkoutSession.Customer.ID
			billingAccount.StripeCustomerID = &customerID
			if err := database.DB.Save(billingAccount).Error; err != nil {
				log.Printf("[Billing] Failed to update billing account with customer ID: %v", err)
			}
		}
	}

	return connect.NewResponse(&billingv1.CreateCheckoutSessionResponse{
		SessionId:   checkoutSession.ID,
		CheckoutUrl: checkoutSession.URL,
	}), nil
}

func (s *Service) CreatePortalSession(ctx context.Context, req *connect.Request[billingv1.CreatePortalSessionRequest]) (*connect.Response[billingv1.CreatePortalSessionResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can access portal
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	// Get billing account
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("billing account not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	if billingAccount.StripeCustomerID == nil || *billingAccount.StripeCustomerID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no Stripe customer found. Please make a purchase first"))
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	returnURL := req.Msg.GetReturnUrl()

	portalSession, err := s.stripeClient.CreatePortalSession(ctx, *billingAccount.StripeCustomerID, returnURL)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create portal session: %w", err))
	}

	return connect.NewResponse(&billingv1.CreatePortalSessionResponse{
		PortalUrl: portalSession.URL,
	}), nil
}

func (s *Service) GetBillingAccount(ctx context.Context, req *connect.Request[billingv1.GetBillingAccountRequest]) (*connect.Response[billingv1.GetBillingAccountResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
	}

	// Get billing account
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return empty billing account if not found
			return connect.NewResponse(&billingv1.GetBillingAccountResponse{
				Account: &billingv1.BillingAccount{
					OrganizationId: orgID,
					Status:         "INACTIVE",
				},
			}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	protoAccount := s.billingAccountToProto(&billingAccount)

	return connect.NewResponse(&billingv1.GetBillingAccountResponse{
		Account: protoAccount,
	}), nil
}

func (s *Service) UpdateBillingAccount(ctx context.Context, req *connect.Request[billingv1.UpdateBillingAccountRequest]) (*connect.Response[billingv1.UpdateBillingAccountResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can update billing account
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	// Get or create billing account
	billingAccount, err := s.getOrCreateBillingAccount(orgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	// Update fields if provided
	if req.Msg.GetBillingEmail() != "" {
		email := req.Msg.GetBillingEmail()
		billingAccount.BillingEmail = &email
	}
	if req.Msg.GetCompanyName() != "" {
		name := req.Msg.GetCompanyName()
		billingAccount.CompanyName = &name
	}
	if req.Msg.GetTaxId() != "" {
		taxID := req.Msg.GetTaxId()
		billingAccount.TaxID = &taxID
	}
	if req.Msg.Address != nil {
		addressJSON, err := json.Marshal(req.Msg.Address)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid address: %w", err))
		}
		addressStr := string(addressJSON)
		billingAccount.Address = &addressStr
	}

	billingAccount.UpdatedAt = time.Now()

	if err := database.DB.Save(billingAccount).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update billing account: %w", err))
	}

	// Update Stripe customer if customer ID exists
	if billingAccount.StripeCustomerID != nil && *billingAccount.StripeCustomerID != "" {
		// Update Stripe customer with new information
		// This would require updating the Stripe client
	}

	protoAccount := s.billingAccountToProto(billingAccount)

	return connect.NewResponse(&billingv1.UpdateBillingAccountResponse{
		Account: protoAccount,
	}), nil
}

func (s *Service) CreateSetupIntent(ctx context.Context, req *connect.Request[billingv1.CreateSetupIntentRequest]) (*connect.Response[billingv1.CreateSetupIntentResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can create setup intents
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	// Get or create billing account
	billingAccount, err := s.getOrCreateBillingAccount(orgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	// Get user email for Stripe customer
	var userEmail string
	if user.Email != "" {
		userEmail = user.Email
	} else if billingAccount.BillingEmail != nil {
		userEmail = *billingAccount.BillingEmail
	}

	if userEmail == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email is required for billing"))
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	// Get or create Stripe customer
	var customerID string
	if billingAccount.StripeCustomerID != nil && *billingAccount.StripeCustomerID != "" {
		customerID = *billingAccount.StripeCustomerID
	} else {
		// Create customer if doesn't exist
		cust, err := s.stripeClient.CreateCustomer(ctx, userEmail, orgID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create customer: %w", err))
		}
		customerID = cust.ID
		// Update billing account
		billingAccount.StripeCustomerID = &customerID
		if err := database.DB.Save(billingAccount).Error; err != nil {
			log.Printf("[Billing] Failed to update billing account with customer ID: %v", err)
		}
	}

	// Create setup intent
	setupIntent, err := s.stripeClient.CreateSetupIntent(ctx, customerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create setup intent: %w", err))
	}

	return connect.NewResponse(&billingv1.CreateSetupIntentResponse{
		ClientSecret:  setupIntent.ClientSecret,
		SetupIntentId: setupIntent.ID,
	}), nil
}

func (s *Service) ListPaymentMethods(ctx context.Context, req *connect.Request[billingv1.ListPaymentMethodsRequest]) (*connect.Response[billingv1.ListPaymentMethodsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
	}

	// Get billing account
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return connect.NewResponse(&billingv1.ListPaymentMethodsResponse{
				PaymentMethods: []*billingv1.PaymentMethod{},
			}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	if billingAccount.StripeCustomerID == nil || *billingAccount.StripeCustomerID == "" {
		return connect.NewResponse(&billingv1.ListPaymentMethodsResponse{
			PaymentMethods: []*billingv1.PaymentMethod{},
		}), nil
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	// Get payment methods from Stripe
	stripeMethods, err := s.stripeClient.ListPaymentMethods(ctx, *billingAccount.StripeCustomerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list payment methods: %w", err))
	}

	// Get customer to find default payment method
	cust, err := s.stripeClient.GetCustomer(ctx, *billingAccount.StripeCustomerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get customer: %w", err))
	}

	var defaultPaymentMethodID string
	if cust.InvoiceSettings != nil && cust.InvoiceSettings.DefaultPaymentMethod != nil {
		defaultPaymentMethodID = cust.InvoiceSettings.DefaultPaymentMethod.ID
	}

	// Convert to proto
	protoMethods := make([]*billingv1.PaymentMethod, 0, len(stripeMethods))
	for _, pm := range stripeMethods {
		isDefault := pm.ID == defaultPaymentMethodID
		protoPM := &billingv1.PaymentMethod{
			Id:        pm.ID,
			Type:      string(pm.Type),
			IsDefault: isDefault,
		}

		if pm.Card != nil {
			protoPM.Card = &billingv1.CardDetails{
				Brand:    string(pm.Card.Brand),
				Last4:    pm.Card.Last4,
				ExpMonth: int32(pm.Card.ExpMonth),
				ExpYear:  int32(pm.Card.ExpYear),
			}
			if pm.BillingDetails != nil && pm.BillingDetails.Name != "" {
				protoPM.Card.Name = &pm.BillingDetails.Name
			}
		}

		protoMethods = append(protoMethods, protoPM)
	}

	return connect.NewResponse(&billingv1.ListPaymentMethodsResponse{
		PaymentMethods: protoMethods,
	}), nil
}

func (s *Service) AttachPaymentMethod(ctx context.Context, req *connect.Request[billingv1.AttachPaymentMethodRequest]) (*connect.Response[billingv1.AttachPaymentMethodResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	paymentMethodID := strings.TrimSpace(req.Msg.GetPaymentMethodId())
	if paymentMethodID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payment_method_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can attach payment methods
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	// Get billing account
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("billing account not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	if billingAccount.StripeCustomerID == nil || *billingAccount.StripeCustomerID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no Stripe customer found"))
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	// Attach payment method
	pm, err := s.stripeClient.AttachPaymentMethod(ctx, paymentMethodID, *billingAccount.StripeCustomerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("attach payment method: %w", err))
	}

	// Convert to proto
	protoPM := &billingv1.PaymentMethod{
		Id:   pm.ID,
		Type: string(pm.Type),
	}

	if pm.Card != nil {
		protoPM.Card = &billingv1.CardDetails{
			Brand:    string(pm.Card.Brand),
			Last4:    pm.Card.Last4,
			ExpMonth: int32(pm.Card.ExpMonth),
			ExpYear:  int32(pm.Card.ExpYear),
		}
		if pm.BillingDetails != nil && pm.BillingDetails.Name != "" {
			protoPM.Card.Name = &pm.BillingDetails.Name
		}
	}

	return connect.NewResponse(&billingv1.AttachPaymentMethodResponse{
		PaymentMethod: protoPM,
	}), nil
}

func (s *Service) DetachPaymentMethod(ctx context.Context, req *connect.Request[billingv1.DetachPaymentMethodRequest]) (*connect.Response[billingv1.DetachPaymentMethodResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	paymentMethodID := strings.TrimSpace(req.Msg.GetPaymentMethodId())
	if paymentMethodID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payment_method_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can detach payment methods
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	// Get billing account to verify customer
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("billing account not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	if billingAccount.StripeCustomerID == nil || *billingAccount.StripeCustomerID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no Stripe customer found"))
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	// Verify payment method belongs to this customer
	paymentMethods, err := s.stripeClient.ListPaymentMethods(ctx, *billingAccount.StripeCustomerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list payment methods: %w", err))
	}

	belongsToCustomer := false
	for _, pm := range paymentMethods {
		if pm.ID == paymentMethodID {
			belongsToCustomer = true
			break
		}
	}

	if !belongsToCustomer {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("payment method does not belong to this customer"))
	}

	// Detach payment method
	if err := s.stripeClient.DetachPaymentMethod(ctx, paymentMethodID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("detach payment method: %w", err))
	}

	return connect.NewResponse(&billingv1.DetachPaymentMethodResponse{
		Success: true,
	}), nil
}

func (s *Service) SetDefaultPaymentMethod(ctx context.Context, req *connect.Request[billingv1.SetDefaultPaymentMethodRequest]) (*connect.Response[billingv1.SetDefaultPaymentMethodResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	paymentMethodID := strings.TrimSpace(req.Msg.GetPaymentMethodId())
	if paymentMethodID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payment_method_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can set default payment method
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	// Get billing account
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("billing account not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	if billingAccount.StripeCustomerID == nil || *billingAccount.StripeCustomerID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no Stripe customer found"))
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	// Verify payment method belongs to this customer
	paymentMethods, err := s.stripeClient.ListPaymentMethods(ctx, *billingAccount.StripeCustomerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list payment methods: %w", err))
	}

	belongsToCustomer := false
	for _, pm := range paymentMethods {
		if pm.ID == paymentMethodID {
			belongsToCustomer = true
			break
		}
	}

	if !belongsToCustomer {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("payment method does not belong to this customer"))
	}

	// Set default payment method
	if err := s.stripeClient.SetDefaultPaymentMethod(ctx, *billingAccount.StripeCustomerID, paymentMethodID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("set default payment method: %w", err))
	}

	return connect.NewResponse(&billingv1.SetDefaultPaymentMethodResponse{
		Success: true,
	}), nil
}

func (s *Service) GetPaymentStatus(ctx context.Context, req *connect.Request[billingv1.GetPaymentStatusRequest]) (*connect.Response[billingv1.GetPaymentStatusResponse], error) {
	paymentIntentID := strings.TrimSpace(req.Msg.GetPaymentIntentId())
	if paymentIntentID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payment_intent_id is required"))
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	paymentIntent, err := s.stripeClient.GetPaymentIntent(ctx, paymentIntentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("payment intent not found: %w", err))
	}

	status := string(paymentIntent.Status)
	var errorMsg *string
	if paymentIntent.LastPaymentError != nil {
		msg := paymentIntent.LastPaymentError.Error()
		errorMsg = &msg
	}

	return connect.NewResponse(&billingv1.GetPaymentStatusResponse{
		Status:       status,
		ErrorMessage: errorMsg,
	}), nil
}

func (s *Service) ListInvoices(ctx context.Context, req *connect.Request[billingv1.ListInvoicesRequest]) (*connect.Response[billingv1.ListInvoicesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can view invoices
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	// Get billing account
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return empty list if no billing account
			return connect.NewResponse(&billingv1.ListInvoicesResponse{
				Invoices: []*billingv1.Invoice{},
				HasMore:  false,
			}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	if billingAccount.StripeCustomerID == nil || *billingAccount.StripeCustomerID == "" {
		// No Stripe customer ID, return empty list
		return connect.NewResponse(&billingv1.ListInvoicesResponse{
			Invoices: []*billingv1.Invoice{},
			HasMore:  false,
		}), nil
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 10
	}

	// List invoices from Stripe
	stripeInvoices, hasMore, err := s.stripeClient.ListInvoices(ctx, *billingAccount.StripeCustomerID, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list invoices: %w", err))
	}

	// Convert Stripe invoices to proto
	invoices := make([]*billingv1.Invoice, 0, len(stripeInvoices))
	for _, inv := range stripeInvoices {
		protoInvoice := &billingv1.Invoice{
			Id:         inv.ID,
			Number:     inv.Number,
			Status:     string(inv.Status),
			AmountDue:  inv.AmountDue,
			AmountPaid: inv.AmountPaid,
			Currency:   strings.ToUpper(string(inv.Currency)),
		}

		if inv.Created > 0 {
			protoInvoice.Date = timestamppb.New(time.Unix(inv.Created, 0))
		}

		if inv.DueDate > 0 {
			protoInvoice.DueDate = timestamppb.New(time.Unix(inv.DueDate, 0))
		}

		if inv.InvoicePDF != "" {
			protoInvoice.InvoicePdf = &inv.InvoicePDF
		}

		if inv.HostedInvoiceURL != "" {
			protoInvoice.HostedInvoiceUrl = &inv.HostedInvoiceURL
		}

		if inv.Description != "" {
			protoInvoice.Description = &inv.Description
		}

		invoices = append(invoices, protoInvoice)
	}

	return connect.NewResponse(&billingv1.ListInvoicesResponse{
		Invoices: invoices,
		HasMore:  hasMore,
	}), nil
}

// Helper functions

func (s *Service) getOrCreateBillingAccount(orgID string) (*database.BillingAccount, error) {
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new billing account
			billingAccount = database.BillingAccount{
				ID:             generateID("ba"),
				OrganizationID: orgID,
				Status:         "ACTIVE",
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := database.DB.Create(&billingAccount).Error; err != nil {
				return nil, fmt.Errorf("create billing account: %w", err)
			}
		} else {
			return nil, err
		}
	}
	return &billingAccount, nil
}

func (s *Service) billingAccountToProto(ba *database.BillingAccount) *billingv1.BillingAccount {
	proto := &billingv1.BillingAccount{
		Id:             ba.ID,
		OrganizationId: ba.OrganizationID,
		Status:         ba.Status,
		CreatedAt:      timestamppb.New(ba.CreatedAt),
		UpdatedAt:      timestamppb.New(ba.UpdatedAt),
	}

	if ba.StripeCustomerID != nil {
		proto.StripeCustomerId = ba.StripeCustomerID
	}
	if ba.BillingEmail != nil {
		proto.BillingEmail = ba.BillingEmail
	}
	if ba.CompanyName != nil {
		proto.CompanyName = ba.CompanyName
	}
	if ba.TaxID != nil {
		proto.TaxId = ba.TaxID
	}
	if ba.Address != nil && *ba.Address != "" {
		var address billingv1.Address
		if err := json.Unmarshal([]byte(*ba.Address), &address); err == nil {
			proto.Address = &address
		}
	}

	return proto
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func (s *Service) CreateDNSDelegationSubscriptionCheckout(ctx context.Context, req *connect.Request[billingv1.CreateDNSDelegationSubscriptionCheckoutRequest]) (*connect.Response[billingv1.CreateDNSDelegationSubscriptionCheckoutResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can create subscription checkout
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient permissions"))
		}
	}

	// Get or create billing account
	billingAccount, err := s.getOrCreateBillingAccount(orgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get billing account: %w", err))
	}

	// Get user email for Stripe customer
	var userEmail string
	if user.Email != "" {
		userEmail = user.Email
	} else if billingAccount.BillingEmail != nil {
		userEmail = *billingAccount.BillingEmail
	}

	if userEmail == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email is required for billing"))
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	successURL := req.Msg.GetSuccessUrl()
	cancelURL := req.Msg.GetCancelUrl()

	// Create subscription checkout session
	sessionParams := &stripe.SubscriptionCheckoutSessionParams{
		OrganizationID: orgID,
		CustomerEmail:  userEmail,
		SuccessURL:     successURL,
		CancelURL:      cancelURL,
	}

	// If billing account already has a Stripe customer ID, use it
	if billingAccount.StripeCustomerID != nil && *billingAccount.StripeCustomerID != "" {
		sessionParams.CustomerID = *billingAccount.StripeCustomerID
	}

	checkoutSession, err := s.stripeClient.CreateSubscriptionCheckoutSession(ctx, sessionParams)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create subscription checkout session: %w", err))
	}

	// Update billing account with Stripe customer ID if not set
	if billingAccount.StripeCustomerID == nil || *billingAccount.StripeCustomerID == "" {
		if checkoutSession.Customer != nil {
			customerID := checkoutSession.Customer.ID
			billingAccount.StripeCustomerID = &customerID
			if err := database.DB.Save(billingAccount).Error; err != nil {
				log.Printf("[Billing] Failed to update billing account with customer ID: %v", err)
			}
		}
	}

	return connect.NewResponse(&billingv1.CreateDNSDelegationSubscriptionCheckoutResponse{
		SessionId:   checkoutSession.ID,
		CheckoutUrl: checkoutSession.URL,
	}), nil
}

func (s *Service) GetDNSDelegationSubscriptionStatus(ctx context.Context, req *connect.Request[billingv1.GetDNSDelegationSubscriptionStatusRequest]) (*connect.Response[billingv1.GetDNSDelegationSubscriptionStatusResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
	}

	// Check subscription status - first try database (API key method), then check Stripe directly
	hasSubscription, subscriptionID, err := database.HasActiveDNSDelegationSubscription(orgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check subscription status: %w", err))
	}

	var cancelAtPeriodEnd bool
	var currentPeriodEnd *timestamppb.Timestamp

	// If not found in database, check Stripe directly (webhook might not have processed yet)
	if !hasSubscription || subscriptionID == "" {
		// Get billing account to find Stripe customer ID
		var billingAccount database.BillingAccount
		if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err == nil {
			if billingAccount.StripeCustomerID != nil && *billingAccount.StripeCustomerID != "" {
				// Check Stripe for active DNS delegation subscription
				sub, err := s.stripeClient.FindDNSDelegationSubscription(ctx, *billingAccount.StripeCustomerID)
				if err == nil && sub != nil {
					hasSubscription = true
					subscriptionID = sub.ID
					cancelAtPeriodEnd = sub.CancelAtPeriodEnd
					// PeriodEnd not available in Stripe Go SDK, will be set when webhook processes
				}
			}
		}
	} else {
		// Subscription found in database, get details from Stripe
		sub, err := s.stripeClient.GetSubscription(ctx, subscriptionID)
		if err == nil && sub != nil {
			cancelAtPeriodEnd = sub.CancelAtPeriodEnd
			// PeriodEnd not available in Stripe Go SDK, will be set when webhook processes
		}
	}

	// Check if organization has an active API key
	apiKey, err := database.GetActiveDNSDelegationAPIKeyForOrganization(orgID)
	hasAPIKey := err == nil && apiKey != nil
	var apiKeyCreatedAt *timestamppb.Timestamp
	var apiKeyDescription string
	if hasAPIKey && apiKey != nil {
		apiKeyCreatedAt = timestamppb.New(apiKey.CreatedAt)
		apiKeyDescription = apiKey.Description
	}

	return connect.NewResponse(&billingv1.GetDNSDelegationSubscriptionStatusResponse{
		HasActiveSubscription: hasSubscription,
		StripeSubscriptionId:   subscriptionID,
		HasApiKey:             hasAPIKey,
		ApiKeyCreatedAt:       apiKeyCreatedAt,
		CancelAtPeriodEnd:     cancelAtPeriodEnd,
		CurrentPeriodEnd:      currentPeriodEnd,
		ApiKeyDescription:     apiKeyDescription,
	}), nil
}

func (s *Service) CancelDNSDelegationSubscription(ctx context.Context, req *connect.Request[billingv1.CancelDNSDelegationSubscriptionRequest]) (*connect.Response[billingv1.CancelDNSDelegationSubscriptionResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if err := s.checkStripeConfigured(); err != nil {
		return nil, err
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has access to this organization and is owner/admin
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		var member database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
		}
		// Only owners and admins can cancel subscriptions
		if member.Role != "owner" && member.Role != "admin" {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("only organization owners and admins can cancel subscriptions"))
		}
	}

	// Get subscription ID
	hasSubscription, subscriptionID, err := database.HasActiveDNSDelegationSubscription(orgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check subscription status: %w", err))
	}
	if !hasSubscription || subscriptionID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no active subscription found"))
	}

	// Cancel subscription via Stripe
	canceledSub, err := s.stripeClient.CancelSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("cancel subscription: %w", err))
	}

	var canceledAt *timestamppb.Timestamp
	var message string
	if canceledSub.CancelAtPeriodEnd {
		// Subscription will cancel at end of billing period
		// PeriodEnd not available in Stripe Go SDK, will be set when webhook processes
		message = "Subscription will be canceled at the end of the current billing period."
	} else {
		// Subscription canceled immediately
		message = "Subscription has been canceled."
	}

	return connect.NewResponse(&billingv1.CancelDNSDelegationSubscriptionResponse{
		Success:    true,
		Message:    message,
		CanceledAt: canceledAt,
	}), nil
}
