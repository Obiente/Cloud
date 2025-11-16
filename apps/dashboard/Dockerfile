# syntax=docker/dockerfile:1.4
FROM node:22-alpine AS base

# Install pnpm in a single layer
RUN corepack enable && \
    COREPACK_ENABLE_STRICT=0 corepack prepare pnpm@10.20.0 --activate

WORKDIR /app

# Dependencies stage - optimized for caching
FROM base AS deps
# Copy only files needed for dependency resolution (changes infrequently)
# Copy workspace root files first - this layer is cached unless lockfile changes
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json ./
# Copy package.json files from all packages (for workspace dependency resolution)
# Copy entire packages/tools directories but .dockerignore excludes source files
COPY packages ./packages
COPY apps/dashboard/package.json ./apps/dashboard/
COPY tools ./tools

# Install dependencies with aggressive caching
# Use BuildKit cache mounts for pnpm store and Nx cache
# Remove retry logic - if install fails, fail fast
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    --mount=type=cache,id=nx-cache,target=/app/.nx/cache \
    pnpm install --frozen-lockfile --prefer-offline

# Copy config files needed for builds
COPY nx.json tsconfig.base.json tsconfig.json ./

# Copy source code (this layer invalidates when source changes)
# Combine multiple COPYs to reduce layers
COPY apps/dashboard ./apps/dashboard
COPY packages ./packages
COPY tools ./tools

# Set Node options once
ENV NODE_OPTIONS="--max-old-space-size=8192"

# Build workspace dependencies if needed
# Most dependencies are already built, so this is usually a no-op
# Suppress errors since build-deps may not exist or may fail gracefully
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    --mount=type=cache,id=nx-cache,target=/app/.nx/cache \
    pnpm --filter "@obiente/dashboard" build-deps || true

# Build stage - reuse everything from deps
FROM deps AS builder

# Build the application with optimized cache mounts
# Combine nuxt prepare and build into a single RUN to reduce layers
# Use all available cache mounts for maximum speed
# Skip nuxt prepare if .nuxt already exists (from cache)
RUN --mount=type=cache,id=nuxt-build,target=/app/apps/dashboard/.nuxt \
    --mount=type=cache,id=nuxt-nitro,target=/app/apps/dashboard/.nitro \
    --mount=type=cache,id=vite-cache,target=/app/apps/dashboard/node_modules/.vite \
    --mount=type=cache,id=nx-cache,target=/app/.nx/cache \
    --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    cd apps/dashboard && \
    ([ -d .nuxt ] || pnpm exec nuxt prepare) && \
    SKIP_TYPE_CHECK=true pnpm exec nx nuxt:build

# Production image - minimal runtime
FROM node:20-alpine AS runner

# Accept build arguments
ARG DASHBOARD_COMMIT
ARG DASHBOARD_COMMIT_MESSAGE

# Create user in a single layer
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nuxtjs

WORKDIR /app

# Copy only what's needed for runtime (minimize image size)
# Only copy node_modules if runtime dependencies are needed
# Most Nuxt apps bundle everything, so we might not need node_modules
COPY --from=builder --chown=nuxtjs:nodejs /app/apps/dashboard/.output ./apps/dashboard/.output

# Set environment variables
ENV NODE_ENV=production \
    HOST=0.0.0.0 \
    PORT=3000 \
    DASHBOARD_COMMIT=${DASHBOARD_COMMIT} \
    DASHBOARD_COMMIT_MESSAGE=${DASHBOARD_COMMIT_MESSAGE}

# Switch to non-root user
USER nuxtjs

EXPOSE 3000

WORKDIR /app/apps/dashboard
CMD ["node", ".output/server/index.mjs"]

