# API Gateway

Single entry point for all client requests, routing to appropriate microservices.

## Features

- Request routing to microservices
- Authentication/authorization (JWT validation)
- CORS handling
- Request/response logging
- Health check aggregation
- WebSocket forwarding

## Port

Default port: `3001`

## Environment Variables

See shared configuration in `docker-compose.yml` for common variables.

### Service-Specific Variables

- `PORT` - Service port (default: 3001)

## Routing

The gateway routes requests based on path prefixes:

- `/obiente.cloud.auth.v1.AuthService/*` → `auth-service:3002`
- `/obiente.cloud.organizations.v1.OrganizationService/*` → `organizations-service:3003`
- `/obiente.cloud.billing.v1.BillingService/*` → `billing-service:3004`
- `/obiente.cloud.deployments.v1.DeploymentService/*` → `deployments-service:3005`
- `/obiente.cloud.gameservers.v1.GameServerService/*` → `gameservers-service:3006`
- `/obiente.cloud.vps.v1.VPSService/*` → `vps-service:3008`
- `/obiente.cloud.superadmin.v1.SuperadminService/*` → `superadmin-service:3009`
- `/obiente.cloud.support.v1.SupportService/*` → `support-service:3009`
- `/obiente.cloud.audit.v1.AuditService/*` → `audit-service:3010`
- `/webhooks/stripe` → `billing-service:3004`
- `/dns/push` → `api:3001` (DNS delegation push endpoint - main API)
- `/dns/push/batch` → `api:3001` (DNS delegation batch push endpoint - main API)
- `/terminal/ws` → `deployments-service:3005`
- `/gameservers/terminal/ws` → `gameservers-service:3006`
- `/vps/terminal/ws` → `vps-service:3008`
- `/vps/ssh/` → `vps-service:3008`

## Dependencies

- All microservices (for routing)
- Auth service (for token validation)

## Notes

- This is the main entry point for all API requests
- Clients should connect to the API Gateway, not individual services
- The gateway forwards requests to backend services and returns responses

