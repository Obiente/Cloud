# Billing Service

Microservice for handling billing, payments, and Stripe integration.

## Features

- Stripe checkout session creation
- Stripe customer portal management
- Billing account management
- Monthly billing processing
- Monthly free credits grants
- Stripe webhook handling
- Invoice and bill management

## Port

Default port: `3004`

## Environment Variables

See shared configuration in `docker-compose.yml` for common variables.

### Service-Specific Variables

- `PORT` - Service port (default: 3004)
- `BILLING_ENABLED` - Enable/disable billing features (default: true)
- `STRIPE_SECRET_KEY` - Stripe API secret key (required for Stripe features)
- `STRIPE_WEBHOOK_SECRET` - Stripe webhook signing secret (required for webhook verification)
- `DASHBOARD_URL` - Dashboard URL for redirects (default: https://obiente.cloud)

## Endpoints

- `/obiente.cloud.billing.v1.BillingService/*` - Connect RPC endpoints
- `/webhooks/stripe` - Stripe webhook endpoint (no auth, uses signature verification)
- `/health` - Health check endpoint
- `/` - Service info

## Background Services

- **Monthly Billing**: Processes monthly bills for organizations (runs daily)
- **Monthly Credits**: Grants monthly free credits to organizations (runs daily)

## Dependencies

- PostgreSQL (main database)
- TimescaleDB (metrics database for usage stats)
- Stripe API (for payment processing)

