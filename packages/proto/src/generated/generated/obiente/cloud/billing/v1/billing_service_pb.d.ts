import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/billing/v1/billing_service.proto.
 */
export declare const file_obiente_cloud_billing_v1_billing_service: GenFile;
/**
 * @generated from message obiente.cloud.billing.v1.CreateCheckoutSessionRequest
 */
export type CreateCheckoutSessionRequest = Message<"obiente.cloud.billing.v1.CreateCheckoutSessionRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Amount in cents ($0.01 units). Must be positive.
     *
     * @generated from field: int64 amount_cents = 2;
     */
    amountCents: bigint;
    /**
     * Optional: success URL (defaults to console URL)
     *
     * @generated from field: optional string success_url = 3;
     */
    successUrl?: string;
    /**
     * Optional: cancel URL (defaults to console URL)
     *
     * @generated from field: optional string cancel_url = 4;
     */
    cancelUrl?: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreateCheckoutSessionRequest.
 * Use `create(CreateCheckoutSessionRequestSchema)` to create a new message.
 */
export declare const CreateCheckoutSessionRequestSchema: GenMessage<CreateCheckoutSessionRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.CreateCheckoutSessionResponse
 */
export type CreateCheckoutSessionResponse = Message<"obiente.cloud.billing.v1.CreateCheckoutSessionResponse"> & {
    /**
     * Stripe Checkout Session ID
     *
     * @generated from field: string session_id = 1;
     */
    sessionId: string;
    /**
     * Stripe Checkout Session URL for redirect
     *
     * @generated from field: string checkout_url = 2;
     */
    checkoutUrl: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreateCheckoutSessionResponse.
 * Use `create(CreateCheckoutSessionResponseSchema)` to create a new message.
 */
export declare const CreateCheckoutSessionResponseSchema: GenMessage<CreateCheckoutSessionResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.CreatePaymentIntentRequest
 */
export type CreatePaymentIntentRequest = Message<"obiente.cloud.billing.v1.CreatePaymentIntentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Amount in cents ($0.01 units). Must be positive.
     *
     * @generated from field: int64 amount_cents = 2;
     */
    amountCents: bigint;
    /**
     * Optional: payment method ID to use (defaults to customer's default payment method)
     *
     * @generated from field: optional string payment_method_id = 3;
     */
    paymentMethodId?: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreatePaymentIntentRequest.
 * Use `create(CreatePaymentIntentRequestSchema)` to create a new message.
 */
export declare const CreatePaymentIntentRequestSchema: GenMessage<CreatePaymentIntentRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.CreatePaymentIntentResponse
 */
export type CreatePaymentIntentResponse = Message<"obiente.cloud.billing.v1.CreatePaymentIntentResponse"> & {
    /**
     * Stripe Payment Intent ID
     *
     * @generated from field: string payment_intent_id = 1;
     */
    paymentIntentId: string;
    /**
     * Stripe Payment Intent client secret for frontend confirmation
     *
     * @generated from field: string client_secret = 2;
     */
    clientSecret: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreatePaymentIntentResponse.
 * Use `create(CreatePaymentIntentResponseSchema)` to create a new message.
 */
export declare const CreatePaymentIntentResponseSchema: GenMessage<CreatePaymentIntentResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.CreatePortalSessionRequest
 */
export type CreatePortalSessionRequest = Message<"obiente.cloud.billing.v1.CreatePortalSessionRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: return URL (defaults to console URL)
     *
     * @generated from field: optional string return_url = 2;
     */
    returnUrl?: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreatePortalSessionRequest.
 * Use `create(CreatePortalSessionRequestSchema)` to create a new message.
 */
export declare const CreatePortalSessionRequestSchema: GenMessage<CreatePortalSessionRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.CreatePortalSessionResponse
 */
export type CreatePortalSessionResponse = Message<"obiente.cloud.billing.v1.CreatePortalSessionResponse"> & {
    /**
     * Stripe Customer Portal Session URL for redirect
     *
     * @generated from field: string portal_url = 1;
     */
    portalUrl: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreatePortalSessionResponse.
 * Use `create(CreatePortalSessionResponseSchema)` to create a new message.
 */
export declare const CreatePortalSessionResponseSchema: GenMessage<CreatePortalSessionResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.GetBillingAccountRequest
 */
export type GetBillingAccountRequest = Message<"obiente.cloud.billing.v1.GetBillingAccountRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.GetBillingAccountRequest.
 * Use `create(GetBillingAccountRequestSchema)` to create a new message.
 */
export declare const GetBillingAccountRequestSchema: GenMessage<GetBillingAccountRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.GetBillingAccountResponse
 */
export type GetBillingAccountResponse = Message<"obiente.cloud.billing.v1.GetBillingAccountResponse"> & {
    /**
     * @generated from field: obiente.cloud.billing.v1.BillingAccount account = 1;
     */
    account?: BillingAccount;
};
/**
 * Describes the message obiente.cloud.billing.v1.GetBillingAccountResponse.
 * Use `create(GetBillingAccountResponseSchema)` to create a new message.
 */
export declare const GetBillingAccountResponseSchema: GenMessage<GetBillingAccountResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.UpdateBillingAccountRequest
 */
export type UpdateBillingAccountRequest = Message<"obiente.cloud.billing.v1.UpdateBillingAccountRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: optional string billing_email = 2;
     */
    billingEmail?: string;
    /**
     * @generated from field: optional string company_name = 3;
     */
    companyName?: string;
    /**
     * @generated from field: optional string tax_id = 4;
     */
    taxId?: string;
    /**
     * @generated from field: optional obiente.cloud.billing.v1.Address address = 5;
     */
    address?: Address;
    /**
     * Day of month (1-31) when billing occurs
     *
     * @generated from field: optional int32 billing_date = 6;
     */
    billingDate?: number;
};
/**
 * Describes the message obiente.cloud.billing.v1.UpdateBillingAccountRequest.
 * Use `create(UpdateBillingAccountRequestSchema)` to create a new message.
 */
export declare const UpdateBillingAccountRequestSchema: GenMessage<UpdateBillingAccountRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.UpdateBillingAccountResponse
 */
export type UpdateBillingAccountResponse = Message<"obiente.cloud.billing.v1.UpdateBillingAccountResponse"> & {
    /**
     * @generated from field: obiente.cloud.billing.v1.BillingAccount account = 1;
     */
    account?: BillingAccount;
};
/**
 * Describes the message obiente.cloud.billing.v1.UpdateBillingAccountResponse.
 * Use `create(UpdateBillingAccountResponseSchema)` to create a new message.
 */
export declare const UpdateBillingAccountResponseSchema: GenMessage<UpdateBillingAccountResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.ListPaymentMethodsRequest
 */
export type ListPaymentMethodsRequest = Message<"obiente.cloud.billing.v1.ListPaymentMethodsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.ListPaymentMethodsRequest.
 * Use `create(ListPaymentMethodsRequestSchema)` to create a new message.
 */
export declare const ListPaymentMethodsRequestSchema: GenMessage<ListPaymentMethodsRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.ListPaymentMethodsResponse
 */
export type ListPaymentMethodsResponse = Message<"obiente.cloud.billing.v1.ListPaymentMethodsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.billing.v1.PaymentMethod payment_methods = 1;
     */
    paymentMethods: PaymentMethod[];
};
/**
 * Describes the message obiente.cloud.billing.v1.ListPaymentMethodsResponse.
 * Use `create(ListPaymentMethodsResponseSchema)` to create a new message.
 */
export declare const ListPaymentMethodsResponseSchema: GenMessage<ListPaymentMethodsResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.GetPaymentStatusRequest
 */
export type GetPaymentStatusRequest = Message<"obiente.cloud.billing.v1.GetPaymentStatusRequest"> & {
    /**
     * @generated from field: string payment_intent_id = 1;
     */
    paymentIntentId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.GetPaymentStatusRequest.
 * Use `create(GetPaymentStatusRequestSchema)` to create a new message.
 */
export declare const GetPaymentStatusRequestSchema: GenMessage<GetPaymentStatusRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.GetPaymentStatusResponse
 */
export type GetPaymentStatusResponse = Message<"obiente.cloud.billing.v1.GetPaymentStatusResponse"> & {
    /**
     * "succeeded", "pending", "failed", "canceled"
     *
     * @generated from field: string status = 1;
     */
    status: string;
    /**
     * @generated from field: optional string error_message = 2;
     */
    errorMessage?: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.GetPaymentStatusResponse.
 * Use `create(GetPaymentStatusResponseSchema)` to create a new message.
 */
export declare const GetPaymentStatusResponseSchema: GenMessage<GetPaymentStatusResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.CreateSetupIntentRequest
 */
export type CreateSetupIntentRequest = Message<"obiente.cloud.billing.v1.CreateSetupIntentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: return URL after setup completion
     *
     * @generated from field: optional string return_url = 2;
     */
    returnUrl?: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreateSetupIntentRequest.
 * Use `create(CreateSetupIntentRequestSchema)` to create a new message.
 */
export declare const CreateSetupIntentRequestSchema: GenMessage<CreateSetupIntentRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.CreateSetupIntentResponse
 */
export type CreateSetupIntentResponse = Message<"obiente.cloud.billing.v1.CreateSetupIntentResponse"> & {
    /**
     * Stripe Setup Intent client secret for frontend
     *
     * @generated from field: string client_secret = 1;
     */
    clientSecret: string;
    /**
     * Setup Intent ID
     *
     * @generated from field: string setup_intent_id = 2;
     */
    setupIntentId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreateSetupIntentResponse.
 * Use `create(CreateSetupIntentResponseSchema)` to create a new message.
 */
export declare const CreateSetupIntentResponseSchema: GenMessage<CreateSetupIntentResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.AttachPaymentMethodRequest
 */
export type AttachPaymentMethodRequest = Message<"obiente.cloud.billing.v1.AttachPaymentMethodRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string payment_method_id = 2;
     */
    paymentMethodId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.AttachPaymentMethodRequest.
 * Use `create(AttachPaymentMethodRequestSchema)` to create a new message.
 */
export declare const AttachPaymentMethodRequestSchema: GenMessage<AttachPaymentMethodRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.AttachPaymentMethodResponse
 */
export type AttachPaymentMethodResponse = Message<"obiente.cloud.billing.v1.AttachPaymentMethodResponse"> & {
    /**
     * @generated from field: obiente.cloud.billing.v1.PaymentMethod payment_method = 1;
     */
    paymentMethod?: PaymentMethod;
};
/**
 * Describes the message obiente.cloud.billing.v1.AttachPaymentMethodResponse.
 * Use `create(AttachPaymentMethodResponseSchema)` to create a new message.
 */
export declare const AttachPaymentMethodResponseSchema: GenMessage<AttachPaymentMethodResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.DetachPaymentMethodRequest
 */
export type DetachPaymentMethodRequest = Message<"obiente.cloud.billing.v1.DetachPaymentMethodRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string payment_method_id = 2;
     */
    paymentMethodId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.DetachPaymentMethodRequest.
 * Use `create(DetachPaymentMethodRequestSchema)` to create a new message.
 */
export declare const DetachPaymentMethodRequestSchema: GenMessage<DetachPaymentMethodRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.DetachPaymentMethodResponse
 */
export type DetachPaymentMethodResponse = Message<"obiente.cloud.billing.v1.DetachPaymentMethodResponse"> & {
    /**
     * Success indicator
     *
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.billing.v1.DetachPaymentMethodResponse.
 * Use `create(DetachPaymentMethodResponseSchema)` to create a new message.
 */
export declare const DetachPaymentMethodResponseSchema: GenMessage<DetachPaymentMethodResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.SetDefaultPaymentMethodRequest
 */
export type SetDefaultPaymentMethodRequest = Message<"obiente.cloud.billing.v1.SetDefaultPaymentMethodRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string payment_method_id = 2;
     */
    paymentMethodId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.SetDefaultPaymentMethodRequest.
 * Use `create(SetDefaultPaymentMethodRequestSchema)` to create a new message.
 */
export declare const SetDefaultPaymentMethodRequestSchema: GenMessage<SetDefaultPaymentMethodRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.SetDefaultPaymentMethodResponse
 */
export type SetDefaultPaymentMethodResponse = Message<"obiente.cloud.billing.v1.SetDefaultPaymentMethodResponse"> & {
    /**
     * Success indicator
     *
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.billing.v1.SetDefaultPaymentMethodResponse.
 * Use `create(SetDefaultPaymentMethodResponseSchema)` to create a new message.
 */
export declare const SetDefaultPaymentMethodResponseSchema: GenMessage<SetDefaultPaymentMethodResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.ListInvoicesRequest
 */
export type ListInvoicesRequest = Message<"obiente.cloud.billing.v1.ListInvoicesRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: limit the number of invoices returned (default: 10, max: 100)
     *
     * @generated from field: optional int32 limit = 2;
     */
    limit?: number;
};
/**
 * Describes the message obiente.cloud.billing.v1.ListInvoicesRequest.
 * Use `create(ListInvoicesRequestSchema)` to create a new message.
 */
export declare const ListInvoicesRequestSchema: GenMessage<ListInvoicesRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.ListInvoicesResponse
 */
export type ListInvoicesResponse = Message<"obiente.cloud.billing.v1.ListInvoicesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.billing.v1.Invoice invoices = 1;
     */
    invoices: Invoice[];
    /**
     * Whether there are more invoices available
     *
     * @generated from field: bool has_more = 2;
     */
    hasMore: boolean;
};
/**
 * Describes the message obiente.cloud.billing.v1.ListInvoicesResponse.
 * Use `create(ListInvoicesResponseSchema)` to create a new message.
 */
export declare const ListInvoicesResponseSchema: GenMessage<ListInvoicesResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.Invoice
 */
export type Invoice = Message<"obiente.cloud.billing.v1.Invoice"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * Invoice number (human-readable)
     *
     * @generated from field: string number = 2;
     */
    number: string;
    /**
     * Invoice status: "draft", "open", "paid", "uncollectible", "void"
     *
     * @generated from field: string status = 3;
     */
    status: string;
    /**
     * Total amount in cents
     *
     * @generated from field: int64 amount_due = 4;
     */
    amountDue: bigint;
    /**
     * Amount paid in cents
     *
     * @generated from field: int64 amount_paid = 5;
     */
    amountPaid: bigint;
    /**
     * Currency code (e.g., "usd")
     *
     * @generated from field: string currency = 6;
     */
    currency: string;
    /**
     * Invoice date
     *
     * @generated from field: google.protobuf.Timestamp date = 7;
     */
    date?: Timestamp;
    /**
     * Due date (if applicable)
     *
     * @generated from field: optional google.protobuf.Timestamp due_date = 8;
     */
    dueDate?: Timestamp;
    /**
     * PDF download URL (if available)
     *
     * @generated from field: optional string invoice_pdf = 9;
     */
    invoicePdf?: string;
    /**
     * Hosted invoice URL (if available)
     *
     * @generated from field: optional string hosted_invoice_url = 10;
     */
    hostedInvoiceUrl?: string;
    /**
     * Description or memo
     *
     * @generated from field: optional string description = 11;
     */
    description?: string;
    /**
     * Subtotal before tax/credits/discount adjustments
     *
     * @generated from field: optional int64 subtotal = 12;
     */
    subtotal?: bigint;
    /**
     * Final invoice total
     *
     * @generated from field: optional int64 total = 13;
     */
    total?: bigint;
    /**
     * Remaining amount still owed
     *
     * @generated from field: optional int64 amount_remaining = 14;
     */
    amountRemaining?: bigint;
    /**
     * When the invoice was fully paid
     *
     * @generated from field: optional google.protobuf.Timestamp paid_at = 15;
     */
    paidAt?: Timestamp;
    /**
     * Number of attempted collections
     *
     * @generated from field: optional int32 attempt_count = 16;
     */
    attemptCount?: number;
    /**
     * Stripe collection method, e.g. charge_automatically or send_invoice
     *
     * @generated from field: optional string collection_method = 17;
     */
    collectionMethod?: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.Invoice.
 * Use `create(InvoiceSchema)` to create a new message.
 */
export declare const InvoiceSchema: GenMessage<Invoice>;
/**
 * @generated from message obiente.cloud.billing.v1.BillingAccount
 */
export type BillingAccount = Message<"obiente.cloud.billing.v1.BillingAccount"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: optional string stripe_customer_id = 3;
     */
    stripeCustomerId?: string;
    /**
     * "ACTIVE", "INACTIVE", "PAST_DUE", etc.
     *
     * @generated from field: string status = 4;
     */
    status: string;
    /**
     * @generated from field: optional string billing_email = 5;
     */
    billingEmail?: string;
    /**
     * @generated from field: optional string company_name = 6;
     */
    companyName?: string;
    /**
     * @generated from field: optional string tax_id = 7;
     */
    taxId?: string;
    /**
     * @generated from field: optional obiente.cloud.billing.v1.Address address = 8;
     */
    address?: Address;
    /**
     * Day of month (1-31) when billing occurs
     *
     * @generated from field: optional int32 billing_date = 9;
     */
    billingDate?: number;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 10;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 11;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.billing.v1.BillingAccount.
 * Use `create(BillingAccountSchema)` to create a new message.
 */
export declare const BillingAccountSchema: GenMessage<BillingAccount>;
/**
 * @generated from message obiente.cloud.billing.v1.PaymentMethod
 */
export type PaymentMethod = Message<"obiente.cloud.billing.v1.PaymentMethod"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * "card", "bank_account", etc.
     *
     * @generated from field: string type = 2;
     */
    type: string;
    /**
     * Card details (if type is "card")
     *
     * @generated from field: optional obiente.cloud.billing.v1.CardDetails card = 3;
     */
    card?: CardDetails;
    /**
     * @generated from field: bool is_default = 4;
     */
    isDefault: boolean;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 5;
     */
    createdAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.billing.v1.PaymentMethod.
 * Use `create(PaymentMethodSchema)` to create a new message.
 */
export declare const PaymentMethodSchema: GenMessage<PaymentMethod>;
/**
 * @generated from message obiente.cloud.billing.v1.CardDetails
 */
export type CardDetails = Message<"obiente.cloud.billing.v1.CardDetails"> & {
    /**
     * "visa", "mastercard", "amex", etc.
     *
     * @generated from field: string brand = 1;
     */
    brand: string;
    /**
     * @generated from field: string last4 = 2;
     */
    last4: string;
    /**
     * @generated from field: int32 exp_month = 3;
     */
    expMonth: number;
    /**
     * @generated from field: int32 exp_year = 4;
     */
    expYear: number;
    /**
     * @generated from field: optional string name = 5;
     */
    name?: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CardDetails.
 * Use `create(CardDetailsSchema)` to create a new message.
 */
export declare const CardDetailsSchema: GenMessage<CardDetails>;
/**
 * @generated from message obiente.cloud.billing.v1.Address
 */
export type Address = Message<"obiente.cloud.billing.v1.Address"> & {
    /**
     * @generated from field: string line1 = 1;
     */
    line1: string;
    /**
     * @generated from field: optional string line2 = 2;
     */
    line2?: string;
    /**
     * @generated from field: string city = 3;
     */
    city: string;
    /**
     * @generated from field: optional string state = 4;
     */
    state?: string;
    /**
     * @generated from field: string postal_code = 5;
     */
    postalCode: string;
    /**
     * @generated from field: string country = 6;
     */
    country: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.Address.
 * Use `create(AddressSchema)` to create a new message.
 */
export declare const AddressSchema: GenMessage<Address>;
/**
 * @generated from message obiente.cloud.billing.v1.CreateDNSDelegationSubscriptionCheckoutRequest
 */
export type CreateDNSDelegationSubscriptionCheckoutRequest = Message<"obiente.cloud.billing.v1.CreateDNSDelegationSubscriptionCheckoutRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: success URL (defaults to console URL)
     *
     * @generated from field: optional string success_url = 2;
     */
    successUrl?: string;
    /**
     * Optional: cancel URL (defaults to console URL)
     *
     * @generated from field: optional string cancel_url = 3;
     */
    cancelUrl?: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreateDNSDelegationSubscriptionCheckoutRequest.
 * Use `create(CreateDNSDelegationSubscriptionCheckoutRequestSchema)` to create a new message.
 */
export declare const CreateDNSDelegationSubscriptionCheckoutRequestSchema: GenMessage<CreateDNSDelegationSubscriptionCheckoutRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.CreateDNSDelegationSubscriptionCheckoutResponse
 */
export type CreateDNSDelegationSubscriptionCheckoutResponse = Message<"obiente.cloud.billing.v1.CreateDNSDelegationSubscriptionCheckoutResponse"> & {
    /**
     * @generated from field: string session_id = 1;
     */
    sessionId: string;
    /**
     * @generated from field: string checkout_url = 2;
     */
    checkoutUrl: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CreateDNSDelegationSubscriptionCheckoutResponse.
 * Use `create(CreateDNSDelegationSubscriptionCheckoutResponseSchema)` to create a new message.
 */
export declare const CreateDNSDelegationSubscriptionCheckoutResponseSchema: GenMessage<CreateDNSDelegationSubscriptionCheckoutResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.GetDNSDelegationSubscriptionStatusRequest
 */
export type GetDNSDelegationSubscriptionStatusRequest = Message<"obiente.cloud.billing.v1.GetDNSDelegationSubscriptionStatusRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.GetDNSDelegationSubscriptionStatusRequest.
 * Use `create(GetDNSDelegationSubscriptionStatusRequestSchema)` to create a new message.
 */
export declare const GetDNSDelegationSubscriptionStatusRequestSchema: GenMessage<GetDNSDelegationSubscriptionStatusRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.GetDNSDelegationSubscriptionStatusResponse
 */
export type GetDNSDelegationSubscriptionStatusResponse = Message<"obiente.cloud.billing.v1.GetDNSDelegationSubscriptionStatusResponse"> & {
    /**
     * @generated from field: bool has_active_subscription = 1;
     */
    hasActiveSubscription: boolean;
    /**
     * Empty if no subscription
     *
     * @generated from field: string stripe_subscription_id = 2;
     */
    stripeSubscriptionId: string;
    /**
     * Whether organization has an active API key
     *
     * @generated from field: bool has_api_key = 3;
     */
    hasApiKey: boolean;
    /**
     * When API key was created (if exists)
     *
     * @generated from field: google.protobuf.Timestamp api_key_created_at = 4;
     */
    apiKeyCreatedAt?: Timestamp;
    /**
     * Whether subscription is scheduled to cancel
     *
     * @generated from field: bool cancel_at_period_end = 5;
     */
    cancelAtPeriodEnd: boolean;
    /**
     * When current billing period ends
     *
     * @generated from field: google.protobuf.Timestamp current_period_end = 6;
     */
    currentPeriodEnd?: Timestamp;
    /**
     * Description of the API key (if exists)
     *
     * @generated from field: string api_key_description = 7;
     */
    apiKeyDescription: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.GetDNSDelegationSubscriptionStatusResponse.
 * Use `create(GetDNSDelegationSubscriptionStatusResponseSchema)` to create a new message.
 */
export declare const GetDNSDelegationSubscriptionStatusResponseSchema: GenMessage<GetDNSDelegationSubscriptionStatusResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.CancelDNSDelegationSubscriptionRequest
 */
export type CancelDNSDelegationSubscriptionRequest = Message<"obiente.cloud.billing.v1.CancelDNSDelegationSubscriptionRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CancelDNSDelegationSubscriptionRequest.
 * Use `create(CancelDNSDelegationSubscriptionRequestSchema)` to create a new message.
 */
export declare const CancelDNSDelegationSubscriptionRequestSchema: GenMessage<CancelDNSDelegationSubscriptionRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.CancelDNSDelegationSubscriptionResponse
 */
export type CancelDNSDelegationSubscriptionResponse = Message<"obiente.cloud.billing.v1.CancelDNSDelegationSubscriptionResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * Status message
     *
     * @generated from field: string message = 2;
     */
    message: string;
    /**
     * When subscription will be canceled (end of billing period)
     *
     * @generated from field: google.protobuf.Timestamp canceled_at = 3;
     */
    canceledAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.billing.v1.CancelDNSDelegationSubscriptionResponse.
 * Use `create(CancelDNSDelegationSubscriptionResponseSchema)` to create a new message.
 */
export declare const CancelDNSDelegationSubscriptionResponseSchema: GenMessage<CancelDNSDelegationSubscriptionResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.ListSubscriptionsRequest
 */
export type ListSubscriptionsRequest = Message<"obiente.cloud.billing.v1.ListSubscriptionsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.ListSubscriptionsRequest.
 * Use `create(ListSubscriptionsRequestSchema)` to create a new message.
 */
export declare const ListSubscriptionsRequestSchema: GenMessage<ListSubscriptionsRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.ListSubscriptionsResponse
 */
export type ListSubscriptionsResponse = Message<"obiente.cloud.billing.v1.ListSubscriptionsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.billing.v1.Subscription subscriptions = 1;
     */
    subscriptions: Subscription[];
};
/**
 * Describes the message obiente.cloud.billing.v1.ListSubscriptionsResponse.
 * Use `create(ListSubscriptionsResponseSchema)` to create a new message.
 */
export declare const ListSubscriptionsResponseSchema: GenMessage<ListSubscriptionsResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.Subscription
 */
export type Subscription = Message<"obiente.cloud.billing.v1.Subscription"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * "active", "canceled", "past_due", "unpaid", "trialing", etc.
     *
     * @generated from field: string status = 2;
     */
    status: string;
    /**
     * @generated from field: google.protobuf.Timestamp current_period_start = 3;
     */
    currentPeriodStart?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp current_period_end = 4;
     */
    currentPeriodEnd?: Timestamp;
    /**
     * If subscription is canceled
     *
     * @generated from field: google.protobuf.Timestamp canceled_at = 5;
     */
    canceledAt?: Timestamp;
    /**
     * Whether subscription will cancel at period end
     *
     * @generated from field: bool cancel_at_period_end = 6;
     */
    cancelAtPeriodEnd: boolean;
    /**
     * Amount in cents
     *
     * @generated from field: int64 amount = 7;
     */
    amount: bigint;
    /**
     * @generated from field: string currency = 8;
     */
    currency: string;
    /**
     * "month", "year", etc.
     *
     * @generated from field: string interval = 9;
     */
    interval: string;
    /**
     * Number of intervals
     *
     * @generated from field: int32 interval_count = 10;
     */
    intervalCount: number;
    /**
     * @generated from field: string description = 11;
     */
    description: string;
    /**
     * @generated from field: google.protobuf.Timestamp created = 12;
     */
    created?: Timestamp;
};
/**
 * Describes the message obiente.cloud.billing.v1.Subscription.
 * Use `create(SubscriptionSchema)` to create a new message.
 */
export declare const SubscriptionSchema: GenMessage<Subscription>;
/**
 * @generated from message obiente.cloud.billing.v1.UpdateSubscriptionPaymentMethodRequest
 */
export type UpdateSubscriptionPaymentMethodRequest = Message<"obiente.cloud.billing.v1.UpdateSubscriptionPaymentMethodRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string subscription_id = 2;
     */
    subscriptionId: string;
    /**
     * @generated from field: string payment_method_id = 3;
     */
    paymentMethodId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.UpdateSubscriptionPaymentMethodRequest.
 * Use `create(UpdateSubscriptionPaymentMethodRequestSchema)` to create a new message.
 */
export declare const UpdateSubscriptionPaymentMethodRequestSchema: GenMessage<UpdateSubscriptionPaymentMethodRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.UpdateSubscriptionPaymentMethodResponse
 */
export type UpdateSubscriptionPaymentMethodResponse = Message<"obiente.cloud.billing.v1.UpdateSubscriptionPaymentMethodResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: obiente.cloud.billing.v1.Subscription subscription = 2;
     */
    subscription?: Subscription;
};
/**
 * Describes the message obiente.cloud.billing.v1.UpdateSubscriptionPaymentMethodResponse.
 * Use `create(UpdateSubscriptionPaymentMethodResponseSchema)` to create a new message.
 */
export declare const UpdateSubscriptionPaymentMethodResponseSchema: GenMessage<UpdateSubscriptionPaymentMethodResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.CancelSubscriptionRequest
 */
export type CancelSubscriptionRequest = Message<"obiente.cloud.billing.v1.CancelSubscriptionRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string subscription_id = 2;
     */
    subscriptionId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.CancelSubscriptionRequest.
 * Use `create(CancelSubscriptionRequestSchema)` to create a new message.
 */
export declare const CancelSubscriptionRequestSchema: GenMessage<CancelSubscriptionRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.CancelSubscriptionResponse
 */
export type CancelSubscriptionResponse = Message<"obiente.cloud.billing.v1.CancelSubscriptionResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * Status message
     *
     * @generated from field: string message = 2;
     */
    message: string;
    /**
     * @generated from field: obiente.cloud.billing.v1.Subscription subscription = 3;
     */
    subscription?: Subscription;
};
/**
 * Describes the message obiente.cloud.billing.v1.CancelSubscriptionResponse.
 * Use `create(CancelSubscriptionResponseSchema)` to create a new message.
 */
export declare const CancelSubscriptionResponseSchema: GenMessage<CancelSubscriptionResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.PayBillRequest
 */
export type PayBillRequest = Message<"obiente.cloud.billing.v1.PayBillRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string bill_id = 2;
     */
    billId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.PayBillRequest.
 * Use `create(PayBillRequestSchema)` to create a new message.
 */
export declare const PayBillRequestSchema: GenMessage<PayBillRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.PayBillResponse
 */
export type PayBillResponse = Message<"obiente.cloud.billing.v1.PayBillResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * Status message
     *
     * @generated from field: string message = 2;
     */
    message: string;
    /**
     * @generated from field: obiente.cloud.billing.v1.MonthlyBill bill = 3;
     */
    bill?: MonthlyBill;
};
/**
 * Describes the message obiente.cloud.billing.v1.PayBillResponse.
 * Use `create(PayBillResponseSchema)` to create a new message.
 */
export declare const PayBillResponseSchema: GenMessage<PayBillResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.ListBillsRequest
 */
export type ListBillsRequest = Message<"obiente.cloud.billing.v1.ListBillsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: limit the number of bills returned (default: 10, max: 100)
     *
     * @generated from field: optional int32 limit = 2;
     */
    limit?: number;
};
/**
 * Describes the message obiente.cloud.billing.v1.ListBillsRequest.
 * Use `create(ListBillsRequestSchema)` to create a new message.
 */
export declare const ListBillsRequestSchema: GenMessage<ListBillsRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.ListBillsResponse
 */
export type ListBillsResponse = Message<"obiente.cloud.billing.v1.ListBillsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.billing.v1.MonthlyBill bills = 1;
     */
    bills: MonthlyBill[];
    /**
     * Whether there are more bills available
     *
     * @generated from field: bool has_more = 2;
     */
    hasMore: boolean;
};
/**
 * Describes the message obiente.cloud.billing.v1.ListBillsResponse.
 * Use `create(ListBillsResponseSchema)` to create a new message.
 */
export declare const ListBillsResponseSchema: GenMessage<ListBillsResponse>;
/**
 * @generated from message obiente.cloud.billing.v1.MonthlyBill
 */
export type MonthlyBill = Message<"obiente.cloud.billing.v1.MonthlyBill"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: google.protobuf.Timestamp billing_period_start = 3;
     */
    billingPeriodStart?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp billing_period_end = 4;
     */
    billingPeriodEnd?: Timestamp;
    /**
     * @generated from field: int64 amount_cents = 5;
     */
    amountCents: bigint;
    /**
     * "PENDING", "PAID", "FAILED", "CANCELLED"
     *
     * @generated from field: string status = 6;
     */
    status: string;
    /**
     * @generated from field: optional google.protobuf.Timestamp paid_at = 7;
     */
    paidAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp due_date = 8;
     */
    dueDate?: Timestamp;
    /**
     * Usage breakdown (JSON string with cost breakdown)
     *
     * @generated from field: optional string usage_breakdown = 9;
     */
    usageBreakdown?: string;
    /**
     * @generated from field: optional string note = 10;
     */
    note?: string;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 11;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 12;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.billing.v1.MonthlyBill.
 * Use `create(MonthlyBillSchema)` to create a new message.
 */
export declare const MonthlyBillSchema: GenMessage<MonthlyBill>;
/**
 * @generated from message obiente.cloud.billing.v1.GenerateCurrentBillRequest
 */
export type GenerateCurrentBillRequest = Message<"obiente.cloud.billing.v1.GenerateCurrentBillRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.billing.v1.GenerateCurrentBillRequest.
 * Use `create(GenerateCurrentBillRequestSchema)` to create a new message.
 */
export declare const GenerateCurrentBillRequestSchema: GenMessage<GenerateCurrentBillRequest>;
/**
 * @generated from message obiente.cloud.billing.v1.GenerateCurrentBillResponse
 */
export type GenerateCurrentBillResponse = Message<"obiente.cloud.billing.v1.GenerateCurrentBillResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * Status message
     *
     * @generated from field: string message = 2;
     */
    message: string;
    /**
     * The generated bill (or existing bill if one already exists)
     *
     * @generated from field: obiente.cloud.billing.v1.MonthlyBill bill = 3;
     */
    bill?: MonthlyBill;
    /**
     * Whether the bill already existed
     *
     * @generated from field: bool already_exists = 4;
     */
    alreadyExists: boolean;
};
/**
 * Describes the message obiente.cloud.billing.v1.GenerateCurrentBillResponse.
 * Use `create(GenerateCurrentBillResponseSchema)` to create a new message.
 */
