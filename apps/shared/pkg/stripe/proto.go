package stripe

import (
	"strings"
	"time"

	billingv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/billing/v1"

	stripego "github.com/stripe/stripe-go/v83"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InvoiceToProto converts a Stripe invoice into the shared billing proto shape.
func InvoiceToProto(inv *stripego.Invoice) *billingv1.Invoice {
	if inv == nil {
		return nil
	}

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

	subtotal := inv.Subtotal
	protoInvoice.Subtotal = &subtotal

	total := inv.Total
	protoInvoice.Total = &total

	amountRemaining := inv.AmountRemaining
	protoInvoice.AmountRemaining = &amountRemaining

	attemptCount := int32(inv.AttemptCount)
	protoInvoice.AttemptCount = &attemptCount

	if inv.CollectionMethod != "" {
		collectionMethod := string(inv.CollectionMethod)
		protoInvoice.CollectionMethod = &collectionMethod
	}

	if inv.StatusTransitions != nil && inv.StatusTransitions.PaidAt > 0 {
		paidAt := timestamppb.New(time.Unix(inv.StatusTransitions.PaidAt, 0))
		protoInvoice.PaidAt = paidAt
	}

	return protoInvoice
}

// AddressToProto converts a Stripe address to the shared billing proto shape.
func AddressToProto(address *stripego.Address) *billingv1.Address {
	if address == nil {
		return nil
	}

	protoAddress := &billingv1.Address{
		Line1:      address.Line1,
		City:       address.City,
		PostalCode: address.PostalCode,
		Country:    address.Country,
	}

	if address.Line2 != "" {
		protoAddress.Line2 = &address.Line2
	}

	if address.State != "" {
		protoAddress.State = &address.State
	}

	return protoAddress
}
