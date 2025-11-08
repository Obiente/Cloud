# Obiente Cloud Dashboard

Frontend dashboard application for the Obiente Cloud platform built with Nuxt 3.

## Overview

This is the main user interface for the Obiente Cloud multi-tenant dashboard. It provides a modern, responsive web application for managing cloud resources including deployments, VPS instances, databases, and billing.

## Tech Stack

### Core Framework

- **Nuxt 3** - Vue.js meta-framework with SSR/SSG
- **Vue 3** - Progressive JavaScript framework
- **TypeScript** - Type safety throughout the application
- **Pinia** - State management for Vue

### UI & Styling

- **Nuxt UI** - Component library built on Tailwind CSS
- **Tailwind CSS** - Utility-first CSS framework
- **Headless UI** - Unstyled, accessible UI components
- **Heroicons** - Beautiful hand-crafted SVG icons
- **@obiente/ui** - Custom component library with Ark UI

### Development Tools

- **Vue DevTools** - Browser extension for debugging
- **ESLint** - Code linting and formatting
- **TypeScript compiler** - Type checking

## Features

### Multi-tenant Architecture

- Organization-based tenant isolation
- Dynamic organization switching
- Role-based access control (RBAC)

### Authentication & Authorization

- Zitadel OIDC/OAuth2 integration
- JWT token management
- SSO (Single Sign-On) support
- Secure session handling

### Resource Management

- **Deployments**: Web application hosting management
- **VPS Instances**: Virtual private server administration
- **Databases**: Managed database service control
- **Billing**: Subscription and usage tracking

### User Experience

- Responsive design for all devices
- Real-time status updates
- Intuitive navigation and workflows
- Comprehensive search and filtering

## Project Structure

```
├── assets/              # Static assets (CSS, images)
├── components/          # Vue components
├── composables/         # Vue composables (business logic)
├── layouts/             # Nuxt layouts
├── middleware/          # Route middleware
├── pages/               # File-based routing
├── plugins/             # Nuxt plugins
├── stores/              # Pinia stores (state management)
├── types/               # TypeScript type definitions
├── utils/               # Utility functions
├── nuxt.config.ts       # Nuxt configuration
└── package.json         # Dependencies and scripts
```

## Development

### Prerequisites

- Node.js 18+
- pnpm (package manager)

### Environment Setup

1. Copy `.env.example` to `.env`
2. Configure API and authentication endpoints
3. Set up Zitadel OIDC configuration

### Available Scripts

```bash
# Development
pnpm dev              # Start development server
pnpm build            # Build for production
pnpm preview          # Preview production build
pnpm generate         # Generate static site

# Code Quality
pnpm typecheck        # Run TypeScript checks
pnpm lint             # Run ESLint
pnpm lint:fix         # Fix ESLint issues
pnpm clean            # Clean build artifacts
```

### Development Server

The application runs on `http://localhost:3000` by default.

## Configuration

### Runtime Config

Environment variables are managed through Nuxt's runtime config:

```typescript
// nuxt.config.ts
runtimeConfig: {
  // Private (server-side only)
  apiSecret: process.env.API_SECRET,

  // Public (client-side)
  public: {
    apiBaseUrl: process.env.API_BASE_URL || 'http://localhost:3001',
    zitadelUrl: process.env.ZITADEL_URL,
    zitadelClientId: process.env.ZITADEL_CLIENT_ID,
  }
}
```

### Key Environment Variables

- `API_BASE_URL` - Backend API endpoint
- `ZITADEL_URL` - Zitadel authentication server
- `ZITADEL_CLIENT_ID` - OAuth2 client identifier

## Authentication Flow

1. **Login Page**: User enters credentials or uses SSO
2. **OIDC Redirect**: Zitadel handles authentication
3. **Token Exchange**: Receive JWT tokens
4. **Dashboard Access**: Authenticated user accesses resources

## State Management

### Pinia Stores

- **Auth Store**: User authentication and session
- **Organization Store**: Current organization context
- **Resource Stores**: Deployments, VPS, databases state

### Composables

- `useAuth()` - Authentication logic
- `useApi()` - API communication
- `useNotifications()` - User feedback system

## Styling Guidelines

### Tailwind CSS Classes

- Use utility classes for styling
- Custom components in `@obiente/ui` package
- Consistent color palette and spacing

### Component Architecture

- Small, focused components
- Props and events for communication
- TypeScript interfaces for type safety