export declare const GenerateCurrentBillResponseSchema: GenMessage<GenerateCurrentBillResponse>;
/**
 * @generated from service obiente.cloud.billing.v1.BillingService
 */
export declare const BillingService: GenService<{
    /**
     * Create a Stripe Checkout Session for purchasing credits
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.CreateCheckoutSession
     */
    createCheckoutSession: {
        methodKind: "unary";
        input: typeof CreateCheckoutSessionRequestSchema;
        output: typeof CreateCheckoutSessionResponseSchema;
    };
    /**
     * Create a Payment Intent for purchasing credits using an existing payment method
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.CreatePaymentIntent
     */
    createPaymentIntent: {
        methodKind: "unary";
        input: typeof CreatePaymentIntentRequestSchema;
        output: typeof CreatePaymentIntentResponseSchema;
    };
    /**
     * Create a Stripe Customer Portal session for managing billing
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.CreatePortalSession
     */
    createPortalSession: {
        methodKind: "unary";
        input: typeof CreatePortalSessionRequestSchema;
        output: typeof CreatePortalSessionResponseSchema;
    };
    /**
     * Create a Setup Intent for collecting payment methods without a payment
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.CreateSetupIntent
     */
    createSetupIntent: {
        methodKind: "unary";
        input: typeof CreateSetupIntentRequestSchema;
        output: typeof CreateSetupIntentResponseSchema;
    };
    /**
     * Get billing account information for an organization
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.GetBillingAccount
     */
    getBillingAccount: {
        methodKind: "unary";
        input: typeof GetBillingAccountRequestSchema;
        output: typeof GetBillingAccountResponseSchema;
    };
    /**
     * Create or update billing account
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.UpdateBillingAccount
     */
    updateBillingAccount: {
        methodKind: "unary";
        input: typeof UpdateBillingAccountRequestSchema;
        output: typeof UpdateBillingAccountResponseSchema;
    };
    /**
     * List payment methods for an organization
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.ListPaymentMethods
     */
    listPaymentMethods: {
        methodKind: "unary";
        input: typeof ListPaymentMethodsRequestSchema;
        output: typeof ListPaymentMethodsResponseSchema;
    };
    /**
     * Attach a payment method to customer
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.AttachPaymentMethod
     */
    attachPaymentMethod: {
        methodKind: "unary";
        input: typeof AttachPaymentMethodRequestSchema;
        output: typeof AttachPaymentMethodResponseSchema;
    };
    /**
     * Detach a payment method from customer
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.DetachPaymentMethod
     */
    detachPaymentMethod: {
        methodKind: "unary";
        input: typeof DetachPaymentMethodRequestSchema;
        output: typeof DetachPaymentMethodResponseSchema;
    };
    /**
     * Set default payment method for customer
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.SetDefaultPaymentMethod
     */
    setDefaultPaymentMethod: {
        methodKind: "unary";
        input: typeof SetDefaultPaymentMethodRequestSchema;
        output: typeof SetDefaultPaymentMethodResponseSchema;
    };
    /**
     * Get payment intent status
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.GetPaymentStatus
     */
    getPaymentStatus: {
        methodKind: "unary";
        input: typeof GetPaymentStatusRequestSchema;
        output: typeof GetPaymentStatusResponseSchema;
    };
    /**
     * List invoices for an organization
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.ListInvoices
     */
    listInvoices: {
        methodKind: "unary";
        input: typeof ListInvoicesRequestSchema;
        output: typeof ListInvoicesResponseSchema;
    };
    /**
     * Create a Stripe Checkout Session for DNS Delegation subscription ($2/month)
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.CreateDNSDelegationSubscriptionCheckout
     */
    createDNSDelegationSubscriptionCheckout: {
        methodKind: "unary";
        input: typeof CreateDNSDelegationSubscriptionCheckoutRequestSchema;
        output: typeof CreateDNSDelegationSubscriptionCheckoutResponseSchema;
    };
    /**
     * Get DNS delegation subscription status for an organization
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.GetDNSDelegationSubscriptionStatus
     */
    getDNSDelegationSubscriptionStatus: {
        methodKind: "unary";
        input: typeof GetDNSDelegationSubscriptionStatusRequestSchema;
        output: typeof GetDNSDelegationSubscriptionStatusResponseSchema;
    };
    /**
     * Cancel DNS delegation subscription
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.CancelDNSDelegationSubscription
     */
    cancelDNSDelegationSubscription: {
        methodKind: "unary";
        input: typeof CancelDNSDelegationSubscriptionRequestSchema;
        output: typeof CancelDNSDelegationSubscriptionResponseSchema;
    };
    /**
     * List all subscriptions for an organization
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.ListSubscriptions
     */
    listSubscriptions: {
        methodKind: "unary";
        input: typeof ListSubscriptionsRequestSchema;
        output: typeof ListSubscriptionsResponseSchema;
    };
    /**
     * Update subscription payment method
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.UpdateSubscriptionPaymentMethod
     */
    updateSubscriptionPaymentMethod: {
        methodKind: "unary";
        input: typeof UpdateSubscriptionPaymentMethodRequestSchema;
        output: typeof UpdateSubscriptionPaymentMethodResponseSchema;
    };
    /**
     * Cancel a subscription by ID
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.CancelSubscription
     */
    cancelSubscription: {
        methodKind: "unary";
        input: typeof CancelSubscriptionRequestSchema;
        output: typeof CancelSubscriptionResponseSchema;
    };
    /**
     * Pay a monthly bill prematurely (before due date)
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.PayBill
     */
    payBill: {
        methodKind: "unary";
        input: typeof PayBillRequestSchema;
        output: typeof PayBillResponseSchema;
    };
    /**
     * List monthly bills for an organization
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.ListBills
     */
    listBills: {
        methodKind: "unary";
        input: typeof ListBillsRequestSchema;
        output: typeof ListBillsResponseSchema;
    };
    /**
     * Generate the current bill early (before billing date)
     * This allows users to create and pay their current bill before their scheduled billing date
     *
     * @generated from rpc obiente.cloud.billing.v1.BillingService.GenerateCurrentBill
     */
    generateCurrentBill: {
        methodKind: "unary";
        input: typeof GenerateCurrentBillRequestSchema;
        output: typeof GenerateCurrentBillResponseSchema;
    };
}>;
