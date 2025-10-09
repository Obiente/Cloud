# Obiente Cloud API

Backend API server for the Obiente Cloud Dashboard platform.

## Overview

This API server provides the backend services for the multi-tenant cloud dashboard. It's built with:

- **Fastify** - High-performance web framework
- **ConnectRPC** - Type-safe RPC protocol with Protobuf
- **Zitadel** - Authentication and authorization
- **Drizzle ORM** - Database operations with PostgreSQL
- **TypeScript** - Type safety throughout

## Architecture

### Tech Stack
- **Framework**: Fastify for HTTP server
- **RPC Protocol**: ConnectRPC with Protocol Buffers
- **Authentication**: Zitadel OIDC/OAuth2
- **Database**: PostgreSQL with Drizzle ORM
- **Validation**: Zod for runtime validation
- **Security**: JWT tokens, rate limiting, CORS

### Project Structure
```
src/
├── config/           # Configuration management
├── plugins/          # Fastify plugins (auth, logging, error handling)
├── routes/           # ConnectRPC route handlers
├── services/         # Business logic services
├── middleware/       # Custom middleware
└── server.ts         # Main server entry point
```

## Features

### Multi-tenancy
- Organization-based tenant isolation
- Row Level Security (RLS) in database
- Tenant-specific resource access control

### Authentication & Authorization
- Zitadel integration for OIDC/OAuth2
- JWT token validation
- Role-based access control (RBAC)
- API key authentication for service-to-service calls

### API Protocol
- ConnectRPC for type-safe communication
- Protocol Buffer schema definitions
- Automatic TypeScript client generation
- Streaming support for real-time updates

### Security
- Rate limiting per IP/user
- CORS configuration
- Request validation with Zod
- Security headers with Helmet
- Audit logging

## Development

### Prerequisites
- Node.js 18+ 
- PostgreSQL 14+
- Zitadel instance (for authentication)

### Environment Setup
1. Copy `.env.example` to `.env`
2. Configure database connection
3. Set up Zitadel authentication
4. Configure Stripe for billing (optional)

### Available Scripts
```bash
# Development
pnpm dev              # Start development server with hot reload
pnpm build            # Build for production
pnpm start            # Start production server

# Database
pnpm db:migrate       # Run database migrations
pnpm db:seed          # Seed database with test data

# Code Quality
pnpm typecheck        # Type checking
pnpm clean            # Clean build artifacts
```

### API Documentation
When running in development mode, Swagger documentation is available at:
- http://localhost:3001/docs

## Deployment

### Production Build
```bash
pnpm build
pnpm start
```

### Environment Variables
See `.env.example` for all required environment variables.

### Health Checks
- `GET /health` - Basic health check endpoint

## Future Enhancements

1. **ConnectRPC Services**: Implement full service handlers for all business domains
2. **Real-time Updates**: Add server streaming for live resource status
3. **Monitoring**: Add metrics collection and observability
4. **Caching**: Implement Redis for session and data caching
5. **Testing**: Add comprehensive test coverage
6. **Docker**: Add containerization for deployment