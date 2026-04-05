package billing

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/stripe"
	billingv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/billing/v1"

	stripego "github.com/stripe/stripe-go/v83"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) enrichBillingAccountFromStripe(ctx context.Context, account *billingv1.BillingAccount) {
	if s.stripeClient == nil || account == nil || account.GetStripeCustomerId() == "" {
		return
	}

	customer, err := s.stripeClient.GetCustomer(ctx, account.GetStripeCustomerId())
	if err != nil {
		log.Printf("[Billing] Failed to hydrate billing account from Stripe customer %s: %v", account.GetStripeCustomerId(), err)
		return
	}

	if account.GetBillingEmail() == "" && customer.Email != "" {
		account.BillingEmail = &customer.Email
	}

	if account.GetCompanyName() == "" {
		if name := stripeCustomerDisplayName(customer); name != "" {
			account.CompanyName = &name
		}
	}

	if account.Address == nil {
		account.Address = stripe.AddressToProto(customer.Address)
	}

	if account.GetTaxId() == "" {
		if taxID := strings.TrimSpace(customer.Metadata["tax_id"]); taxID != "" {
			account.TaxId = &taxID
		}
	}

	if customer.Delinquent {
		account.Status = "PAST_DUE"
	} else if strings.TrimSpace(account.Status) == "" || strings.EqualFold(account.Status, "inactive") {
		account.Status = "ACTIVE"
	}
}

func stripeSubscriptionToProto(sub *stripego.Subscription) *billingv1.Subscription {
	if sub == nil {
		return nil
	}

	amount := int64(0)
	currency := "usd"
	interval := ""
	intervalCount := int32(1)
	description := ""

	if item := primarySubscriptionItem(sub); item != nil && item.Price != nil {
		amount = item.Price.UnitAmount
		currency = string(item.Price.Currency)
		if item.Price.Recurring != nil {
			interval = string(item.Price.Recurring.Interval)
			intervalCount = int32(item.Price.Recurring.IntervalCount)
		}
		description = subscriptionDescription(item)
	}

	protoSub := &billingv1.Subscription{
		Id:                sub.ID,
		Status:            string(sub.Status),
		Amount:            amount,
		Currency:          currency,
		Interval:          interval,
		IntervalCount:     intervalCount,
		Description:       description,
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}

	if item := primarySubscriptionItem(sub); item != nil {
		if item.CurrentPeriodStart > 0 {
			protoSub.CurrentPeriodStart = timestamppb.New(time.Unix(item.CurrentPeriodStart, 0))
		}
		if item.CurrentPeriodEnd > 0 {
			protoSub.CurrentPeriodEnd = timestamppb.New(time.Unix(item.CurrentPeriodEnd, 0))
		}
	}

	if sub.CanceledAt > 0 {
		protoSub.CanceledAt = timestamppb.New(time.Unix(sub.CanceledAt, 0))
	}
	if sub.Created > 0 {
		protoSub.Created = timestamppb.New(time.Unix(sub.Created, 0))
	}

	return protoSub
}

func primarySubscriptionItem(sub *stripego.Subscription) *stripego.SubscriptionItem {
	if sub == nil || sub.Items == nil || len(sub.Items.Data) == 0 {
		return nil
	}
	return sub.Items.Data[0]
}

func subscriptionDescription(item *stripego.SubscriptionItem) string {
	if item == nil || item.Price == nil {
		return ""
	}
	if item.Price.Nickname != "" {
		return item.Price.Nickname
	}
	if item.Price.Product != nil && item.Price.Product.Name != "" {
		return item.Price.Product.Name
	}
	if item.Plan != nil && item.Plan.Nickname != "" {
		return item.Plan.Nickname
	}
	if item.Price.LookupKey != "" {
		return item.Price.LookupKey
	}
	if item.Price.ID != "" {
		return item.Price.ID
	}
	return "Subscription"
}

func stripeCustomerDisplayName(customer *stripego.Customer) string {
	if customer == nil {
		return ""
	}
	for _, candidate := range []string{
		customer.BusinessName,
		customer.Name,
		customer.IndividualName,
	} {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}
