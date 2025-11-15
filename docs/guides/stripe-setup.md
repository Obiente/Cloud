# Stripe Payment Setup

Complete guide for setting up Stripe payment processing in Obiente Cloud.

## Overview

Obiente Cloud uses Stripe for payment processing, supporting:
- **Credit purchases** - One-time payments to add credits to organization accounts
- **Subscriptions** - Recurring payments for DNS delegation and other services
- **Automatic billing** - Usage-based billing with automatic invoice generation

## Prerequisites

- A Stripe account (create one at [stripe.com](https://stripe.com))
- Access to your Stripe Dashboard
- HTTPS endpoint for webhook delivery (required for production)

## Step 1: Get Stripe API Keys

1. Log in to your [Stripe Dashboard](https://dashboard.stripe.com)
2. Navigate to **Developers** > **API keys**
3. Copy the following keys:
   - **Secret key** (starts with `sk_test_` for test mode or `sk_live_` for live mode)
   - **Publishable key** (starts with `pk_test_` for test mode or `pk_live_` for live mode)

## Step 2: Configure Environment Variables

Add the following environment variables to your `.env` file:

```bash
# Stripe Configuration
STRIPE_SECRET_KEY=sk_test_...  # or sk_live_... for production
STRIPE_WEBHOOK_SECRET=whsec_...  # Will be set after webhook configuration
NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...  # or pk_live_... for production
```

**Important:**
- Use test keys (`sk_test_`, `pk_test_`) for development
- Use live keys (`sk_live_`, `pk_live_`) for production
- Never commit these keys to version control
- The publishable key is safe to expose (used client-side)

## Step 3: Configure Webhook Endpoint

### Production Setup

1. In Stripe Dashboard, go to **Developers** > **Webhooks**
2. Click **Add endpoint**
3. Enter your webhook URL:
   ```
   https://your-domain.com/webhooks/stripe
   ```
4. **Select events to listen to:**
   
   **Checkout Events:**
   - `checkout.session.completed` - Payment completed
   - `checkout.session.expired` - Checkout session expired
   - `checkout.session.async_payment_succeeded` - Async payment succeeded
   - `checkout.session.async_payment_failed` - Async payment failed
   
   **Subscription Events:**
   - `customer.subscription.created` - Subscription created (grants monthly free credits)
   - `customer.subscription.updated` - Subscription updated (grants monthly free credits if active)
   - `customer.subscription.deleted` - Subscription deleted
   - `customer.subscription.paused` - Subscription paused
   - `customer.subscription.resumed` - Subscription resumed (grants monthly free credits if active)
   - `customer.subscription.pending_update_applied` - Pending subscription update applied
   - `customer.subscription.pending_update_expired` - Pending subscription update expired
   - `customer.subscription.trial_will_end` - Trial ending soon
   
   **Invoice Events:**
   - `invoice.created` - Invoice created
   - `invoice.deleted` - Invoice deleted
   - `invoice.finalized` - Invoice finalized
   - `invoice.finalization_failed` - Invoice finalization failed
   - `invoice.marked_uncollectible` - Invoice marked uncollectible
   - `invoice.overdue` - Invoice overdue
   - `invoice.overpaid` - Invoice overpaid
   - `invoice.paid` - Invoice paid (adds credits)
   - `invoice.payment_action_required` - Payment requires action
   - `invoice.payment_failed` - Invoice payment failed
   - `invoice.payment_succeeded` - Invoice payment succeeded
   - `invoice.sent` - Invoice sent to customer
   - `invoice.upcoming` - Upcoming invoice
   - `invoice.updated` - Invoice updated
   - `invoice.voided` - Invoice voided
   
   **Payment Intent Events:**
   - `payment_intent.succeeded` - Payment succeeded
   - `payment_intent.payment_failed` - Payment failed
   
   **Subscription Schedule Events:**
   - `subscription_schedule.aborted` - Subscription schedule aborted
   - `subscription_schedule.canceled` - Subscription schedule canceled
   - `subscription_schedule.completed` - Subscription schedule completed
   - `subscription_schedule.created` - Subscription schedule created
   - `subscription_schedule.expiring` - Subscription schedule expiring
   - `subscription_schedule.released` - Subscription schedule released
   - `subscription_schedule.updated` - Subscription schedule updated

5. **Set API version:** Select `2025-10-29.clover` (must match SDK version)
6. **Set payload format:** Select **Snapshot** (recommended)
7. Click **Add endpoint**
8. Copy the **Signing secret** (starts with `whsec_`) and set it as `STRIPE_WEBHOOK_SECRET`

### Development Setup

For local development, use Stripe CLI to forward webhooks:

1. **Install Stripe CLI:**
   ```bash
   # macOS
   brew install stripe/stripe-cli/stripe
   
   # Linux
   wget https://github.com/stripe/stripe-cli/releases/latest/download/stripe_*_linux_x86_64.tar.gz
   tar -xvf stripe_*_linux_x86_64.tar.gz
   sudo mv stripe /usr/local/bin/
   
   # Windows
   # Download from https://github.com/stripe/stripe-cli/releases
   ```

2. **Login to Stripe:**
   ```bash
   stripe login
   ```

3. **Forward webhooks to local server:**
   ```bash
   stripe listen --forward-to localhost:3001/webhooks/stripe
   ```

4. **Copy the webhook signing secret** from the CLI output and set it as `STRIPE_WEBHOOK_SECRET` in your `.env` file

## Step 4: Verify Webhook Configuration

1. In Stripe Dashboard, go to **Developers** > **Webhooks**
2. Click on your webhook endpoint
3. Click **Send test webhook**
4. Select an event type (e.g., `checkout.session.completed`)
5. Click **Send test webhook**
6. Check your application logs to verify the webhook was received and processed

## Webhook Event Handling

The system handles the following webhook events:

### Payment Events

- **`checkout.session.completed`** - Processes one-time credit purchases
- **`checkout.session.expired`** - Logs expired checkout sessions
- **`checkout.session.async_payment_succeeded`** - Processes successful async payments (bank transfers, etc.)
- **`checkout.session.async_payment_failed`** - Logs failed async payments
- **`payment_intent.succeeded`** - Logs successful payments
- **`payment_intent.payment_failed`** - Logs failed payments

### Subscription Events

- **`customer.subscription.created`** - Creates DNS delegation API keys and grants monthly free credits from plan
- **`customer.subscription.updated`** - Updates subscription status, API keys, and grants monthly free credits if active
- **`customer.subscription.deleted`** - Revokes API keys when subscription is deleted
- **`customer.subscription.paused`** - Logs when subscription is paused
- **`customer.subscription.resumed`** - Grants monthly free credits when subscription is resumed and active
- **`customer.subscription.pending_update_applied`** - Logs when pending subscription update is applied
- **`customer.subscription.pending_update_expired`** - Logs when pending subscription update expires
- **`customer.subscription.trial_will_end`** - Logs when trial is ending (with days remaining)

### Invoice Events

- **`invoice.created`** - Logs when invoice is created
- **`invoice.deleted`** - Logs when invoice is deleted
- **`invoice.finalized`** - Logs when invoice is finalized
- **`invoice.finalization_failed`** - Logs invoice finalization failures
- **`invoice.marked_uncollectible`** - Updates billing account status to UNCOLLECTIBLE
- **`invoice.overdue`** - Logs when invoice becomes overdue
- **`invoice.overpaid`** - Logs when invoice is overpaid
- **`invoice.paid`** - Processes invoice payments and adds credits
- **`invoice.payment_action_required`** - Logs when payment requires user action (3D Secure, etc.)
- **`invoice.payment_failed`** - Updates billing account status on payment failure
- **`invoice.payment_succeeded`** - Logs successful invoice payments
- **`invoice.sent`** - Logs when invoice is sent to customer
- **`invoice.upcoming`** - Logs upcoming invoices (can be used for notifications)
- **`invoice.updated`** - Logs when invoice is updated
- **`invoice.voided`** - Logs when invoice is voided

### Subscription Schedule Events

- **`subscription_schedule.aborted`** - Logs when subscription schedule is aborted
- **`subscription_schedule.canceled`** - Logs when subscription schedule is canceled
- **`subscription_schedule.completed`** - Logs when subscription schedule is completed
- **`subscription_schedule.created`** - Logs when subscription schedule is created
- **`subscription_schedule.expiring`** - Logs when subscription schedule is expiring
- **`subscription_schedule.released`** - Logs when subscription schedule is released
- **`subscription_schedule.updated`** - Logs when subscription schedule is updated

## Monthly Free Credits via Stripe

The system automatically grants monthly free credits from organization plans when:

1. **Subscription Created** - When a subscription becomes active, monthly free credits are granted based on the organization's plan
2. **Subscription Updated** - When a subscription becomes active again (after being paused/canceled), monthly free credits are granted
3. **Subscription Resumed** - When a subscription is resumed and becomes active, monthly free credits are granted

**Important Notes:**
- Credits are granted only once per month per organization
- The amount is determined by the `MonthlyFreeCreditsCents` field in the organization's plan
- Credits are tracked in the `monthly_credit_grants` table to prevent duplicate grants
- If an organization has no plan assigned or the plan has no free credits, no credits are granted

## Webhook Security

The webhook handler implements several security measures:

1. **Signature Verification** - All webhooks are verified using the signing secret
2. **Idempotency** - Events are tracked in the database to prevent duplicate processing
3. **HTTPS Only** - Production webhooks must use HTTPS
4. **Rate Limiting** - Stripe automatically retries failed webhooks with exponential backoff

## Testing

### Test Mode

1. Use test API keys (`sk_test_`, `pk_test_`)
2. Use Stripe CLI for local webhook forwarding
3. Use test card numbers from [Stripe Testing](https://stripe.com/docs/testing)

**Test Card Numbers:**
- Success: `4242 4242 4242 4242`
- Decline: `4000 0000 0000 0002`
- Requires authentication: `4000 0025 0000 3155`

### Live Mode

1. Switch to live API keys (`sk_live_`, `pk_live_`)
2. Configure production webhook endpoint
3. Test with small amounts first

## Troubleshooting

### Webhook Not Received

1. **Check endpoint URL** - Ensure it's accessible via HTTPS (production) or HTTP (local with Stripe CLI)
2. **Verify webhook secret** - Ensure `STRIPE_WEBHOOK_SECRET` matches the signing secret from Stripe
3. **Check firewall** - Ensure Stripe can reach your endpoint
4. **Check logs** - Look for webhook signature verification errors

### Webhook Signature Verification Failed

1. **Verify webhook secret** - Ensure `STRIPE_WEBHOOK_SECRET` is correct
2. **Check API version** - Ensure webhook endpoint uses API version `2025-10-29.clover`
3. **Check payload format** - Ensure using "Snapshot" format

### Events Not Processing

1. **Check event types** - Ensure all required events are selected in Stripe Dashboard
2. **Check logs** - Look for parsing or processing errors
3. **Check idempotency** - Events may have already been processed (check `stripe_webhook_events` table)

### Payment Not Adding Credits

1. **Check metadata** - Ensure `organization_id` is included in checkout session metadata
2. **Check customer ID** - Ensure customer is linked to billing account
3. **Check logs** - Look for errors in `addCreditsFromPayment` function

## Best Practices

1. **Use Test Mode First** - Always test in test mode before going live
2. **Monitor Webhooks** - Set up alerts for webhook failures
3. **Log All Events** - Keep logs of all webhook events for debugging
4. **Handle Failures Gracefully** - Implement retry logic for transient failures
5. **Keep Secrets Secure** - Never commit API keys or webhook secrets to version control
6. **Use Environment Variables** - Store all Stripe configuration in environment variables
7. **Test Webhook Delivery** - Use Stripe Dashboard to send test webhooks
8. **Monitor Stripe Dashboard** - Regularly check webhook delivery status

## API Version Compatibility

The system uses Stripe API version `2025-10-29.clover`. When configuring webhooks in Stripe Dashboard:

1. Go to **Developers** > **Webhooks**
2. Select your webhook endpoint
3. Click **Settings**
4. Under **API version**, select `2025-10-29.clover`
5. Save changes

**Important:** The webhook API version must match the SDK version to ensure compatibility.

## Payload Format: Snapshot vs Thin

The system is configured to use **Snapshot** payloads (recommended):

- **Snapshot** - Full object data included (current implementation)
- **Thin** - Only object ID included (requires additional API calls)

**Recommendation:** Use Snapshot format for simplicity and reliability.

## Additional Resources

- [Stripe Webhooks Documentation](https://stripe.com/docs/webhooks)
- [Stripe Testing Guide](https://stripe.com/docs/testing)
- [Stripe API Reference](https://stripe.com/docs/api)
- [Stripe CLI Documentation](https://stripe.com/docs/stripe-cli)

---

[← Back to Guides](index.md)