## Deployment

### Production Build

```bash
pnpm build
pnpm preview
```

### Static Generation

```bash
pnpm generate
```

### Docker Deployment

The dashboard includes a production-ready Dockerfile with heavy caching optimizations for faster builds.

#### Prerequisites

- Docker 24.0+ with BuildKit enabled
- Docker Compose (optional)

#### Build with Docker

**Basic Build Command:**

```bash
# From the workspace root directory
# Enable BuildKit for optimal caching (required for cache mounts)
export DOCKER_BUILDKIT=1

# Build the image
docker build \
  -f apps/dashboard/Dockerfile \
  -t obiente/cloud-dashboard:test \
  --build-arg DASHBOARD_COMMIT=$(git rev-parse HEAD) \
  .
```

**Using Docker Buildx (Recommended):**

```bash
# Create a buildx builder if you don't have one
docker buildx create --name mybuilder --use

# Build with BuildKit (enabled by default with buildx)
docker buildx build \
  -f apps/dashboard/Dockerfile \
  -t obiente/cloud-dashboard:test \
  --build-arg DASHBOARD_COMMIT=$(git rev-parse HEAD) \
  --load \
  .
```

**Test Cache Effectiveness:**

```bash
# First build (no cache)
echo "=== First build (cold cache) ==="
time DOCKER_BUILDKIT=1 docker build \
  -f apps/dashboard/Dockerfile \
  -t obiente/cloud-dashboard:test \
  --build-arg DASHBOARD_COMMIT=$(git rev-parse HEAD) \
  .

# Make a small change to trigger rebuild (e.g., add a comment to a file)
# Then rebuild to see cache effectiveness
echo "=== Second build (warm cache) ==="
time DOCKER_BUILDKIT=1 docker build \
  -f apps/dashboard/Dockerfile \
  -t obiente/cloud-dashboard:test \
  --build-arg DASHBOARD_COMMIT=$(git rev-parse HEAD) \
  .
```

**Build and Run Locally:**

```bash
# Build
DOCKER_BUILDKIT=1 docker build \
  -f apps/dashboard/Dockerfile \
  -t obiente/cloud-dashboard:test \
  --build-arg DASHBOARD_COMMIT=$(git rev-parse HEAD) \
  .

# Run
docker run -p 3000:3000 obiente/cloud-dashboard:test
```

**Quick One-Liner:**

```bash
DOCKER_BUILDKIT=1 docker build \
  -f apps/dashboard/Dockerfile \
  -t dashboard:test \
  --build-arg DASHBOARD_COMMIT=$(git rev-parse --short HEAD) \
  .
```

**Using Docker Compose:**

```bash
# Build with docker-compose
docker compose -f docker-compose.dashboard.yml build
```

**Notes:**
- `DOCKER_BUILDKIT=1` is required to enable BuildKit cache mounts for optimal performance
- The context is `.` (workspace root) because the Dockerfile copies from the workspace
- `--build-arg DASHBOARD_COMMIT` is optional but recommended for tracking builds
- Cache mounts (pnpm store, Nuxt, Nitro, Vite, Nx) persist across builds on the same machine

#### Development with Docker Swarm

**For development on manager nodes only** (worker nodes cannot deploy stacks):

```bash
# Verify you're a manager node (not a worker)
docker node ls  # Should work, not show "not a manager" error

# Build images first (required before deploying)
export DOCKER_BUILDKIT=1
docker build -f apps/api/Dockerfile -t obiente/cloud-api:latest .

# Set main deployment DNS server IP (replace with your actual DNS server IP)
export MAIN_DNS_IP=10.0.9.10  # Replace with your main deployment's DNS server IP

# Deploy development stack using docker stack deploy (NOT docker compose)
docker stack deploy -c docker-compose.swarm.dev.yml obiente-dev

# View logs
docker service logs -f obiente-dev_api

# List services
docker stack services obiente-dev

# Remove stack
docker stack rm obiente-dev
```

**Note**: The `docker-compose.swarm.dev.yml` file uses Swarm-specific features (overlay networks) and **must** be deployed with `docker stack deploy`, not `docker compose`. Docker Swarm doesn't support building images during deployment - you must build them first.

**Worker Nodes**: If you're on a worker node, use regular `docker compose` with production DNS instead (see Docker Compose section above).

