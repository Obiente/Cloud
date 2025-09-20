# GitHub Copilot Instructions - Obiente Cloud Dashboard

## Project Overview
Multi-tenant cloud dashboard platform for Obiente Cloud offering web hosting and VPS management. Users can manage deployments, invite team members with role-based permissions, and monitor billing across all services.

## Tech Stack
- **Frontend**: Vue.js 3, Nuxt 3, TypeScript
- **Backend**: Node.js, TypeScript, Protobuf ConnectRPC
- **Database**: PostgreSQL (future: CockroachDB)
- **Authentication**: Zitadel (OIDC/OAuth2)
- **Billing**: Stripe
- **UI Components**: Ark UI (headless components)
- **Styling**: Minimal for now, future: obiente-ui
- **Testing**: Vitest, Playwright
- **State Management**: Pinia
- **Monorepo**: Turborepo with pnpm workspaces
- **API Protocol**: Protobuf ConnectRPC (type-safe)

## Architecture
- **Project Structure**: Turborepo monorepo with pnpm workspaces
- **Multi-tenancy**: Schema-based with Row Level Security (RLS)
- **API Design**: Protobuf ConnectRPC with type-safe generated clients
- **Real-time**: Server streaming via ConnectRPC for status updates

## Key Entities
- **Organization**: Multi-tenant container
- **User**: Individual accounts with org memberships
- **Deployment**: Web hosting instances
- **VPSInstance**: Virtual private servers
- **Database**: Managed database instances
- **BillingAccount**: Financial tracking
- **Permission**: Granular access control

## Development Guidelines
- Follow TDD approach: tests first, then implementation
- Use TypeScript for type safety
- Implement proper error handling and validation
- Follow REST API conventions
- Ensure multi-tenant data isolation
- Use RLS for database security

## Recent Changes
- Phase 0: Research completed with technology stack decisions
- Phase 1: Data model and API contracts designed
- Contract tests to be implemented (failing initially)
- Quickstart guide created for development workflow

## File Structure
```
apps/
├── dashboard/            # Nuxt 3 frontend app
│   ├── components/      # Vue components
│   ├── pages/          # Nuxt pages
│   ├── stores/         # Pinia stores
│   └── composables/    # Vue composables
├── api/                # Node.js backend API
│   ├── src/
│   │   ├── services/   # Business logic
│   │   ├── models/     # Database models
│   │   ├── middleware/ # Auth, validation
│   │   └── proto/      # Protobuf definitions
│   └── tests/
└── docs/               # Documentation

packages/
├── ui/                 # Shared UI components
├── proto/              # Shared protobuf definitions
├── config/             # Shared configs
└── types/              # Shared TypeScript types

specs/001-obiente-cloud-dashboard/
├── spec.md             # Feature specification
├── plan.md             # Implementation plan
├── research.md         # Technology research
├── data-model.md       # Database design
├── quickstart.md       # Development guide
└── contracts/          # API specifications
```

## Key Implementation Notes
- Use Zitadel SDK for authentication flows
- Implement Stripe webhooks for billing events
- Create multi-tenant middleware for request isolation
- Use Ark UI components as base for custom components
- Implement real-time updates via ConnectRPC streaming
- Follow protobuf schema definitions in contracts/ directory
- Use turborepo for build caching and task parallelization
- Generate type-safe API clients from protobuf definitions

## Testing Strategy
- Contract tests for API endpoints
- Unit tests for business logic
- Integration tests for user flows
- E2E tests with Playwright
- Load testing for performance validation

---
*Generated for feature 001-obiente-cloud-dashboard | Updated: 2025-09-18*