**DNS Configuration**: The development stack uses the main deployment's DNS server. **Important**: Production DNS queries the production database, so it can only resolve production deployments, not dev deployments. For dev deployments to resolve, use local DNS (default) or see [DNS Development Guide](../../docs/deployment/dns-development.md).

You have two options:

**Option 1: Set DNS IP directly** (simple but requires knowing the IP):
```bash
export MAIN_DNS_IP=10.0.9.10  # Replace with your main deployment's DNS server IP
```

**Option 2: Connect to main deployment's network** (recommended):
1. Find your main deployment's network name:
   ```bash
   docker network ls | grep obiente
   ```

2. In `docker-compose.swarm.dev.yml`, uncomment the main deployment network:
   ```yaml
   networks:
     main-deployment-network:
       external: true
       name: obiente_obiente-network  # Replace with your actual network name
   ```

3. Update the API service to connect to it:
   ```yaml
   api:
     networks:
       - obiente-network
       - main-deployment-network  # Add this line
   ```

   This allows DNS resolution via service name (e.g., `obiente_dns` if your main stack is named `obiente`).

#### Run with Docker Compose

```bash
# Start the dashboard
docker compose -f docker-compose.dashboard.yml up -d

# View logs
docker compose -f docker-compose.dashboard.yml logs -f

# Stop the dashboard
docker compose -f docker-compose.dashboard.yml down
```

#### Docker Caching Features

The Dockerfile is optimized with multiple caching strategies for maximum build speed:

1. **pnpm Store Caching**: Uses BuildKit cache mounts to persist the pnpm store across builds, dramatically speeding up dependency installation. The `--prefer-offline` flag ensures cached packages are used when available.

2. **Build Artifact Caching**: Multiple BuildKit cache mounts persist build artifacts:
   - **`.nuxt`**: Nuxt build cache (persists across builds)
   - **`.nitro`**: Nitro build cache (persists across builds)
   - **`.vite`**: Vite dependency pre-bundling cache (speeds up dependency processing)
   - **`.nx/cache`**: Nx computation cache (speeds up task execution)

3. **Layer Optimization**: Dependencies are installed in a separate layer that's cached independently from source code changes. This means dependency installation only runs when `package.json` or `pnpm-lock.yaml` changes.

4. **Workspace Support**: Properly handles Nx workspace dependencies and builds them efficiently with cached Nx computation results.

**Expected Performance:**
- **First build**: Normal time (no cache)
- **Subsequent builds with no changes**: 60-80% faster (cache hits for dependencies and build artifacts)
- **Builds with only source code changes**: 40-60% faster (dependencies and most build artifacts cached)
- **Builds with dependency changes**: 20-30% faster (build artifacts still cached)

#### Environment Variables

Configure the dashboard using environment variables in `docker-compose.dashboard.yml`:

```yaml
environment:
  NODE_ENV: production
  PORT: 3000
  NUXT_SESSION_PASSWORD: ${NUXT_SESSION_PASSWORD:-changeme}
  NUXT_PUBLIC_API_HOST: ${API_URL:-http://api.${DOMAIN:-localhost}}
  NUXT_PUBLIC_OIDC_ISSUER: ${ZITADEL_URL:-https://obiente.cloud}
  # ... see docker-compose.dashboard.yml for full list
```

**Important**: Set `NUXT_SESSION_PASSWORD` to a secure random string in production (minimum 32 characters).

#### Multi-Stage Build

The Dockerfile uses a multi-stage build process:

1. **Base Stage**: Sets up Node.js and pnpm
2. **Deps Stage**: Installs dependencies with caching
3. **Builder Stage**: Builds the application
4. **Runner Stage**: Minimal production image with only runtime files

This approach results in a smaller final image and faster builds through layer caching.

### Environment Variables

Ensure all required environment variables are set in production.

## Security

### Content Security Policy

- Configured in `nuxt.config.ts`
- Prevents XSS attacks
- Restricts resource loading

### Authentication

- JWT tokens stored securely
- Automatic token refresh
- Secure HTTP-only cookies

## Future Enhancements

1. **Real-time Updates**: WebSocket integration for live status
2. **PWA Support**: Offline capabilities and app-like experience
3. **Advanced Analytics**: Resource usage dashboards and insights
4. **Mobile App**: React Native companion application
5. **Internationalization**: Multi-language support
6. **Advanced Search**: Full-text search across resources
7. **Team Collaboration**: Real-time collaboration features
